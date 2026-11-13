// Package campusway records the proposed product architecture; it implements no VPN runtime.
//
// 2026-11-13 / 产品暂定名：校径 CampusWay。名称不限定学校；尚未进行商标核查。
// 本目录是设计交付，不能用其连接示意宣称真实线路已经接通。
// 用户要求所有修改说明放入代码注释，因此不新建明文方案或变更日志页面。
// 本稿替代 2026-09 方案中的单学校产品范围；旧模块路径、用户凭据目录、
// 历史许可证和现有 EXE 不在设计阶段直接重命名，防止破坏导入与数据发现。
//
// 架构依据（2026-11-13 读取，借鉴边界，未复制实现）：
// https://github.com/chenx-dust/EZ4Connect/blob/master/docs/ARCHITECTURE.md
// https://github.com/chenx-dust/EZ4Connect/blob/master/application/coreprocess.h
// https://github.com/chenx-dust/EZ4Connect/blob/master/application/profileservice.h
// EZ4Connect 将领域、用例、基础设施和展示分开，通过 Coordinator 注入实现，
// ProfileService 管理配置上下文，CoreProcess 用独立事件表达认证和连接状态。
// 校径采用同样的依赖边界，以 Go 端口代替对 Qt 类型的耦合。
//
// https://github.com/Mythologyli/ZJU-Connect-for-Windows
// https://github.com/Mythologyli/ZJU-Connect-for-Windows/blob/master/zjuconnectcontroller/zjuconnectcontroller.h
// 该项目区分网页认证、L2TP、RVPN，并由控制器管理核心进程与认证信号。
// 校径将接入方式、传输协议、网关地址、转发模式分开，不把新增 URL 等同于新增协议。
//
// 目标依赖：
//
//	presentation/desktop (Wails + TypeScript) ─┐
//	presentation/android (Kotlin + Compose) ─┼→ application → core
//	presentation/ios (SwiftUI) ──────────────┘       ↑
//	infrastructure/{protocol,storage,platform} ──────┘
//
// Go 领域和用例层不能引用 Wails、JNI、Swift、Qt 或文件系统具体实现。
// 应用启动组合根负责注入适配器；桌面和手机共用配置/事件契约而非共用窗口代码。
// Android 先做绑定验证；iOS 先验证 tunnel adapter，不能假设现有 Go TUN 栈直接兼容。
//
// core/Profile：
//
//	id：稳定 UUID，不用线路名作为索引；rename 不改变凭据归属。
//	name / institution / endpoint(host, port)：用户可增删改的线路信息。
//	accessKind：vpn | web-auth | system-vpn，区分隧道和门户认证。
//	protocol：easyconnect | atrust；其它协议仅在适配器实现后注册。
//	mode：proxy | tun；分流策略单独保存，避免将不同层次混成一个下拉框。
//	credentialRef：系统凭据存储引用，禁止在 Profile 中保存密码或验证码。
//	routePolicy：gateway / custom，规则包含来源和生效结果。
//	version：乐观并发检查；schemaVersion 用于持久配置迁移。
//
// Campus 不代表协议；同一学校可保存多个网关、身份或访问用途。
// “多线路”是多配置，可随时选择与切换，首版只允许一个活动隧道，不承诺并发聚合。
//
// application：
//
//	ProfileService 管理 create/update/delete/list/duplicate/import/export。
//	ConnectionCoordinator 串行处理 connect/disconnect/switch/cancel。
//	AuthCoordinator 将图形验证码、短信、TOTP、SSO 转换为 Challenge 事件。
//	Session 拥有 client、resolver、listeners、TUN、cancel、done 与清理栈。
//	CapabilityRegistry 返回每个平台/协议支持的模式、认证方式和不可用原因。
//	EventStore 提供递增 seq 的有界日志与 revision 快照，避免数组长度游标停滞。
//
// infrastructure：
//
//	protocol/easyconnect 与 protocol/atrust 适配现有 Go 客户端，返回结构化事件。
//	ProcessAdapter 仅为必要外部核心保留，固定程序白名单，禁止用户任意命令。
//	不从一般日志文字猜测 connected；listener 与隧道 ready 才报告 connected。
//	Windows：Wails 壳、托盘、DPAPI、WinTun、按需提权 helper。
//	Android：VpnService、前台服务通知、Keystore 加密凭据、gomobile/JNI 核心绑定。
//	iOS：NetworkExtension + Packet Tunnel Provider、Keychain、App Group 配置。
//	UI 不能拥有 VPN 生命周期；切到后台或关闭窗口不应直接终止系统服务。
//
// 平台证据：
// https://developer.android.com/develop/connectivity/vpn
// Android 需要系统 VPN 授权，外层隧道 socket 必须 protect，避免流量回环；
// 每个用户/工作配置仅有一个活动 VPN service，切换其它 VPN 可能替换当前服务。
// https://developer.apple.com/documentation/networkextension/nepackettunnelprovider
// iOS 的自定义包隧道运行于扩展，需要 Network Extension entitlement；
// 实现需用 packetFlow 传递 IP 包，并设置 DNS/路由。网页或响应式 UI 不等于手机 VPN。
//
// 切换事务：
//  1. Validate(target)；不支持的协议、错误地址或缺凭据在停止旧连接前提示。
//  2. 显示从 A 切换到 B 会短暂断开；允许取消，不声称无缝切换。
//  3. Freeze operationId，停止 A，等待其 done，并确认 DNS/路由/代理恢复。
//  4. 若清理失败，进入 cleanup_required，禁止启动 B。
//  5. Start(B)，处理挑战与权限，只在必需组件 ready 后提交 activeProfileId。
//  6. B 失败则明确未连接，并提供重新连接 A 的操作；不静默反复认证造成锁号。
//
// selectedProfileId 与 activeProfileId 分离；浏览或编辑 B 不会改变 A 的运行配置。
// 旧 sessionId 的迟到事件被丢弃；高频切换不并发启动，明确取消/替换待执行请求。
// 删除活动线路须先断开；导入配置限制大小、校验字段，并且不能携带明文秘密。
//
// 数据安全：
// 凭据按 profileId + canonical gateway + username 绑定，更换网关必须重新确认。
// 系统 TLS 信任为默认，例外/指纹须可核验；身份校验覆盖认证及所有隧道握手。
// 日志/诊断脱敏 password、TWFID、token、验证码；配置写入原子化并返回错误。
// 桌面 UI binding 只开放有限命令；移动桥只传 DTO 和受限操作，不能执行任意代码。
// 系统代理只在服务 ready 后设置，断开恢复先前值；不覆盖用户随后主动修改的设置。
// 路由/DNS 操作按会话记账并逆序回滚。切换不同线路不复用旧 DNS 缓存。
//
// 界面契约：
// 桌面：独立侧栏、线路列表和详情工作区；可搜索、添加、编辑、选中并连接。
// 手机：独立单列布局、底部导航、44pt/48dp 操作区、全屏新增表单与切换确认 sheet。
// 两端都包含线路名、学校、协议、网关、账户、模式及高级分流设置。
// 首页不重复密码表单；首次添加/缺少凭据/认证挑战时才要求输入。
// 主题为 system/light/dark，默认 system，系统主题变化实时响应；用文字和 SVG
// 区分状态，不依赖颜色。所有图标来自 Lucide SVG；没有 emoji 或位图装饰。
// 三组 SVG 是同一需求的三种布局候选，每组有单独桌面/手机文件及配套总览。
// 本阶段只出设计，不用静态图模拟真实连接，也不将未实现协议展示为生产可用。
//
// 交付阶段与验收：
// P0：选定视觉方案和名称；冻结 Profile/Session/Challenge/Capability 契约。
// P1：配置多条线路与迁移；写入/删除失败可见；新身份不借用旧身份密码。
// P2：串行 Session 与切换事务；端口冲突、取消、挑战、断开失败、迟到事件测试。
// P3：桌面壳接入；Go 核心 EasyConnect 与 aTrust 分别做真实网关验收，能力按结果开放。
// P4：Android 授权、protect、后台生命周期和 JNI 验收；不将 Windows go test 当移动验收。
// P5：iOS entitlement、packetFlow、扩展生命周期和真机网络验证，再决定发行范围。
// P6：更新二进制名、窗口/托盘/安装资源为 CampusWay；旧用户目录单次迁移，失败保留源。
//
//	保留上游版权/许可证声明；发行名改变不能丢掉现有 AGPL 来源与其它组件许可。
//
// 新协议适配器需分别实现并验收；第一轮不把 OpenVPN/WireGuard/L2TP 当已支持功能。
// 尚未执行这些运行时改造；方案选择后才进入功能实现。
//
// 用户选定的合并流程（覆盖前述三组候选布局）：
// 首次无配置：采用方案 3 的单焦点登录，服务器/协议/账号/密码为最小输入。
// 一条配置：保留单线路连接页；账户已保存时不重复展现密码输入。
// 保存第二条后：进入方案 1 衍生的线路管理页，支持表格和方块两种呈现。
// 两种呈现共享同一 selectedProfileId、过滤、排序与 activeProfileId，切视图不重连。
// 桌面登录窗口 460x600 DIP，多线路窗口默认 900x620 DIP，正常窗口启动，不全屏，
// 不默认最大化；用户调整后保存正常窗口尺寸，新增线路不得覆盖用户已保存尺寸。
// 桌面最低尺寸建议 760x560 DIP，小窗口通过滚动与收起侧栏容纳，禁止缩小字体补偿。
// 小屏/高 DPI 时以可用工作区限制窗口大小；登录到管理页的首次过渡仅在必要时扩展。
// 网格默认桌面三列、手机两列，宽度不足时减列；桌面表格含线路/协议/延迟/状态，
// 手机表格将学校/协议合并为两行以避免横向滚动。当前选择的操作区在底部固定。
// 点击条目只选择，连接按钮才操作 VPN；已有活动线路时显示切换确认与短暂断线提示。
// 显示设置：桌面紧凑弹层、手机底部 sheet，包含视图、排序、密度、主题。
// 参考图的“布局/尺寸”合并为“密度”，避免两个控件调整同一间距；紧凑也保留触控热区。
// 延迟只显示真实测量结果，无数据为未检测；按延迟排序时未测/失败置后，稳定同序。
// 网关探测与隧道内可达性明确区分，不能用网关延迟宣称 VPN 已连接；不自动发送密码测速。
// 显示偏好按设备保存，不嵌入线路配置；system 模式监听系统变化，light/dark 显式覆盖。
// 手机 sheet 支持关闭按钮、返回手势、背景遮罩关闭，焦点限制在面板并恢复到触发按钮。
// 本轮是已选方向的细化 SVG，位于 refined/{system,light,dark}/；尚未执行实际界面代码。
package campusway

// Build verification: Windows x64 desktop and portable ZIP built locally; go test ./...
// and go vet ./... passed. Worker cancellation smoke test verified that its output
// did not contain the input password sentinel. Native UI rendering, real gateway
// authentication, NSIS, macOS and Linux packages still require runtime validation.
// Mobile layouts are implemented; native Android/iOS VPN service integration is pending.
// GUI currently exposes loopback HTTP/SOCKS proxies; system routing is not enabled.

// Navigation revision: startup opens the Activity overview without onboarding.
// Both form factors use Home/Connections/Settings in the bottom navigation.
// Appearance controls belong to settings or the connection toolbar only.
// zh-CN is the default language; en is persisted through preferences.
// Legacy account migration is disabled; route exports exclude identities and secrets.
// Settings include route import/export, per-profile account editing/credential removal,
// connection management, an explicit GitHub release source, and language selection.
// Detailed protocol options are validated and forwarded to the isolated worker.
// Verification: all Go tests passed, GUI/engine vet passed, JS syntax passed.
// Real aTrust/EasyConnect authentication and native UI rendering remain unverified.
