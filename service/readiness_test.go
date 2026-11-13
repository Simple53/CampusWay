package service

import (
	"net"
	"testing"
)

func TestBusyProxyPortsReturnErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().String()
	if err := StartHTTP(address, nil); err == nil {
		t.Fatal("HTTP reported ready on occupied port")
	}
	if err := StartSocks5(address, nil, nil, "", ""); err == nil {
		t.Fatal("SOCKS reported ready on occupied port")
	}
}
