//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const internetSettings = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type proxyBackup struct {
	URL          string `json:"url"`
	HadURL       bool   `json:"hadURL"`
	Enabled      uint32 `json:"enabled"`
	HadEnabled   bool   `json:"hadEnabled"`
	InstalledURL string `json:"installedURL"`
}

func notifyProxyChange() {
	proc := windows.NewLazySystemDLL("wininet.dll").NewProc("InternetSetOptionW")
	proc.Call(0, 39, 0, 0)
	proc.Call(0, 37, 0, 0)
	// WinHTTP is deliberately untouched; only browsers using system proxy settings
	// are routed, and the VPN worker continues using its direct transport.
}

func manualProxyFallback(server string, enabled bool) string {
	if !enabled || server == "" {
		return "DIRECT"
	}
	if !strings.Contains(server, "=") {
		return "PROXY " + server
	}
	return server
}

func restoreBrowserProxy(dir string) error {
	path := filepath.Join(dir, "browser-proxy-backup.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var backup proxyBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettings, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	currentURL, _, _ := key.GetStringValue("AutoConfigURL")
	// A later user/Clash change wins over our old backup.
	if currentURL == backup.InstalledURL {
		if backup.HadURL {
			err = key.SetStringValue("AutoConfigURL", backup.URL)
		} else {
			err = key.DeleteValue("AutoConfigURL")
		}
		if err != nil && err != registry.ErrNotExist {
			return err
		}
		if backup.HadEnabled {
			err = key.SetDWordValue("ProxyEnable", backup.Enabled)
		} else {
			err = key.DeleteValue("ProxyEnable")
		}
		if err != nil && err != registry.ErrNotExist {
			return err
		}
		notifyProxyChange()
	}
	return os.Remove(path)
}

func enableBrowserProxy(dir string, p Profile) (func() error, error) {
	if err := restoreBrowserProxy(dir); err != nil {
		return nil, err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettings, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return nil, err
	}
	defer key.Close()
	previousURL, _, urlErr := key.GetStringValue("AutoConfigURL")
	if previousURL != "" {
		return nil, errors.New("An existing browser PAC is configured; configure the CampusWay proxy in your browser")
	}
	enabled, _, enabledErr := key.GetIntegerValue("ProxyEnable")
	server, _, _ := key.GetStringValue("ProxyServer")
	url, closeServer, err := serveBrowserPAC(p, manualProxyFallback(server, enabled != 0))
	if err != nil {
		return nil, err
	}
	backup := proxyBackup{previousURL, urlErr == nil, uint32(enabled), enabledErr == nil, url}
	data, _ := json.Marshal(backup)
	path := filepath.Join(dir, "browser-proxy-backup.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		closeServer()
		return nil, err
	}
	if err = key.SetStringValue("AutoConfigURL", url); err == nil {
		err = key.SetDWordValue("ProxyEnable", 0)
	}
	if err != nil {
		_ = restoreBrowserProxy(dir)
		closeServer()
		return nil, err
	}
	notifyProxyChange()
	return func() error { err := restoreBrowserProxy(dir); closeServer(); return err }, nil
}
