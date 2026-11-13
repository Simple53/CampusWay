# NNU-Connect：重新评估与实现计划

日期：2026-09-06。依据：任务 `01a062a0-81fc-74b2-b247-f01d9083a926` 的用户要求、上一轮交付记录，以及本地源码静态复核。本文是待实施设计，不能作为已完成能力清单。

## 1. 产品结论

保留现有 Go / EasyConnect 核心，重构桌面应用层和会话资源管理。目标是 Windows 上简洁、可信、可重复连接的南京师范大学远程访问工具。默认本地代理；TUN 在完整回滚验收前保留实验性标记。aTrust 保留为上游兼容代码，不进入 NJNU 桌面产品流程。

上一轮完成了回环监听、DPAPI 基础存储、移除宽泛私网路由、图标和文档修订；这些成果可保留。但全局清理变量复位只是临时补丁，尚未完成 Session 架构，也没有证明真实 VPN 端到端可用。现有 EXE 使用 Edge App 窗口。

## 2. 新发现与优先级

| 优先级 | 证据 | 影响 | 实施决策 |
| --- | --- | --- | --- |
| P0 | `gui/static/app.js:128` 异步保存未等待；连接载荷不含 remember_password / auto_connect；`gui/server.go:180` 用请求覆盖配置 | 用户选项会被零值覆盖，已存凭据可能被删除，保存与连接存在竞态 | 配置与连接命令拆分；连接引用已提交配置版本；凭据更新使用明确操作 |
| P0 | `gui/server.go:182` 未阻止 disconnecting；connecting 状态在解锁之后设置；全局 BeginTerminalSession 清空 hook | 并发连接或旧会话清理可能影响新会话 | 单控制器串行处理命令；会话独立拥有资源；清理结束才释放运行槽位 |
| P0 | `engine/engine.go:315` 异步启动代理后直接 connected；`service/socks.go:40` bind 失败 panic；HTTP bind 错误只日志输出 | 端口冲突可退出整个应用，或显示已连接但代理不可用 | 同步获取 listener；运行错误上报监督器；全部必需组件 ready 后 connected |
| P0 | `client/easyconnect/request.go:166,275,326` 使用 fmt.Scan | 无控制台 GUI 无法正常完成交互认证 | 抽出 AuthInteractor，桌面显示验证码/短信/TOTP 输入，CLI 保留终端适配器 |
| P0 | `gui/server.go:257` 控制 API 无会话授权，disconnect 未限定方法 | 外部网页可尝试触发本地控制请求 | 过渡期加令牌、Host/Origin、请求类型与大小校验；桌面版改为受限 Go binding |
| P0 | EasyConnect 多处 InsecureSkipVerify；账号和服务器更换仍可能复用 currentConf.Password | 服务端身份和凭据归属边界不足 | 凭据按规范化网关+端口+账号绑定；所有认证/隧道 TLS 共用信任策略 |
| P1 | `gui/server.go:303` 使用数组长度作日志游标，数组最多 500 条 | 达到 500 后客户端游标可能永久停住 | 单调 seq + 有界缓冲，显式报告日志缺口 |
| P1 | `service/http.go:70,101` 劫持连接缺乏完整关闭管理，响应 Body 未关闭 | 长连接/资源泄漏；Shutdown 不能替代所有隧道连接清理 | 跟踪活动连接，两侧关闭，读取后关闭 Body，清理带期限 |
| P1 | `engine/engine.go:242` 忽略 AddRoute 错误；断开状态早于 defer 清理 | TUN 部分失败仍可能显示成功，网络恢复没有明确结果 | 路由操作事务化；只回滚本会话创建的状态，清理失败显示待修复 |
| P1 | gui/main.go 端口冲突后选新端口；托盘启动 Edge App | 缺乏单实例及可靠窗口生命周期 | Wails 主窗口、单实例唤醒、关闭隐藏、退出等待清理 |

这些是源码发现与风险推断；未使用真实账号复现全部路径。已有测试通过也不能覆盖目前没有测试的 engine 和 GUI 编排。

## 3. 技术选型

- Go 保留协议、网络栈、DNS 和代理；不整体重写协议，也不继续扩散无关 import 重命名。
- 桌面采用 Wails v2 稳定版本线 + WebView2。实施时锁定验证后的精确版本；v3 官网当前标为 Beta，本轮不以迁移 Beta 为目标。
- 前端采用 React + TypeScript + Vite + CSS variables。按页面和组件拆分，初期使用 reducer/context，无需引入大型状态库或完整企业组件库。
- Go 方法负责命令，事件负责更新，GetSnapshot 负责首次挂载和重连恢复。应用层不依赖 Wails，便于 CLI 共用和无窗口测试。
- 保留已有原创图标；视觉设计用 CSS/SVG，无需安装设计插件。
- 代理版资源嵌入 EXE，但依赖系统 WebView2 Runtime；不承诺任意 Windows 环境下零依赖单文件。TUN 版优先 portable ZIP 明确包含 WinTun。

官方依据：[Wails 窗口、单实例与绑定选项](https://v2.wails.io/docs/reference/options/)、[Windows Runtime 指南](https://v2.wails.io/docs/guides/windows/)、[Wails v3 当前阶段](https://v3.wails.io/)。

## 4. 目标代码结构

```text
cmd/desktop/                 Wails 启动、绑定、窗口生命周期
cmd/cli/                     CLI 参数与终端交互
internal/app/                Controller、命令、DTO、事件、快照
internal/session/            Session、资源注册/逆序释放、运行监督
internal/auth/               AuthInteractor、Challenge、认证错误分类
internal/profile/            配置版本、迁移、规范化和校验
internal/credential/         DPAPI 实现与网关/账号绑定
internal/trust/              统一 TLS 信任和指纹策略
internal/diagnostics/        结构化脱敏日志、探测、导出
internal/platform/windows/   托盘、单实例、权限、TUN helper/IPC
client/easyconnect/          保留并逐步适配 context 与认证交互
service/                    返回可关闭资源及错误，移除全局 hook 依赖
stack/                      保留网络栈，经 Session 统一管理
frontend/src/
  app/                      Shell、主题、状态订阅
  pages/                    Connect、Resources、Activity、Settings
  components/               Sidebar、Status、ConnectionForm、AuthDialog
  bridge/                   Go bindings 封装；仅测试/预览用 mock
build/                      manifest、图标源、资源生成、打包
docs/                       设计、测试矩阵、上游变更记录
```

迁移按调用链逐步进行；旧 engine.Run 暂做兼容封装。CLI 和桌面最终都调用 Controller/Session；不保留两套认证和网络启动流程。

### 状态与资源规则

```text
idle → connecting → awaiting_auth → connecting → connected
           ↓              ↓                        ↓
          stopping ← cancel / error / disconnect / shutdown
           ↓
     idle / failed / cleanup_required
```

认证挑战可重复出现。所有命令由 Controller 串行化，每次连接分配 sessionId，过期会话事件不能覆盖新状态。stopping 和 cleanup_required 禁止新连接；后者修复并确认资源释放后才可恢复。

Session 持有 context/cancel、done、client、stack、resolver、listeners、活动连接及 cleanup stack。资源取得后立即登记；失败逆序释放。认证请求、TLS 拨号、后台保活必须遵守 session context；Cancel 不能只取消最外层 Run。

Connected 表示认证、网络栈和必需监听器已就绪；校内目标可达性是独立探测结果，不用任意网站探测成功替代服务就绪，也不虚构延迟或流量。

### 前后端契约

- GetSnapshot → state、sessionId、revision、connectedAt、运行配置摘要、端口、最后日志 seq、当前 challenge（不含秘密）。
- SaveProfile(profile, expectedRevision) → 已提交 revision；写入失败返回明确错误。
- SetCredential(profileKey, action=replace|delete, secret) → 仅返回是否存在；不向 UI 回传秘密。
- Connect(profileId, revision, optionalEphemeralSecret) → 接受的 sessionId；重复请求不会创建并行会话。
- SubmitChallenge(sessionId, challengeId, response)、CancelConnect(sessionId)、Disconnect(sessionId)。
- GetLogs(afterSeq, limit) → entries、nextSeq、gap；订阅事件携带 sessionId/revision/seq，过期事件丢弃，缺口重新取快照。
- ExportDiagnostics 只导出脱敏信息。密码、TWFID、短信/TOTP、认证响应不进入日志和前端持久存储。

订阅先建立，再取快照并按 revision 对齐，避免初始化时漏事件。单会话短期明文只在使用期间保留；不宣称 Go 字符串可保证物理内存清零。

### 凭据、TLS 与权限

配置 DTO 不含 password 字段；临时密码只属于 Connect 命令。更换网关或账号时重新查询独立凭据，不复用上一身份密码。DPAPI 写入、删除失败不能伪报成功；旧配置备份后迁移，成功才移除明文旧字段。

TLS 默认使用系统信任链。网关兼容例外按主机保存，首次指纹必须提供可核验来源，不能盲目自动信任；证书变更阻断并由用户重新核验。HTTP、uTLS 和所有隧道必须一致执行策略。

代理默认普通权限。TUN 在后续阶段由按需启动的受限 helper 管理；命名管道限定当前用户/会话，验证对端、握手令牌、消息大小与命令白名单，不接受任意 shell 命令。具体哪些网络栈操作必须驻留提权进程由技术验证决定。helper 崩溃/父进程退出通过租约与已记录操作恢复，仅撤销本应用变更。首版不引入常驻系统服务。

## 5. Codex 式界面设计

视觉稿见 `ui-direction.svg`。这是布局设计，画面中没有真实连接数据。

- 初始 1120×760，最小 860×620；左栏 208，详情栏 264 可收起，中央内容宽度约 520–640；窄窗口将详情改抽屉。
- 灰黑主背景 #202020，侧栏 #181818，输入层 #282828，分隔 #383838，主文字 #ECECEC，次文字 #A5A5A5；浅色主题以同一套语义 token 提供，默认跟随系统。
- 标题 22–26px，正文 14px，辅助 12px；圆角 8–12px，8px 间距基线。主动作中性浅色，绿色仅表达成功；不添加大幅宣传横幅。
- 左栏：当前校园配置、连接、资源与分流、活动；设置和关于固定底部。MVP 仅显示已实现入口。
- 中央连接页：简短标题、学工号和密码、接入模式、一行凭据选项、一个主动作；网关编辑进入设置。
- 右栏：会话状态、就绪后的代理端口及复制、分流说明、最近活动摘要；完整日志单独页面。
- 等待认证时就地弹出挑战输入；连接中主按钮改“取消连接”，停止中禁用；错误解释原因并给具体恢复动作。
- 未连接显示“未启动”，指标无数据显示“—”。时长由后端 connectedAt 计算，重新打开窗口不重置。
- 资源页呈现网关下发/用户配置的来源与生效状态，TUN 路由写入失败可定位；新增域名不承诺获得额外授权。
- 设置分为账号、网络、外观、启动行为；界面只写“记住密码”，DPAPI 说明置于辅助信息。
- 托盘动作与主窗口共用 Controller；关闭窗口隐藏，托盘退出执行停止并等待结果。
- 支持 Tab 顺序、明显焦点、Enter 提交、Esc 关闭非阻塞面板、减少动画、高对比；125/150/200% 缩放验收。

## 6. 可独立审核的实施顺序

| 阶段 | 代码工作 | 完成标准 |
| --- | --- | --- |
| 0 基线 | 保存现有变更边界；统一忽略 cache/dist；记录源码 revision 与产物；增加当前缺陷回归用例 | 不夹带宠物资源等无关目录；现有产物可回退；失败用例能说明缺陷 |
| 1 正确性 | 修配置/凭据契约、串行控制器、Session、服务 ready/error、取消和清理 | 连续 50 次连接/断开模拟测试；并发 connect 只起一次；占用端口不崩溃；旧事件不能改变新会话 |
| 2 认证与信任 | AuthInteractor、挑战过期与取消、TLS 统一、账号凭据隔离、日志 seq/脱敏 | 认证挑战 mock 完整覆盖；切换身份不借用密码；错误证书不提交凭据；超过 500 条日志仍更新 |
| 3 桌面纵向切片 | Wails 壳 + React 连接页 + bindings + 托盘 + 单实例；移除产品控制 HTTP API | EXE 开启/隐藏/恢复/退出可靠；旧窗口清除；UI 状态全部来自真实快照；新 shell 可连接适配后的引擎 |
| 4 产品界面 | 资源、活动、设置、浅深色、详情栏、错误/空/认证状态 | 可见入口全部可用；键盘与 DPI 验收；不将 mock 数据编入生产 |
| 5 TUN | 提权边界验证、helper、路由/DNS 操作记录与恢复、休眠/网络切换 | 管理员拒绝、部分写入失败、进程异常时恢复正确；与已有代理工具共存场景验收 |
| 6 发布 | 干净 Windows 构建、资源生成脚本、版本号/哈希、依赖说明、中英文 README | 从 clean checkout 可重建；WinTun 缺失不能伪称 TUN 包完整；真实 NJNU 认证及校内资源验收完成后再提升发布状态 |

阶段 1→2→3 是首个可用桌面里程碑；阶段 4 可在应用契约冻结后开展。TUN 不阻塞代理版里程碑。暂不做多学校、多会话并行、插件系统、自动更新服务或跨平台 GUI。

## 7. 验证边界

单元测试覆盖控制器/配置/认证，集成测试覆盖临时监听端口与资源关闭；Windows 上在工具链允许时做 race 检测。GUI 验证连接、取消、挑战、错误、隐藏与恢复。真实网关测试另外记录时间、环境和结果：密码登录、实际出现的挑战类型、校内 DNS/HTTP/TCP 目标、断网/休眠恢复、TUN 回滚。

本次不把前次 test/vet 通过等同于新版架构验收；测试执行结果另见本次对话。代码阶段不能替代需要实际账号和网络的验收。
