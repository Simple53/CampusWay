package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DiagnosticTarget is one row of the home-page latency chart.
type DiagnosticTarget struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	OK      bool   `json:"ok"`
	Skipped bool   `json:"skipped,omitempty"` // not tested because the tunnel is down
	MS      int64  `json:"ms"`
	Err     string `json:"err,omitempty"`
}

// DiagnosticReport feeds the diagnostics widget. No credentials are involved.
type DiagnosticReport struct {
	Gateway   DiagnosticTarget   `json:"gateway"`   // direct TCP probe of the selected gateway
	Targets   []DiagnosticTarget `json:"targets"`   // sites tested through the local HTTP proxy
	LocalIP   string             `json:"localIP"`   // outbound LAN address
	GatewayIP string             `json:"gatewayIP"` // what the local resolver returned for the gateway
	FakeIP    bool               `json:"fakeIP"`    // resolver returned a Clash-style fake IP
	Connected bool               `json:"connected"`
}

// Sites tested by default. The first three are common campus portals; users add
// more through each connection's "additional domains" field.
var defaultDiagnosticSites = []struct{ name, url string }{
	{"ehall", "https://ehallapp.nnu.edu.cn"},
	{"zgzj", "http://zgzj.njnu.edu.cn"},
	{"hsnnu", "https://hsnnu.nnu.edu.cn"},
}

// RunDiagnostics probes the gateway directly and every target through the
// tunnel (when connected). Probes run concurrently so the whole test takes
// about one timeout, not one per site.
func (s *AppServer) RunDiagnostics() (DiagnosticReport, error) {
	s.mu.Lock()
	var prof *Profile
	for _, candidate := range s.config.Profiles {
		if candidate.ID == s.active || (prof == nil && candidate.ID == s.config.Selected) {
			cp := candidate
			prof = &cp
		}
	}
	connected := s.state == "connected"
	s.mu.Unlock()

	report := DiagnosticReport{Connected: connected, Targets: []DiagnosticTarget{}, LocalIP: outboundIP()}

	httpPort := 1081
	if prof != nil {
		if prof.Options.HTTPPort > 0 {
			httpPort = prof.Options.HTTPPort
		}
		report.Gateway = probeGateway(prof.Server, prof.Port)
		report.GatewayIP, report.FakeIP = resolveGateway(prof.Server)
	}

	sites := append([]struct{ name, url string }{}, defaultDiagnosticSites...)
	if prof != nil {
		for _, domain := range strings.FieldsFunc(prof.CustomDomains, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == ' ' }) {
			if len(sites) >= 8 {
				break
			}
			if strings.ContainsAny(domain, "/\\@?# ") {
				continue
			}
			sites = append(sites, struct{ name, url string }{domain, "https://" + domain})
		}
	}

	report.Targets = make([]DiagnosticTarget, len(sites))
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	client := &http.Client{Timeout: 6 * time.Second, Transport: &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   true,
	}}
	var wg sync.WaitGroup
	for i, site := range sites {
		wg.Add(1)
		go func(i int, name, target string) {
			defer wg.Done()
			row := DiagnosticTarget{Name: name, URL: target}
			if !connected {
				row.Skipped = true
				report.Targets[i] = row
				return
			}
			start := time.Now()
			req, err := http.NewRequestWithContext(context.Background(), "GET", target, nil)
			if err == nil {
				req.Header.Set("User-Agent", "Mozilla/5.0 CampusWay")
				var resp *http.Response
				if resp, err = client.Do(req); err == nil {
					_ = resp.Body.Close()
				}
			}
			row.MS = time.Since(start).Milliseconds()
			row.OK = err == nil
			if err != nil {
				row.Err = shortError(err)
			}
			report.Targets[i] = row
		}(i, site.name, site.url)
	}
	wg.Wait()
	return report, nil
}

// probeGateway opens a plain TCP connection straight to the gateway, bypassing
// any proxy, so a failure points at the network path rather than the tunnel.
func probeGateway(server string, port int) DiagnosticTarget {
	row := DiagnosticTarget{Name: server, URL: net.JoinHostPort(server, strconv.Itoa(port))}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", row.URL, 5*time.Second)
	row.MS = time.Since(start).Milliseconds()
	if err != nil {
		row.Err = shortError(err)
		return row
	}
	_ = conn.Close()
	row.OK = true
	return row
}

// resolveGateway returns the address the local resolver gives for the gateway
// and whether it lies in the fake-IP range used by Clash/Mihomo DNS hijacking.
func resolveGateway(server string) (string, bool) {
	ips, err := net.LookupIP(server)
	if err != nil || len(ips) == 0 {
		return "", false
	}
	return ips[0].String(), fakeIPNet.Contains(ips[0])
}

func outboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// shortError keeps the chart labels readable and avoids echoing long URLs.
func shortError(err error) string {
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "timeout"), strings.Contains(text, "deadline"):
		return "timeout"
	case strings.Contains(text, "refused"):
		return "refused"
	case strings.Contains(text, "eof"), strings.Contains(text, "reset"):
		return "closed"
	case strings.Contains(text, "no such host"):
		return "dns"
	}
	return "error"
}
