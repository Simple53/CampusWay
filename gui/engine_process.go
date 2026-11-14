package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nnu-connect/nnu-connect/configs"
	"github.com/nnu-connect/nnu-connect/engine"
	"github.com/nnu-connect/nnu-connect/internal/authinput"
	"github.com/nnu-connect/nnu-connect/internal/traffic"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const eventPrefix = "CAMPUSWAY_EVENT "

type engineRequest struct {
	Profile  Profile
	Password string
}
type engineCommand struct{ Kind, ID, Answer string }

func runEngine() {
	// Secret-bearing upstream logs stay in the worker and are not forwarded to the UI.
	stdlog.SetOutput(io.Discard)
	decoder := json.NewDecoder(os.Stdin)
	var request engineRequest
	if decoder.Decode(&request) != nil {
		os.Exit(2)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var outputMu sync.Mutex
	emit := func(e Event) {
		outputMu.Lock()
		defer outputMu.Unlock()
		b, _ := json.Marshal(e)
		fmt.Fprintln(os.Stdout, eventPrefix+string(b))
	}
	answers := make(chan engineCommand, 1)
	go func() {
		for {
			var c engineCommand
			if decoder.Decode(&c) != nil {
				cancel()
				return
			}
			if c.Kind == "stop" {
				cancel()
				return
			}
			if c.Kind == "answer" {
				select {
				case answers <- c:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	authinput.Handler = func(kind string, data []byte) (string, error) {
		id := identifier()
		e := Event{Kind: "challenge", ID: id, Message: "Enter the " + kind + " requested by your gateway"}
		if len(data) > 0 && len(data) < 1024*1024 {
			mime := http.DetectContentType(data)
			if strings.HasPrefix(mime, "image/") {
				e.Image = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
			}
		}
		emit(e)
		for {
			select {
			case c := <-answers:
				if c.ID == id {
					return c.Answer, nil
				}
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}
	p := request.Profile
	if err := validateProfile(&p); err != nil {
		emit(Event{Kind: "state", State: "error", Message: "Invalid connection configuration"})
		os.Exit(2)
	}

	conf := configs.Config{Protocol: p.Protocol, ServerAddress: p.Server, ServerPort: p.Port, Username: p.Username, Password: request.Password, DisablePresetConfig: false, SocksBind: fmt.Sprintf("127.0.0.1:%d", p.Options.SocksPort), HTTPBind: fmt.Sprintf("127.0.0.1:%d", p.Options.HTTPPort), RemoteDNSServer: p.Options.RemoteDNS, SecondaryDNSServer: p.Options.SecondaryDNS, DNSTTL: uint64(p.Options.DNSTTL), DisableKeepAlive: true, AuthType: p.Options.AuthType, Phone: p.Username, LoginDomain: p.Options.LoginDomain, UpdateBestNodesInterval: p.Options.UpdateInterval, TCPTunnelMode: p.Options.TCPOnly, DisableRemoteDNS: p.Options.DisableRemoteDNS, DisableMultiLine: p.Options.DisableMultiLine, DisableServerConfig: p.Options.DisableServerConfig, SkipDomainResource: p.Options.SkipDomainResource, ProxyAll: p.Options.ProxyAll}
	conf.CustomProxyDomain = strings.FieldsFunc(p.CustomDomains, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == ' ' })
	// A private worker owns this counter lifetime; rates use the real sample interval.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		last := time.Now()
		up, down := traffic.Totals()
		for {
			select {
			case now := <-ticker.C:
				nextUp, nextDown := traffic.Totals()
				seconds := now.Sub(last).Seconds()
				emit(Event{Kind: "traffic", Traffic: &TrafficSnapshot{Uploaded: nextUp, Downloaded: nextDown, UploadRate: float64(nextUp-up) / seconds, DownloadRate: float64(nextDown-down) / seconds, SampledAt: now.UnixMilli()}})
				up, down, last = nextUp, nextDown, now
			case <-ctx.Done():
				return
			}
		}
	}()

	err := engine.Run(ctx, conf, func(state, msg string) {
		if state == "disconnected" {
			return
		}
		if state == "error" {
			emit(Event{Kind: "state", State: "error", Message: msg})
			return
		}
		messages := map[string]string{"connecting": "Connecting to your gateway", "connected": "Connected", "disconnecting": "Disconnecting"}
		m := messages[state]
		if m == "" {
			m = msg
		}
		emit(Event{Kind: "state", State: state, Message: m})
	})
	if err != nil && ctx.Err() == nil {
		emit(Event{Kind: "state", State: "error", Message: err.Error()})
		os.Exit(1)
	}
}
