package dnsutil

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/miekg/dns"
)

type fixedResolver []net.IP

func (r fixedResolver) LookupIP(context.Context, string, string) ([]net.IP, error) { return r, nil }

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFakeIPIsReplacedBeforeExportingToVPN(t *testing.T) {
	previous := dohClient
	t.Cleanup(func() { dohClient = previous; realCache.Flush() })
	calls := 0
	dohClient = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		wire, _ := io.ReadAll(req.Body)
		var query dns.Msg
		if err := query.Unpack(wire); err != nil {
			t.Fatal(err)
		}
		reply := new(dns.Msg)
		reply.SetReply(&query)
		reply.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: query.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60}, A: net.ParseIP("202.119.104.1")}}
		data, _ := reply.Pack()
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}
	for i := 0; i < 2; i++ {
		ips, err := LookupIP(context.Background(), fixedResolver{net.ParseIP("198.18.0.38")}, "ip4", "vpn.example.edu")
		if err != nil || len(ips) != 1 || ips[0].String() != "202.119.104.1" {
			t.Fatalf("ips=%v err=%v", ips, err)
		}
	}
	if calls != 1 {
		t.Fatalf("real DNS answer not cached: %d queries", calls)
	}
	ips, err := LookupIP(context.Background(), fixedResolver{net.ParseIP("198.18.0.1"), net.ParseIP("192.0.2.1")}, "ip4", "mixed.example.edu")
	if err != nil || len(ips) != 1 || calls != 1 {
		t.Fatal("real DNS answer unnecessarily queried externally")
	}
	for _, addr := range []string{"198.18.0.1", "198.19.255.255"} {
		if !IsFakeIP(net.ParseIP(addr)) {
			t.Fatal("fake range missed", addr)
		}
	}
	if IsFakeIP(net.ParseIP("198.20.0.1")) {
		t.Fatal("real IP rejected")
	}
}
