package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/psvmcc/disconter/pkg/discovery"
	"github.com/psvmcc/disconter/pkg/envflags"
	"github.com/psvmcc/disconter/pkg/handlers"

	"github.com/VictoriaMetrics/metrics"
	"github.com/miekg/dns"
)

const (
	appName = "disconter"
)

var (
	version    string
	commit     string
	appVersion = appName + " " + version + "/" + commit

	metricsListen string
	dnsListen     string
	ver           bool
	metricsGo     bool
	debug         bool
	dockerRefresh int
)

func main() {
	flag.StringVar(&metricsListen, "bind.metrics", "0.0.0.0:9553", "Bind the HTTP metrics server")
	flag.StringVar(&dnsListen, "bind.dns", "0.0.0.0:53535", "Bind the DNS server")
	flag.StringVar(&discovery.DockerSocket, "docker.socket", "/var/run/docker.sock", "Docker(Podman) socket path")
	flag.IntVar(&dockerRefresh, "docker.refresh.interval", 100, "Container events refresh interval in milliseconds")
	flag.BoolVar(&metricsGo, "metrics.go", true, "Extend Golang metrics")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&ver, "v", false, "Print version")
	err := envflags.SetFlagsFromEnvironment()
	if err != nil {
		panic(err)
	}
	flag.Parse()
	if ver {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	go func() {
		versionMetric := fmt.Sprintf(`disconter_info{version=%q,commit=%q}`, version, commit)
		metrics.GetOrCreateCounter(versionMetric).Set(0)
		http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
			metrics.WritePrometheus(w, true)
		})

		server := &http.Server{
			Addr:              metricsListen,
			ReadHeaderTimeout: 3 * time.Second,
		}
		log.Fatal(server.ListenAndServe())
	}()

	go func() {
		handlers.Debug = debug
		dns.HandleFunc("disconter.", handlers.HandleDNSRequest)
		server := &dns.Server{Addr: dnsListen, Net: "udp"}
		log.Fatal(server.ListenAndServe())
	}()

	discovery.Debug = debug

	eventChan := make(chan string)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	_, err = discovery.ListContainers(discovery.DockerSocket)
	if err != nil {
		panic(err)
	}

	go discovery.DockerEventListener(discovery.DockerSocket, eventChan)

	for {
		select {
		case <-signalChan:
			fmt.Println("Program terminated.")
			return
		case event, ok := <-eventChan:
			if ok {
				fmt.Println("Received Docker event:", event)
			} else {
				fmt.Println("Event channel closed. Exiting.")
			}
		}
	}
}
