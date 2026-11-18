package atrust

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nnu-connect/nnu-connect/log"
)

func (c *Client) getIP() error {
	addr := c.BestNodes[c.MajorNodeGroup]
	if addr == "" && c.MajorNodeGroup == "" {
		for _, node := range c.BestNodes {
			addr = node
			break
		}
	}
	if addr == "" {
		return fmt.Errorf("no advertised node completed TLS for virtual IP setup")
	}

	if c.SID == "" {
		return fmt.Errorf("missing authenticated session ID")
	}
	ctx, cancel := context.WithTimeout(c.lifecycleCtx, 10*time.Second)
	defer cancel()
	conn, err := c.underlayDialer.DialTLSContext(ctx, "tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return fmt.Errorf("establish TLS to virtual IP node %s: %w", addr, err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	payload, err := json.Marshal(authRequestSID{Sid: c.SID})
	if err != nil {
		return err
	}
	if _, err := conn.Write(wrapAuthReqData(payload, 1)); err != nil {
		return fmt.Errorf("send virtual IP request: %w", err)
	}
	ips, err := readTunnelAuth(conn)
	if err != nil {
		return fmt.Errorf("read virtual IP response: %w", err)
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			c.ip = ip.To4()
			log.Printf("Received IP: %s", c.ip.String())
			return nil
		}
	}
	return fmt.Errorf("gateway did not return a virtual IPv4 address")
}
