package main

import (
	"testing"

	"github.com/nnu-connect/nnu-connect/configs"
)

func TestGlobalDNSOverridesApplyOnlyWhenEnabled(t *testing.T) {
	route := configs.Config{RemoteDNSServer: "auto", SecondaryDNSServer: "1.1.1.1", DNSTTL: 300, DisableRemoteDNS: true}
	override := DNSOverrides{Remote: "10.0.0.53", TTL: 60, Records: "PORTAL.EXAMPLE. 192.0.2.9"}
	override.apply(&route)
	if route.RemoteDNSServer != "auto" || route.DNSTTL != 300 || len(route.CustomDNSList) != 0 {
		t.Fatal("disabled preferences changed the route")
	}
	override.Enabled = true
	override.apply(&route)
	if route.RemoteDNSServer != "10.0.0.53" || route.SecondaryDNSServer != "1.1.1.1" || route.DisableRemoteDNS || route.DNSTTL != 60 {
		t.Fatal("global values did not override the corresponding route options")
	}
	if len(route.CustomDNSList) != 1 || route.CustomDNSList[0].HostName != "portal.example" {
		t.Fatal("hostname override was not normalized")
	}
	if err := (DNSOverrides{Records: "portal.example 198.18.0.1"}).validate(); err == nil {
		t.Fatal("accepted a synthetic address as a static override")
	}
}
