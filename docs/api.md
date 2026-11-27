# 玩家站接口

浏览器里的网页和 ARKFGF 联机助手都用这组接口。所有接口都只读、不需要登录，返回的就是 Commander 写的快照（见 [snapshot.md](snapshot.md)）。

和快照一样，接口随玩家站版本一起改，不做新旧版本兼容。

## 地址

玩家站在容器里监听 `PORTAL_LISTEN`（默认 `:8080`），Commander 把它映射到宿主机上服主设置的端口。

- 发布到收录站的服务器：`http://<公网 IP>:<端口>`，或者服主自己配置的域名（反向代理，可以加 HTTPS）
- P2P 联机房：联机助手加入房间后，把本机的一个空闲端口转发到房间节点的这个端口，访问 `http://127.0.0.1:<本机端口>`

## 接口

错误统一 `{"error": "中文说明", "code": "错误码"}`。请求过于频繁时返回 429 `rate_limited` 和 `Retry-After`。

**`GET /api/v1/portal`** → `portal.json` 的内容。还没有快照时 503 `not_ready`。

**`GET /api/v1/shops/:key`** → `shops/<key>.json` 的内容。没有这台服务器的商店时 404 `shop_not_found`。`key` 只能是数字。

这两个接口带 `ETag`（内容的哈希）和 `Cache-Control: no-cache`，请求带 `If-None-Match` 且内容没变时返回 304。

**`GET /api/v1/events`** Server-Sent Events，快照变化时推送通知，只告诉客户端什么变了，客户端自己重新请求上面的接口：

```
event: portal
data: {}

event: shop
data: {"key": "3"}
```

- 刚连上时不推送，客户端先请求一次 `/api/v1/portal`
- 每 25 秒发一行注释（`: ping`）保持连接
- 连接数太多时 503 `too_many_connections`，客户端稍后重连（浏览器的 `EventSource` 会自动重连）

**`GET /healthz`** → `200 ok`。

其他 `GET` 请求返回网页（单页应用，找不到的路径返回 `index.html`）。

## 限制

- 只接受 `GET` 和 `HEAD`，其他方法 405
- 不设置 CORS 头：网页和接口同源；联机助手在自己的 Go 进程里请求，不经过浏览器
- 按来源 IP 限制请求频率和 SSE 连接数，另有全站的 SSE 连接上限。联机房的玩家经房间节点访问，在玩家站看来来自同一个地址，所以按 IP 的上限比较宽
- 反向代理后面时设置 `PORTAL_TRUSTED_PROXIES`（逗号分隔的 IP 或 CIDR），来自这些地址的请求按 `X-Forwarded-For` 里最后一个不受信任的地址计算来源 IP；没有设置时一律用连接的地址
- 网页的响应带 `Content-Security-Policy`（只允许加载自身的脚本和样式，禁止被嵌入其他页面）、`X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`

## 联机助手怎么用

- 怎么找到玩家站：Commander 发布到 arkfgf.com 收录站时带上玩家站地址（`portal_url`），服务器详情页的「用联机助手打开」按钮是 `arkfgf-link://portal?url=<地址>`；玩家也可以在助手里粘贴服主给的地址。联机房：Commander 登记房间时带上玩家站端口（`portal_port`），加入响应里返回，助手把本机一个 TCP 端口转发到房间节点的这个端口，直连后访问 `http://127.0.0.1:<本机端口>`。网站只经手地址和端口，内容由助手直接向玩家站请求
- 助手在自己的 Go 进程里请求，每 15 秒带 `If-None-Match` 刷新一次（没有用 SSE），不跟随跳转
- 内容来自服主的机器，助手把它当作不可信的输入：请求超时 10 秒，响应最多 4 MB，只接受 JSON，所有字段只当纯文本显示，链接只打开 `http`、`https`，并且要玩家点击
- 不把玩家站的网页加载进助手的窗口，用助手自己的界面显示，另外提供「在浏览器中打开」（联机房的地址只在本机有效，不提供）
