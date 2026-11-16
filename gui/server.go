package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Connection options group protocol, DNS and local-proxy settings per route.
type ConnectionOptions struct {
	AuthType            string `json:"authType"`
	LoginDomain         string `json:"loginDomain"`
	SocksPort           int    `json:"socksPort"`
	HTTPPort            int    `json:"httpPort"`
	RemoteDNS           string `json:"remoteDNS"`
	SecondaryDNS        string `json:"secondaryDNS"`
	DNSTTL              int    `json:"dnsTTL"`
	TCPOnly             bool   `json:"tcpOnly"`
	DisableRemoteDNS    bool   `json:"disableRemoteDNS"`
	DisableMultiLine    bool   `json:"disableMultiLine"`
	DisableServerConfig bool   `json:"disableServerConfig"`
	SkipDomainResource  bool   `json:"skipDomainResource"`
	ProxyAll            bool   `json:"proxyAll"`
	UpdateInterval      int    `json:"updateInterval"`
}
type Profile struct {
	Options       ConnectionOptions `json:"options"`
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Server        string            `json:"server"`
	Port          int               `json:"port"`
	Protocol      string            `json:"protocol"`
	Username      string            `json:"username"`
	CustomDomains string            `json:"customDomains"`
	Remember      bool              `json:"remember"`
	HasPassword   bool              `json:"hasPassword"`
}
type Preferences struct {
	Accent      string            `json:"accent"`
	Language    string            `json:"language"`
	Theme       string            `json:"theme"`
	View        string            `json:"view"`
	Density     string            `json:"density"`
	Sort        string            `json:"sort"`
	Widgets     []string          `json:"widgets"`
	WidgetSizes map[string]string `json:"widgetSizes"`
	CloseAction string            `json:"closeAction"`
}
type storedConfig struct {
	Version     int         `json:"version"`
	Profiles    []Profile   `json:"profiles"`
	Selected    string      `json:"selected"`
	Preferences Preferences `json:"preferences"`
}
type TrafficSnapshot struct {
	Uploaded     uint64  `json:"uploaded"`
	Downloaded   uint64  `json:"downloaded"`
	UploadRate   float64 `json:"uploadRate"`
	DownloadRate float64 `json:"downloadRate"`
	SampledAt    int64   `json:"sampledAt"`
}
type Event struct {
	Traffic *TrafficSnapshot `json:"traffic,omitempty"`
	Kind    string           `json:"kind"`
	State   string           `json:"state,omitempty"`
	Message string           `json:"message,omitempty"`
	ID      string           `json:"id,omitempty"`
	Image   string           `json:"image,omitempty"`
}
type LogEntry struct {
	Seq     uint64 `json:"seq"`
	Time    string `json:"time"`
	Message string `json:"message"`
}
type Snapshot struct {
	Traffic     *TrafficSnapshot `json:"traffic"`
	Profiles    []Profile        `json:"profiles"`
	Preferences Preferences      `json:"preferences"`
	Selected    string           `json:"selected"`
	Active      string           `json:"active"`
	State       string           `json:"state"`
	Message     string           `json:"message"`
	ConnectedAt int64            `json:"connectedAt"`
	Challenge   *Event           `json:"challenge"`
	Logs        []LogEntry       `json:"logs"`
	CanRemember bool             `json:"canRemember"`
}
type AppServer struct {
	traffic                *TrafficSnapshot
	mu                     sync.Mutex
	operation              sync.Mutex
	dir                    string
	config                 storedConfig
	state, message, active string
	connectedAt            int64
	logs                   []LogEntry
	seq                    uint64
	challenge              *Event
	cmd                    *exec.Cmd
	input                  io.WriteCloser
	done                   chan struct{}
	cancelled              bool
	startupErr             error
	resize                 func(bool)
	minimize               func()
}

func newAppAt(dir string) *AppServer {
	s := &AppServer{dir: dir, state: "disconnected", message: "Ready to connect", config: storedConfig{Version: 1, Profiles: []Profile{}, Preferences: Preferences{Accent: "graphite", Language: "zh-CN", Theme: "system", View: "grid", Density: "compact", Sort: "default", Widgets: []string{"status", "speed", "diag", "logs"}, WidgetSizes: map[string]string{}, CloseAction: "minimize"}}, logs: []LogEntry{}}
	if err := os.MkdirAll(dir, 0700); err != nil {
		s.startupErr = err
		return s
	}
	data, err := os.ReadFile(filepath.Join(dir, "profiles.json"))
	if err == nil {
		if err = json.Unmarshal(data, &s.config); err != nil {
			s.startupErr = fmt.Errorf("Cannot read saved profiles: %w", err)
		}
	} else if !os.IsNotExist(err) {
		s.startupErr = err
	}
	if s.config.Version != 1 {
		s.startupErr = errors.New("Unsupported profile version; original data was preserved")
	}
	if s.config.Profiles == nil {
		s.config.Profiles = []Profile{}
	}
	if s.config.Preferences.Accent == "" {
		s.config.Preferences.Accent = "graphite"
	}
	if s.config.Preferences.Language == "" {
		s.config.Preferences.Language = "zh-CN"
	}
	for i := range s.config.Profiles {
		p := &s.config.Profiles[i]
		if len(p.ID) != 32 {
			s.startupErr = errors.New("Invalid saved profile identifier")
			continue
		}
		if _, err := hex.DecodeString(p.ID); err != nil {
			s.startupErr = err
			continue
		}
		p.HasPassword = false
		if p.Remember {
			_, err := os.Stat(s.secretPath(p.ID))
			p.HasPassword = err == nil
		}
	}
	return s
}
func newApp() *AppServer {
	dir, err := os.UserConfigDir()
	if err != nil {
		s := newAppAt("CampusWay")
		s.startupErr = err
		return s
	}
	s := newAppAt(filepath.Join(dir, "CampusWay"))
	// Legacy accounts are imported only through an explicit user action.

	return s
}
func identifier() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func (s *AppServer) secretPath(id string) string {
	return filepath.Join(s.dir, "credential-"+id+".bin")
}
func validateProfile(p *Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Server = strings.TrimSpace(p.Server)
	p.Username = strings.TrimSpace(p.Username)
	if p.Name == "" || len(p.Name) > 80 {
		return errors.New("Enter a profile name (up to 80 characters)")
	}
	if p.Username == "" || len(p.Username) > 256 {
		return errors.New("Enter your campus account")
	}
	if p.Port < 1 || p.Port > 65535 {
		return errors.New("Port must be between 1 and 65535")
	}
	if p.Protocol != "easyconnect" && p.Protocol != "atrust" {
		return errors.New("Unsupported VPN protocol")
	}
	if p.Server == "" || len(p.Server) > 253 || strings.ContainsAny(p.Server, "/\\@?# \t\r\n") {
		return errors.New("Enter a hostname or IP address, without a URL scheme")
	}
	if net.ParseIP(p.Server) == nil {
		for _, label := range strings.Split(p.Server, ".") {
			if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return errors.New("Invalid gateway hostname")
			}
			for _, r := range label {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
					return errors.New("Invalid gateway hostname")
				}
			}
		}
	}
	if len(p.CustomDomains) > 8192 {
		return errors.New("Too many domain rules")
	}

	o := &p.Options
	if o.AuthType == "" {
		o.AuthType = "auth/psw"
	}
	if !oneOf(o.AuthType, "auth/psw", "auth/smsCheckCode") {
		return errors.New("Unsupported authentication method")
	}
	if o.SocksPort == 0 {
		o.SocksPort = 1080
	}
	if o.HTTPPort == 0 {
		o.HTTPPort = 1081
	}
	if o.SocksPort < 1 || o.SocksPort > 65535 || o.HTTPPort < 1 || o.HTTPPort > 65535 || o.SocksPort == o.HTTPPort {
		return errors.New("Invalid or conflicting proxy ports")
	}
	if o.RemoteDNS == "" {
		o.RemoteDNS = "auto"
	}
	if o.SecondaryDNS == "" {
		o.SecondaryDNS = "1.1.1.1"
	}
	for _, host := range []string{o.RemoteDNS, o.SecondaryDNS} {
		if host != "auto" && net.ParseIP(host) == nil {
			return errors.New("DNS must be an IP address or auto")
		}
	}
	if o.DNSTTL == 0 {
		o.DNSTTL = 300
	}
	if o.DNSTTL < 1 || o.DNSTTL > 86400 {
		return errors.New("Invalid DNS TTL")
	}
	if o.UpdateInterval == 0 {
		o.UpdateInterval = 300
	}
	if o.UpdateInterval < 10 || o.UpdateInterval > 86400 {
		return errors.New("Invalid node update interval")
	}
	if len(o.LoginDomain) > 256 {
		return errors.New("Login domain is too long")
	}
	return nil
}
func (s *AppServer) persist(c storedConfig) error {
	if s.startupErr != nil {
		return s.startupErr
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".profiles-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, filepath.Join(s.dir, "profiles.json"))
}
func (s *AppServer) SaveProfile(p Profile, password string) (Profile, error) {
	if !s.operation.TryLock() {
		return Profile{}, errors.New("Wait for the current connection operation")
	}
	defer s.operation.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateProfile(&p); err != nil {
		return Profile{}, err
	}
	if s.startupErr != nil {
		return Profile{}, s.startupErr
	}
	if p.Remember && !canRemember() {
		return Profile{}, errors.New("Password storage is unavailable on this platform; use a session password")
	}
	c := s.config
	c.Profiles = append([]Profile{}, c.Profiles...)
	index := -1
	if p.ID == "" {
		p.ID = identifier()
	} else {
		for i, q := range c.Profiles {
			if q.ID == p.ID {
				index = i
			}
		}
		if index < 0 {
			return Profile{}, errors.New("Profile no longer exists")
		}
	}
	if p.ID == s.active && s.cmd != nil {
		return Profile{}, errors.New("Disconnect before editing the active profile")
	}
	serverChanged := index >= 0 && c.Profiles[index].Server != p.Server
	changed := index >= 0 && (serverChanged || c.Profiles[index].Port != p.Port || c.Profiles[index].Username != p.Username || c.Profiles[index].Protocol != p.Protocol || c.Profiles[index].Options.LoginDomain != p.Options.LoginDomain || (c.Profiles[index].Options.AuthType != "" && c.Profiles[index].Options.AuthType != p.Options.AuthType))
	oldID := ""
	if index >= 0 {
		oldID = c.Profiles[index].ID
	}
	// Identity changes get a fresh credential slot before any secret is written.
	if changed {
		p.ID = identifier()
	}
	p.HasPassword = false
	if p.Remember || password != "" {
		p.Remember = true
		if password == "" && !changed && index >= 0 {
			p.HasPassword = c.Profiles[index].HasPassword
		}
		if password == "" && changed && !serverChanged && index >= 0 && c.Profiles[index].HasPassword {
			// 网关未改变（如修改了账号、端口或选项）且密码留空保留，则安全迁移旧凭据至新槽位
			if oldSecret, err := loadCredential(s.secretPath(oldID)); err == nil && oldSecret != "" {
				if err := saveCredential(s.secretPath(p.ID), oldSecret); err == nil {
					p.HasPassword = true
				}
			}
		}
		if password != "" {
			if err := saveCredential(s.secretPath(p.ID), password); err != nil {
				return Profile{}, err
			}
			// DPAPI 加密即时回读校验
			if got, err := loadCredential(s.secretPath(p.ID)); err == nil && got == password {
				p.HasPassword = true
			} else {
				_ = deleteCredential(s.secretPath(p.ID))
				return Profile{}, errors.New("stored credential failed verification")
			}
		}
	}
	if !p.Remember && oldID != "" {
		if err := deleteCredential(s.secretPath(oldID)); err != nil && !os.IsNotExist(err) {
			return Profile{}, err
		}
	}
	if index < 0 {
		c.Profiles = append(c.Profiles, p)
	} else {
		c.Profiles[index] = p
	}
	c.Selected = p.ID
	if err := s.persist(c); err != nil {
		return Profile{}, err
	}
	s.config = c
	if changed && oldID != "" {
		_ = deleteCredential(s.secretPath(oldID))
	}
	return p, nil
}
func (s *AppServer) DeleteProfile(id string) error {
	if !s.operation.TryLock() {
		return errors.New("Wait for the current connection operation")
	}
	defer s.operation.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.active == id {
		return errors.New("Disconnect before deleting this profile")
	}
	c := s.config
	c.Profiles = []Profile{}
	found := false
	for _, p := range s.config.Profiles {
		if p.ID == id {
			found = true
		} else {
			c.Profiles = append(c.Profiles, p)
		}
	}
	if !found {
		return errors.New("Profile no longer exists")
	}
	if err := deleteCredential(s.secretPath(id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	if c.Selected == id {
		c.Selected = ""
		if len(c.Profiles) > 0 {
			c.Selected = c.Profiles[0].ID
		}
	}
	if err := s.persist(c); err != nil {
		return err
	}
	s.config = c
	return nil
}
func (s *AppServer) SelectProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.config.Profiles {
		if p.ID == id {
			c := s.config
			c.Selected = id
			if err := s.persist(c); err != nil {
				return err
			}
			s.config = c
			return nil
		}
	}
	return errors.New("Profile no longer exists")
}
func (s *AppServer) SavePreferences(p Preferences) error {
	if p.Accent == "" {
		p.Accent = "graphite"
	}
	if !oneOf(p.Accent, "graphite", "ocean", "jade", "violet", "rose", "amber") {
		return errors.New("Invalid color theme")
	}
	if p.Language == "" {
		p.Language = "zh-CN"
	}
	if !oneOf(p.Language, "zh-CN", "en") {
		return errors.New("Unsupported language")
	}
	if !oneOf(p.Theme, "system", "light", "dark") || !oneOf(p.View, "table", "grid") || !oneOf(p.Density, "comfortable", "standard", "compact", "ultra-compact") || !oneOf(p.Sort, "default", "name") {
		return errors.New("Invalid display preferences")
	}
	if p.CloseAction == "" {
		p.CloseAction = "minimize"
	}
	if !oneOf(p.CloseAction, "minimize", "exit") {
		return errors.New("Invalid close action")
	}
	if p.WidgetSizes == nil {
		p.WidgetSizes = map[string]string{}
	}
	cleanSizes := map[string]string{}
	for k, v := range p.WidgetSizes {
		if oneOf(k, "status", "speed", "diag", "logs") && oneOf(v, "half", "full") {
			cleanSizes[k] = v
		}
	}
	p.WidgetSizes = cleanSizes
	// nil means "never customised" (show everything); an empty list is a valid
	// choice and hides every widget. Unknown ids and duplicates are dropped.
	if p.Widgets == nil {
		p.Widgets = []string{"status", "speed", "diag", "logs"}
	}
	seen := map[string]bool{}
	clean := []string{}
	for _, w := range p.Widgets {
		if oneOf(w, "status", "speed", "diag", "logs") && !seen[w] {
			seen[w] = true
			clean = append(clean, w)
		}
	}
	p.Widgets = clean
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.config
	c.Preferences = p
	if err := s.persist(c); err != nil {
		return err
	}
	s.config = c
	return nil
}
// MinimizeWindow 响应前端最小化按钮操作，将窗口最小化或隐藏至托盘
func (s *AppServer) MinimizeWindow() error {
	s.mu.Lock()
	fn := s.minimize
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
	return nil
}

func oneOf(v string, choices ...string) bool {
	for _, c := range choices {
		if v == c {
			return true
		}
	}
	return false
}
func (s *AppServer) GetSnapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := s.message
	if s.startupErr != nil {
		msg = s.startupErr.Error()
	}
	var trafficCopy *TrafficSnapshot
	if s.traffic != nil {
		v := *s.traffic
		trafficCopy = &v
	}
	return Snapshot{Traffic: trafficCopy, Profiles: append([]Profile{}, s.config.Profiles...), Preferences: s.config.Preferences, Selected: s.config.Selected, Active: s.active, State: s.state, Message: msg, ConnectedAt: s.connectedAt, Challenge: s.challenge, Logs: append([]LogEntry{}, s.logs...), CanRemember: canRemember()}
}
func (s *AppServer) record(state, message string) {
	s.state = state
	s.message = message
	s.seq++
	s.logs = append(s.logs, LogEntry{s.seq, time.Now().Format(time.RFC3339), message})
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
}
// rememberPassword stores the credential with DPAPI, reads it back to prove the
// round trip works, and only then flips the profile flags on disk.
// The caller must hold s.mu.
func (s *AppServer) rememberPassword(id, password string) error {
	if err := saveCredential(s.secretPath(id), password); err != nil {
		return err
	}
	if got, err := loadCredential(s.secretPath(id)); err != nil || got != password {
		_ = deleteCredential(s.secretPath(id))
		return errors.New("stored credential failed verification")
	}
	next := s.config
	next.Profiles = append([]Profile{}, s.config.Profiles...)
	for i := range next.Profiles {
		if next.Profiles[i].ID == id {
			next.Profiles[i].Remember, next.Profiles[i].HasPassword = true, true
			if err := s.persist(next); err != nil {
				return err
			}
			s.config = next
			return nil
		}
	}
	return errors.New("Profile no longer exists")
}

// warn appends an event-log line without touching the connection state.
// The caller must hold s.mu.
func (s *AppServer) warn(message string) {
	s.seq++
	s.logs = append(s.logs, LogEntry{s.seq, time.Now().Format(time.RFC3339), message})
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
}

func (s *AppServer) Connect(id, password string, remember bool) error {
	if !s.operation.TryLock() {
		return errors.New("Another connection operation is in progress")
	}
	defer s.operation.Unlock()
	s.mu.Lock()
	var p Profile
	for _, v := range s.config.Profiles {
		if v.ID == id {
			p = v
		}
	}
	if p.ID == "" {
		s.mu.Unlock()
		return errors.New("Choose a saved profile")
	}
	if s.startupErr != nil {
		s.mu.Unlock()
		return s.startupErr
	}
	if s.active == id && s.cmd != nil {
		s.mu.Unlock()
		return errors.New("This profile is already running")
	}
	if password == "" && p.HasPassword {
		var err error
		password, err = loadCredential(s.secretPath(id))
		if err != nil {
			s.mu.Unlock()
			return errors.New("Saved password could not be opened; enter it again")
		}
	}
	if password == "" && !(p.Protocol == "atrust" && p.Options.AuthType == "auth/smsCheckCode") {
		s.mu.Unlock()
		return errors.New("Enter your password")
	}

	// Persist the password when the user asked for it (dialog checkbox) or the
	// profile already opted in. A failure never blocks the connection, but it is
	// recorded in the event log so the user can see why nothing was remembered.
	if password != "" && (remember || p.Remember) && canRemember() {
		if err := s.rememberPassword(id, password); err != nil {
			s.warn("Password could not be saved on this device")
		} else {
			p.Remember, p.HasPassword = true, true
		}
	}

	s.mu.Unlock()
	if err := s.stop(); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, "--engine")
	hideChild(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		in.Close()
		out.Close()
		return err
	}
	if err = cmd.Start(); err != nil {
		in.Close()
		out.Close()
		return err
	}
	s.mu.Lock()
	s.cmd = cmd
	s.input = in
	s.done = make(chan struct{})
	done := s.done
	s.active = id
	s.cancelled = false
	s.challenge = nil
	s.connectedAt = 0
	s.traffic = nil
	s.record("connecting", fmt.Sprintf("正在连接网关: %s (%s, 协议: %s)", p.Name, p.Server, p.Protocol))
	s.mu.Unlock()
	go s.readChild(cmd, out, done)
	go s.readStderr(cmd, errPipe)
	if err = json.NewEncoder(in).Encode(engineRequest{Profile: p, Password: password}); err != nil {
		_ = cmd.Process.Kill()
		return errors.New("Could not initialize the connection process")
	}
	return nil
}

// readStderr 读取引擎子进程的错误输出，并将每一行作为告警日志存入内存日志缓冲区
func (s *AppServer) readStderr(cmd *exec.Cmd, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s.mu.Lock()
		if s.cmd == cmd && !s.cancelled {
			s.warn("[stderr] " + line)
		}
		s.mu.Unlock()
	}
}

func (s *AppServer) readChild(cmd *exec.Cmd, out io.Reader, done chan struct{}) {
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		idx := strings.Index(raw, eventPrefix)
		if idx < 0 {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(strings.TrimSpace(raw[idx+len(eventPrefix):])), &e) != nil {
			continue
		}
		s.mu.Lock()
		if s.cmd == cmd && !s.cancelled {
			switch e.Kind {
			case "traffic":
				if s.state == "connected" {
					s.traffic = e.Traffic
				}
			case "state":
				s.record(e.State, e.Message)
				if e.State == "connected" {
					s.connectedAt = time.Now().UnixMilli()
				}
			case "challenge":
				s.challenge = &e
				s.record("awaiting_auth", e.Message)
			case "log":
				s.warn(e.Message)
			}
		}
		s.mu.Unlock()
	}
	err := cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == cmd {
		if s.cancelled {
			s.record("disconnected", "Disconnected")
		} else if s.state != "error" {
			if err != nil {
				s.record("error", "Connection ended unexpectedly. Check the gateway, credentials and proxy ports.")
			} else {
				s.record("disconnected", "Disconnected")
			}
		}
		s.cmd = nil
		s.input = nil
		s.active = ""
		s.challenge = nil
		s.connectedAt = 0
		s.traffic = nil
	}
	close(done)
}
func (s *AppServer) stop() error {
	s.mu.Lock()
	if s.cmd == nil {
		s.mu.Unlock()
		return nil
	}
	s.cancelled = true
	in, done, cmd := s.input, s.done, s.cmd
	s.challenge = nil
	s.record("disconnecting", "Disconnecting")
	s.mu.Unlock()
	_ = json.NewEncoder(in).Encode(engineCommand{Kind: "stop"})
	select {
	case <-done:
		return nil
	case <-time.After(8 * time.Second):
		// Desktop workers own only process-local proxies; killing one cannot leave system routes behind.
		_ = cmd.Process.Kill()
		select {
		case <-done:
			return nil
		case <-time.After(3 * time.Second):
			return errors.New("Connection process is still stopping")
		}
	}
}
func (s *AppServer) Disconnect() error {
	if !s.operation.TryLock() {
		return errors.New("Wait for the current operation")
	}
	defer s.operation.Unlock()
	return s.stop()
}
func (s *AppServer) SubmitChallenge(id, response string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.challenge == nil || s.challenge.ID != id || s.input == nil {
		return errors.New("This verification request has expired")
	}
	if len(response) == 0 || len(response) > 8192 {
		return errors.New("Enter a valid verification response")
	}
	if err := json.NewEncoder(s.input).Encode(engineCommand{Kind: "answer", ID: id, Answer: response}); err != nil {
		return err
	}
	s.challenge = nil
	s.record("connecting", "Verifying your response")
	return nil
}
func (s *AppServer) SetLayout(manager bool) {
	if s.resize != nil {
		s.resize(manager)
	}
}
