package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nnu-connect/nnu-connect/configs"
	"github.com/nnu-connect/nnu-connect/engine"
	"github.com/nnu-connect/nnu-connect/internal/authinput"
	"github.com/nnu-connect/nnu-connect/internal/traffic"
)

const eventPrefix = "CAMPUSWAY_EVENT "

type engineRequest struct {
	Profile  Profile
	Password string
	Session  engine.Session
	DNS      DNSOverrides
}
type engineCommand struct{ Kind, ID, Answer string }

// logPipeWriter 将标准日志流 (log.Printf) 按行解析并通过 IPC 事件转发给主 GUI 进程。
// 对敏感数据进行实时过滤，严禁将明文密码发送至界面展示。
type logPipeWriter struct {
	emit     func(Event)
	password string
	buf      []byte
	mu       sync.Mutex
}

func (w *logPipeWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r\n")
		w.buf = w.buf[idx+1:]
		if line == "" {
			continue
		}
		// 严禁泄露密码凭据：如果日志中含有明文密码，脱敏过滤
		if w.password != "" {
			line = strings.ReplaceAll(line, w.password, "******")
		}
		w.emit(Event{Kind: "log", Message: line})
	}
	return len(p), nil
}

func runEngine() {
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
		fmt.Fprintf(os.Stdout, "\n%s%s\n", eventPrefix, string(b))
	}

	// 接管标准库与核心引擎日志输出管道，进行脱敏后实时通过 Event 上报前端
	pipeWriter := &logPipeWriter{
		emit:     emit,
		password: request.Password,
	}
	stdlog.SetOutput(pipeWriter)

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
		stdlog.Printf("[认证验证] 网关要求输入: %s，正在等待客户端输入...", kind)
		e := Event{Kind: "challenge", ID: id, Message: "Enter the " + kind + " requested by your gateway"}
		if len(data) > 0 && len(data) < 1024*1024 {
			mime := http.DetectContentType(data)
			if !strings.HasPrefix(mime, "image/") {
				mime = "image/jpeg"
			}
			e.Image = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
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

	// 打印前置环境诊断日志（网关地址、协议、DNS 解析与 Fake-IP 侦测）
	stdlog.Printf("[Init] 准备连接网关: %s (端口: %d, 协议: %s, 账户: %s)", p.Server, p.Port, p.Protocol, p.Username)
	if ips, err := net.LookupIP(p.Server); err != nil {
		stdlog.Printf("[Init] 域名 DNS 解析异常 (%s): %v", p.Server, err)
	} else {
		var ipStrs []string
		hasFakeIP := false
		for _, ip := range ips {
			ipStrs = append(ipStrs, ip.String())
			if fakeIPNet.Contains(ip) {
				hasFakeIP = true
			}
		}
		stdlog.Printf("[Init] 域名解析成功: %s -> [%s]", p.Server, strings.Join(ipStrs, ", "))
		if hasFakeIP {
			stdlog.Printf("[Init] 系统 DNS 返回 Fake-IP；底层连接将解析真实地址。PROCESS-NAME 规则只控制流量路由，不改变 DNS 响应。")
		}
	}
	stdlog.Printf("[Init] 本地代理预定端口: SOCKS5 127.0.0.1:%d, HTTP 127.0.0.1:%d (将在网关握手认证通过后启动绑定)", p.Options.SocksPort, p.Options.HTTPPort)

	conf := configs.Config{Protocol: p.Protocol, ServerAddress: p.Server, ServerPort: p.Port, Username: p.Username, Password: request.Password, DisablePresetConfig: false, SocksBind: fmt.Sprintf("127.0.0.1:%d", p.Options.SocksPort), HTTPBind: fmt.Sprintf("127.0.0.1:%d", p.Options.HTTPPort), RemoteDNSServer: p.Options.RemoteDNS, SecondaryDNSServer: p.Options.SecondaryDNS, DNSTTL: uint64(p.Options.DNSTTL), DisableKeepAlive: true, AuthType: p.Options.AuthType, Phone: p.Username, LoginDomain: p.Options.LoginDomain, UpdateBestNodesInterval: p.Options.UpdateInterval, TCPTunnelMode: p.Options.TCPOnly, DisableRemoteDNS: p.Options.DisableRemoteDNS, DisableMultiLine: p.Options.DisableMultiLine, DisableServerConfig: p.Options.DisableServerConfig, SkipDomainResource: p.Options.SkipDomainResource, ProxyAll: p.Options.ProxyAll}
	if request.DNS.Enabled {
		request.DNS.apply(&conf)
	}
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

	err := engine.RunWithSession(ctx, conf, func(state, msg string) {
		if state == "disconnected" {
			return
		}
		if state == "tunnel_ip" {
			emit(Event{Kind: "tunnel_ip", Message: msg})
			return
		}
		if state == "error" {
			return
		}
		statusMsg := msg
		if statusMsg == "" {
			messages := map[string]string{
				"connecting":    "Connecting to your gateway",
				"connected":     "Connected",
				"disconnecting": "Disconnecting",
			}
			statusMsg = messages[state]
		}
		emit(Event{Kind: "state", State: state, Message: statusMsg})
	}, request.Session, func(session engine.Session) {
		emit(Event{Kind: "session", Session: &session})
	})
	if err != nil && ctx.Err() == nil {
		emit(Event{Kind: "state", State: "error", Message: explainConnectError(p.Server, err)})
		os.Exit(1)
	}
}

// fakeIPNet is the benchmark range (RFC 2544) that Clash/Mihomo-style tools
// hand out as "fake IPs" when their DNS hijacking is active.
var fakeIPNet = func() *net.IPNet { _, n, _ := net.ParseCIDR("198.18.0.0/15"); return n }()

// gatewayIntercepted reports whether the local resolver maps the gateway host to
// a fake IP, meaning another proxy/TUN program sits between us and the gateway.
func gatewayIntercepted(server string) bool {
	ips, err := net.LookupIP(server)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if fakeIPNet.Contains(ip) {
			return true
		}
	}
	return false
}

// explainConnectError turns low-level failures into one of a few stable
// messages the UI can translate. Credentials are never part of the output.
func explainConnectError(server string, err error) string {
	raw := err.Error()
	stdlog.Printf("[错误代码] 连接网关异常 (%s): %v", server, err)
	text := strings.ToLower(raw)
	transport := strings.Contains(text, "eof") || strings.Contains(text, "timeout") || strings.Contains(text, "reset") || strings.Contains(text, "refused") || strings.Contains(text, "deadline")
	switch {
	case strings.Contains(text, "fake-ip"):
		return "Real DNS lookup failed while local DNS returns fake-IP"
	case strings.Contains(text, "no such host") || strings.Contains(text, "lookup"):
		return "Gateway hostname could not be resolved"
	case strings.Contains(text, "atrust virtual ip setup"):
		return "aTrust virtual IP setup failed. See connection logs for the node and stage."
	case transport:
		return "Gateway is unreachable or closed the connection"
	case strings.Contains(text, "login failed") || strings.Contains(text, "password") || strings.Contains(text, "auth"):
		return "Gateway rejected the account or password"
	}
	return "Could not connect. Check your gateway, credentials, verification response and local ports."
}
