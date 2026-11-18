package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdatesOnlyOfferNewerSemanticVersions(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		newer           bool
	}{
		{"0.1.2", "v0.1.1", false}, {"v0.1.2", "v0.1.2", false}, {"0.1.9", "v0.1.10", true},
		{"0.1.2", "0.1.2-beta.1", false}, {"0.1.2-beta.2", "0.1.2-beta.10", true},
		{"0.1.2+local", "v0.1.2", false}, {"dev", "v0.1.2", false},
	} {
		if got := releaseStatus(tc.current, tc.latest) == "A different release is available"; got != tc.newer {
			t.Errorf("%s -> %s: newer=%v", tc.current, tc.latest, got)
		}
	}
}

func TestEasyConnectMigratesTCPOnlyAndInvalidatesOldSessions(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.Options.TCPOnly = true
	p, err := s.SaveProfile(p, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Options.TCPOnly {
		t.Fatal("unsupported TCP-only option retained")
	}
	// Simulate a profile written by the previous release.
	s.config.Profiles[0].Options.TCPOnly = true
	if err := s.persist(s.config); err != nil {
		t.Fatal(err)
	}
	reopened := newAppAt(s.dir)
	if reopened.config.Profiles[0].Options.TCPOnly {
		t.Fatal("old option not migrated")
	}
	if canRemember() {
		if err := saveCredential(s.sessionPath(p.ID), `{"twfID":"secret-session"}`); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(mustJSON(t, s.GetSnapshot())), "secret-session") {
			t.Fatal("session leaked to UI")
		}
		p.Username = "other-student"
		if _, err := s.SaveProfile(p, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(s.sessionPath(p.ID)); !os.IsNotExist(err) {
			t.Fatal("old identity session retained")
		}
	}
}

func TestLogsHaveLevels(t *testing.T) {
	s := newAppAt(t.TempDir())
	s.warn("VPN connected")
	s.warn("[warn] DNS fallback")
	s.record("error", "Connection rejected")
	logs := s.GetSnapshot().Logs
	for i, want := range []string{"info", "warn", "error"} {
		if logs[i].Level != want {
			t.Errorf("level=%s want=%s", logs[i].Level, want)
		}
	}
	data, _ := os.ReadFile(filepath.Join(s.dir, "logs", logs[0].Time[:10]+".log"))
	if len(data) == 0 {
		t.Fatal("daily log missing")
	}
}

func TestPACRoutesBothNNUDomainsAndKeepsClashFallback(t *testing.T) {
	p := sampleProfile()
	p.Server = "vpn.nnu.edu.cn"
	p.Options.HTTPPort = 1081
	pac := browserPAC(p, "PROXY 127.0.0.1:7897")
	for _, want := range []string{`"nnu.edu.cn"`, `"njnu.edu.cn"`, `PROXY 127.0.0.1:1081`, `PROXY 127.0.0.1:7897`} {
		if !strings.Contains(pac, want) {
			t.Fatal("missing PAC rule", want)
		}
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required to execute PAC assertions")
	}
	cmd := exec.Command("node", "-e", `const vm=require('node:vm'),fs=require('node:fs'),assert=require('node:assert/strict');
const sandbox={dnsDomainIs:(host,suffix)=>host.endsWith(suffix)};
vm.createContext(sandbox);vm.runInContext(fs.readFileSync(0,'utf8'),sandbox);
for(const host of ['zgzj.njnu.edu.cn','ehallapp.nnu.edu.cn','NNU.EDU.CN.']) assert.equal(sandbox.FindProxyForURL('http://'+host,host),'PROXY 127.0.0.1:1081');
assert.equal(sandbox.FindProxyForURL('http://evilnnu.edu.cn','evilnnu.edu.cn'),'PROXY 127.0.0.1:7897');
assert.equal(sandbox.FindProxyForURL('http://localhost','localhost'),'DIRECT');`)
	cmd.Stdin = strings.NewReader(pac)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("PAC execution: %v\n%s", err, output)
	}
}
