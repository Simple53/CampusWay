package engine

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/nnu-connect/nnu-connect/client"
	"github.com/nnu-connect/nnu-connect/internal/hook_func"
	"github.com/nnu-connect/nnu-connect/stack/gvisor"
)

type pendingClient struct {
	client.Client
	conn net.Conn
	t    *testing.T
}

func (c pendingClient) IP() (net.IP, error)                    { return net.ParseIP("192.0.2.7").To4(), nil }
func (c pendingClient) NewL3Conn() (io.ReadWriteCloser, error) { return c.conn, nil }
func (c pendingClient) Close() {
	if !hook_func.IsTerminal() {
		c.t.Error("VPN sockets closed before shutdown was marked")
	}
	_ = c.conn.Close()
}

func TestDisconnectWhileDataTunnelReadIsPending(t *testing.T) {
	hook_func.BeginTerminalSession()
	local, remote := net.Pipe()
	defer remote.Close()
	c := pendingClient{conn: local, t: t}
	s, err := gvisor.NewStack(c)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { s.Run(); close(done) }()
	ctx, cancel := context.WithCancel(context.Background())
	finish := cancelClientOnContext(ctx, c)
	cancel()
	finish()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("data tunnel reader did not stop")
	}
}
