package engine

import (
	"context"
	"crypto"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/nnu-connect/nnu-connect/client"
	atrustclient "github.com/nnu-connect/nnu-connect/client/atrust"
	easyconnectclient "github.com/nnu-connect/nnu-connect/client/easyconnect"
	"github.com/nnu-connect/nnu-connect/configs"
	"github.com/nnu-connect/nnu-connect/dial"
	"github.com/nnu-connect/nnu-connect/internal/hook_func"
	"github.com/nnu-connect/nnu-connect/log"
	"github.com/nnu-connect/nnu-connect/resolve"
	"github.com/nnu-connect/nnu-connect/service"
	"github.com/nnu-connect/nnu-connect/stack"
	"github.com/nnu-connect/nnu-connect/stack/gvisor"
	"github.com/nnu-connect/nnu-connect/stack/tcptunnel"
	"github.com/nnu-connect/nnu-connect/stack/tun"
	"golang.org/x/crypto/pkcs12"
)

type StatusCallback func(status string, message string)

// Session contains gateway-issued login state; desktop callers encrypt it at rest.
type Session struct {
	TwfID    string `json:"twfID,omitempty"`
	AuthData []byte `json:"authData,omitempty"`
}

func deriveGatewayRootDomain(serverAddr string) string {
	host := strings.TrimSpace(serverAddr)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" || net.ParseIP(host) != nil {
		return ""
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	if len(labels) >= 3 {
		lastTwo := labels[len(labels)-2] + "." + labels[len(labels)-1]
		switch lastTwo {
		case "edu.cn", "ac.cn", "org.cn", "com.cn", "gov.cn", "net.cn":
			return strings.Join(labels[len(labels)-3:], ".")
		}
	}
	return strings.Join(labels[len(labels)-2:], ".")
}

func CampusDomains(serverAddr string) []string {
	root := deriveGatewayRootDomain(serverAddr)
	if root == "nnu.edu.cn" || root == "njnu.edu.cn" {
		return []string{"nnu.edu.cn", "njnu.edu.cn"}
	}
	if root == "zju.edu.cn" {
		return []string{root, "cc98.org"}
	}
	if root == "" {
		return nil
	}
	return []string{root}
}

// Run runs the VPN engine lifecycle until ctx is cancelled.
func Run(ctx context.Context, conf configs.Config, onStatus StatusCallback) error {
	return RunWithSession(ctx, conf, onStatus, Session{}, nil)
}

func RunWithSession(ctx context.Context, conf configs.Config, onStatus StatusCallback, session Session, onSession func(Session)) error {
	var vpnClient client.Client
	hook_func.BeginTerminalSession()
	defer func() { shutdownClient(vpnClient) }()

	if onStatus != nil {
		onStatus("connecting", "正在初始化 VPN 核心引擎...")
	}

	if conf.DebugDump {
		log.EnableDebug()
	}

	if errs := hook_func.ExecInitialFunc(ctx, conf); errs != nil {
		for _, err := range errs {
			log.Printf("Initial RouteDesk failed: %s", err)
		}
		if onStatus != nil {
			onStatus("error", fmt.Sprintf("初始化失败: %v", errs[0]))
		}
		return fmt.Errorf("initial failed: %v", errs)
	}

	switch conf.Protocol {
	case "easyconnect":
		if onStatus != nil {
			onStatus("connecting", fmt.Sprintf("正在握手认证: %s (%s)...", conf.ServerAddress, conf.Username))
		}
		tlsCert := tls.Certificate{}
		if conf.CertFile != "" {
			p12Data, err := os.ReadFile(conf.CertFile)
			if err != nil {
				return fmt.Errorf("read certificate file error: %w", err)
			}

			key, cert, err := pkcs12.Decode(p12Data, conf.CertPassword)
			if err != nil {
				return fmt.Errorf("decode certificate file error: %w", err)
			}

			tlsCert = tls.Certificate{
				Certificate: [][]byte{cert.Raw},
				PrivateKey:  key.(crypto.PrivateKey),
				Leaf:        cert,
			}
		}

		serverAddr := conf.ServerAddress
		if conf.ServerPort > 0 && conf.ServerPort != 443 {
			serverAddr = net.JoinHostPort(conf.ServerAddress, fmt.Sprint(conf.ServerPort))
		} else {
			serverAddr = net.JoinHostPort(conf.ServerAddress, "443")
		}

		if conf.TwfID == "" {
			conf.TwfID = session.TwfID
		}
		vpnClient = easyconnectclient.NewClient(
			serverAddr,
			conf.Username,
			conf.Password,
			conf.TOTPSecret,
			tlsCert,
			conf.TwfID,
			!conf.DisableMultiLine,
			!conf.DisableServerConfig,
			!conf.SkipDomainResource,
		)

		log.Printf("VPN protocol: %s, server: %s", conf.Protocol, serverAddr)
		defer cancelClientOnContext(ctx, vpnClient)()
		err := vpnClient.(*easyconnectclient.Client).Setup(conf.GraphCodeFile, conf.BindInterface, conf.AutoDetectInterface)
		if err != nil {
			if onStatus != nil {
				onStatus("error", fmt.Sprintf("EasyConnect 认证失败: %s", err.Error()))
			}
			return fmt.Errorf("vpn client setup error: %w", err)
		}

	case "atrust":
		if onStatus != nil {
			onStatus("connecting", fmt.Sprintf("正在通过 aTrust 认证: %s...", conf.ServerAddress))
		}
		var err error
		var resourceData []byte
		if conf.ResourceFile != "" {
			resourceData, err = os.ReadFile(conf.ResourceFile)
			if err != nil {
				return fmt.Errorf("read resource file error: %w", err)
			}
		}

		var clientData []byte
		clientData = session.AuthData
		if conf.ClientDataFile != "" {
			clientData, err = os.ReadFile(conf.ClientDataFile)
			if err != nil {
				log.Printf("Read client data file error: %s", err)
			}
		}

		vpnClient = atrustclient.NewClient(conf.Username, conf.SID, conf.DeviceID, conf.SignKey)
		defer cancelClientOnContext(ctx, vpnClient)()
		vpnClient.(*atrustclient.Client).SetSkipTCPTunnelWait(conf.SkipTCPTunnelWait)

		clientData, err = vpnClient.(*atrustclient.Client).Setup(
			conf.ServerAddress,
			conf.ServerPort,
			conf.Username,
			conf.Password,
			conf.Phone,
			conf.LoginDomain,
			conf.AuthType,
			conf.GraphCodeFile,
			conf.CasTicket,
			conf.OAuth2Code,
			clientData,
			resourceData,
			conf.UpdateBestNodesInterval,
			conf.BindInterface,
			conf.AutoDetectInterface,
		)
		if len(clientData) != 0 {
			if conf.ClientDataFile != "" {
				_ = os.WriteFile(conf.ClientDataFile, clientData, 0600)
			}
			if onSession != nil {
				onSession(Session{AuthData: clientData})
			}
		}
		if err != nil {
			return fmt.Errorf("aTrust setup error: %w", err)
		}

	default:
		return fmt.Errorf("unsupported VPN protocol: %s", conf.Protocol)
	}

	log.Printf("VPN client authentication success")
	if err := ctx.Err(); err != nil {
		return err
	}
	if conf.Protocol == "easyconnect" && onSession != nil {
		onSession(Session{TwfID: vpnClient.(*easyconnectclient.Client).SessionID()})
	}
	if closer, ok := vpnClient.(interface{ Close() }); ok {
		hook_func.RegisterTerminalFunc("CloseVPNClient", func(ctx context.Context) error {
			closer.Close()
			return nil
		})
	}

	ipResources, _ := vpnClient.IPResources()
	ipSet, _ := vpnClient.IPSet()
	domainResources, _ := vpnClient.DomainResources()
	dnsResource, _ := vpnClient.DNSResource()

	if domainResources == nil {
		domainResources = make(map[string]client.DomainResource)
	}
	addDomain := func(domain string) {
		domain = strings.Trim(strings.ToLower(strings.TrimSpace(domain)), ".")
		if domain != "" {
			if _, exists := domainResources[domain]; !exists {
				domainResources[domain] = client.DomainResource{PortMin: 1, PortMax: 65535, Protocol: "all"}
			}
		}
	}
	if !conf.DisablePresetConfig {
		for _, domain := range CampusDomains(conf.ServerAddress) {
			addDomain(domain)
		}
	}
	for _, domain := range conf.CustomProxyDomain {
		addDomain(domain)
	}
	if conf.Protocol == "easyconnect" && !conf.DisablePresetConfig && ipResources == nil {
		ipResources = []client.IPResource{}
	}

	var vpnStack stack.Stack
	var err error
	if conf.TCPTunnelMode && !vpnClient.CanUseTCPTunnel() {
		log.Println("TCP-only mode is unavailable for EasyConnect; using the L3 tunnel")
		conf.TCPTunnelMode = false
	}
	if conf.TCPTunnelMode {
		vpnStack, err = tcptunnel.NewStack(vpnClient)
		if err != nil {
			return fmt.Errorf("TCP tunnel stack setup error: %w", err)
		}
	} else if conf.TUNMode {
		if onStatus != nil {
			onStatus("connecting", "正在配置 TUN 虚拟网卡...")
		}
		vpnTUNStack, err := tun.NewStack(vpnClient, conf.DNSHijack, conf.FakeIP, ipResources)
		if err != nil {
			return fmt.Errorf("TUN stack setup error (please run as Administrator): %w", err)
		}

		if conf.AddRoute && ipSet != nil {
			for _, prefix := range ipSet.Prefixes() {
				log.Printf("Add route to %s", prefix.String())
				_ = vpnTUNStack.AddRoute(prefix.String())
			}
		}

		if conf.FakeIP {
			_ = vpnTUNStack.AddRoute("198.18.0.0/16")
		}

		vpnStack = vpnTUNStack
	} else {
		vpnStack, err = gvisor.NewStack(vpnClient)
		if err != nil {
			return fmt.Errorf("gVisor stack setup error: %w", err)
		}
	}

	useRemoteDNS := !conf.DisableRemoteDNS
	remoteDNSServer := conf.RemoteDNSServer
	if useRemoteDNS && remoteDNSServer == "auto" {
		remoteDNSServer, err = vpnClient.DNSServer()
		if err != nil {
			useRemoteDNS = false
			remoteDNSServer = "114.114.114.114"
			log.Println("No remote DNS server provided by server. Using local/fallback DNS")
		} else {
			log.Printf("Use DNS server %s provided by server", remoteDNSServer)
		}
	}

	vpnResolver := resolve.NewResolver(
		vpnStack,
		remoteDNSServer,
		conf.SecondaryDNSServer,
		conf.DNSTTL,
		domainResources,
		dnsResource,
		useRemoteDNS,
	)
	hook_func.RegisterTerminalFunc("CloseResolver", func(ctx context.Context) error {
		vpnResolver.Close()
		return nil
	})

	for _, customDns := range conf.CustomDNSList {
		ipAddr := net.ParseIP(customDns.IP)
		if ipAddr == nil {
			log.Printf("Custom DNS for host %s is invalid, SKIP", customDns.HostName)
			continue
		}
		vpnResolver.SetPermanentDNS(customDns.HostName, ipAddr)
		log.Printf("Add custom DNS: %s -> %s\n", customDns.HostName, customDns.IP)
	}

	localResolver := service.NewDnsServer(vpnResolver, []string{remoteDNSServer, conf.SecondaryDNSServer})
	vpnStack.SetupResolve(localResolver)
	vpnStack.SetupIPPool(vpnResolver.IPPool)

	go vpnStack.Run()

	if conf.Protocol == "atrust" {
		conf.ProxyAll = false
	}
	vpnDialer := dial.NewDialer(vpnStack, vpnResolver, ipResources, conf.ProxyAll, conf.DialDirectProxy)

	if conf.DNSServerBind != "" {
		go service.ServeDNS(conf.DNSServerBind, localResolver)
	}
	if conf.TUNMode {
		clientIP, _ := vpnClient.IP()
		go service.ServeDNS(clientIP.String()+":53", localResolver)
	}

	if conf.SocksBind != "" {
		if err := service.StartSocks5(conf.SocksBind, vpnDialer, vpnResolver, conf.SocksUser, conf.SocksPasswd); err != nil {
			return err
		}
	}

	if conf.HTTPBind != "" {
		if err := service.StartHTTP(conf.HTTPBind, vpnDialer); err != nil {
			return err
		}
	}

	if conf.ShadowsocksURL != "" {
		go service.ServeShadowsocks(vpnDialer, conf.ShadowsocksURL)
	}

	for _, portForwarding := range conf.PortForwardingList {
		switch portForwarding.NetworkType {
		case "tcp":
			go service.ServeTCPForwarding(vpnStack, portForwarding.BindAddress, portForwarding.RemoteAddress)
		case "udp":
			go service.ServeUDPForwarding(vpnStack, portForwarding.BindAddress, portForwarding.RemoteAddress)
		default:
			log.Printf("Port forwarding: unknown network type %s", portForwarding.NetworkType)
		}
	}

	if !conf.DisableKeepAlive {
		keepAliveURL := conf.KeepAliveURL
		if keepAliveURL == "" && conf.ServerAddress != "" {
			if conf.ServerPort > 0 && conf.ServerPort != 443 {
				keepAliveURL = fmt.Sprintf("https://%s:%d", conf.ServerAddress, conf.ServerPort)
			} else {
				keepAliveURL = "https://" + conf.ServerAddress
			}
		}
		if keepAliveURL != "" {
			keepAliveCtx, keepAliveCancel := context.WithCancel(context.Background())
			hook_func.RegisterTerminalFunc("CloseKeepAlive", func(ctx context.Context) error {
				keepAliveCancel()
				return nil
			})
			go service.KeepAlive(keepAliveCtx, vpnResolver, vpnDialer, keepAliveURL)
		}
	}

	if onStatus != nil {
		if ip, err := vpnClient.IP(); err == nil && ip != nil {
			onStatus("tunnel_ip", ip.String())
		}
		onStatus("connected", fmt.Sprintf("已成功连接至 %s (SOCKS5: %s, HTTP: %s)", conf.ServerAddress, conf.SocksBind, conf.HTTPBind))
	}

	// Block until context canceled
	<-ctx.Done()

	log.Println("Shutting down RouteDesk ......")
	if onStatus != nil {
		onStatus("disconnecting", "正在断开连接并清理网络规则...")
	}

	log.Println("RouteDesk shutdown complete. Bye~")
	if onStatus != nil {
		onStatus("disconnected", "已断开连接")
	}

	return nil
}

func shutdownClient(vpnClient client.Client) {
	// Readers must see shutdown before cancellation closes their sockets.
	for _, err := range hook_func.ExecTerminalFunc(context.Background()) {
		log.Printf("Shutdown RouteDesk hook error: %s", err)
	}
	if closer, ok := vpnClient.(interface{ Close() }); ok {
		closer.Close()
	}
}

func cancelClientOnContext(ctx context.Context, vpnClient client.Client) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		shutdownClient(vpnClient)
		close(done)
	})
	return func() {
		if !stop() {
			<-done
		}
	}
}
