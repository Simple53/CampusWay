package resolve

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/miekg/dns"
	"github.com/nnu-connect/nnu-connect/client"
)

func TestDomainResourcesRespectLabelBoundaries(t *testing.T) {
	r := NewResolver(nil, "", "", 300, map[string]client.DomainResource{"nnu.edu.cn": {PortMin: 1, PortMax: 65535}}, map[string]net.IP{"www.nnu.edu.cn": net.ParseIP("192.0.2.1"), "evilnnu.edu.cn": net.ParseIP("192.0.2.2")}, false)
	defer r.Close()
	ctx, _, err := r.Resolve(context.Background(), "WWW.NNU.EDU.CN.")
	if err != nil || ctx.Value(ContextKeyDomainResource) == nil {
		t.Fatal("campus subdomain not routed", err)
	}
	ctx, _, err = r.Resolve(context.Background(), "evilnnu.edu.cn")
	if err != nil || ctx.Value(ContextKeyDomainResource) != nil {
		t.Fatal("unrelated domain routed through VPN", err)
	}
}

func TestSecondaryDNSAnswersAreSharedAndCached(t *testing.T) {
	socket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := &dns.Server{PacketConn: socket, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
		requests.Add(1)
		response := new(dns.Msg)
		response.SetReply(q)
		response.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: q.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60}, A: net.ParseIP("192.0.2.5")}}
		_ = w.WriteMsg(response)
	})}
	go server.ActivateAndServe()
	defer server.Shutdown()
	r := NewResolver(nil, "", "", 300, nil, nil, false)
	defer r.Close()
	r.secondaryResolver.Dial = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, socket.LocalAddr().String())
	}
	for i := 0; i < 3; i++ {
		_, ip, err := r.Resolve(context.Background(), "cached.example.")
		if err != nil || ip.String() != "192.0.2.5" {
			t.Fatalf("resolve %d: %v, %v", i, ip, err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("DNS requests = %d, want 1", got)
	}
}
