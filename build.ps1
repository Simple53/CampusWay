param(
    [string]$GoBin = "",
    [string]$Version = "0.1.1",
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
$out = Join-Path $PSScriptRoot 'dist/campusway'
New-Item -ItemType Directory -Force $out | Out-Null

# Desktop embeds its frontend and launches a private engine process; no separate runtime files are needed.
& $GoBin build -buildvcs=false -tags desktop,production -ldflags "-H=windowsgui -s -w -X main.version=$Version" -o "$out/CampusWay.exe" ./gui
if ($LASTEXITCODE) { throw 'Desktop build failed' }

# Copy to root directory for easy access
try {
    Copy-Item "$out/CampusWay.exe" "$PSScriptRoot/CampusWay.exe" -Force -ErrorAction Stop
} catch {
    Write-Host "提示: 根目录 CampusWay.exe 正被运行中的程序占用，最新编译产物已生成至: $out/CampusWay.exe" -ForegroundColor Yellow
}

Copy-Item LICENSE "$out/LICENSE"
Copy-Item gui/static/icons/LICENSE "$out/ICON-LICENSE" -ErrorAction Stop

$zip = "$out/CampusWay-$Version-windows-x64-portable.zip"
Compress-Archive -Force -Path "$out/CampusWay.exe", "$out/LICENSE", "$out/ICON-LICENSE" -DestinationPath $zip

$hashTargets = @("$out/CampusWay.exe", $zip)

if ($WithCLI) {
    & $GoBin build -buildvcs=false -ldflags "-s -w -X 'main.CommitID=$Version'" -o "$out/campusway-cli.exe" .
    if ($LASTEXITCODE) { throw 'CLI build failed' }
    try {
        Copy-Item "$out/campusway-cli.exe" "$PSScriptRoot/campusway-cli.exe" -Force -ErrorAction Stop
    } catch {
        Write-Host "提示: 根目录 campusway-cli.exe 正被占用，最新 CLI 产物已生成至: $out/campusway-cli.exe" -ForegroundColor Yellow
    }
    $cliZip = "$out/campusway-cli-$Version-windows-x64.zip"
    Compress-Archive -Force -Path "$out/campusway-cli.exe", "$out/LICENSE" -DestinationPath $cliZip
    $hashTargets += @("$out/campusway-cli.exe", $cliZip)
}

Get-FileHash $hashTargets | ForEach-Object { "$($_.Hash.ToLower())  $([IO.Path]::GetFileName($_.Path))" } | Set-Content -Encoding ascii "$out/SHA256SUMS.txt"
Get-ChildItem $out
