package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nnu-connect/nnu-connect/engine"
)

// Campus traffic goes to the VPN HTTP proxy; other browser traffic keeps the
// previous manual proxy (including Clash), or the system's normal TUN route.
func browserPAC(p Profile, fallback string) string {
	domains := append([]string{}, engine.CampusDomains(p.Server)...)
	for _, domain := range strings.FieldsFunc(p.CustomDomains, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == ' ' }) {
		domain = strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(domain), "*"), ".")
		if domain != "" && !strings.ContainsAny(domain, "/\\@?#:\"'") {
			domains = append(domains, domain)
		}
	}
	encoded, _ := json.Marshal(domains)
	encodedFallback, _ := json.Marshal(fallback)
	return fmt.Sprintf(`function FindProxyForURL(url, host) {
 host = host.toLowerCase().replace(/\.$/, "");
 if (host === "localhost" || host === "127.0.0.1" || host === "::1") return "DIRECT";
 var domains = %s;
 for (var i = 0; i < domains.length; i++) {
  if (host === domains[i] || dnsDomainIs(host, "." + domains[i])) return "PROXY 127.0.0.1:%d";
 }
 var fallback = %s;
 if (fallback.indexOf("=") >= 0) {
  var protocol = url.substring(0, url.indexOf(":")), socks = "";
  var parts = fallback.split(";");
  for (var j = 0; j < parts.length; j++) {
   var rule = parts[j].trim().split("=");
   if (rule[0] === protocol) return "PROXY " + rule[1];
   if (rule[0] === "socks") socks = "SOCKS5 " + rule[1];
  }
  return socks || "DIRECT";
 }
 return fallback;
}`, encoded, p.Options.HTTPPort, encodedFallback)
}

func serveBrowserPAC(p Profile, fallback string) (string, func(), error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	pac := browserPAC(p, fallback)
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(pac))
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go server.Serve(listener)
	return "http://" + listener.Addr().String() + "/proxy.pac", func() { _ = server.Close() }, nil
}
