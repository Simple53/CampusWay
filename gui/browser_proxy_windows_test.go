//go:build windows

package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// This check changes the real system browser proxy briefly and is opt-in.
func TestSystemBrowserProxyRoundTrip(t *testing.T) {
	if os.Getenv("CAMPUSWAY_TEST_SYSTEM_PROXY") != "1" {
		t.Skip("opt-in system proxy check")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettings, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	oldURL, _, oldURLErr := key.GetStringValue("AutoConfigURL")
	oldEnabled, _, oldEnabledErr := key.GetIntegerValue("ProxyEnable")
	if oldURL != "" {
		t.Skip("another PAC is active")
	}
	p := sampleProfile()
	p.Server = "vpn.nnu.edu.cn"
	p.Options.HTTPPort = 1081
	restore, err := enableBrowserProxy(t.TempDir(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()
	url, _, err := key.GetStringValue("AutoConfigURL")
	if err != nil || !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatal("system PAC was not set")
	}
	client := http.Client{Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	client.CloseIdleConnections()
	if err != nil || !strings.Contains(string(data), "njnu.edu.cn") {
		t.Fatal("campus PAC unavailable")
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	gotURL, _, gotURLErr := key.GetStringValue("AutoConfigURL")
	gotEnabled, _, gotEnabledErr := key.GetIntegerValue("ProxyEnable")
	if gotURL != oldURL || (gotURLErr == nil) != (oldURLErr == nil) || gotEnabled != oldEnabled || (gotEnabledErr == nil) != (oldEnabledErr == nil) {
		t.Fatal("original browser proxy settings were not restored")
	}
}
