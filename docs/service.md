## 作为系统服务运行 (CampusWay CLI)

**请先在终端直接手动运行 `campusway-cli` 并确认认证、代理与分流无误后再注册为系统服务，避免配置错误反复重试导致账号或 IP 被临时锁定。**

### Linux (systemd)

对于 Ubuntu/Debian、RHEL、Arch 等基于 Systemd 的 Linux 发行版，可按以下步骤将 `campusway-cli` 注册为后台守护服务：

1. 前往 [GitHub Releases](https://github.com/Simple53/CampusWay/releases) 下载对应架构的 `campusway-cli-vX.Y.Z-linux-*.tar.gz`，解压并将 `campusway-cli` 放置到 `/opt/campusway-cli`，赋予执行权限：`chmod +x /opt/campusway-cli`。
2. 在 `/etc/campusway/` 下创建配置文件 `/etc/campusway/config.toml`（可参考仓库根目录的 `config.toml.example`）。
3. 创建 `/lib/systemd/system/campusway.service`：

   ```ini
   [Unit]
   Description=CampusWay CLI VPN Service
   After=network-online.target
   Wants=network-online.target

   [Service]
   Restart=on-failure
   RestartSec=5s
   ExecStart=/opt/campusway-cli -config /etc/campusway/config.toml

   [Install]
   WantedBy=multi-user.target
   ```

4. 启用并启动服务：

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now campusway
   ```

### macOS (launchd)

1. 前往 [GitHub Releases](https://github.com/Simple53/CampusWay/releases) 下载对应架构的 `campusway-cli-vX.Y.Z-macos-*.tar.gz`，解压并将 `campusway-cli` 放置到 `/usr/local/bin/campusway-cli`。
2. 若系统提示未签名限制，执行：`sudo xattr -rd com.apple.quarantine /usr/local/bin/campusway-cli`。
3. 准备配置文件 `/etc/campusway/config.toml`，并在 `/Library/LaunchDaemons/com.campusway.cli.plist` 创建如下内容：

   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
       <dict>
           <key>Label</key>
           <string>com.campusway.cli</string>
           <key>ProgramArguments</key>
           <array>
               <string>/usr/local/bin/campusway-cli</string>
               <string>-config</string>
               <string>/etc/campusway/config.toml</string>
           </array>
           <key>RunAtLoad</key>
           <true/>
           <key>KeepAlive</key>
           <true/>
       </dict>
   </plist>
   ```

4. 加载或卸载服务：

   ```zsh
   sudo chown root:wheel /Library/LaunchDaemons/com.campusway.cli.plist
   sudo launchctl load /Library/LaunchDaemons/com.campusway.cli.plist
   # 停用服务:
   # sudo launchctl unload /Library/LaunchDaemons/com.campusway.cli.plist
   ```

### OpenWrt (procd)

1. 将编译好的 Linux `campusway-cli` 二进制放入 `/usr/bin/campusway-cli` 并赋予执行权限。
2. 参考 `config.toml.example` 创建 `/etc/campusway.toml`，建议设置 `proxy_all = true` 以配合路由器代理插件分流。
3. 保存 `/etc/init.d/campusway` 并赋予执行权限：

   ```shell
   #!/bin/sh /etc/rc.common

   USE_PROCD=1
   START=60
   STOP=03

   PROGRAM="/usr/bin/campusway-cli"
   CONFIG_FILE="/etc/campusway.toml"
   LOG_FILE="/var/log/campusway.log"

   start_service() {
       procd_open_instance
       procd_set_param command /bin/sh -c "${PROGRAM} -config ${CONFIG_FILE} >>${LOG_FILE} 2>&1"
       procd_set_param respawn 3600 5 3
       procd_set_param file ${CONFIG_FILE}
       procd_close_instance
   }
   ```

4. 启用服务：`/etc/init.d/campusway enable && /etc/init.d/campusway start`，随后在代理插件中添加 `127.0.0.1:1080`（SOCKS5）或 `127.0.0.1:1081`（HTTP）节点并注意确保校园 VPN 网关域名本身走直连。
