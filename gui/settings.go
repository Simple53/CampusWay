package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type DiagnosticTarget struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	OK   bool   `json:"ok"`
	MS   int64  `json:"ms"`
	Err  string `json:"err,omitempty"`
}

type DiagnosticReport struct {
	Targets []DiagnosticTarget `json:"targets"`
	LocalIP string             `json:"localIP"`
	DNS     string             `json:"dns"`
}

// RunDiagnostics tests connectivity to selected campus & internet targets through the active local proxy.
func (s *AppServer) RunDiagnostics() (DiagnosticReport, error) {
	s.mu.Lock()
	p := s.config.Preferences
	var activeProf *Profile
	for _, prof := range s.config.Profiles {
		if prof.ID == s.active {
			cp := prof
			activeProf = &cp
			break
		}
	}
	s.mu.Unlock()

	httpPort := 1081
	if activeProf != nil && activeProf.Options.HTTPPort > 0 {
		httpPort = activeProf.Options.HTTPPort
	}

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 3 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   4 * time.Second,
	}

	defaultTargets := []struct{ name, targetURL string }{
		{"教务系统 (JWGL)", "http://jwgl.njnu.edu.cn"},
		{"办事大厅 (E-Hall)", "https://ehallapp.nnu.edu.cn"},
		{"中国知网 (CNKI)", "https://www.cnki.net"},
		{"公共网关 (Baidu)", "https://www.baidu.com"},
	}

	report := DiagnosticReport{
		Targets: make([]DiagnosticTarget, 0, len(defaultTargets)),
		DNS:     "127.0.0.1 (VPN DNS / Fallback)",
	}

	for _, dt := range defaultTargets {
		start := time.Now()
		req, err := http.NewRequestWithContext(context.Background(), "GET", dt.targetURL, nil)
		resItem := DiagnosticTarget{Name: dt.name, URL: dt.targetURL}
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 CampusWay/1.0")
			resp, reqErr := client.Do(req)
			elapsed := time.Since(start).Milliseconds()
			resItem.MS = elapsed
			if reqErr == nil {
				resItem.OK = true
				_ = resp.Body.Close()
			} else {
				resItem.OK = false
				resItem.Err = reqErr.Error()
			}
		} else {
			resItem.OK = false
			resItem.Err = err.Error()
		}
		report.Targets = append(report.Targets, resItem)
	}

	// Retrieve local outbound IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		report.LocalIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
		_ = conn.Close()
	} else {
		report.LocalIP = "127.0.0.1"
	}

	_ = p
	return report, nil
}

// Exports deliberately omit usernames and all saved credentials. Imported routes
// remain drafts until the user supplies an account in the connection editor.
func (s *AppServer) ExportData() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles := append([]Profile{}, s.config.Profiles...)
	for i := range profiles {
		profiles[i].Username = ""
		profiles[i].Remember = false
		profiles[i].HasPassword = false
	}
	b, e := json.MarshalIndent(storedConfig{Version: 1, Profiles: profiles, Preferences: s.config.Preferences}, "", "  ")
	return string(b), e
}
func (s *AppServer) ImportData(data string) error {
	if len(data) > 1024*1024 {
		return errors.New("Import exceeds 1 MB")
	}
	var c storedConfig
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return errors.New("Invalid profile file")
	}
	if c.Version != 1 || len(c.Profiles) > 100 {
		return errors.New("Unsupported profile file")
	}
	for i := range c.Profiles {
		p := &c.Profiles[i]
		p.ID = identifier()
		p.Remember = false
		p.HasPassword = false
		// Imported identities are never trusted or silently restored.
		p.Username = "import-draft"
		if err := validateProfile(p); err != nil {
			return err
		}
		p.Username = ""
	}
	if !s.operation.TryLock() {
		return errors.New("Wait for the current connection operation")
	}
	defer s.operation.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.config
	next.Profiles = append(append([]Profile{}, s.config.Profiles...), c.Profiles...)
	if len(next.Profiles) > 100 {
		return errors.New("Too many profiles")
	}
	if err := s.persist(next); err != nil {
		return err
	}
	s.config = next
	return nil
}
func (s *AppServer) ForgetPassword(id string) error {
	if !s.operation.TryLock() {
		return errors.New("Wait for the current connection operation")
	}
	defer s.operation.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.config
	next.Profiles = append([]Profile{}, s.config.Profiles...)
	for i := range next.Profiles {
		if next.Profiles[i].ID == id {
			if err := deleteCredential(s.secretPath(id)); err != nil {
				return err
			}
			next.Profiles[i].Remember = false
			next.Profiles[i].HasPassword = false
			if err := s.persist(next); err != nil {
				return err
			}
			s.config = next
			return nil
		}
	}
	return errors.New("Profile no longer exists")
}

const defaultUpdateRepo = "Simple53/CampusWay"

// Only an explicitly supplied or default CampusWay repository is queried. No account or
// gateway configuration is sent to the update service.
func (s *AppServer) CheckVersion(repository string) (map[string]string, error) {
	if repository == "" {
		repository = defaultUpdateRepo
	}
	result := map[string]string{"current": version, "repository": repository, "status": "Update source is not configured"}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repository) {
		return nil, errors.New("Invalid GitHub repository")
	}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repository+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CampusWay")
	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Unable to check for updates")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New("No published release found")
	}
	var release struct {
		Tag string `json:"tag_name"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&release); err != nil {
		return nil, errors.New("Invalid release response")
	}
	result["latest"] = release.Tag
	result["status"] = "A different release is available"
	if strings.TrimPrefix(release.Tag, "v") == version {
		result["status"] = "You are on the latest release"
	}
	return result, nil
}
