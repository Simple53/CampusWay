## Run as a System Service (RouteDesk CLI)

**Always verify your configuration manually in the terminal before enabling a background service, so repeated login failures do not trigger a temporary account or IP lock.**

### Linux (systemd)

1. Download the `routedesk-cli-vX.Y.Z-linux-*.tar.gz` archive matching your architecture from [GitHub Releases](https://github.com/Simple53/CampusWay/releases), extract it, and place the binary at `/opt/routedesk-cli` with `chmod +x /opt/routedesk-cli`.
2. Create `/etc/routedesk/config.toml` based on `config.toml.example` in the repository root.
3. Create `/lib/systemd/system/routedesk.service`:

   ```ini
   [Unit]
   Description=RouteDesk CLI VPN Service
   After=network-online.target
   Wants=network-online.target

   [Service]
   Restart=on-failure
   RestartSec=5s
   ExecStart=/opt/routedesk-cli -config /etc/routedesk/config.toml

   [Install]
   WantedBy=multi-user.target
   ```

4. Enable and start the service:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now routedesk
   ```

### macOS (launchd)

1. Download `routedesk-cli-vX.Y.Z-macos-*.tar.gz` from [GitHub Releases](https://github.com/Simple53/CampusWay/releases) and place `routedesk-cli` in `/usr/local/bin/routedesk-cli`.
2. Remove quarantine attributes if needed: `sudo xattr -rd com.apple.quarantine /usr/local/bin/routedesk-cli`.
3. Create `/etc/routedesk/config.toml` and `/Library/LaunchDaemons/com.routedesk.cli.plist`:

   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
       <dict>
           <key>Label</key>
           <string>com.routedesk.cli</string>
           <key>ProgramArguments</key>
           <array>
               <string>/usr/local/bin/routedesk-cli</string>
               <string>-config</string>
               <string>/etc/routedesk/config.toml</string>
           </array>
           <key>RunAtLoad</key>
           <true/>
           <key>KeepAlive</key>
           <true/>
       </dict>
   </plist>
   ```

4. Load or unload the service:

   ```zsh
   sudo chown root:wheel /Library/LaunchDaemons/com.routedesk.cli.plist
   sudo launchctl load /Library/LaunchDaemons/com.routedesk.cli.plist
   # Disable service:
   # sudo launchctl unload /Library/LaunchDaemons/com.routedesk.cli.plist
   ```

### OpenWrt (procd)

1. Place the Linux `routedesk-cli` binary at `/usr/bin/routedesk-cli` with executable permissions.
2. Create `/etc/routedesk.toml` based on `config.toml.example` (setting `proxy_all = true` is recommended when routing through OpenWrt proxy plugins).
3. Save `/etc/init.d/routedesk` with executable permissions:

   ```shell
   #!/bin/sh /etc/rc.common

   USE_PROCD=1
   START=60
   STOP=03

   PROGRAM="/usr/bin/routedesk-cli"
   CONFIG_FILE="/etc/routedesk.toml"
   LOG_FILE="/var/log/routedesk.log"

   start_service() {
       procd_open_instance
       procd_set_param command /bin/sh -c "${PROGRAM} -config ${CONFIG_FILE} >>${LOG_FILE} 2>&1"
       procd_set_param respawn 3600 5 3
       procd_set_param file ${CONFIG_FILE}
       procd_close_instance
   }
   ```

4. Enable the service via `/etc/init.d/routedesk enable && /etc/init.d/routedesk start`, configure `127.0.0.1:1080` (SOCKS5) or `127.0.0.1:1081` (HTTP) in your proxy plugin, and keep your campus VPN gateway hostname set to direct connection to avoid routing loops.
