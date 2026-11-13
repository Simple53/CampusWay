package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleProfile() Profile {
	return Profile{Name: "Campus", Server: "vpn.example.edu", Port: 443, Protocol: "easyconnect", Username: "student"}
}
func TestProfilesPersistWithoutSecrets(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.Remember = canRemember()
	saved, err := s.SaveProfile(p, "sentinel-password")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.dir, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "sentinel-password") {
		t.Fatal("plaintext password persisted")
	}
	if strings.Contains(string(mustJSON(t, s.GetSnapshot())), "sentinel-password") {
		t.Fatal("secret leaked to UI")
	}
	reopened := newAppAt(s.dir)
	if len(reopened.GetSnapshot().Profiles) != 1 {
		t.Fatal("profile not restored")
	}
	saved.Server = "other.example.edu"
	saved, err = s.SaveProfile(saved, "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.HasPassword {
		t.Fatal("password reused after gateway change")
	}
	if err = s.DeleteProfile(saved.ID); err != nil {
		t.Fatal(err)
	}
	if len(newAppAt(s.dir).GetSnapshot().Profiles) != 0 {
		t.Fatal("delete not persisted")
	}
}
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRejectUnknownIdentityAndInvalidGateway(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.ID = "../outside"
	if _, err := s.SaveProfile(p, ""); err == nil {
		t.Fatal("unknown ID accepted")
	}
	for _, host := range []string{"https://vpn.example.edu", "host/secret", "user@host", "a:123", "-invalid.example"} {
		p = sampleProfile()
		p.Server = host
		if _, err := s.SaveProfile(p, ""); err == nil {
			t.Fatalf("accepted %q", host)
		}
	}
}
func TestPreferencesAndBoundedLogCursor(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := Preferences{Accent: "graphite", Language: "zh-CN", Theme: "dark", View: "table", Density: "compact", Sort: "name"}
	if err := s.SavePreferences(p); err != nil {
		t.Fatal(err)
	}
	if newAppAt(s.dir).GetSnapshot().Preferences != p {
		t.Fatal("preferences lost")
	}
	for i := 0; i < 510; i++ {
		s.record("disconnected", "test")
	}
	snap := s.GetSnapshot()
	if len(snap.Logs) != 500 || snap.Logs[0].Seq != 11 || snap.Logs[499].Seq != 510 {
		t.Fatal("log cursor stopped advancing")
	}
	p.Theme = "arbitrary"
	if err := s.SavePreferences(p); err == nil {
		t.Fatal("invalid theme accepted")
	}
}
func TestDamagedStoreIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	s := newAppAt(dir)
	if _, err := s.SaveProfile(sampleProfile(), ""); err == nil {
		t.Fatal("corrupt data overwritten")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "damaged" {
		t.Fatal("source lost")
	}
}
func TestConnectionOperationBlocksEditing(t *testing.T) {
	s := newAppAt(t.TempDir())
	s.operation.Lock()
	defer s.operation.Unlock()
	if _, err := s.SaveProfile(sampleProfile(), ""); err == nil {
		t.Fatal("edit raced with switch")
	}
}

func TestRouteExportImportPrivacy(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.Protocol = "atrust"
	p.Options = ConnectionOptions{LoginDomain: "campus", SocksPort: 2080, HTTPPort: 2081}
	if _, err := s.SaveProfile(p, "secret-sentinel"); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportData()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(data, "student") || strings.Contains(data, "secret-sentinel") {
		t.Fatal("export contains identity or password")
	}
	dest := newAppAt(t.TempDir())
	if err := dest.ImportData(data); err != nil {
		t.Fatal(err)
	}
	got := dest.GetSnapshot().Profiles[0]
	if got.Username != "" || got.Remember || got.HasPassword || got.Options.SocksPort != 2080 || got.Protocol != "atrust" {
		t.Fatal("unsafe or lossy import")
	}
	if _, err := dest.SaveProfile(got, ""); err == nil {
		t.Fatal("import draft accepted without account")
	}
}
func TestConnectionOptionsAndLanguage(t *testing.T) {
	s := newAppAt(t.TempDir())
	if s.GetSnapshot().Preferences.Language != "zh-CN" {
		t.Fatal("wrong default language")
	}
	p := sampleProfile()
	p.Options.SocksPort = 1081
	p.Options.HTTPPort = 1081
	if _, e := s.SaveProfile(p, ""); e == nil {
		t.Fatal("conflicting proxy ports accepted")
	}
	prefs := s.GetSnapshot().Preferences
	prefs.Language = "en"
	if e := s.SavePreferences(prefs); e != nil {
		t.Fatal(e)
	}
	if newAppAt(s.dir).GetSnapshot().Preferences.Language != "en" {
		t.Fatal("language not persisted")
	}
}

func TestTrafficSnapshotDoesNotExposeMutableState(t *testing.T) {
	s := newAppAt(t.TempDir())
	s.traffic = &TrafficSnapshot{Uploaded: 17, Downloaded: 23}
	got := s.GetSnapshot()
	got.Traffic.Uploaded = 99
	if s.GetSnapshot().Traffic.Uploaded != 17 {
		t.Fatal("snapshot shares mutable traffic state")
	}
}

func TestColorThemesPersistAndRejectUnknown(t *testing.T) {
	s := newAppAt(t.TempDir())
	for _, accent := range []string{"graphite", "ocean", "jade", "violet", "rose", "amber"} {
		p := s.GetSnapshot().Preferences
		p.Accent = accent
		if e := s.SavePreferences(p); e != nil {
			t.Fatal(e)
		}
		if newAppAt(s.dir).GetSnapshot().Preferences.Accent != accent {
			t.Fatal("accent lost", accent)
		}
	}
	p := s.GetSnapshot().Preferences
	p.Accent = "invalid"
	if e := s.SavePreferences(p); e == nil {
		t.Fatal("unknown accent accepted")
	}
}
