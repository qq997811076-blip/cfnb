$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error '未检测到 Go 环境，请先安装 Go。'
    exit 1
}
go run .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
