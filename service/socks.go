package service

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/nnu-connect/nnu-connect/dial"
	"github.com/nnu-connect/nnu-connect/internal/hook_func"
	"github.com/nnu-connect/nnu-connect/internal/traffic"
	"github.com/nnu-connect/nnu-connect/log"
	"github.com/nnu-connect/nnu-connect/resolve"
	"github.com/things-go/go-socks5"
)

func ServeSocks5(bindAddr string, dialer *dial.Dialer, resolver *resolve.Resolver, user string, password string) {
	if err := StartSocks5(bindAddr, dialer, resolver, user, password); err != nil {
		log.Println(err)
	}
}

func StartSocks5(bindAddr string, dialer *dial.Dialer, resolver *resolve.Resolver, user string, password string) error {
	var authMethods []socks5.Authenticator
	if user != "" && password != "" {
		authMethods = append(authMethods, socks5.UserPassAuthenticator{
			Credentials: socks5.StaticCredentials{user: password},
		})

		log.Println("Neither traffic nor credentials are encrypted in the SOCKS5 protocol!")
		log.Println("DO NOT deploy it to the public network. All consequences and responsibilities have nothing to do with the developer")
	} else {
		authMethods = append(authMethods, socks5.NoAuthAuthenticator{})
	}

	server := socks5.NewServer(
		socks5.WithAuthMethods(authMethods),
		socks5.WithResolver(resolver),
		socks5.WithDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := dialer.DialIPPort(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if network != "tcp" {
				return c, nil
			}
			return traffic.Wrap(c), nil
		}),
		socks5.WithLogger(socks5.NewLogger(log.NewLogger("[SOCKS5] "))),
	)

	log.Printf("SOCKS5 server listening on %s", bindAddr)

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("SOCKS5 listen failed: %w", err)
	}

	hook_func.RegisterTerminalFunc("CloseSocks5Listener", func(ctx context.Context) error {
		log.Println("Closing SOCKS5 listener...")
		if err := listener.Close(); err != nil {
			return fmt.Errorf("close SOCKS5 listener failed: %w", err)
		}
		return nil
	})

	go func() {
		if err := server.Serve(listener); err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("SOCKS5 server closed")
			} else {
				log.Println("SOCKS5 listen failed: " + err.Error())
			}
		}
	}()
	return nil
}
