package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nnu-connect/nnu-connect/client/atrust"
	"github.com/nnu-connect/nnu-connect/configs"
)

var CommitID string

const nnuConnectVersion = "1.0.0"

func getTOMLVal[T int | uint64 | string | bool](valPointer *T, defaultVal T) T {
	if valPointer == nil {
		return defaultVal
	} else {
		return *valPointer
	}
}

func getTOMLAliasVal[T int | uint64 | string | bool](primary, legacy *T, defaultVal T) T {
	if primary != nil {
		return *primary
	}
	return getTOMLVal(legacy, defaultVal)
}

func parseTOMLConfig(configFile string, conf *configs.Config) error {
	var confTOML configs.ConfigTOML

	_, err := toml.DecodeFile(configFile, &confTOML)
	if err != nil {
		return errors.New("NNU Connect: error parsing the config file: " + err.Error())
	}

	conf.Protocol = getTOMLVal(confTOML.Protocol, "easyconnect")
	conf.ServerAddress = getTOMLVal(confTOML.ServerAddress, "vpn.njnu.edu.cn")
	conf.ServerPort = getTOMLVal(confTOML.ServerPort, 443)
	conf.Username = getTOMLVal(confTOML.Username, "")
	conf.Password = getTOMLVal(confTOML.Password, "")
	conf.TOTPSecret = getTOMLVal(confTOML.TOTPSecret, "")
	conf.CertFile = getTOMLVal(confTOML.CertFile, "")
	conf.CertPassword = getTOMLVal(confTOML.CertPassword, "")
	conf.DisableServerConfig = getTOMLVal(confTOML.DisableServerConfig, false)
	conf.SkipDomainResource = getTOMLVal(confTOML.SkipDomainResource, false)
	conf.DisableZJUConfig = getTOMLAliasVal(confTOML.DisablePresetConfig, confTOML.DisableZJUConfig, false)
	conf.DisableRemoteDNS = getTOMLAliasVal(confTOML.DisableRemoteDNS, confTOML.DisableZJUDNS, false)
	conf.DisableMultiLine = getTOMLVal(confTOML.DisableMultiLine, false)
	conf.ProxyAll = getTOMLVal(confTOML.ProxyAll, false)
	conf.SocksBind = getTOMLVal(confTOML.SocksBind, "127.0.0.1:1080")
	conf.SocksUser = getTOMLVal(confTOML.SocksUser, "")
	conf.SocksPasswd = getTOMLVal(confTOML.SocksPasswd, "")
	conf.HTTPBind = getTOMLVal(confTOML.HTTPBind, "127.0.0.1:1081")
	conf.ShadowsocksURL = getTOMLVal(confTOML.ShadowsocksURL, "")
	conf.DialDirectProxy = getTOMLVal(confTOML.DialDirectProxy, "")
	conf.TCPTunnelMode = getTOMLVal(confTOML.TCPTunnelMode, false)
	conf.TUNMode = getTOMLVal(confTOML.TUNMode, false)
	conf.AddRoute = getTOMLVal(confTOML.AddRoute, false)
	conf.DNSTTL = getTOMLVal(confTOML.DNSTTL, uint64(3600))
	conf.DebugDump = getTOMLVal(confTOML.DebugDump, false)
	conf.DisableKeepAlive = getTOMLVal(confTOML.DisableKeepAlive, false)
	conf.KeepAliveURL = getTOMLVal(confTOML.KeepAliveURL, "")
	conf.RemoteDNSServer = getTOMLAliasVal(confTOML.RemoteDNSServer, confTOML.ZJUDNSServer, "auto")
	conf.SecondaryDNSServer = getTOMLVal(confTOML.SecondaryDNSServer, "114.114.114.114")
	conf.DNSServerBind = getTOMLVal(confTOML.DNSServerBind, "")
	conf.DNSHijack = getTOMLVal(confTOML.DNSHijack, false)
	conf.FakeIP = getTOMLVal(confTOML.FakeIP, false)
	conf.GraphCodeFile = getTOMLVal(confTOML.GraphCodeFile, "")
	conf.BindInterface = getTOMLVal(confTOML.BindInterface, "")
	conf.AutoDetectInterface = getTOMLVal(confTOML.AutoDetectInterface, false)
	conf.AuthType = getTOMLVal(confTOML.AuthType, "")
	conf.Phone = getTOMLVal(confTOML.Phone, "")
	conf.LoginDomain = getTOMLVal(confTOML.LoginDomain, "Radius")
	conf.ClientDataFile = getTOMLVal(confTOML.ClientDataFile, "")
	conf.CasTicket = getTOMLVal(confTOML.CasTicket, "")
	conf.OAuth2Code = getTOMLVal(confTOML.OAuth2Code, "")
	conf.SID = getTOMLVal(confTOML.SID, "")
	conf.DeviceID = getTOMLVal(confTOML.DeviceID, "")
	conf.SignKey = getTOMLVal(confTOML.SignKey, "")
	conf.ResourceFile = getTOMLVal(confTOML.ResourceFile, "")
	conf.UpdateBestNodesInterval = getTOMLVal(confTOML.UpdateBestNodesInterval, 300)
	conf.SkipTCPTunnelWait = getTOMLVal(confTOML.SkipTCPTunnelWait, false)

	for _, singlePortForwarding := range confTOML.PortForwarding {
		if singlePortForwarding.NetworkType == nil {
			return errors.New("NNU Connect: network type is not set")
		}

		if singlePortForwarding.BindAddress == nil {
			return errors.New("NNU Connect: bind address is not set")
		}

		if singlePortForwarding.RemoteAddress == nil {
			return errors.New("NNU Connect: remote address is not set")
		}

		conf.PortForwardingList = append(conf.PortForwardingList, configs.SinglePortForwarding{
			NetworkType:   *singlePortForwarding.NetworkType,
			BindAddress:   *singlePortForwarding.BindAddress,
			RemoteAddress: *singlePortForwarding.RemoteAddress,
		})
	}

	for _, singleCustomDns := range confTOML.CustomDNS {
		if singleCustomDns.HostName == nil {
			return errors.New("NNU Connect: host name is not set")
		}

		if singleCustomDns.IP == nil {
			return errors.New("NNU Connect: IP is not set")
		}

		conf.CustomDNSList = append(conf.CustomDNSList, configs.SingleCustomDNS{
			HostName: *singleCustomDns.HostName,
			IP:       *singleCustomDns.IP,
		})
	}

	for _, singleCustomProxyDomain := range confTOML.CustomProxyDomain {
		var domainRegex = regexp.MustCompile(`^[a-zA-Z\d-]+(\.[a-zA-Z\d-]+)*\.[a-zA-Z]{2,}$`)
		if !domainRegex.MatchString(singleCustomProxyDomain) {
			return fmt.Errorf("NNU Connect: %s is not a valid domain", singleCustomProxyDomain)
		}
		conf.CustomProxyDomain = append(conf.CustomProxyDomain, singleCustomProxyDomain)
	}

	return nil
}

func init() {
	configFile, tcpPortForwarding, udpPortForwarding, customDns, customProxyDomain := "", "", "", "", ""
	showVersion := false
	atrustAuthInfo := false
	atrustTrustDevice := false
	atrustUntrustDevice := false

	flag.StringVar(&conf.Protocol, "protocol", "easyconnect", "Protocol (easyconnect, atrust)")
	flag.StringVar(&conf.ServerAddress, "server", "vpn.njnu.edu.cn", "EasyConnect/aTrust server address")
	flag.IntVar(&conf.ServerPort, "port", 443, "EasyConnect/aTrust port address")
	flag.StringVar(&conf.Username, "username", "", "Your unified authentication username/student ID")
	flag.StringVar(&conf.Password, "password", "", "Your password")
	flag.StringVar(&conf.TOTPSecret, "totp-secret", "", "TOTP secret")
	flag.StringVar(&conf.CertFile, "cert-file", "", "Client certificate p12 file path for certificate login")
	flag.StringVar(&conf.CertPassword, "cert-password", "", "Client certificate password")
	flag.BoolVar(&conf.DisableServerConfig, "disable-server-config", false, "Don't parse server config")
	flag.BoolVar(&conf.SkipDomainResource, "skip-domain-resource", false, "Don't use server-provided domain resources")
	flag.BoolVar(&conf.DisableZJUConfig, "disable-preset-config", false, "Don't use NJNU preset config (domains and subnets)")
	flag.BoolVar(&conf.DisableZJUConfig, "disable-zju-config", false, "Alias for disable-preset-config")
	flag.BoolVar(&conf.DisableRemoteDNS, "disable-remote-dns", false, "Use local DNS instead of remote DNS")
	flag.BoolVar(&conf.DisableRemoteDNS, "disable-zju-dns", false, "Alias for disable-remote-dns")
	flag.BoolVar(&conf.DisableMultiLine, "disable-multi-line", false, "Disable multi line auto select")
	flag.BoolVar(&conf.ProxyAll, "proxy-all", false, "Proxy all IPv4 traffic")
	flag.StringVar(&conf.SocksBind, "socks-bind", "127.0.0.1:1080", "The address SOCKS5 server listens on")
	flag.StringVar(&conf.SocksUser, "socks-user", "", "SOCKS5 username, default is don't use auth")
	flag.StringVar(&conf.SocksPasswd, "socks-passwd", "", "SOCKS5 password, default is don't use auth")
	flag.StringVar(&conf.HTTPBind, "http-bind", "127.0.0.1:1081", "The address HTTP server listens on")
	flag.StringVar(&conf.ShadowsocksURL, "shadowsocks-url", "", "The address Shadowsocks server listens on (e.g. ss://method:password@host:port)")
	flag.StringVar(&conf.DialDirectProxy, "dial-direct-proxy", "", "Dial with proxy when the connection doesn't match rules (e.g. http://127.0.0.1:7890)")
	flag.BoolVar(&conf.TCPTunnelMode, "tcp-tunnel-mode", false, "Use TCP tunnel only and disable L3 tunnel, only works with atrust protocol")
	flag.BoolVar(&conf.TUNMode, "tun-mode", false, "Enable TUN mode (requires administrator privilege)")
	flag.BoolVar(&conf.AddRoute, "add-route", false, "Add route from rules for TUN interface")
	flag.Uint64Var(&conf.DNSTTL, "dns-ttl", 3600, "DNS record time to live, unit is second")
	flag.BoolVar(&conf.DebugDump, "debug-dump", false, "Enable traffic debug dump (only for debug usage)")
	flag.BoolVar(&conf.DisableKeepAlive, "disable-keep-alive", false, "Disable keep alive")
	flag.StringVar(&conf.KeepAliveURL, "keep-alive-url", "https://vpn.njnu.edu.cn", "Keep alive URL")
	flag.StringVar(&conf.RemoteDNSServer, "remote-dns-server", "auto", "Remote DNS server address. Set to 'auto' to use remote DNS server provided by server")
	flag.StringVar(&conf.RemoteDNSServer, "zju-dns-server", "auto", "Alias for remote-dns-server")
	flag.StringVar(&conf.SecondaryDNSServer, "secondary-dns-server", "114.114.114.114", "Secondary DNS server address. Leave empty to use system default DNS server")
	flag.StringVar(&conf.DNSServerBind, "dns-server-bind", "", "The address DNS server listens on (e.g. 127.0.0.1:53)")
	flag.BoolVar(&conf.DNSHijack, "dns-hijack", false, "Hijack all dns query to NNU Connect")
	flag.BoolVar(&conf.FakeIP, "fake-ip", false, "Enable Fake IP for DNS hijack")
	flag.StringVar(&conf.GraphCodeFile, "graph-code-file", "", "Graph Check Code File")
	flag.StringVar(&conf.BindInterface, "bind-interface", "", "Bind VPN underlay connections to this network interface")
	flag.BoolVar(&conf.AutoDetectInterface, "auto-detect-interface", false, "Automatically detect and bind the VPN underlay interface")
	flag.StringVar(&conf.TwfID, "twf-id", "", "Login using twfID captured")
	flag.StringVar(&conf.AuthType, "auth-type", "", "aTrust authentication type (auth/psw, auth/cas, auth/httpsOauth2, auth/smsCheckCode)")
	flag.StringVar(&conf.Phone, "phone", "", "Phone number for aTrust SMS check code login")
	flag.StringVar(&conf.LoginDomain, "login-domain", "Radius", "aTrust login domain")
	flag.StringVar(&conf.ClientDataFile, "client-data-file", "", "aTrust Client Data File")
	flag.StringVar(&conf.CasTicket, "cas-ticket", "", "aTrust CAS Ticket")
	flag.StringVar(&conf.OAuth2Code, "oauth2-code", "", "aTrust OAuth2 code")
	flag.StringVar(&conf.SID, "sid", "", "aTrust SID")
	flag.StringVar(&conf.DeviceID, "device-id", "", "aTrust Device ID")
	flag.StringVar(&conf.SignKey, "sign-key", "", "aTrust Sign Key")
	flag.StringVar(&conf.ResourceFile, "resource-file", "", "aTrust Resource File")
	flag.IntVar(&conf.UpdateBestNodesInterval, "update-best-nodes-interval", 300, "Interval to update best nodes in seconds")
	flag.BoolVar(&conf.SkipTCPTunnelWait, "skip-tcp-tunnel-wait", false, "Don't wait for aTrust TCP tunnel connection status")
	flag.StringVar(&tcpPortForwarding, "tcp-port-forwarding", "", "TCP port forwarding (e.g. 0.0.0.0:9898-10.10.98.98:80)")
	flag.StringVar(&udpPortForwarding, "udp-port-forwarding", "", "UDP port forwarding (e.g. 127.0.0.1:53-10.10.0.21:53)")
	flag.StringVar(&customDns, "custom-dns", "", "Custom set dns lookup (e.g. ehallapp.nnu.edu.cn:10.10.1.1)")
	flag.StringVar(&customProxyDomain, "custom-proxy-domain", "", "Custom set domains which force use VPN proxy (e.g. ehallapp.nnu.edu.cn,zgzj.njnu.edu.cn)")
	flag.StringVar(&configFile, "config", "", "Config file path")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&atrustAuthInfo, "auth-info", false, "Fetch aTrust authentication information, but not login")
	flag.BoolVar(&atrustTrustDevice, "trust-device", false, "Trust the current device for aTrust")
	flag.BoolVar(&atrustUntrustDevice, "untrust-device", false, "Untrust the current device for aTrust")

	flag.Parse()

	if showVersion {
		fmt.Printf("NNU Connect v%s\n", nnuConnectVersion)
		os.Exit(0)
	}

	if atrustAuthInfo {
		if conf.Protocol != "atrust" {
			fmt.Fprintln(os.Stderr, "Auth info is only supported by the atrust protocol")
			os.Exit(1)
		}
		log.SetOutput(io.Discard)
		info, err := atrust.GetAuthInfoList(conf.ServerAddress, conf.ServerPort, conf.BindInterface, conf.AutoDetectInterface)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Get auth info list error:", err)
			os.Exit(1)
		}
		jsonInfo, err := json.Marshal(info)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error marshaling auth info:", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonInfo))
		os.Exit(0)
	}

	if atrustTrustDevice || atrustUntrustDevice {
		if conf.Protocol != "atrust" {
			fmt.Fprintln(os.Stderr, "Trust/Untrust device is only supported by the atrust protocol")
			os.Exit(1)
		}
		if conf.ClientDataFile == "" {
			fmt.Fprintln(os.Stderr, "Client data file is required for trust/untrust device")
			os.Exit(1)
		}
		clientData, err := os.ReadFile(conf.ClientDataFile)
		if err != nil {
			log.Printf("Read client data file error: %s", err)
			os.Exit(1)
		}

		err = atrust.SetTrusted(conf.ServerAddress, conf.ServerPort, clientData, atrustTrustDevice, conf.BindInterface, conf.AutoDetectInterface)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Trust/Untrust device error:", err)
			os.Exit(1)
		}
		if atrustTrustDevice {
			log.Println("Device trusted successfully")
		} else {
			log.Println("Device untrusted successfully")
		}
		os.Exit(0)
	}

	if configFile != "" {
		err := parseTOMLConfig(configFile, &conf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		if tcpPortForwarding != "" {
			forwardingStringList := strings.Split(tcpPortForwarding, ",")
			for _, forwardingString := range forwardingStringList {
				addressStringList := strings.Split(forwardingString, "-")
				if len(addressStringList) != 2 {
					fmt.Fprintln(os.Stderr, "NNU Connect: wrong tcp port forwarding format")
					os.Exit(1)
				}

				conf.PortForwardingList = append(conf.PortForwardingList, configs.SinglePortForwarding{
					NetworkType:   "tcp",
					BindAddress:   addressStringList[0],
					RemoteAddress: addressStringList[1],
				})
			}
		}

		if udpPortForwarding != "" {
			forwardingStringList := strings.Split(udpPortForwarding, ",")
			for _, forwardingString := range forwardingStringList {
				addressStringList := strings.Split(forwardingString, "-")
				if len(addressStringList) != 2 {
					fmt.Fprintln(os.Stderr, "NNU Connect: wrong udp port forwarding format")
					os.Exit(1)
				}

				conf.PortForwardingList = append(conf.PortForwardingList, configs.SinglePortForwarding{
					NetworkType:   "udp",
					BindAddress:   addressStringList[0],
					RemoteAddress: addressStringList[1],
				})
			}
		}

		if customDns != "" {
			dnsList := strings.Split(customDns, ",")
			for _, dnsString := range dnsList {
				dnsStringSplit := strings.Split(dnsString, ":")
				if len(dnsStringSplit) != 2 {
					fmt.Fprintln(os.Stderr, "NNU Connect: wrong custom dns format")
					os.Exit(1)
				}

				conf.CustomDNSList = append(conf.CustomDNSList, configs.SingleCustomDNS{
					HostName: dnsStringSplit[0],
					IP:       dnsStringSplit[1],
				})
			}
		}

		if customProxyDomain != "" {
			domainList := strings.Split(customProxyDomain, ",")
			for _, domain := range domainList {
				var domainRegex = regexp.MustCompile(`^[a-zA-Z\d-]+(\.[a-zA-Z\d-]+)*\.[a-zA-Z]{2,}$`)
				if !domainRegex.MatchString(domain) {
					fmt.Fprintf(os.Stderr, "NNU Connect: %s is not a valid domain\n", domain)
					os.Exit(1)
				}
				conf.CustomProxyDomain = append(conf.CustomProxyDomain, domain)
			}
		}
	}
	if password, ok := os.LookupEnv("NNU_CONNECT_PASSWORD"); ok {
		conf.Password = password
	}

	missing := conf.ServerAddress == ""
	if !missing && conf.Protocol == "easyconnect" {
		missing = (conf.Username == "" || conf.Password == "") && conf.TwfID == ""
	}
	if !missing && conf.Protocol == "atrust" {
		switch conf.AuthType {
		case "auth/psw":
			missing = conf.Username == "" || conf.Password == ""
		case "auth/smsCheckCode":
			missing = conf.Phone == ""
		}
		if missing {
			missing = conf.SID == "" || conf.DeviceID == "" || conf.ResourceFile == ""
		}
	}
	if missing {
		fmt.Println("NNU Connect: missing required arguments")
		fmt.Println("Please see: https://github.com/nnu-connect/nnu-connect")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()

		os.Exit(1)
	}
}
