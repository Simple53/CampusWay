package main

import (
	"errors"
	"net"
	"strings"
	"testing"
)

// A password entered at connect time must survive a restart and never reach
// profiles.json in plaintext; this is the regression guard for "cannot save".
func TestRememberPasswordRoundTrip(t *testing.T) {
	if !canRemember() {
		t.Skip("credential storage unavailable on this platform")
	}
	s := newAppAt(t.TempDir())
	p, err := s.SaveProfile(sampleProfile(), "")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	err = s.rememberPassword(p.ID, "s3cret!")
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	reopened := newAppAt(s.dir)
	snap := reopened.GetSnapshot()
	if len(snap.Profiles) != 1 || !snap.Profiles[0].Remember || !snap.Profiles[0].HasPassword {
		t.Fatalf("remember flags lost after restart: %+v", snap.Profiles)
	}
	if got, err := loadCredential(reopened.secretPath(p.ID)); err != nil || got != "s3cret!" {
		t.Fatalf("credential did not round trip: %q %v", got, err)
	}
}

func TestWidgetPreferencesValidated(t *testing.T) {
	s := newAppAt(t.TempDir())
	p := Preferences{Accent: "graphite", Language: "zh-CN", Theme: "system", View: "grid", Density: "compact", Sort: "default"}
	p.Widgets = []string{"logs", "bogus", "status", "logs"}
	if err := s.SavePreferences(p); err != nil {
		t.Fatal(err)
	}
	got := s.GetSnapshot().Preferences.Widgets
	if len(got) != 2 || got[0] != "logs" || got[1] != "status" {
		t.Fatalf("order not kept or junk not dropped: %v", got)
	}
	p.Widgets = []string{}
	if err := s.SavePreferences(p); err != nil {
		t.Fatal(err)
	}
	if got := s.GetSnapshot().Preferences.Widgets; got == nil || len(got) != 0 {
		t.Fatalf("an empty widget list must be allowed, got %v", got)
	}
}

func TestExplainConnectErrorNeverLeaksGatewayResponse(t *testing.T) {
	msg := explainConnectError("vpn.example.edu", errors.New("Login failed: <Result>0</Result><Password>hunter2</Password>"))
	if msg != "Gateway rejected the account or password" {
		t.Fatalf("unexpected message %q", msg)
	}
	if got := explainConnectError("vpn.example.edu", errors.New("EOF")); got != "Gateway is unreachable or closed the connection" && got != "Gateway traffic is intercepted by a local proxy (fake-IP DNS)" {
		t.Fatalf("transport error not classified: %q", got)
	}
}

func TestFakeIPRange(t *testing.T) {
	if !fakeIPNet.Contains(net.ParseIP("198.18.0.174")) || fakeIPNet.Contains(net.ParseIP("10.0.0.1")) {
		t.Fatal("fake IP range misclassifies addresses")
	}
}

// 只要用户在表单中填写了密码，无论是否显式勾选记住密码，都必须自动持久化保存且不可丢失
func TestSaveProfileWithPasswordAlwaysRemembers(t *testing.T) {
	if !canRemember() {
		t.Skip("credential storage unavailable on this platform")
	}
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.Remember = false
	saved, err := s.SaveProfile(p, "autoSavePass123")
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Remember || !saved.HasPassword {
		t.Fatalf("expected Remember and HasPassword to be true, got remember=%v hasPass=%v", saved.Remember, saved.HasPassword)
	}
	cred, err := loadCredential(s.secretPath(saved.ID))
	if err != nil || cred != "autoSavePass123" {
		t.Fatalf("saved credential mismatch: %q, err: %v", cred, err)
	}
}

// 当用户修改了账号（身份标识改变）且密码留空时，必须将原已保存的密码完整迁移到新凭据槽位，绝不丢失
func TestSaveProfileIdentityChangeKeepsPassword(t *testing.T) {
	if !canRemember() {
		t.Skip("credential storage unavailable on this platform")
	}
	s := newAppAt(t.TempDir())
	p := sampleProfile()
	p.Username = "initialUser"
	saved, err := s.SaveProfile(p, "keepSecret456")
	if err != nil {
		t.Fatal(err)
	}

	// 用户仅修改用户名，密码字段留空（代表保留旧密码）
	p2 := saved
	p2.Username = "updatedUser"
	updated, err := s.SaveProfile(p2, "")
	if err != nil {
		t.Fatal(err)
	}
	if !updated.HasPassword || !updated.Remember {
		t.Fatalf("expected HasPassword=true after username change with blank password, got %v", updated.HasPassword)
	}
	cred, err := loadCredential(s.secretPath(updated.ID))
	if err != nil || cred != "keepSecret456" {
		t.Fatalf("credential lost after username change: %q, err: %v", cred, err)
	}
}

// 验证 logPipeWriter 能够正确按行切割日志并严格对用户密码执行脱敏屏蔽，防止凭据泄露
func TestLogPipeWriterMasksPasswordAndSplitsLines(t *testing.T) {
	var captured []string
	writer := &logPipeWriter{
		emit: func(e Event) {
			if e.Kind == "log" {
				captured = append(captured, e.Message)
			}
		},
		password: "SuperSecretPassword123!",
	}

	raw := "Request: https://vpn.nnu.edu.cn/login?pass=SuperSecretPassword123!\r\nAdd route 10.0.0.0/8\n"
	n, err := writer.Write([]byte(raw))
	if err != nil || n != len(raw) {
		t.Fatalf("unexpected write result: n=%d, err=%v", n, err)
	}

	if len(captured) != 2 {
		t.Fatalf("expected 2 lines captured, got %d: %v", len(captured), captured)
	}

	if strings.Contains(captured[0], "SuperSecretPassword123!") {
		t.Fatalf("password leaked in log output: %q", captured[0])
	}
	if !strings.Contains(captured[0], "******") {
		t.Fatalf("expected password to be masked with '******', got %q", captured[0])
	}
	if captured[1] != "Add route 10.0.0.0/8" {
		t.Fatalf("expected route line preserved, got %q", captured[1])
	}
}
