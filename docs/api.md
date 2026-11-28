# 玩家站接口

浏览器里的网页和 ARKFGF 联机助手都用这组接口。快照接口只读、不需要登录，返回的就是 Commander 写的快照（见 [snapshot.md](snapshot.md)）；登录后能看自己的数据（见下文「登录」）。

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

## 登录

登录只用来看自己的数据（ArkShop 点数、礼包剩余次数），玩家站只拿到 SteamID、昵称和头像。

**`GET /api/v1/me`**（会话用 `Authorization: Bearer <session>` 或 cookie）→ `{"user": {"steam_id", "name", "avatar"} 或 null, "steam_login": true, "link_login": true, "data": {"<服务器 key>": {"points", "total_spent", "kits"}}}`。没登录时 `user` 为 `null`、没有 `data`。`link_login` 为 false 时（Commander 关闭了和 arkfgf.com 的连接）联机助手不能登录。

**网页：Steam 登录**

- `GET /auth/steam/login` 跳到 Steam 登录页（OpenID），同时设置一个 10 分钟的 state cookie
- `GET /auth/steam/callback` 校验 state 和回调地址，向 Steam 核验（`check_authentication`）后设置 HttpOnly 的会话 cookie（30 天），跳回 `/#/me`；失败时跳到 `/#/me?error=<原因>`
- 回调地址用 `PORTAL_PUBLIC_URL`（Commander 里填的访问地址），没有时按请求的 Host 算（受信任的反向代理给的 `X-Forwarded-Proto` 决定 http / https）

**联机助手：复用 arkfgf.com 的登录**

1. 助手生成随机 nonce，`GET /api/v1/identity?nonce=<16–128 位 [A-Za-z0-9_-]>` → `{"public_key", "audience", "signature"}`。助手验证签名（Ed25519，签的是 `"arkfgf-portal-identity
" + nonce`），并确认 `audience` 是 `public_key` 的 SHA-256（十六进制）
2. 助手用自己的 arkfgf.com 登录 `POST https://arkfgf.com/api/v1/auth/portal-token {"audience"}` → `{"token", "expires_at"}`（5 分钟有效，EdDSA 签名，`aud` 是这个玩家站）
3. `POST /api/v1/login {"token"}` → `{"session", "expires_at", "user"}`。玩家站用 `PORTAL_ARKFGF_KEYS_URL`（arkfgf.com 的 `GET /api/v1/auth/portal-keys`）的公钥校验签名、签发方、`aud` 和有效期。错误：401 `invalid_token`，没有配置公钥地址时 404 `link_login_disabled`
4. 之后用 `Authorization: Bearer <session>` 请求 `/api/v1/me`

arkfgf.com 的登录令牌不会发给玩家站，玩家站拿到的令牌只对它自己有效，别的玩家站用不了。

**`POST /api/v1/logout`** 清除会话 cookie，204。

其他 `GET` 请求返回网页（单页应用，找不到的路径返回 `index.html`）。

## 限制

- 只接受 `GET` 和 `HEAD`（另外 `POST /api/v1/login`、`POST /api/v1/logout`），其他方法 405
- 不设置 CORS 头：网页和接口同源；联机助手在自己的 Go 进程里请求，不经过浏览器
- 按来源 IP 限制请求频率和 SSE 连接数，另有全站的 SSE 连接上限。联机房的玩家经房间节点访问，在玩家站看来来自同一个地址，所以按 IP 的上限比较宽
- 反向代理后面时设置 `PORTAL_TRUSTED_PROXIES`（逗号分隔的 IP 或 CIDR），来自这些地址的请求按 `X-Forwarded-For` 里最后一个不受信任的地址计算来源 IP；没有设置时一律用连接的地址
- 网页的响应带 `Content-Security-Policy`（只允许加载自身的脚本和样式，禁止被嵌入其他页面）、`X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`

## 联机助手怎么用

- 怎么找到玩家站：Commander 发布到 arkfgf.com 收录站时带上玩家站地址（`portal_url`），服务器详情页的「用联机助手打开」按钮是 `arkfgf-link://portal?url=<地址>`；玩家也可以在助手里粘贴服主给的地址。联机房：Commander 登记房间时带上玩家站端口（`portal_port`），加入响应里返回，助手把本机一个 TCP 端口转发到房间节点的这个端口，直连后访问 `http://127.0.0.1:<本机端口>`。网站只经手地址和端口，内容由助手直接向玩家站请求
- 助手在自己的 Go 进程里请求，每 15 秒带 `If-None-Match` 刷新一次（没有用 SSE），不跟随跳转
- 助手登录了 arkfgf.com 时自动按上面的步骤登录玩家站，会话只在内存里
- 内容来自服主的机器，助手把它当作不可信的输入：请求超时 10 秒，响应最多 4 MB，只接受 JSON，所有字段只当纯文本显示，链接只打开 `http`、`https`，并且要玩家点击
- 不把玩家站的网页加载进助手的窗口，用助手自己的界面显示，另外提供「在浏览器中打开」（联机房的地址只在本机有效，不提供）
