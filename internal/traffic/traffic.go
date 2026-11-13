// Package traffic counts payload bytes on outbound local-proxy connections only.
// It deliberately excludes authentication, device-wide traffic and packet inspection.
package traffic

import (
	"net"
	"sync/atomic"
)

var uploaded, downloaded atomic.Uint64

func Totals() (uint64, uint64) { return uploaded.Load(), downloaded.Load() }
func Wrap(c net.Conn) net.Conn { return &countedConn{Conn: c} }

type countedConn struct{ net.Conn }

func (c *countedConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	downloaded.Add(uint64(n))
	return n, e
}
func (c *countedConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	uploaded.Add(uint64(n))
	return n, e
}
