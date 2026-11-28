# ArkPlayerPortal

[ArkServerCommander](https://github.com/TBro1998/ArkServerCommander) 的玩家站：服主的 Commander 为玩家开一个网页（中文 / 英文），玩家在上面看公告、服务器状态和设置、服主写的规则和指南、商店里的商品（ArkShop），用 Steam 登录后看自己的点数，也可以用 [ARKFGF 联机助手](https://github.com/TBro1998/ArkServerLink) 直接查看。内容变化时打开着的页面自动更新。

商店只用来看：玩家在游戏里用聊天命令（`/buy`、`/kit` 等）购买，玩家站给出可以直接复制的命令。

## 怎么运行

玩家站由 Commander 创建和管理，服主不用手动部署：在 Commander 里打开玩家站后，Commander 拉取镜像并启动容器。

玩家站只读取 Commander 写好的快照文件，不连接 Commander、游戏服务器和 RCON，也没有任何凭据。镜像里只有一个静态程序（没有 shell），以非 root 用户运行，可以用只读文件系统、去掉所有 capability 运行：

```bash
docker run -d --name ark-player-portal --read-only --cap-drop ALL --security-opt no-new-privileges \
  -p 8090:8080 -v /path/to/commander/data/portal:/data:ro tbro98/ark-player-portal:latest
```

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `PORTAL_DATA_DIR` | `/data` | 快照目录，见 [docs/snapshot.md](docs/snapshot.md) |
| `PORTAL_LISTEN` | `:8080` | 监听地址 |
| `PORTAL_TRUSTED_PROXIES` | 空 | 反向代理的 IP 或 CIDR（逗号分隔），来自它们的请求按 `X-Forwarded-For` 取来源 IP |
| `PORTAL_PUBLIC_URL` | 空 | 玩家访问的地址，Steam 登录的回调地址用；为空时按请求的 Host 算 |
| `PORTAL_ARKFGF_KEYS_URL` | 空 | arkfgf.com 的公钥地址（`https://arkfgf.com/api/v1/auth/portal-keys`），设置了才能用联机助手登录 |

接口见 [docs/api.md](docs/api.md)。

## 开发

```bash
# 前端（web/），Vite + React + Tailwind CSS
cd web && npm ci && npm run build   # 构建到 web/dist，打包进 Go 程序

# 后端，用示例数据运行
PORTAL_DATA_DIR=dev/data PORTAL_LISTEN=127.0.0.1:8080 go run .

# 改前端时：上面的后端开着，另开一个终端
cd web && npm run dev               # http://127.0.0.1:5173，接口转给 :8080

# 检查
gofmt -l . && go vet ./... && go test ./...
cd web && npx tsc -b
```

`dev/data/` 是示例快照（三台服务器、公告、一个商店），改里面的文件可以看到页面实时更新。没有构建前端时 `go build` 也能通过，网页显示「还没有构建」。

## 镜像

推送 `v*` tag 时由 GitHub Actions 构建：

- `tbro98/ark-player-portal:<版本>`，例如 `1.4.2`。Commander 固定使用某个版本，更新 Commander 时一起更新玩家站
- `tbro98/ark-player-portal:latest`，最新的正式版本，用于手动部署和测试

快照和接口随版本直接修改，不做新旧版本兼容。

## 许可

[Apache License 2.0](LICENSE)
