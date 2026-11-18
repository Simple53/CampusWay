#!/usr/bin/env bash
set -euo pipefail
platform="$1"
version="${2:-${RELEASE_VERSION:-2.0.0}}"
version="${version#v}"
if [[ "${GITHUB_REF_TYPE:-}" == tag ]]; then version="${GITHUB_REF_NAME#v}"; fi
[[ "$version" =~ ^[0-9A-Za-z.-]+$ ]] || exit 2
short_version="${version%%-*}"
mkdir -p dist/packages build/staging
if [[ "$platform" == macos-* ]]; then
  dmg_dir="build/staging/dmg"
  app="$dmg_dir/RouteDesk.app"
  rm -rf "$dmg_dir"
  mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
  CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -buildvcs=false -tags desktop,production -ldflags "-s -w -X main.version=$version" -o "$app/Contents/MacOS/RouteDesk" ./gui
  cp build/assets/campusway.icns "$app/Contents/Resources/routedesk.icns"
  cp LICENSE "$app/Contents/Resources/LICENSE"
  cp gui/static/icons/LICENSE "$app/Contents/Resources/ICON-LICENSE"
  cat > "$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>RouteDesk</string>
<key>CFBundleIdentifier</key><string>app.campusway.desktop</string>
<key>CFBundleName</key><string>RouteDesk</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>$short_version</string>
<key>CFBundleVersion</key><string>$short_version</string>
<key>CFBundleIconFile</key><string>routedesk</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
EOF
  codesign --force --deep --sign - "$app"
  dmg="dist/packages/RouteDesk-$version-$platform.dmg"
  for attempt in 1 2 3 4 5; do
    rm -f "$dmg"
    if hdiutil create -volname RouteDesk -srcfolder "$dmg_dir" -fs HFS+ -ov -format UDZO "$dmg"; then
      break
    fi
    sleep 3
  done
  [[ -f "$dmg" ]]
else
  root="build/staging/linux"
  rm -rf "$root"
  mkdir -p "$root/usr/bin" "$root/usr/share/applications" "$root/usr/share/icons/hicolor/512x512/apps" "$root/usr/share/doc/routedesk" "$root/DEBIAN"
  go build -buildvcs=false -tags desktop,production,webkit2_41 -ldflags "-s -w -X main.version=$version" -o "$root/usr/bin/routedesk" ./gui
  cp build/assets/campusway.png "$root/usr/share/icons/hicolor/512x512/apps/routedesk.png"
  cp LICENSE "$root/usr/share/doc/routedesk/copyright"
  cp gui/static/icons/LICENSE "$root/usr/share/doc/routedesk/icon-license"
  printf '[Desktop Entry]\nType=Application\nName=RouteDesk\nExec=routedesk\nIcon=routedesk\nCategories=Network;\nTerminal=false\n' > "$root/usr/share/applications/routedesk.desktop"
  printf 'Package: routedesk\nVersion: %s\nArchitecture: amd64\nMaintainer: RouteDesk contributors\nDescription: VPN connection manager\nDepends: libgtk-3-0, libwebkit2gtk-4.1-0\n' "$version" > "$root/DEBIAN/control"
  dpkg-deb --build --root-owner-group "$root" "dist/packages/RouteDesk-$version-$platform.deb"
  tar -czf "dist/packages/RouteDesk-$version-$platform.tar.gz" -C "$root" usr
fi
