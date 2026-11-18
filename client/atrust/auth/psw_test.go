package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordUsesSelectedDomainAndReturnsGatewayRejection(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["username"] != "student@Radius" {
			t.Errorf("username = %q", body["username"])
		}
		_, _ = io.WriteString(w, `{"code":75500000,"message":"The username or password is incorrect. You still have 7 attempts left","data":{"graphCheckCodeEnable":0}}`)
	}))
	defer server.Close()
	s := NewSession("unused")
	s.baseURL, s.client, s.pubKey, s.pubKeyExp = server.URL, server.Client(), key.N.Text(16), "65537"
	err = (PasswordLogin{Username: "student", Password: "password", Domain: ""}).login(s, AuthInfo{LoginDomain: "Radius"})
	if err == nil || !strings.Contains(err.Error(), "7 attempts left") || strings.Contains(err.Error(), "ticket is empty") {
		t.Fatalf("lost gateway rejection: %v", err)
	}
	if calls != 1 {
		t.Fatalf("credential rejection retried %d times", calls)
	}
}
