# 2026-11-17 连接修复与验证

## NNU EasyConnect

旧版本允许 EasyConnect 使用 aTrust 专有的 TCP 隧道模式。此时认证和本地代理端口都可以成功，但转发接口始终返回 `not implemented`，校园 DNS 也无法工作。读取旧配置、保存配置和启动引擎时均已兼容处理，EasyConnect 使用 L3 数据通道；数据通道握手完成后才显示已连接。

保活请求原先在发送前就取消了 context，已调整为请求完成后取消。关闭连接时先标记清理状态，再关闭 VPN 和数据通道，避免未完成请求触发后台 panic。TLS 数据流按 IPv4 包长度读取，支持拆包和合包。

勾选记住密码时，在 Windows 上同时通过 DPAPI 加密保存网关会话。有效会话用于重连，失效后退回一次完整认证。修改账户或密码、清除密码和删除线路会清除对应的会话。会话和令牌不显示在界面、不随配置导出。验证码与登录有效期仍由网关决定，不能承诺永久免验证。

## Clash 与浏览器接入

`PROCESS-NAME,RouteDesk.exe,DIRECT` 决定进程流量走哪条路径，不改变 Clash DNS 返回的 Fake-IP，也不会把浏览器流量转发给 RouteDesk。

网关解析或备用 DNS 返回 `198.18.0.0/15` 时，客户端会过滤虚拟地址，通过 HTTPS DNS 获取真实地址，避免将 Clash 的虚拟地址发入校园 VPN。校园 DNS 优先通过已建立的数据通道访问。

Windows 桌面连接成功后，默认启用本地 PAC，将校园域名和附加域名交给本线路的 HTTP 代理；NNU 同时识别 `nnu.edu.cn`、`njnu.edu.cn`。其他域名保留原手动代理路径；没有手动代理时使用原系统路由，因此可与 Clash TUN 共存。断开、切换和正常退出时恢复原设置，异常退出后在下次启动恢复。高级配置可关闭此行为。

自动接入适用于遵循 Windows 系统代理设置的浏览器；独立代理扩展或忽略系统代理的应用需单独配置。已有其他 PAC 时不会覆盖，会在日志提示手动接入。

## aTrust、更新和界面

- 密码和短信认证提交实际选中的登录域，例如 Radius。显式指定但不存在的域不再静默替换。
- 密码认证被拒绝时保留网关的错误原因，不再继续流程后误报 `ticket is empty`。
- 更新按语义版本比较，仅远程版本更高时提供更新，包括预发布版本和数字分段比较。
- 每条线路的开关直接连接或断开，切换时先停止原线路；等待验证时可以取消。右上角仅显示线路名称。
- 首页取消右下角缩放箭头，大小设置保留在编辑模式。复制按钮补齐 SVG，日志及导出文件含 INFO/WARN/ERROR/DEBUG 级别，目录及导出操作在设置中。

## 验证记录

2026-11-17，在用户现有 Clash 环境中使用独立本地测试端口验证：

- NNU 网关真实地址解析成功，认证、发送及接收隧道握手成功。
- 通过本地 HTTP 代理访问 `http://zgzj.njnu.edu.cn/` 返回 HTTP 200，标题为“欢迎登录 - 政工之家”。这验证的是登录页面可达，业务系统自身登录仍需用户账户。
- `http://jwgl.njnu.edu.cn/` 本次请求超时；`https://ehallapp.nnu.edu.cn/` 返回 HTTP 200，但标题为 OpenResty 默认页面，不视为业务验证通过。
- 保持连接跨过保活周期后，再次连接成功复用会话，没有第二次验证码。
- 实网测试发现的退出顺序问题随后修复，并增加“数据通道仍在等待读取时断开”的回归测试。
- Go 全量测试、前端模块检查通过。Playwright 验证逐线路连接、全部开关关闭，以及验证码弹窗取消后断开。

ZJU 本次未进行真实账户登录验证；已通过本地模拟网关验证 Radius 提交及密码拒绝错误传播。自动 PAC 的路由脚本有执行断言；Windows 实机测试已验证 PAC 设置、脚本服务可用及原代理设置恢复。浏览器经系统 PAC 访问真实校园站点仍需桌面连接后复测。

## 参考实现

- [ZJU-Connect-for-Windows 控制器](https://github.com/Mythologyli/ZJU-Connect-for-Windows/blob/master/zjuconnectcontroller/zjuconnectcontroller.cpp)：独立核心进程、标准输入验证响应及终止清理。
- [docker-easyconnect 启动逻辑](https://github.com/docker-easyconnect/docker-easyconnect/blob/master/docker-root/usr/local/bin/start-sangfor.sh)：连接生命周期和残余进程清理。
- [docker-easyconnect 用法](https://github.com/docker-easyconnect/docker-easyconnect/blob/master/doc/usage.md)：持久化登录状态，以及代理服务与宿主流量接入是分别需要处理的环节。
