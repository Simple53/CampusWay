<div align="center">
  <img src="build/assets/campusway.png" width="96" height="96" alt="CampusWay Icon" />
  <h1>CampusWay</h1>
  <p><strong>Cross-platform Campus VPN Connection Manager for EasyConnect and aTrust</strong></p>
  <p><strong>支持 EasyConnect 与 aTrust 协议的跨平台校园网连接客户端与多线路管理器</strong></p>
  <p>
    <a href="#key-features">Features</a> ·
    <a href="#downloads--installation">Downloads</a> ·
    <a href="#web-interactive-preview">Web Preview</a> ·
    <a href="README.md">中文文档</a> ·
    <a href="https://simple53.github.io/CampusWay/">Live Preview</a> ·
    <a href="https://github.com/Simple53/CampusWay/releases">GitHub Releases</a>
  </p>
</div>

---

> ⚠️ **Disclaimer & Scope Notes**
>
> This software is provided **as-is**, without any express or implied warranty. Users are responsible for complying with their university network policies.
> - The desktop GUI currently provides **loopback proxy mode** (`SOCKS5 127.0.0.1:1080` and `HTTP 127.0.0.1:1081` by default, configurable per profile). Traffic counters measure TCP traffic forwarded through the local proxies rather than device-wide system interfaces.
> - For full system TUN routing, port forwarding, Docker containers, or headless router/daemon deployments, use the **`campusway-cli`** binary included in releases.

---

## 📖 Overview

**CampusWay** is a lightweight, standalone campus network connection manager that eliminates the need for bloated vendor EasyConnect/aTrust background services. Built in Go with a compact **Wails** desktop interface and an offline-capable browser interaction preview, it lets you manage multiple campus gateways, authentication profiles, and custom routing rules cleanly in one place.

- **Repository**: [https://github.com/Simple53/CampusWay](https://github.com/Simple53/CampusWay)
- **Live Interactive Web Preview**: [https://simple53.github.io/CampusWay/](https://simple53.github.io/CampusWay/)
- **All Platform Releases**: [GitHub Releases](https://github.com/Simple53/CampusWay/releases) (Windows, macOS, Linux, CLI, and Android AAR)

---

<h2 id="key-features">1. Key Features</h2>

### 1. Dual Protocol Support & Flexible Authentication
- **EasyConnect & aTrust**: Create, edit, and switch between EasyConnect and aTrust gateway profiles from a single client.
- **Interactive Challenges**: Supports password login, SMS verification (`auth/smsCheckCode`), graphical captcha verification dialogs, custom login domains, and multi-line node refresh intervals.
- **Per-Profile Network Tuning**: Configure custom SOCKS5/HTTP proxy ports, additional proxy domains (`customDomains`), remote/fallback DNS servers, DNS TTL, TCP-only tunneling, and `proxyAll` behavior independently for each route.

### 2. Multi-Route Management & Safe Switching
- **Grid & Table Views**: Manage routes in either a visual card grid or a compact table with instant keyword search and sorting.
- **Session Isolation**: Separates the *selected* profile from the *active* running profile so browsing or editing inactive profiles never disrupts an active VPN session. Switching to another route prompts for confirmation before cleanly tearing down the active worker.

### 3. Compact Native Desktop & Customization
- **Six Accent Palettes**: Built-in **Graphite**, **Ocean**, **Jade**, **Violet**, **Rose**, and **Amber** high-contrast themes.
- **Adaptive Appearance & Bilingual UI**: Supports **System / Light / Dark** appearance, **Compact / Standard / Comfortable** layout densities, and instant switching between **Simplified Chinese** and **English**.

### 4. Real-Time Throughput & Diagnostics
- **Home Activity Dashboard**: Displays connection status, local loopback proxy endpoints, elapsed connection duration, live download/upload speeds, and cumulative proxy TCP byte counters.
- **Structured Event Logs**: Captures connection and verification events with one-click copy and JSON export while keeping raw authentication responses private.

### 5. Security & Credential Hygiene
- **OS-Protected Credentials**: Uses native Windows **DPAPI (`CryptProtectData`)** to encrypt remembered passwords on disk; plaintext passwords are never saved to `profiles.json` or emitted to the UI.
- **Isolated Worker Process**: Runs the VPN core in a dedicated child worker (`--engine`) with strict secret-output suppression and clean process termination.
- **Sanitized Import / Export**: Route exports automatically strip usernames, remember flags, and stored credentials; imported routes remain drafts until the user explicitly supplies an account.

---

<h2 id="downloads--installation">2. Downloads & Installation</h2>

Official release packages are built automatically via GitHub Actions and published on [GitHub Releases](https://github.com/Simple53/CampusWay/releases) along with `SHA256SUMS.txt`.

> ⚠️ Desktop packages are currently unsigned. You may need to allow execution once in Windows SmartScreen or macOS Gatekeeper.

### Windows (x64) Desktop
- **Installer**: Download **`CampusWay-<version>-windows-x64-setup.exe`** for a standard NSIS installation with Start Menu shortcuts and an uninstaller (requires WebView2 Runtime, preinstalled on Windows 10/11).
- **Portable**: Download **`CampusWay-<version>-windows-x64-portable.zip`**, extract, and launch **`CampusWay.exe`** directly.

### macOS (Apple Silicon & Intel) Desktop
- **Apple Silicon**: Download **`CampusWay-<version>-macos-arm64.dmg`**.
- **Intel Mac**: Download **`CampusWay-<version>-macos-x64.dmg`**.
- Drag `CampusWay.app` into `/Applications`. If Gatekeeper flags the unsigned app on first launch, run `xattr -cr /Applications/CampusWay.app` or allow it in *System Settings -> Privacy & Security*.

### Linux (x64) Desktop
- **Debian / Ubuntu**: Download **`CampusWay-<version>-linux-x64.deb`** and install via:
  ```bash
  sudo apt install ./CampusWay-<version>-linux-x64.deb
  ```
- **Tarball**: Download **`CampusWay-<version>-linux-x64.tar.gz`** (requires `libgtk-3-0` and `libwebkit2gtk-4.1-0`).

### Command-Line Client (`campusway-cli`), Docker & System Services
Recommended for headless servers, OpenWrt routers, Docker containers, or full TUN mode:
- **Archives**:
  - Windows: `campusway-cli-<version>-windows-x64.zip`, `campusway-cli-<version>-windows-arm64.zip`
  - macOS: `campusway-cli-<version>-macos-arm64.tar.gz`, `campusway-cli-<version>-macos-x64.tar.gz`
  - Linux: `campusway-cli-<version>-linux-x64.tar.gz`, `campusway-cli-<version>-linux-arm64.tar.gz`
- **Quick Start**:
  ```bash
  ./campusway-cli -protocol easyconnect -server vpn.example.edu.cn -username <student_id> -password <password>
  # Or load from a TOML configuration file (see config.toml.example):
  ./campusway-cli -config config.toml
  ```
- See [Docker Deployment Guide](docs/docker_en.md) and [System Service Guide (Linux / macOS / OpenWrt)](docs/service_en.md) for daemon setups.

### Android Core Binding (`.aar`)
- **`CampusWay-<version>-android-aar.zip`** is built with `gomobile` for integrating the core EasyConnect tunnel into native Android `VpnService` clients. Full mobile UI apps remain under development.

---

<h2 id="web-interactive-preview">3. Web Interactive Preview</h2>

Explore the full UI flow directly in your browser without connecting to any live VPN gateway:

- **Live Preview Workbench**: [https://simple53.github.io/CampusWay/](https://simple53.github.io/CampusWay/)
- Switch seamlessly across **Desktop (800 × 620)**, **Mobile (390 × 844)**, and **Adaptive Web** canvases.
- Test 5 interactive scenarios (*Multi-route ready*, *First-run empty*, *Connected with live throughput simulation*, *Verification challenge*, and *Auth failure*)—all purely simulated in memory.

---

## 4. Local Development, Preview & Builds

### Prerequisites
- **Go** matching `go.mod` (`1.25+`)
- **Node.js 22+** (JS syntax, theme contrast, and preview bridge tests)
- **Python 3.9+** (local preview server, GitHub Pages build, and smoke checks)
- **WebView2 Runtime** (for running the Windows desktop GUI)

### Commands

```powershell
# 1. Start the local interactive preview workbench (http://127.0.0.1:4174)
./preview.ps1

# 2. Run all automated checks (Go test/vet, JS tests, preview HTTP & Pages tests)
./scripts/check.ps1

# 3. Build Windows Desktop & CLI locally into dist/campusway/
./build.ps1 -WithCLI

# 4. Run engine worker cancellation and secret-redaction smoke check
python build/smoke.py

# 5. Build the static GitHub Pages bundle into dist/pages/
python scripts/build-pages.py
```

---

## 5. Origins & Licenses

- Core Go protocol engines and desktop interaction patterns build upon open-source work by [@Mythologyli](https://github.com/Mythologyli) and [EZ4Connect](https://github.com/chenx-dust/EZ4Connect).
- Released under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
- UI icons use [Lucide](gui/static/icons/LICENSE).
