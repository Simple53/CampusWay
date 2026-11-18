param(
    [string]$GoBin = "",
    [string]$Version = "2.0.0",
    [switch]$WithCLI
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
if ($Version -notmatch '^[0-9A-Za-z.-]+$') { throw "Invalid version: $Version" }

if (-not $GoBin -or -not (Test-Path $GoBin)) {
    $candidates = @(
        (Join-Path $PSScriptRoot '.tools\go\bin\go.exe'),
        (Join-Path $PSScriptRoot '..\.tools\go\bin\go.exe')
    )
    $found = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
    if ($found) {
        $GoBin = $found
    } elseif (Get-Command go -ErrorAction SilentlyContinue) {
        $GoBin = 'go'
    } else {
        throw "Go toolchain not found in .tools, ../.tools, or PATH"
    }
}

$env:GOCACHE = Join-Path $PSScriptRoot '.gocache'
$out = Join-Path $PSScriptRoot 'dist/routedesk'
New-Item -ItemType Directory -Force $out | Out-Null

# Desktop embeds its frontend and launches a private engine process; no separate runtime files are needed.
& $GoBin build -buildvcs=false -tags desktop,production -ldflags "-H=windowsgui -s -w -X main.version=$Version" -o "$out/RouteDesk.exe" ./gui
if ($LASTEXITCODE) { throw 'Desktop build failed' }

# Copy to root directory for easy access
try {
    Copy-Item "$out/RouteDesk.exe" "$PSScriptRoot/RouteDesk.exe" -Force -ErrorAction Stop
} catch {
    Write-Host "提示: 根目录 RouteDesk.exe 正被运行中的程序占用，最新编译产物已生成至: $out/RouteDesk.exe" -ForegroundColor Yellow
}

Copy-Item LICENSE "$out/LICENSE"
Copy-Item gui/static/icons/LICENSE "$out/ICON-LICENSE" -ErrorAction Stop

$zip = "$out/RouteDesk-$Version-windows-x64-portable.zip"
Compress-Archive -Force -Path "$out/RouteDesk.exe", "$out/LICENSE", "$out/ICON-LICENSE" -DestinationPath $zip

$hashTargets = @("$out/RouteDesk.exe", $zip)

if ($WithCLI) {
    & $GoBin build -buildvcs=false -ldflags "-s -w -X 'main.CommitID=$Version'" -o "$out/routedesk-cli.exe" .
    if ($LASTEXITCODE) { throw 'CLI build failed' }
    try {
        Copy-Item "$out/routedesk-cli.exe" "$PSScriptRoot/routedesk-cli.exe" -Force -ErrorAction Stop
    } catch {
        Write-Host "提示: 根目录 routedesk-cli.exe 正被占用，最新 CLI 产物已生成至: $out/routedesk-cli.exe" -ForegroundColor Yellow
    }
    $cliZip = "$out/routedesk-cli-$Version-windows-x64.zip"
    Compress-Archive -Force -Path "$out/routedesk-cli.exe", "$out/LICENSE" -DestinationPath $cliZip
    $hashTargets += @("$out/routedesk-cli.exe", $cliZip)
}

Get-FileHash $hashTargets | ForEach-Object { "$($_.Hash.ToLower())  $([IO.Path]::GetFileName($_.Path))" } | Set-Content -Encoding ascii "$out/SHA256SUMS.txt"
Get-ChildItem $out
