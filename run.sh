#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
if ! command -v go >/dev/null 2>&1; then
  echo "未检测到 Go 环境。请先安装 Go（Termux：pkg install golang）。" >&2
  exit 1
fi
exec go run .
