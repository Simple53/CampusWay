package main

import (
	"errors"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/nnu-connect/nnu-connect/configs"
	"github.com/nnu-connect/nnu-connect/internal/dnsutil"
)

type DNSOverrides struct {
	Enabled   bool   `json:"enabled"`
	Remote    string `json:"remote"`
	Secondary string `json:"secondary"`
	TTL       int    `json:"ttl"`
	Records   string `json:"records"`
}

func (d DNSOverrides) validate() error {
	remote := strings.TrimSpace(d.Remote)
	if remote != "" && remote != "auto" && (!usableDNSServer(remote) || net.ParseIP(remote).To4() == nil) {
		return errors.New("Enter an IPv4 campus DNS or auto")
	}
	secondary := strings.TrimSpace(d.Secondary)
	if secondary != "" && !usableDNSServer(secondary) {
		return errors.New("Enter an IP address for fallback DNS")
	}
	if d.TTL < 0 || d.TTL > 86400 {
		return errors.New("DNS cache TTL must be between 1 and 86400 seconds")
	}
	_, err := d.parseRecords()
	return err
}

func usableDNSServer(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.IsGlobalUnicast() && !dnsutil.IsFakeIP(ip)
}

func (d DNSOverrides) parseRecords() ([]configs.SingleCustomDNS, error) {
	if len(d.Records) > 8192 {
		return nil, errors.New("DNS overrides exceed 8 KB")
	}
	var records []configs.SingleCustomDNS
	for _, line := range strings.Split(d.Records, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 || len(records) == 64 || !usableDNSServer(fields[1]) {
			return nil, errors.New("Enter up to 64 DNS records, one domain and IP per line")
		}
		name := strings.TrimSuffix(strings.ToLower(fields[0]), ".")
		if _, ok := dns.IsDomainName(name); !ok || !strings.Contains(name, ".") ||
			strings.ContainsAny(name, "*_/\\:") || net.ParseIP(name) != nil {
			return nil, errors.New("Invalid domain in DNS override")
		}
		records = append(records, configs.SingleCustomDNS{HostName: name, IP: net.ParseIP(fields[1]).String()})
	}
	return records, nil
}

func (d DNSOverrides) apply(conf *configs.Config) {
	if !d.Enabled || d.validate() != nil {
		return
	}
	if remote := strings.TrimSpace(d.Remote); remote != "" {
		conf.RemoteDNSServer = remote
		conf.DisableRemoteDNS = false
	}
	if secondary := strings.TrimSpace(d.Secondary); secondary != "" {
		conf.SecondaryDNSServer = secondary
	}
	if d.TTL != 0 {
		conf.DNSTTL = uint64(d.TTL)
	}
	conf.CustomDNSList, _ = d.parseRecords()
}
