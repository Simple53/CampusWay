# CampusWay 整体重做计划

状态：计划已落盘；P0 源码研究及现版界面基线完成；P1 本地预览与前端分层完成；P2 主路径首版已实现，等待更多交互验收。后续阶段尚未验收。
此文件替代旧 redesign-plan-2026-09.md 中的前端实施安排。用户此次明确要求 plan 文件，因此计划采用可读 Markdown；实施变更说明继续保留在代码注释。

## 目标与约束

- CampusWay 保留英文品牌名。中文默认，英文可切换；所有界面文案使用消息键，不再扫描 DOM 翻译英文。
- 保留 Go 协议核心及 Wails 桌面壳。先修正交互与数据契约，再重写页面，避免一次改动协议、存储和视觉。
- 桌面默认 800×620、可缩放非全屏；手机目标 390×844，并检查 360px 宽。
- 黑白灰、跟随系统/浅色/深色主题；SVG 图标，禁止 emoji。
- 三个稳定目的地：首页、连接、设置。底部栏只导航，视图/排序/密度不进入底部栏。
- 启动直接进入首页，不弹登录。仅用户执行连接且认证需要时请求凭据。
- 不注入学校、网关、账户、密码，不自动导入历史身份。运行测试只使用独立临时目录。
- 不展示虚构流量、延迟、资源数、网络可达状态；没有真实数据就不渲染指标。

## P0：研究证据与当前缺陷

上游源码已下载到工作区 ../.references，未并入产品源码。Git clone 网络失败后改用 GitHub API 和 raw 源码读取。
已读文件（master 快照；tree.json 保存树 SHA，后续需锁定 SHA 再补齐测试）：

| 上游 | 已读源码 | 具体结论与采用边界 |
| --- | --- | --- |
| EZ4Connect | core/connectionprofile.h | endpoint/credentials/dns/proxy/tunnel/behavior/debug 分组；采用分组模型，不照搬默认学校和密码保存方式 |
| EZ4Connect | core/connectionsessionstate.cpp | Starting/Running/Stopping/Reconnecting/Failed/Interrupted；操作可用性从状态派生，不靠多处按钮临时判断 |
| EZ4Connect | application/connectionsession.cpp | 生命周期与认证挑战通过会话管理；取消输入必须停止会话 |
| EZ4Connect | application/profileservice.cpp | 配置管理与 UI 分离；选中线路与运行线路需要分别建模 |
| EZ4Connect | presentation/coordinators/connectionuicontroller.cpp | 会话输出送 ApplicationLogger；认证独立协调；系统代理与 VPN 连接是两类状态 |
| EZ4Connect | application/applicationlogger.cpp | appendCoreOutput → publishEntry → entryAdded + 输出；不能用几个状态消息冒充诊断日志 |
| EZ4Connect | presentation/dialogs/settingwindow/settingwindow.cpp 和 .ui | 密码/SMS/CAS/OAuth2 条件认证、DNS 自动选项、TUN 关联选项、附加配置及日志目录入口 |
| EZ4Connect | presentation/dialogs/extrasettingwindow/extrasettingwindow.cpp、presentation/main/mainwindow.cpp | 高级规则与主操作分离；需要继续逐项建立能力映射 |
| ZJU-Connect-for-Windows | settingwindow/settingwindow.cpp | 连接、代理、自动重连等独立配置；不会把所有选项放在主连接界面 |
| ZJU-Connect-for-Windows | zjuconnectcontroller/zjuconnectcontroller.cpp | stdout/stderr 都读取，认证请求转为独立信号；不采用命令行传密码做法 |
| ZJU-Connect-for-Windows | mainwindow.cpp | 实际日志文件生命周期、连接操作和错误反馈；学校专属 L2TP/WebLogin 不直接迁入通用校园客户端 |

原始来源：
- https://github.com/chenx-dust/EZ4Connect
- https://github.com/Mythologyli/ZJU-Connect-for-Windows
- https://www.nngroup.com/articles/progressive-disclosure/ ：常用操作优先，高级功能按需呈现，避免超过两层。
- https://developer.android.com/develop/ui/compose/components/navigation-bar ：导航项对应稳定页面，而不是弹出动作。

当前源码可证实的问题（不是截图审查结论）：
1. gui/static/app.js 同时堆积模板、路由、桥接、事件、翻译和状态，后续以函数覆盖追加实现。
2. 首页用品牌、状态、添加按钮、完整日志拼接，缺乏稳定的信息优先级。
3. 设置将多个完整表单直接平铺；版本检查让普通用户填写仓库，增加与任务无关的信息。
4. 连接编辑默认展开高级项，且 EasyConnect 与 aTrust 字段没有按协议分组/联动。
5. engine_process.go 丢弃普通上游日志，server.go 仅记录状态；没有诊断读取/导出入口和持久化链路。
6. 每秒快照变化触发整页 innerHTML 重建，可能导致滚动、焦点和输入状态丢失。
7. DOM 文本替换翻译不能可靠覆盖动态错误、组合字符串与辅助标签。
8. 上次交付只有测试和构建，没有真实 UI 验收，不应称为完成的设计。

## 页面职责与操作路径

### 首页

单个连接摘要：状态、当前运行线路或待连接线路、一个主要操作。无线路时：短说明及“添加连接”；有线路但未连接时：所选线路及“连接”；连接中可取消，已连接可断开，失败显示可执行的原因及“查看诊断”。
仅已连接时显示真实代理监听地址和会话时长。底部最多一条最近事件，不铺完整日志，不堆指标卡。不显示账户密码。首页快捷线路选择只切换待连接目标，不暗中切断当前连接。

### 连接

标题 + 添加；搜索按线路数量按需显示；表格/方块在本页工具栏。线路记录呈现名称、网关、协议、真实状态；运行与选中分别标记。编辑/复制/删除进入记录操作区，不在每张卡上堆满按钮。空态不出现 undefined 或空选择面板。

编辑为页面内子视图（手机全宽），只有保存/取消一组动作：
- 基础：名称、协议、网关、端口。
- 认证：按协议与认证方式显示账户/手机号/登录域；密码始终空白输入，已保存仅显示状态。
- 高级默认折叠，展开后分 DNS、代理、连接行为；不以一个超长表单堆砌。
- aTrust 密码、SMS 先做端到端验收；CAS/OAuth2 需要认证发现和浏览器回调闭环，未完成前明确不可用，不伪装成已支持。
- EasyConnect 专属项与 aTrust 专属项互斥呈现。TUN/系统代理/转发必须有实际平台服务才开放。

### 设置

根页只显示分类行和当前值，进入后有标题及返回：
- 账户管理：按线路编辑用户自行输入的账户、清除已保存凭据；不新增全局账户体系。
- 连接管理：进入连接页，管理默认线路和实际已实现的行为。
- 数据管理：脱敏线路导入/导出、操作结果；导入预检查，无隐式身份恢复。
- 日志与诊断：实时事件列表、时间/级别/阶段、筛选、复制/脱敏导出、清空（确认）。明确本次会话/历史日志范围。
- 外观：主题、列表布局、密度；统一一个配置来源。
- 语言：中文/English，切换后全部页面、对话框与错误更新。
- 关于与更新：当前应用/核心版本；发布源由构建配置决定；未配置时显示说明，不要求用户输入 GitHub 仓库，不制造检查成功。

## 数据和前端架构

保留无构建依赖的 ES modules，先拆为：
- gui/static/app.js：启动、订阅、路由。
- gui/static/core/bridge.js：Wails 命令与异常适配；生产无 mock fallback。
- gui/static/core/store.js：snapshot、选择、会话状态；更新与页面草稿分离。
- gui/static/core/i18n.js：zh-CN/en 词典、错误码映射、日期格式。
- gui/static/components/：shell、navigation、status、field、dialog、setting-row。
- gui/static/pages/：home、connections、connection-editor、settings、diagnostics。
- gui/static/styles/：tokens、shell、components、pages、responsive。
- gui/diagnostics.go：有界诊断事件及脱敏导出；记录代码和阶段，不转发原始认证响应。

连接状态和主按钮由单一 view-model 计算。页面跳转与保存不改变运行会话；快照更新只刷新状态区域，不重建用户正在编辑的表单。
诊断事件至少含 seq/time/level/code/phase/sessionId；日志与会话状态分离。不得通过直接展示 raw stdout 修复“无日志”：密码、OTP、Cookie、token、认证响应永不进入可导出的日志。

## 分阶段执行与验收门槛

每阶段先更新本文件状态及证据，再进入下一阶段；未通过不打包发布。

- [x] P0-A：实际读取两个上游的关键源码，并记录采用/不采用边界。
- [x] P0-B：搜索交互设计原则，形成页面职责、能力边界和技术分层。
- [x] P0-C：当前版本桌面/手机截图及关键路径审查；浏览器桥接预览必须明确为测试夹具，不等同于原生 Wails 验证。
- [x] P1：实现独立预览与测试夹具，冻结现版截图；拆出 i18n、桥接、store 和 shell，不同时扩张功能。
- [ ] P2：首页/连接/设置主路径重建，空态及有线路状态桌面/手机对照验收；不生成安装包。
- [ ] P3：配置分组与协议联动；字段→Profile→engine Config 映射测试；取消编辑不改数据，运行线路不可被无提示改写。
- [ ] P4：补真实诊断链路、导出、级别/会话过滤及持久化策略，测试故障分类与脱敏。
- [ ] P5：数据/账户/外观/语言/版本设置闭环；切换语言不丢草稿；未配置更新源正确反馈。
- [ ] P6：真实桌面与手机尺寸交互审查、键盘导航、主题/缩放检查；原生 Wails 启动与故障注入验证；然后重新打包。

验收矩阵：首次启动、无线路、单线路、多线路、切换待连接、切换运行连接、连接中取消、验证码取消、认证失败、代理端口占用、导入草稿、删除运行线路、中文/英文、浅/深/系统主题。
重点尺寸：800×620、640×560、390×844、360×740；100%/125%/150% 桌面缩放。
完成证据：页面截图 + 实际操作结果 + 后端测试，不能用“编译通过”代替。

## 当前进度与限制

现已开始按阶段实施，本地网页预览为当前审阅入口。当前 exe/zip 是旧实现，不代表本计划已完成。真实校园账户登录与 Android/iOS 原生 VPN 服务不在已验证范围。


## 本地网页预览（当前交付入口）

- 启动：在仓库根目录运行 `./preview.ps1`；或 `python scripts/preview-ui.py`。
- 地址：http://127.0.0.1:4174；直接界面：http://127.0.0.1:4174/app。
- 控制台：桌面 800×620、手机 390×844、网页自适应；画布缩放不改变 iframe 的实际布局尺寸。
- 场景：无线路、多线路、已连接、验证码、认证失败；切换场景重置内存数据。
- 默认中文，应用设置内可切换 English；模拟路由使用 .invalid，账户为空。
- 模拟器位于 scripts/preview，不嵌入发布应用；前端生产入口只使用 Wails bridge。
- 无需 npm 安装或前端打包；改 gui/static 后点击控制台刷新即可。

### 执行记录

- P0 完成：两个上游关键源码、交互研究、现版基线截图；见 BASELINE.md 和 SOURCES.json。
- P1 完成：bridge/store/i18n/ui/pages 拆分；消除旧版覆盖函数和 DOM 翻译；新增独立本地预览控制台和内存模拟器。
- P2 首版：重建首页连接摘要、线路方块/表格、分类设置入口和页面式编辑；高级配置默认收起，aTrust/EasyConnect 字段联动。尚未标记整个 P2 验收完成。
- 浏览器实际操作：模拟连接成功、验证码输入与继续、中文切换英文再切回、创建无个人信息的演示线路、手机表格/编辑页、场景切换；390px 视口 document scrollWidth=390。
- 自动检查：模拟器成功/失败/验证码/取消与重置时序、导入失败无部分写入、导出不含身份密码；预览 HTTP 路由与路径隔离；JS 语法；go test ./gui。
- 下一步：继续 P2 状态矩阵与主题/尺寸审查，随后做 P3 字段映射和 P4 真实诊断。当前日志页仅显示后端已有事件，模拟日志不等于真实 VPN 诊断链路完成。
- 此轮未打安装包，也未验证真实校园登录、原生 Wails 窗口或手机原生 VPN 服务。

### 线路与首页模块调整（2026-11-14）

- 连接页选择线路即启动/切换连接，移除底部独立连接区；点击当前运行线路幂等。凭据仍需由用户提供，获取凭据前保留原连接。
- 每条线路分别提供状态圆点（灰/黄/绿/红）及 SVG 编辑按钮；颜色同时配有状态文字，选中不代表已连接。
- 首页拆分 connection-status、network-speed、connection-logs 三个模块；桌面两列概览加日志，手机纵向排列。
- 新增 internal/traffic：统计本地 HTTP/SOCKS TCP 出站连接的成功读写字节，通过 worker 每秒采样进入快照，不统计整机网速、认证响应或 UDP。
- 预览通过独立模拟器提供标注清晰的网速样本；正式运行不使用模拟值，过期采样显示不可用。
- 验证：Go GUI/service/traffic 测试及 vet、模拟器测试和线路/网速视图回归通过；浏览器验证单击连接、自动切换、重复点击、独立编辑和表格行为。真实校园网络吞吐仍待登录后验证。

### 铅笔图标、主题与发布前检查（2026-11-14）

- 线路编辑统一使用 Lucide pencil SVG，保留图标许可。
- 配色独立于亮暗模式：石墨、海蓝、青绿、紫藤、玫瑰、琥珀；后端校验并持久化，旧配置迁移默认石墨。
- 新增 scripts/check.ps1，统一执行 Go test/vet、JS 语法、模拟器/首页/配色对比度回归和隔离 HTTP 测试；GitHub main 推送时执行同一检查及 Windows 构建/worker smoke。
- 已实际查看海蓝深色桌面与手机线路页；编辑仍不触发连接，状态圆点保持独立的成功/警告/错误语义。
- 本次推送目标为 Simple53/CampusWay；保留上游 origin，使用 campusway 远程；自动化通过不代表真实校园账号及全部系统运行环境已经验收。

发布前检查结果：scripts/check.ps1 全部通过；Windows 构建与 worker smoke 通过；actionlint v1.7.12 校验全部工作流通过；暂存区 diff --check 通过。修复旧 action 版本，macOS ARM 构建使用当前受支持的 macos-15。Git push --dry-run 已验证目标仓库可写，首次推送无需覆盖远端历史。
