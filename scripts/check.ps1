param([string]$GoBin = '')
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $PSScriptRoot)

if (-not $GoBin -or -not (Test-Path $GoBin)) {
    $candidates = @(
        (Join-Path (Get-Location) '.tools\go\bin\go.exe'),
        (Join-Path (Get-Location) '..\.tools\go\bin\go.exe')
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

$env:GOCACHE = Join-Path (Get-Location) '.gocache'
& $GoBin test ./... -timeout 180s
if ($LASTEXITCODE) { throw 'Go tests failed' }
& $GoBin vet ./...
if ($LASTEXITCODE) { throw 'Go vet failed' }

$sourceFiles = Get-ChildItem gui/static, scripts/preview -Recurse -File | Where-Object { $_.Extension -in '.js', '.mjs' }
foreach ($file in $sourceFiles) {
    node --check $file.FullName
    if ($LASTEXITCODE) { throw "Syntax failed: $($file.Name)" }
}
foreach ($testFile in Get-ChildItem scripts/preview/*.test.mjs) {
    node $testFile.FullName
    if ($LASTEXITCODE) { throw "Failed: $($testFile.Name)" }
}
python scripts/test_preview.py
if ($LASTEXITCODE) { throw 'Preview HTTP tests failed' }
Write-Output 'All local automated checks passed.'
