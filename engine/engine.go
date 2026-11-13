package engine

import (
	"context"
	"crypto"
	"crypto/tls"
	"fmt"
	"net"
	"os"

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

// Run runs the VPN engine lifecycle until ctx is cancelled.
func Run(ctx context.Context, conf configs.Config, onStatus StatusCallback) error {
	hook_func.BeginTerminalSession()
	defer func() {
		if errs := hook_func.ExecTerminalFunc(context.Background()); errs != nil {
			for _, err := range errs {
				log.Printf("Shutdown NNU-Connect hook error: %s", err)
			}
		}
	}()

	if onStatus != nil {
		onStatus("connecting", "正在初始化 VPN 核心引擎...")
	}

	if conf.DebugDump {
		log.EnableDebug()
	}

	if errs := hook_func.ExecInitialFunc(ctx, conf); errs != nil {
		for _, err := range errs {
			log.Printf("Initial NNU-Connect failed: %s", err)
		}
		if onStatus != nil {
			onStatus("error", fmt.Sprintf("初始化失败: %v", errs[0]))
		}
		return fmt.Errorf("initial failed: %v", errs)
	}

	var vpnClient client.Client
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
		if conf.ClientDataFile != "" {
			clientData, err = os.ReadFile(conf.ClientDataFile)
			if err != nil {
				log.Printf("Read client data file error: %s", err)
			}
		}

		vpnClient = atrustclient.NewClient(conf.Username, conf.SID, conf.DeviceID, conf.SignKey)
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
		if err != nil {
			if onStatus != nil {
				onStatus("error", fmt.Sprintf("aTrust 认证失败: %s", err.Error()))
			}
			return fmt.Errorf("aTrust setup error: %w", err)
		}

		if conf.ClientDataFile != "" {
			_ = os.WriteFile(conf.ClientDataFile, clientData, 0644)
		}

	default:
		return fmt.Errorf("unsupported VPN protocol: %s", conf.Protocol)
	}

	log.Printf("VPN client authentication success")
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

	if conf.Protocol == "easyconnect" {
		if !conf.DisableZJUConfig {
			if domainResources == nil {
				domainResources = make(map[string]client.DomainResource)
			}

			// Pre-configure NJNU core domains
			nnuDomains := []string{
				"njnu.edu.cn",
				"nnu.edu.cn",
			}
			for _, d := range nnuDomains {
				domainResources[d] = client.DomainResource{
					PortMin:  1,
					PortMax:  65535,
					Protocol: "all",
				}
			}

			if ipResources == nil {
				ipResources = []client.IPResource{}
			}

			// IP routes come from the VPN gateway. Broad private ranges such as
			// 10.0.0.0/8 can overlap with the user's LAN and are not safe defaults.
		}

		// Add custom proxy domains (e.g. ehallapp.nnu.edu.cn, zgzj.njnu.edu.cn, etc.)
		if domainResources == nil {
			domainResources = make(map[string]client.DomainResource)
		}
		for _, customProxyDomain := range conf.CustomProxyDomain {
			if customProxyDomain != "" {
				domainResources[customProxyDomain] = client.DomainResource{
					PortMin:  1,
					PortMax:  65535,
					Protocol: "all",
				}
				log.Printf("Added custom proxy domain: %s", customProxyDomain)
			}
		}
	}

	var vpnStack stack.Stack
	var err error
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
		if keepAliveURL == "" {
			keepAliveURL = "https://vpn.njnu.edu.cn"
		}
		keepAliveCtx, keepAliveCancel := context.WithCancel(context.Background())
		hook_func.RegisterTerminalFunc("CloseKeepAlive", func(ctx context.Context) error {
			keepAliveCancel()
			return nil
		})
		go service.KeepAlive(keepAliveCtx, vpnResolver, vpnDialer, keepAliveURL)
	}

	if onStatus != nil {
		onStatus("connected", fmt.Sprintf("已成功连接至 %s (SOCKS5: %s, HTTP: %s)", conf.ServerAddress, conf.SocksBind, conf.HTTPBind))
	}

	// Block until context canceled
	<-ctx.Done()

	log.Println("Shutting down NNU-Connect ......")
	if onStatus != nil {
		onStatus("disconnecting", "正在断开连接并清理网络规则...")
	}

	log.Println("NNU-Connect shutdown complete. Bye~")
	if onStatus != nil {
		onStatus("disconnected", "已断开连接")
	}

	return nil
}
