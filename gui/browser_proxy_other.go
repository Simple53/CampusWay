//go:build !windows

package main

import "errors"

func restoreBrowserProxy(string) error { return nil }
func enableBrowserProxy(string, Profile) (func() error, error) {
	return nil, errors.New("Configure the CampusWay HTTP proxy in your browser on this platform")
}
