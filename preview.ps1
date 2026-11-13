param([int]$Port = 4174)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
python scripts/preview-ui.py --port $Port
