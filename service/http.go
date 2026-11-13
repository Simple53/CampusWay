package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nnu-connect/nnu-connect/dial"
	"github.com/nnu-connect/nnu-connect/internal/hook_func"
	"github.com/nnu-connect/nnu-connect/internal/traffic"
	"github.com/nnu-connect/nnu-connect/log"
)

// The MIT License (MIT)
//
// Copyright (c) 2016 Ian Denhardt <ian@zenhack.net>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

func ServeHTTP(bindAddr string, dialer *dial.Dialer) {
	if err := StartHTTP(bindAddr, dialer); err != nil {
		log.Println(err)
	}
}

func StartHTTP(bindAddr string, dialer *dial.Dialer) error {
	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return err
	}
	var connections sync.Map
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
				c, err := dialer.Dial(ctx, net, addr)
				if err != nil {
					return nil, err
				}
				return traffic.Wrap(c), nil
			},
		},
		// We must pass redirect response to browser
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "CONNECT" {
			serverConn, err := dialer.Dial(context.Background(), "tcp", req.Host)
			if err != nil {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(err.Error() + "\n"))
				return
			}

			serverConn = traffic.Wrap(serverConn)
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				_ = serverConn.Close()
				w.WriteHeader(500)
				_, _ = w.Write([]byte("Failed cast to hijacker\n"))
				return
			}

			w.WriteHeader(200)

			clientConn, bio, err := hijacker.Hijack()
			if err != nil {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(err.Error() + "\n"))
				_ = serverConn.Close()
				return
			}

			connections.Store(clientConn, serverConn)
			closePair := func() { clientConn.Close(); serverConn.Close(); connections.Delete(clientConn) }
			go func() { defer closePair(); _, _ = io.Copy(serverConn, bio) }()
			go func() { defer closePair(); _, _ = io.Copy(clientConn, serverConn) }()
		} else {
			req.RequestURI = ""

			resp, err := client.Do(req)
			if err != nil {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(err.Error() + "\n"))
				return
			}

			defer resp.Body.Close()
			hdr := w.Header()
			for k, v := range resp.Header {
				hdr[k] = v
			}

			w.WriteHeader(resp.StatusCode)

			_, _ = io.Copy(w, resp.Body)
		}
	})

	log.Printf("HTTP server listening on %s", bindAddr)

	server := &http.Server{Addr: bindAddr, Handler: handlerFunc}

	hook_func.RegisterTerminalFunc("CloseHTTPListener", func(ctx context.Context) error {
		log.Println("Closing HTTP listener...")
		connections.Range(func(k, v any) bool { k.(net.Conn).Close(); v.(net.Conn).Close(); return true })
		client.CloseIdleConnections()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("close HTTP listener failed: %w", err)
		}
		return nil
	})

	go func() {
		if err := server.Serve(listener); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Println("HTTP server closed")
			} else {
				log.Println("HTTP listen failed: " + err.Error())
			}
		}
	}()
	return nil
}
