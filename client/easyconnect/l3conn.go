package easyconnect

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/nnu-connect/nnu-connect/log"
)

type L3Conn struct {
	easyConnectClient *Client

	sendConn     io.WriteCloser
	sendLock     sync.Mutex
	sendErrCount int

	recvConn     io.ReadCloser
	recvLock     sync.Mutex
	recvErrCount int
}

// TLS is a byte stream: one Read may contain part of an IP packet or several.
func readIPPacket(reader io.Reader, p []byte) (int, error) {
	if len(p) < 4 {
		return 0, io.ErrShortBuffer
	}
	if _, err := io.ReadFull(reader, p[:4]); err != nil {
		return 0, err
	}
	size := int(binary.BigEndian.Uint16(p[2:4]))
	if p[0]>>4 != 4 || size < 20 || size < int(p[0]&15)*4 {
		return 0, fmt.Errorf("invalid VPN IPv4 packet length %d", size)
	}
	if size > len(p) {
		return 0, io.ErrShortBuffer
	}
	_, err := io.ReadFull(reader, p[4:size])
	return size, err
}

func (c *L3Conn) Read(p []byte) (int, error) {
	c.recvLock.Lock()
	defer c.recvLock.Unlock()
	for {
		n, err := readIPPacket(c.recvConn, p)
		if err == nil {
			c.recvErrCount = 0
			return n, nil
		}
		if c.easyConnectClient.lifecycleCtx.Err() != nil || c.recvErrCount >= 5 {
			return 0, err
		}
		c.recvErrCount++
		log.Printf("Receive tunnel failed; reconnecting: %v", err)
		_ = c.recvConn.Close()
		next, err := c.easyConnectClient.RecvConn()
		if err != nil {
			return 0, err
		}
		c.recvConn = next
	}
}

func (c *L3Conn) Write(p []byte) (int, error) {
	c.sendLock.Lock()
	defer c.sendLock.Unlock()
	for {
		n, err := c.sendConn.Write(p)
		if err == nil {
			c.sendErrCount = 0
			return n, nil
		}
		if c.easyConnectClient.lifecycleCtx.Err() != nil || c.sendErrCount >= 5 {
			return n, err
		}
		c.sendErrCount++
		log.Printf("Send tunnel failed; reconnecting: %v", err)
		_ = c.sendConn.Close()
		next, err := c.easyConnectClient.SendConn()
		if err != nil {
			return n, err
		}
		c.sendConn = next
	}
}

func (c *L3Conn) Close() error {
	if c.sendConn != nil {
		_ = c.sendConn.Close()
	}
	if c.recvConn != nil {
		_ = c.recvConn.Close()
	}
	return nil
}

func (c *Client) NewL3Conn() (io.ReadWriteCloser, error) {
	conn := &L3Conn{
		easyConnectClient: c,
		sendErrCount:      0,
		recvErrCount:      0,
	}

	var err error
	conn.sendConn, err = c.SendConn()
	if err != nil {
		log.Printf("Error occurred while creating sendConn: %v", err)
		return nil, err
	}

	conn.recvConn, err = c.RecvConn()
	if err != nil {
		log.Printf("Error occurred while creating recvConn: %v", err)
		_ = conn.sendConn.Close()
		return nil, err
	}
	return conn, nil
}
