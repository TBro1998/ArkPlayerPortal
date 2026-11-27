# ArkPlayerPortal

[ArkServerCommander](https://github.com/TBro1998/ArkServerCommander) 的玩家站：服主的 Commander 为玩家开一个网页，玩家在上面看公告、服务器状态和信息、商店里的商品（ArkShop），也可以用 [ARKFGF 联机助手](https://github.com/TBro1998/ArkServerLink) 直接查看。

> 开发中，还不能使用。

## 怎么运行

玩家站由 Commander 创建和管理，服主不用手动部署：在 Commander 里打开玩家站后，Commander 拉取镜像 `tbro98/ark-player-portal` 并启动容器。

玩家站只读取 Commander 写好的数据，不连接 Commander、游戏服务器和 RCON，也没有任何凭据。容器不提权、文件系统只读。

## 镜像

推送 `v*` tag 时由 GitHub Actions 构建：

- `tbro98/ark-player-portal:<版本>`，例如 `1.4.2`。Commander 固定使用某个版本，更新 Commander 时一起更新玩家站
- `tbro98/ark-player-portal:latest`，最新的正式版本，用于手动部署和测试

## 许可

[Apache License 2.0](LICENSE)
