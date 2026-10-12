# CloudFlare 广东电信优选 IP

> ⚡ 每小时更新一次

---

### 📥 订阅地址 [ip.txt](https://raw.githubusercontent.com/RikkaSaiko1/CloudFlare-ip/refs/heads/main/ip.txt)

```
https://raw.githubusercontent.com/RikkaSaiko1/CloudFlare-ip/refs/heads/main/ip.txt
```


### 💬 交流反馈

[📢 频道](https://t.me/Rikkasaiko0)　[👥 群组](https://t.me/rikkasaiko)

---

**欢迎反馈～**

---

### 🔗 项目来源

本项目基于 [xinyitang3/cfnb](https://github.com/xinyitang3/cfnb) 部署，感谢原作者。


## Go 迁移版

当前分支 `go-full-migration` 提供 Go 入口，保留原有 `config.json` 参数名称与配置文件格式；Go 代码仅使用标准库，不需要下载 Go 第三方模块。

- Go 1.22 或更高版本。
- Linux / Termux：运行 `sh run.sh`，或在项目目录执行 `go run .`。
- Windows PowerShell：运行 `.\\run.ps1`。
- 带宽测速仍需要系统已安装 `curl`；GitHub 同步需要 `git` 命令可用。
- GitHub 同步的认证参数从现有 `git_sync.sh`（Linux / Termux）或 `git_sync.ps1`（Windows）读取，实际同步流程由 Go 程序执行。
- 原 `main.py` 暂时保留在此迁移分支中，供逐项对照原逻辑；运行入口是 `main.go`。
