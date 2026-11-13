# CampusWay

Campus network connection manager for EasyConnect and aTrust, with a compact Wails desktop interface and a browser-based interaction preview.

## Local UI preview

Requires Python 3.9 or later. From the repository root:

```powershell
./preview.ps1
```

Open http://127.0.0.1:4174. Switch between desktop, phone and adaptive web canvases. Test empty routes, connection success, verification and failure without a real VPN connection. Chinese is the default UI language; English is available in Settings. The preview discards passwords and resets its in-memory data when reloaded.

Frontend files are in `gui/static`; simulation-only files are in `scripts/preview` and are never embedded in the desktop application. Edit frontend files and refresh the preview to review changes.

## Build and checks

Requires Go matching `go.mod`, Node.js 22+ and Python 3.9+. Windows desktop requires WebView2 Runtime.

```powershell
./scripts/check.ps1 -GoBin go
./build.ps1 -GoBin go
```

The check script runs Go tests and vet, JavaScript syntax and interaction-model tests, palette contrast tests, and isolated preview HTTP tests. Windows build output is under `dist/campusway`.

The desktop currently provides loopback HTTP/SOCKS proxies. Its traffic counters cover local-proxy TCP traffic, not device-wide throughput. Real campus authentication requires your gateway and credentials. Native Android/iOS VPN integration remains in development; responsive phone previews are UI simulations.

## Packaging

GitHub Actions validates pushes to `main`. The separate package workflow generates Windows portable ZIP and NSIS installer, macOS DMG, and Linux DEB and tar.gz packages on manual dispatch or version tags. Tag releases are created as drafts. Desktop packages are currently unsigned; cross-platform runtime behavior requires platform-specific verification.

## Origins and licenses

The Go core derives from [ZJU-Connect](https://github.com/Mythologyli/ZJU-Connect). Architecture and connection flows reference [EZ4Connect](https://github.com/chenx-dust/EZ4Connect) and [ZJU-Connect-for-Windows](https://github.com/Mythologyli/ZJU-Connect-for-Windows). CampusWay retains the upstream GPL-3.0 license in `LICENSE`. Lucide icons retain their license in `gui/static/icons/LICENSE`.
