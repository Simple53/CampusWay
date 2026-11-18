//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWindowsUpdateSelectsDesktopAsset(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("installed desktop asset is x64")
	}
	desktop := "RouteDesk-9.0.1-windows-x64-portable.zip"
	release := githubRelease{Tag: "v9.0.1", Assets: []releaseAsset{
		{Name: "routedesk-cli-9.0.1-windows-x64.zip"},
		{Name: "RouteDesk-9.0.1-windows-x64-setup.exe"},
		{Name: desktop},
	}}
	if asset, ok := desktopUpdateAsset(release); !ok || asset.Name != desktop {
		t.Fatalf("selected asset: %+v, %t", asset, ok)
	}
}

func TestWindowsUpdateReplacesFileAndKeepsRollback(t *testing.T) {
	dir := t.TempDir()
	target, stage, backup := filepath.Join(dir, "current.exe"), filepath.Join(dir, "new.exe"), filepath.Join(dir, "previous.exe")
	for path, body := range map[string]string{target: "old", stage: "new"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := replaceUpdateFile(target, stage, backup); err != nil {
		t.Fatal(err)
	}
	for path, expected := range map[string]string{target: "new", backup: "old"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != expected {
			t.Fatalf("%s = %q, %v", path, got, err)
		}
	}
	if err := replaceUpdateFile(target, backup, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "old" {
		t.Fatalf("rollback = %q, %v", got, err)
	}
}
