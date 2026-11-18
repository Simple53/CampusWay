package atrust

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nnu-connect/nnu-connect/internal/underlay"
)

func TestProbeNodesFallsBackWhenTCPClosesDuringTLS(t *testing.T) {
	tcpOnly, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcpOnly.Close()
	go func() {
		conn, err := tcpOnly.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()
	valid := httptest.NewTLSServer(http.NotFoundHandler())
	defer valid.Close()
	validAddr := valid.Listener.Addr().String()

	c := NewClient("", "", "", "")
	defer c.Close()
	c.underlayDialer = underlay.New(validAddr, underlay.Options{})
	c.NodeGroups = map[string]nodeGroup{
		"main": {wan: []string{tcpOnly.Addr().String()}, lan: []string{validAddr}},
	}
	if got := c.probeNodes(context.Background())["main"]; got != validAddr {
		t.Fatalf("selected node = %q, want %q", got, validAddr)
	}
}

func TestGetIPWithVariableLengthSession(t *testing.T) {
	const sid = "short-test-session-id"
	certServer := httptest.NewTLSServer(http.NotFoundHandler())
	config := certServer.TLS
	certServer.Close()
	node, err := tls.Listen("tcp4", "127.0.0.1:0", config)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()
	go func() {
		conn, err := node.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var header [7]byte
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			return
		}
		payload := make([]byte, binary.BigEndian.Uint16(header[5:]))
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}
		var request authRequestSID
		if err := json.Unmarshal(payload, &request); err != nil || request.Sid != sid {
			return
		}
		var vipRequest [10]byte
		if _, err := io.ReadFull(conn, vipRequest[:]); err != nil {
			return
		}
		if vipRequest != [10]byte{5, 4, 0, 1} {
			return
		}
		response := []byte{5, 0xd0, 0x53, 0, 0, 10}
		response = append(response, `{"code":0}`...)
		response = append(response, 5, 0, 0, 1, 192, 0, 2, 1, 0, 0)
		_, _ = conn.Write(response)
	}()

	addr := node.Addr().String()
	c := NewClient("", sid, "", "")
	defer c.Close()
	c.underlayDialer = underlay.New(addr, underlay.Options{})
	c.BestNodes = map[string]string{"test": addr}
	c.MajorNodeGroup = "test"
	if err := c.getIP(); err != nil {
		t.Fatal(err)
	}
	if got := c.ip.String(); got != "192.0.2.1" {
		t.Fatalf("virtual IP = %s", got)
	}
}
