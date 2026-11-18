<div align="center">
  <img src="build/assets/campusway.png" width="96" height="96" alt="RouteDesk Icon" />
  <h1>RouteDesk (校径)</h1>
  <p><strong>支持 EasyConnect 与 aTrust 协议的跨平台校园网连接客户端与多线路管理器</strong></p>
  <p><strong>Cross-platform Campus VPN Connection Manager for EasyConnect and aTrust</strong></p>
  <p>
    <a href="#核心功能特性">核心功能</a> ·
    <a href="#客户端下载与安装指南">多平台下载</a> ·
    <a href="#浏览器在线交互预览">在线预览</a> ·
    <a href="README_en.md">English Documentation</a> ·
    <a href="https://simple53.github.io/CampusWay/">Web Preview</a> ·
    <a href="https://github.com/Simple53/CampusWay/releases">GitHub Releases</a>
  </p>
</div>

---

> ⚠️ **免责声明与适用说明**
>
> 本程序按原样提供，开发者不对程序的可用性、稳定性或适用性作任何明示或暗示的担保。请严格遵守所在高校的网络安全与管理规定使用本工具。
> - 桌面图形客户端当前以**本地回环代理模式**运行（默认提供 `SOCKS5 127.0.0.1:1080` 与 `HTTP 127.0.0.1:1081`，可按线路自定义端口），流量统计反映的是本地代理转发的 TCP 流量。
> - Windows 桌面连接后默认将校园浏览器流量接入本地代理，断开后恢复原代理设置；可在高级配置中关闭。[EasyConnect、Clash 共存与验证说明](docs/connectivity-fixes.md)。
> - 若需全系统 TUN 虚拟网卡、端口转发或路由器后台常驻，可使用同仓库提供的命令行工具 **`routedesk-cli`**。

---

## 📖 项目简介

**RouteDesk（校径）** 是一款轻量、纯净、无需安装官方臃肿客户端与后台驻留服务的校园网连接管理器。它基于纯 Go 语言实现深信服 EasyConnect 与 aTrust 双协议核心，搭配基于 **Wails** 构建的原生桌面界面与可离线的浏览器交互预览工作台，让您能够在同一客户端内安全管理多所高校、多网关或不同用途的校园网连接线路。

- **开源仓库**：[https://github.com/Simple53/CampusWay](https://github.com/Simple53/CampusWay)
- **在线交互预览**：[https://simple53.github.io/CampusWay/](https://simple53.github.io/CampusWay/)
- **全平台下载**：前往 [Releases 页面](https://github.com/Simple53/CampusWay/releases) 获取 Windows / macOS / Linux 桌面客户端、全架构 CLI 及 Android 核心 AAR 包。

---

<h2 id="核心功能特性">一、核心功能特性</h2>

### 1. 双协议支持与灵活认证
- **EasyConnect & aTrust 全兼容**：同一界面自由添加、配置与切换 EasyConnect 和 aTrust 网关线路。
- **多种挑战与验证方式**：支持密码认证、短信验证码（SMS）、登录域（Login Domain）设置、图形验证码挑战交互弹窗以及节点延迟自动优选间隔配置。
- **高级网络微调**：支持按线路独立配置 SOCKS5 / HTTP 代理端口、自定义分流域名列表（`customDomains`）、远程/备用 DNS、DNS 缓存 TTL、aTrust 专用 TCP 隧道及 `proxyAll` 等行为选项。

### 2. 多线路管理与安全切换
- **双视图自由切换**：提供**方块网格（Grid）**与**信息表格（Table）**两种线路视图，支持按名称实时搜索与自定义排序。
- **一键切换与状态隔离**：区分「当前选中线路」与「正在运行线路」，浏览或编辑其他配置时不会干扰正在进行的会话；每条线路通过独立开关连接或断开，切换时先停止原线路。

### 3. 原生轻量桌面与个性化外观
- **六种精选配色主题**：内置**石墨（Graphite）**、**海蓝（Ocean）**、**青绿（Jade）**、**紫藤（Violet）**、**玫瑰（Rose）**、**琥珀（Amber）**六套高对比度配色主题。
- **自适应明暗与密度**：支持**跟随系统 / 浅色 / 深色**三态主题实时响应，支持**紧凑 / 标准 / 宽松**三种显示密度，以及**简体中文 / English** 一键切换。

### 4. 实时网速监控与诊断日志
- **首页实时仪表盘**：集中展示当前连接状态、本地代理地址与端口、已连接时长、本地代理 TCP 实时下载/上传速率与累计流量。
- **会话事件诊断台**：按时间序列记录连接与验证事件，支持一键**复制日志**或**导出 JSON 格式诊断记录**；原始敏感认证响应绝不落盘或明文输出。

### 5. 隐私与凭据安全设计
- **系统级凭据加密**：Windows 平台采用原生 **DPAPI（`CryptProtectData`）** 加密保存记住的密码，绝不将明文密码写入 `profiles.json`。
- **工作进程安全隔离**：底层 VPN 引擎以独立子进程（`--engine`）运行，UI 层与核心引擎通过受控事件协议通信，关闭主窗口或切换线路时严格回收隧道会话。
- **无敏感信息导入导出**：导出线路配置（JSON）时自动剥离用户名、记住密码标志及所有凭据；导入他人分享的线路配置后仅作为草案保存，需由使用者自行填写账户信息后方可使用。

---

<h2 id="客户端下载与安装指南">二、多平台客户端下载与安装指南</h2>

所有正式版本均通过 GitHub Actions 自动构建并发布至 [GitHub Releases](https://github.com/Simple53/CampusWay/releases)，附件内均附带完整的 `SHA256SUMS.txt` 校验文件。

> ⚠️ 当前桌面端安装包尚未购买商业代码签名证书。Windows SmartScreen 或 macOS Gatekeeper 首次启动时可能会提示未签名应用，按系统指引确认允许运行即可。

### 1. Windows (x64) 桌面端
- **标准安装版**：下载 **`RouteDesk-<version>-windows-x64-setup.exe`**，双击按向导安装，自动创建开始菜单快捷方式及卸载程序（依赖系统内置 WebView2 Runtime，Windows 10/11 通常已预装）。
- **免安装便携版**：下载 **`RouteDesk-<version>-windows-x64-portable.zip`**，解压后直接双击运行 **`RouteDesk.exe`**。

### 2. macOS (Apple Silicon & Intel) 桌面端
- **Apple Silicon (M1/M2/M3/M4)**：下载 **`RouteDesk-<version>-macos-arm64.dmg`**。
- **Intel Mac**：下载 **`RouteDesk-<version>-macos-x64.dmg`**。
- 打开 `.dmg` 将 `RouteDesk.app` 拖入「应用程序」目录；若首次打开提示“无法验证开发者”，可执行 `xattr -cr /Applications/RouteDesk.app` 或在「系统设置 -> 隐私与安全性」中允许打开。

### 3. Linux (x64) 桌面端
- **Debian / Ubuntu**：下载 **`RouteDesk-<version>-linux-x64.deb`**，执行安装：
  ```bash
  sudo apt install ./RouteDesk-<version>-linux-x64.deb
  ```
- **通用压缩包**：下载 **`RouteDesk-<version>-linux-x64.tar.gz`**（需系统已安装 `libgtk-3-0` 与 `libwebkit2gtk-4.1-0`）。

### 4. 命令行工具 (`routedesk-cli`) 与容器/系统服务部署
适用于服务器、无桌面 Linux、OpenWrt 路由器、Docker 或需要全系统 TUN 虚拟网卡模式的场景：
- **下载包名**：
  - Windows：`routedesk-cli-<version>-windows-x64.zip` / `routedesk-cli-<version>-windows-arm64.zip`
  - macOS：`routedesk-cli-<version>-macos-arm64.tar.gz` / `routedesk-cli-<version>-macos-x64.tar.gz`
  - Linux：`routedesk-cli-<version>-linux-x64.tar.gz` / `routedesk-cli-<version>-linux-arm64.tar.gz`
- **快速使用**：
  ```bash
  # 使用参数直接启动本地代理 (SOCKS5 :1080 / HTTP :1081)
  ./routedesk-cli -protocol easyconnect -server vpn.example.edu.cn -username <学工号> -password <密码>

  # 或使用 TOML 配置文件启动 (参考根目录 config.toml.example)
  ./routedesk-cli -config config.toml
  ```
- **Docker 与系统守护服务教程**：详见 [Docker 部署指南](docs/docker.md) 与 [Linux / macOS / OpenWrt 系统服务指南](docs/service.md)。

### 5. Android 核心库 (`.aar`)
- Releases 同时提供 `gomobile` 编译的 **`RouteDesk-<version>-android-aar.zip`**，面向具备安卓原生开发需求的用户集成系统 `VpnService`。完整原生 Android / iOS 客户端仍在持续演进中。

---

<h2 id="浏览器在线交互预览">三、浏览器在线交互预览</h2>

无需下载或连接真实校园网网关，直接通过 GitHub Pages 访问离线模拟工作台：

- **在线预览地址**：[https://simple53.github.io/CampusWay/](https://simple53.github.io/CampusWay/)
- 支持实时切换 **桌面（800 × 620）/ 手机（390 × 844）/ 自适应网页** 三种设备画布；
- 内置 **多线路待连接、首次无配置、已连接流量模拟、图形验证码挑战、网关认证失败** 5 种交互场景（全部为纯前端模拟，绝不记录或发送任何输入密码）。

---

## 四、本地开发、预览与打包指南

### 环境依赖
- **Go**（匹配 `go.mod` 中的 `1.25+`）
- **Node.js 22+**（用于前端语法与交互/主题测试）
- **Python 3.9+**（用于本地预览服务器、GitHub Pages 静态构建与冒烟测试）
- Windows 桌面端运行与调试需系统存在 **Microsoft Edge WebView2 Runtime**

### 常用开发命令

```powershell
# 1. 启动本地交互预览工作台 (默认访问 http://127.0.0.1:4174)
./preview.ps1

# 2. 运行全量自动化校验 (Go 测试/vet + JS 语法/单测 + 预览服务与 Pages 打包测试)
./scripts/check.ps1

# 3. 本地编译 Windows 桌面端与 CLI (产出位于 dist/routedesk/)
./build.ps1 -WithCLI

# 4. 运行引擎隔离与秘钥脱敏冒烟测试
python build/smoke.py

# 5. 构建用于 GitHub Pages 部署的纯静态预览站点 (产出位于 dist/pages/)
python scripts/build-pages.py
```

---

## 五、项目架构目录

```text
RouteDesk/
├── .github/workflows/
│   ├── deploy.yml           # GitHub Pages 在线预览自动构建与部署流水线
│   ├── release.yml          # Windows / macOS / Linux / Android 多平台打包与发布流水线
│   └── validate.yml         # Push / PR 自动化测试与构建验证流水线
├── build/
│   ├── assets/              # 应用图标资源 (ico / icns / png)
│   ├── package-unix.sh      # macOS (.dmg) 与 Linux (.deb / .tar.gz) 打包脚本
│   ├── smoke.py             # --engine 工作进程退出与敏感日志脱敏测试
│   └── windows.nsi          # Windows NSIS 安装包构建脚本
├── client/                  # EasyConnect 与 aTrust 核心协议驱动实现
├── configs/                 # 配置结构体与 TOML 映射定义
├── dial/                    # 规则拨号器
├── docs/                    # Docker 部署与系统守护服务中英文文档
├── engine/                  # 统一 VPN 生命周期调度引擎
├── gui/
│   ├── static/              # Wails 内嵌前端源码 (pages / components / core / icons)
│   ├── main.go              # Wails 桌面壳入口与 --engine 工作进程分派
│   ├── server.go            # 本地配置管理、状态快照、子进程通信与校验接口
│   └── settings.go          # 线路导入导出、凭据清除与 GitHub Release 版本检查
├── internal/                # 认证挑战、Hook 生命周期、IP 池、测速、流量统计与底层 DNS/TCPIP 工具
├── mobile/                  # gomobile Android 绑定接口
├── resolve/                 # 域名分流规则与解析缓存管理器
├── scripts/                 # check.ps1 / preview-ui.py / build-pages.py 与前端单元测试
├── service/                 # SOCKS5、HTTP、DNS、Shadowsocks、端口转发与保活服务
├── stack/                   # gVisor 用户态栈、TCP 隧道栈与各平台 TUN 虚拟网卡栈
├── build.ps1                # Windows 桌面端及 CLI 一键编译打包脚本
├── main.go / init.go        # routedesk-cli 命令行主入口
└── config.toml.example      # CLI 配置文件模板
```

---

## 💖 致谢与开源协议

- **核心协议与技术来源**：
  - [@Mythologyli](https://github.com/Mythologyli) — Go 语言 EasyConnect / aTrust 协议核心与客户端桌面交互参考
  - [chenx-dust/EZ4Connect](https://github.com/chenx-dust/EZ4Connect) — 领域分层与多线路控制流架构参考
- **开源许可证**：
  - 本项目遵循上游 [GNU General Public License v3.0 (GPL-3.0)](LICENSE) 开源协议。
  - 界面图标采用 Lucide 图标库，许可证见 [gui/static/icons/LICENSE](gui/static/icons/LICENSE)。
