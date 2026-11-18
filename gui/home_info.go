package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nnu-connect/nnu-connect/internal/dnsutil"
)

type WebsiteLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func parseWebsiteURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || u == nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("Enter a valid HTTP or HTTPS website")
	}
	return u, nil
}

type IPInfo struct {
	IP      string `json:"ip"`
	Family  string `json:"family"`
	Country string `json:"country"`
	Region  string `json:"region"`
	Public  bool   `json:"public"`
}

func (s *AppServer) RefreshIPInfo() (IPInfo, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ipapi.co/json/")
	if err == nil {
		defer resp.Body.Close()
		var data struct {
			IP      string `json:"ip"`
			Country string `json:"country_name"`
			Region  string `json:"region"`
		}
		if resp.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(resp.Body, 16*1024)).Decode(&data) == nil {
			if ip := net.ParseIP(data.IP); ip != nil {
				info := describeIP(ip, true)
				info.Country, info.Region = data.Country, data.Region
				return info, nil
			}
		}
	}
	if ip := net.ParseIP(outboundIP()); ip != nil {
		return describeIP(ip, false), nil
	}
	return IPInfo{}, errors.New("Could not refresh IP information")
}

func describeIP(ip net.IP, public bool) IPInfo {
	family := "IPv6"
	if ip.To4() != nil {
		family = "IPv4"
	}
	return IPInfo{IP: ip.String(), Family: family, Public: public}
}

// GetLocalIPs reads the device interfaces without sending a network request.
func (s *AppServer) GetLocalIPs() []IPInfo {
	interfaces, _ := net.Interfaces()
	var ipv4, ipv6 net.IP
	for _, iface := range interfaces {
		name := strings.ToLower(iface.Name)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 ||
			strings.Contains(name, "mihomo") || strings.Contains(name, "clash") ||
			strings.Contains(name, "wintun") || strings.Contains(name, "routedesk") || strings.Contains(name, "virtual") || strings.Contains(name, "vethernet") ||
			strings.Contains(name, "campusway") || strings.HasPrefix(name, "utun") ||
			strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap") {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ip, _, _ := net.ParseCIDR(address.String())
			if !ip.IsGlobalUnicast() || dnsutil.IsFakeIP(ip) {
				continue
			}
			if ip.To4() != nil {
				if ipv4 == nil {
					ipv4 = ip
				}
			} else if ipv6 == nil {
				ipv6 = ip
			}
		}
	}
	out := []IPInfo{}
	for _, address := range []net.IP{ipv4, ipv6} {
		if address != nil {
			out = append(out, describeIP(address, false))
		}
	}
	return out
}
