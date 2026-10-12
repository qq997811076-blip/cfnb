#!/usr/bin/env bash
# Cloudflare IP 优选工具 - Go 版 Linux/Termux 部署
set -euo pipefail
TASK_INTERVAL_MINUTES=5
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"
CYAN='\033[0;36m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
echo -e "$CYANCloudflare IP 优选工具 - Go 部署$NC"
echo "工作目录: $SCRIPT_DIR"

PM=""
if command -v pkg >/dev/null 2>&1; then PM="pkg"
elif command -v apt-get >/dev/null 2>&1; then PM="apt"
elif command -v dnf >/dev/null 2>&1; then PM="dnf"
elif command -v yum >/dev/null 2>&1; then PM="yum"
elif command -v pacman >/dev/null 2>&1; then PM="pacman"
fi
install_named() {
  local tool="$1" package="$1"
  case "$PM:$tool" in pkg:go) package="golang";; apt:go) package="golang-go";; dnf:go|yum:go) package="golang";; pacman:go) package="go";; esac
  case "$PM" in
    pkg) pkg install -y "$package" ;;
    apt) if [[ $EUID -eq 0 ]]; then apt-get update && apt-get install -y "$package"; else sudo apt-get update && sudo apt-get install -y "$package"; fi ;;
    dnf|yum|pacman)
      local cmd="$PM"
      if [[ $EUID -ne 0 ]]; then cmd="sudo $PM"; fi
      if [[ "$PM" == "pacman" ]]; then $cmd -S --noconfirm "$package"; else $cmd install -y "$package"; fi ;;
    *) echo -e "$RED未找到支持的包管理器，无法自动安装 $tool。$NC" >&2; exit 1 ;;
  esac
}
if ! command -v go >/dev/null 2>&1; then echo -e "$YELLOW安装 Go...$NC"; install_named go; fi
if ! command -v go >/dev/null 2>&1; then echo "请手动安装 Go 1.22+ 后重试。" >&2; exit 1; fi
GO_VERSION="$(go env GOVERSION | sed 's/^go//')"
GO_MINOR="$(printf '%s' "$GO_VERSION" | awk -F. '{print $2}')"
if [[ -z "$GO_MINOR" || "$GO_MINOR" -lt 22 ]]; then echo "Go $GO_VERSION 太旧，需要 Go 1.22+。" >&2; exit 1; fi
if ! command -v git >/dev/null 2>&1; then install_named git; fi
if ! command -v curl >/dev/null 2>&1; then install_named curl; fi
if [[ ! -f "$SCRIPT_DIR/main.go" || ! -f "$SCRIPT_DIR/go.mod" ]]; then echo "未找到 main.go 或 go.mod，请确认项目目录。" >&2; exit 1; fi

echo -e "$GREEN正在构建 Go 程序...$NC"
go build -o "$SCRIPT_DIR/cfnb" .
chmod +x "$SCRIPT_DIR/cfnb"
cat > "$SCRIPT_DIR/.gitignore" <<'EOF'
config.json
git_sync.ps1
git_sync.sh
__pycache__/
cfnb
cfnb.exe
*.log
EOF
if [[ -f "$SCRIPT_DIR/git_sync.sh" ]]; then chmod +x "$SCRIPT_DIR/git_sync.sh"; fi
echo -e "$GREEN构建成功：$SCRIPT_DIR/cfnb$NC"

if command -v crontab >/dev/null 2>&1; then
  echo -e "$YELLOW配置定时任务：每 $TASK_INTERVAL_MINUTES 分钟$NC"
  NICE_PREFIX=""
  if [[ $EUID -eq 0 ]]; then NICE_PREFIX="nice -n -20 "; fi
  CRON_CMD="*/$TASK_INTERVAL_MINUTES * * * * cd \"$SCRIPT_DIR\" && $NICE_PREFIX\"$SCRIPT_DIR/cfnb\" >> \"$SCRIPT_DIR/cron.log\" 2>&1"
  if crontab -l 2>/dev/null | grep -F "$SCRIPT_DIR/cfnb" >/dev/null; then
    echo "定时任务已存在，跳过添加。"
  else
    (crontab -l 2>/dev/null || true; echo "# Cloudflare IP 优选工具 Go 定时任务（每5分钟）"; echo "$CRON_CMD") | crontab -
    echo "定时任务已添加。"
  fi
else
  echo -e "$YELLOW未检测到 crontab，跳过自动定时任务。$NC"
fi
echo "1. 按需编辑 config.json。"
echo "2. 在 git_sync.sh 中填写 GitHub Token、用户名、仓库名和分支。"
echo "3. 查看日志：tail -f cron.log；管理任务：crontab -e"
read -r -p "是否立即运行一次 Go 程序测试？(y/N) " answer
if [[ "$answer" =~ ^[Yy]$ ]]; then "$SCRIPT_DIR/cfnb"; fi
