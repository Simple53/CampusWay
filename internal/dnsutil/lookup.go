package dnsutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

var fakeRange = &net.IPNet{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)}

func IsFakeIP(ip net.IP) bool { return fakeRange.Contains(ip) }

type IPResolver interface {
	LookupIP(context.Context, string, string) ([]net.IP, error)
}

var realCache = cache.New(5*time.Minute, 0)
var dohClient = &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}

// LookupIP preserves the configured resolver, but never exports another proxy's
// synthetic addresses into a campus tunnel. Literal DoH endpoints avoid DNS loops.
func LookupIP(ctx context.Context, resolver IPResolver, network, host string) ([]net.IP, error) {
	ips, err := resolver.LookupIP(ctx, network, host)
	if err != nil {
		return nil, err
	}
	var real []net.IP
	for _, ip := range ips {
		if !IsFakeIP(ip) {
			real = append(real, ip)
		}
	}
	if len(real) > 0 {
		return real, nil
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses for %s", host)
	}
	key := network + ":" + host
	if cached, ok := realCache.Get(key); ok {
		return cached.([]net.IP), nil
	}
	qtype := uint16(dns.TypeA)
	if network == "ip6" {
		qtype = dns.TypeAAAA
	}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(host), qtype)
	wire, err := msg.Pack()
	if err != nil {
		return nil, err
	}
	for _, endpoint := range []string{"https://223.5.5.5/dns-query", "https://1.1.1.1/dns-query"} {
		real, err = lookupHTTPS(ctx, endpoint, wire, msg.Id, qtype)
		if err == nil && len(real) > 0 {
			realCache.SetDefault(key, real)
			return real, nil
		}
	}
	return nil, fmt.Errorf("DNS for %s returned fake-IP; real DNS lookup failed: %w", host, err)
}

func lookupHTTPS(ctx context.Context, endpoint string, wire []byte, id, qtype uint16) ([]net.IP, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(wire))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	resp, err := dohClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DNS HTTPS status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, err
	}
	var reply dns.Msg
	if err := reply.Unpack(data); err != nil {
		return nil, err
	}
	if !reply.Response || reply.Id != id || reply.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("invalid DNS HTTPS reply")
	}
	var ips []net.IP
	for _, answer := range reply.Answer {
		var ip net.IP
		switch record := answer.(type) {
		case *dns.A:
			if qtype == dns.TypeA {
				ip = record.A
			}
		case *dns.AAAA:
			if qtype == dns.TypeAAAA {
				ip = record.AAAA
			}
		}
		if ip != nil && !IsFakeIP(ip) {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("DNS HTTPS reply contains no real addresses")
	}
	return ips, nil
}
