package handlers

import (
	"fmt"
	"strings"

	"github.com/psvmcc/disconter/pkg/discovery"

	"github.com/VictoriaMetrics/metrics"
	"github.com/miekg/dns"
)

var Debug bool

func HandleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			if Debug {
				fmt.Println(
					"[DEBUG] DNS query record:",
					strings.Split(w.RemoteAddr().String(), ":")[0],
					strings.ToLower(q.Name),
					dns.TypeToString[q.Qtype],
				)
			}
			queryCounter := fmt.Sprintf(`disconter_dns_queries{type=%q}`, dns.TypeToString[q.Qtype])
			metrics.GetOrCreateCounter(queryCounter).Inc()
			metrics.GetOrCreateCounter("disconter_dns_queries_total").Inc()

			if (dns.TypeToString[q.Qtype] == "A" || dns.TypeToString[q.Qtype] == "ANY") && strings.HasSuffix(q.Name, "container.disconter.") {
				for _, c := range discovery.ServiceContainers {
					if q.Name == fmt.Sprintf("%s.container.disconter.", c.Name) {
						rr, err := dns.NewRR(fmt.Sprintf("%s %d A %s", q.Name, c.DisconterService.TTL, c.IP))
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
					}
				}
			}
			if (dns.TypeToString[q.Qtype] == "A" || dns.TypeToString[q.Qtype] == "ANY") && strings.HasSuffix(q.Name, "service.disconter.") {
				for _, c := range discovery.ServiceContainers {
					if q.Name == fmt.Sprintf("%s.service.disconter.", c.DisconterService.Name) {
						rr, err := dns.NewRR(fmt.Sprintf("%s %d A %s", q.Name, c.DisconterService.TTL, c.IP))
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
					}
				}
			}
			if dns.TypeToString[q.Qtype] == "SRV" && strings.HasSuffix(q.Name, "service.disconter.") {
				for _, c := range discovery.ServiceContainers {
					if q.Name == fmt.Sprintf("%s.service.disconter.", c.DisconterService.Name) || q.Name == fmt.Sprintf("_%s._tcp.service.disconter.", c.DisconterService.Name) {
						rr, err := dns.NewRR(fmt.Sprintf("%s %d SRV %d %d %d %s.container.disconter", q.Name, c.DisconterService.TTL, c.DisconterService.Priority, c.DisconterService.Weight, c.DisconterService.Port, c.Name))
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
						rrA, err := dns.NewRR(fmt.Sprintf("%s.container.disconter %d A %s", c.Name, c.DisconterService.TTL, c.IP))
						if err == nil {
							m.Extra = append(m.Extra, rrA)
						}
					}
				}
			}
		}
	}
	err := w.WriteMsg(m)
	if err != nil {
		fmt.Println("[ERROR] DNS reply:", err)
		metrics.GetOrCreateCounter("disconter_dns_errors").Inc()
	}
}
