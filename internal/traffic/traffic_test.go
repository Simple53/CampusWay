package traffic

import (
	"io"
	"net"
	"testing"
)

func TestConnectionCountsBothDirections(t *testing.T) {
	up, down := Totals()
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	c := Wrap(a)
	done := make(chan struct{})
	go func() { defer close(done); buf := make([]byte, 3); io.ReadFull(b, buf); b.Write([]byte("hello")) }()
	if n, e := c.Write([]byte("abc")); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	buf := make([]byte, 5)
	if _, e := io.ReadFull(c, buf); e != nil {
		t.Fatal(e)
	}
	<-done
	u, d := Totals()
	if u-up != 3 || d-down != 5 {
		t.Fatal(u-up, d-down)
	}
}
