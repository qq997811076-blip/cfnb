# Cloudflare IP 优选工具 - Go 版 Windows 部署
$ErrorActionPreference = "Stop"
try { $Host.UI.RawUI.WindowTitle = "Cloudflare IP 优选部署" } catch {}
function Test-Administrator {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal $identity
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
if (-not (Test-Administrator)) {
  Write-Host "本脚本需要管理员权限来创建计划任务。" -ForegroundColor Yellow
  $choice = Read-Host "是否尝试以管理员身份重新运行？(Y/N)"
  if ($choice -eq 'Y' -or $choice -eq 'y') {
    $quotedScript = '"' + $PSCommandPath + '"'
    Start-Process -FilePath "powershell.exe" -Verb RunAs -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File $quotedScript"
    exit 0
  }
  Write-Host "请以管理员身份运行 setup.ps1。" -ForegroundColor Red
  exit 1
}
Set-Location $PSScriptRoot
$ScriptDir = $PSScriptRoot
$TaskName = "Cloudflare IP 优选"
$TaskIntervalMinutes = 5
$GoBinaryPath = Join-Path $ScriptDir "cfnb.exe"
$WorkingDirectory = $ScriptDir
function Refresh-EnvPath {
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
}
function Get-NextAlignedTime {
  param([int]$IntervalMinutes = 5)
  $now = Get-Date
  $total = $now.Hour * 60 + $now.Minute
  return $now.Date.AddMinutes([math]::Ceiling($total / $IntervalMinutes) * $IntervalMinutes)
}

Write-Host "Cloudflare IP 优选工具 - Go 部署" -ForegroundColor Cyan
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { Write-Host "请先安装 Go 1.22+：https://go.dev/dl/" -ForegroundColor Red; exit 1 }
  winget install GoLang.Go --accept-package-agreements --accept-source-agreements
  Refresh-EnvPath
  $goCmd = Get-Command go -ErrorAction SilentlyContinue
}
if (-not $goCmd) { Write-Host "Go 未加入 PATH，请重新打开 PowerShell 后重试。" -ForegroundColor Red; exit 1 }
$GoExePath = $goCmd.Source
$goVersion = (& $GoExePath env GOVERSION).Trim()
if ($goVersion -notmatch '^go1\.(\d+)') { Write-Host "无法识别 Go 版本：$goVersion" -ForegroundColor Red; exit 1 }
if ([int]$Matches[1] -lt 22) { Write-Host "Go $goVersion 太旧，需要 Go 1.22+。" -ForegroundColor Red; exit 1 }

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { Write-Host "请先安装 Git。" -ForegroundColor Red; exit 1 }
  winget install Git.Git --accept-package-agreements --accept-source-agreements
  Refresh-EnvPath
}
if (-not (Get-Command git -ErrorAction SilentlyContinue)) { Write-Host "Git 未加入 PATH，请重试。" -ForegroundColor Red; exit 1 }
if (-not (Get-Command curl -CommandType Application -ErrorAction SilentlyContinue)) {
  if (Get-Command winget -ErrorAction SilentlyContinue) {
    winget install cURL.cURL --accept-package-agreements --accept-source-agreements
    Refresh-EnvPath
  }
}
if (-not (Get-Command curl -CommandType Application -ErrorAction SilentlyContinue)) { Write-Host "带宽测速需要 curl，请安装后重试。" -ForegroundColor Red; exit 1 }
if (-not (Test-Path (Join-Path $ScriptDir "main.go")) -or -not (Test-Path (Join-Path $ScriptDir "go.mod"))) { Write-Host "未找到 main.go 或 go.mod。" -ForegroundColor Red; exit 1 }

Write-Host "正在构建 Go 程序..." -ForegroundColor Green
& $GoExePath build -o $GoBinaryPath .
if ($LASTEXITCODE -ne 0) { throw "Go 构建失败，退出码：$LASTEXITCODE" }
@'
config.json
git_sync.ps1
git_sync.sh
__pycache__/
cfnb
cfnb.exe
*.log
'@ | Set-Content -Path (Join-Path $ScriptDir ".gitignore") -Encoding utf8

$firstRunTime = Get-NextAlignedTime -IntervalMinutes $TaskIntervalMinutes
$startBoundaryStr = $firstRunTime.ToString("yyyy-MM-ddTHH:mm:ss")
$startTimeDisplay = $firstRunTime.ToString("HH:mm")
Write-Host "配置 Windows 计划任务 '$TaskName'，每 $TaskIntervalMinutes 分钟运行。" -ForegroundColor Yellow
try {
  $taskService = New-Object -ComObject Schedule.Service
  $taskService.Connect()
  $rootFolder = $taskService.GetFolder("\")
  try { $rootFolder.DeleteTask($TaskName, 0) } catch {}
  $definition = $taskService.NewTask(0)
  $definition.RegistrationInfo.Description = "每$($TaskIntervalMinutes)分钟运行一次 Cloudflare IP 优选 Go 工具"
  $definition.Principal.LogonType = 5
  $definition.Principal.RunLevel = 1
  $definition.Settings.Enabled = $true
  $definition.Settings.StartWhenAvailable = $false
  $definition.Settings.AllowHardTerminate = $true
  $definition.Settings.ExecutionTimeLimit = "PT72H"
  $definition.Settings.MultipleInstances = 3
  $definition.Settings.Priority = 0
  $definition.Settings.DisallowStartIfOnBatteries = $true
  $definition.Settings.StopIfGoingOnBatteries = $true
  $trigger = $definition.Triggers.Create(1)
  $trigger.StartBoundary = $startBoundaryStr
  $trigger.Repetition.Interval = "PT$($TaskIntervalMinutes)M"
  $trigger.Repetition.StopAtDurationEnd = $false
  $trigger.Enabled = $true
  $action = $definition.Actions.Create(0)
  $action.Path = $GoBinaryPath
  $action.Arguments = ""
  $action.WorkingDirectory = $WorkingDirectory
  $rootFolder.RegisterTaskDefinition($TaskName, $definition, 6, "SYSTEM", $null, 5) | Out-Null
  Write-Host "计划任务 '$TaskName' 创建成功。" -ForegroundColor Green
} catch {
  Write-Host "COM 创建失败，尝试 schtasks：$_" -ForegroundColor Yellow
  $taskCommand = '"' + $GoBinaryPath + '"'
  $args = @("/Create", "/TN", $TaskName, "/TR", $taskCommand, "/SC", "ONCE", "/ST", $startTimeDisplay, "/SD", $firstRunTime.ToString("yyyy/MM/dd"), "/RI", $TaskIntervalMinutes, "/DU", "9999/12/31", "/RU", "SYSTEM", "/F", "/RL", "HIGHEST")
  $result = & schtasks @args 2>&1
  if ($LASTEXITCODE -ne 0) { Write-Host "创建计划任务失败：$result" -ForegroundColor Red; exit 1 }
}
Write-Host "后续步骤：" -ForegroundColor Cyan
Write-Host "1. 按需编辑 config.json。"
Write-Host "2. 在 git_sync.ps1 中填写 GitHub Token、用户名、仓库名和分支。"
Write-Host "3. 查看计划任务：taskschd.msc"
$response = Read-Host "是否立即运行一次 Go 程序测试？(Y/N)"
if ($response -eq 'Y' -or $response -eq 'y') { & $GoBinaryPath }
Read-Host "部署完成，按 Enter 键退出"
