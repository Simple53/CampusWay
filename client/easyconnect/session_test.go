package easyconnect

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKeepAliveSendsLiveRequest(t *testing.T) {
	called := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/por/update_session.csp" || r.Header.Get("Cookie") != "TWFID=0123456789abcdef" {
			t.Errorf("unexpected keepalive request %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `<Auth><Message>success</Message><ErrorCode>1</ErrorCode></Auth>`)
	}))
	defer server.Close()
	c := NewClient(strings.TrimPrefix(server.URL, "https://"), "", "", "", tls.Certificate{}, "0123456789abcdef", false, false, false)
	c.httpClient = server.Client()
	defer c.Close()
	if err := c.keepSessionAlive(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("keepalive never reached the server")
	}
	c.Close()
	if err := c.keepSessionAlive(); err == nil {
		t.Fatal("closed session kept running")
	}
}

type shortReader struct{ io.Reader }

func (r shortReader) Read(p []byte) (int, error) {
	if len(p) > 3 {
		p = p[:3]
	}
	return r.Reader.Read(p)
}

func TestIPPacketsSurviveSplitAndCoalescedTLSReads(t *testing.T) {
	packet := append([]byte{0x45, 0, 0, 24}, make([]byte, 20)...)
	reader := shortReader{bytes.NewReader(append(append([]byte{}, packet...), packet...))}
	buf := make([]byte, 1500)
	for i := 0; i < 2; i++ {
		n, err := readIPPacket(reader, buf)
		if err != nil || n != len(packet) || !bytes.Equal(buf[:n], packet) {
			t.Fatalf("packet %d: n=%d err=%v", i, n, err)
		}
	}
	if _, err := readIPPacket(bytes.NewReader([]byte{0x45, 0, 0, 1}), buf); err == nil {
		t.Fatal("invalid packet accepted")
	}
}

func TestOptionalMultilineConfigDoesNotPanic(t *testing.T) {
	for _, conf := range []string{`<Conf/>`, `<Conf><Mline list=""/></Conf>`, `<Other/>`} {
		if err := (&Client{}).parseLineListFromConfig(conf); err == nil {
			t.Fatal("missing line accepted")
		}
	}
}
