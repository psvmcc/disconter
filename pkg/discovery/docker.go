package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

var ServiceContainers []ContainerInfo
var Debug bool
var DockerSocket string

type DockerEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		Attributes struct {
			Name             string `json:"name"`
			DisconterService string `json:"disconter.service"`
		} `json:"Attributes"`
	} `json:"Actor"`
}

type ContainerInfo struct {
	ID               string
	Name             string
	IP               string
	DisconterService struct {
		Name     string
		Priority uint16
		Weight   uint16
		Port     uint16
		TTL      uint16
	}
}

func DockerEventListener(socketPath string, eventChan chan<- string) {
	done := make(chan bool)

	for {
		var err error
		var conn net.Conn
		conn, err = connectToSocket(socketPath)
		if err != nil {
			metrics.GetOrCreateCounter(`disconter_discovery_errors{type="connectToSocket"}`).Inc()
			continue
		}
		ServiceContainers, err = ListContainers(socketPath)
		if err != nil {
			fmt.Println("Error:", err)
			metrics.GetOrCreateCounter(`disconter_discovery_errors{type="ListContainers"}`).Inc()
		}
		go listenForEvents(conn, eventChan, done)

		<-done
	}
}

func connectToSocket(socketPath string) (net.Conn, error) {
	for {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			return conn, nil
		}
		fmt.Printf("Error connecting to Docker socket: %v. Retrying in 1 second...\n", err)
		time.Sleep(time.Second)
	}
}

func listenForEvents(conn net.Conn, eventChan chan<- string, done chan<- bool) {
	var err error
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Printf("Error: %v", err)
			metrics.GetOrCreateCounter(`disconter_discovery_errors{type="listenForEvents"}`).Inc()
		}
		done <- true
	}()

	req, err := http.NewRequest("GET", "http://localhost/events?filter={\"type\":[\"container\"]}", http.NoBody)
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		metrics.GetOrCreateCounter(`disconter_discovery_errors{type="listenForEvents"}`).Inc()
		return
	}

	client := http.Client{Transport: &http.Transport{Dial: func(_, _ string) (net.Conn, error) { return conn, nil }}}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %v\n", err)
		metrics.GetOrCreateCounter(`disconter_discovery_errors{type="listenForEvents"}`).Inc()
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		eventData := scanner.Text()
		var event DockerEvent
		err = json.Unmarshal([]byte(eventData), &event)
		if err != nil {
			fmt.Printf("Error parsing Docker event: %v\n", err)
			metrics.GetOrCreateCounter(`disconter_discovery_errors{type="listenForEvents"}`).Inc()
			continue
		}

		if event.Type == "container" && event.Actor.Attributes.DisconterService != "" && (event.Action == "start" || event.Action == "die") {
			ServiceContainers, err = ListContainers(DockerSocket)
			if err != nil {
				fmt.Println("Error:", err)
				metrics.GetOrCreateCounter(`disconter_discovery_errors{type="ListContainers"}`).Inc()
			}
			eventChan <- fmt.Sprint(event)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		metrics.GetOrCreateCounter(`disconter_discovery_errors{type="listenForEvents"}`).Inc()
	}
}

func ListContainers(socket string) (containers []ContainerInfo, err error) {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
		Timeout: time.Second * 5,
	}

	resp, err := client.Get("http://localhost/containers/json")
	if err != nil {
		return nil, fmt.Errorf("failed connection to docker socket: %w", err)
	}

	defer resp.Body.Close() // nolint

	if resp.StatusCode != http.StatusOK {
		e := struct {
			Message string `json:"message"`
		}{}

		if err = json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("failed to parse error from docker daemon: %w", err)
		}

		return nil, fmt.Errorf("unexpected error from docker daemon: %s", e.Message)
	}

	var response []struct {
		ID              string `json:"Id"`
		Name            string
		State           string
		Labels          map[string]string
		Created         int64
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string
			}
		}
		Names []string
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response from docker daemon: %w", err)
	}

	containers = make([]ContainerInfo, len(response))

	for i, resp := range response {
		if resp.State != "running" {
			continue
		}

		if v, ok := resp.Labels["disconter.service"]; !ok {
			if v == "" {
				continue
			}
		}
		c := ContainerInfo{}

		c.ID = resp.ID
		c.Name = strings.TrimPrefix(resp.Names[0], "/")
		c.DisconterService.Name = resp.Labels["disconter.service"]
		c.DisconterService.Priority = 1
		c.DisconterService.Weight = 1
		c.DisconterService.Port = 80
		c.DisconterService.TTL = 0

		for _, v := range resp.NetworkSettings.Networks {
			if v.IPAddress != "" {
				c.IP = v.IPAddress
			}
		}

		if c.IP == "" {
			fmt.Printf("[DEBUG] skip container %s no ip on defined networks\n", c.Name)
			continue
		}

		if resp.Labels["disconter.service.priority"] != "" {
			priority, err := strconv.Atoi(resp.Labels["disconter.service.priority"])
			if err == nil {
				c.DisconterService.Priority = uint16(priority)
			}
		}

		if resp.Labels["disconter.service.weight"] != "" {
			weight, err := strconv.Atoi(resp.Labels["disconter.service.weight"])
			if err == nil {
				c.DisconterService.Weight = uint16(weight)
			}
		}

		if resp.Labels["disconter.service.port"] != "" {
			port, err := strconv.Atoi(resp.Labels["disconter.service.port"])
			if err == nil {
				c.DisconterService.Port = uint16(port)
			}
		}

		if resp.Labels["disconter.service.ttl"] != "" {
			ttl, err := strconv.Atoi(resp.Labels["disconter.service.ttl"])
			if err == nil {
				c.DisconterService.TTL = uint16(ttl)
			}
		}
		containers[i] = c
	}

	return containers, nil
}
