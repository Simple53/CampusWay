param([string]$GoBin = "..\.tools\go\bin\go.exe", [string]$Version = "0.2.0-alpha.1")
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
if ($Version -notmatch '^[0-9A-Za-z.-]+$') { throw "Invalid version" }
if (!(Test-Path $GoBin)) { $GoBin = 'go' }
$env:GOCACHE = Join-Path $PSScriptRoot '.gocache'
$out = Join-Path $PSScriptRoot 'dist/campusway'
New-Item -ItemType Directory -Force $out | Out-Null
# Desktop embeds its frontend and launches a private engine process; no separate runtime files are needed.
& $GoBin build -buildvcs=false -tags desktop,production -ldflags "-H=windowsgui -s -w -X main.version=$Version" -o "$out/CampusWay.exe" ./gui
if ($LASTEXITCODE) { throw 'Desktop build failed' }
Copy-Item LICENSE "$out/LICENSE"
Copy-Item gui/static/icons/LICENSE "$out/ICON-LICENSE" -ErrorAction Stop
$zip = "$out/CampusWay-$Version-windows-x64.zip"
Compress-Archive -Force -Path "$out/CampusWay.exe","$out/LICENSE","$out/ICON-LICENSE" -DestinationPath $zip
Get-FileHash "$out/CampusWay.exe",$zip | ForEach-Object { "$($_.Hash.ToLower())  $([IO.Path]::GetFileName($_.Path))" } | Set-Content -Encoding ascii "$out/SHA256SUMS.txt"
Get-ChildItem $out
