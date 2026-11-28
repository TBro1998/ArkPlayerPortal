# 快照格式（Commander → 玩家站）

玩家站显示的所有内容都来自 Commander 写好的快照文件。玩家站只读这些文件，不连接 Commander、游戏服务器和 RCON。

快照格式和玩家站一起改：Commander 固定使用某个版本的玩家站镜像，更新 Commander 时玩家站跟着更新，所以字段可以直接增删改，不做新旧版本兼容。

## 目录

Commander 的数据目录 `portal/` 以只读方式挂到玩家站容器的 `/data`（`PORTAL_DATA_DIR`）：

```
portal/
├── portal.json        站点资料、公告、服务器、页面
├── identity.key       玩家站的身份密钥（Ed25519 种子，base64），不公开
├── shops/
│   └── <key>.json     一台服务器的商店（ArkShop），key 是服务器在 Commander 里的 ID
└── players/
    └── <key>.json     这台服务器上每个玩家自己的数据（点数等），不公开
```

- 挂载整个目录，不要单独挂文件：Commander 先写临时文件再改名替换（同一目录内），单独挂载的文件被替换后容器里看到的还是旧文件
- 玩家站每秒检查一次文件的修改时间和大小，变了就重新读取。读到的不是合法 JSON 时（例如正好读到一半）保留上一次的内容，下一次再读
- `portal.json` 不存在时玩家站显示「还没有数据」；某台服务器的商店文件不存在表示它没有商店
- 内容只在变化时重写：服务器状态、人数变化，发了公告，商店配置变化。没有变化时不要重写，否则会给所有打开页面的玩家推送一次更新

所有文本都是纯文本，玩家站不解析 HTML 和 Markdown。时间都是 RFC 3339（UTC）。

有些文本可以是两种语言：`{"zh": "服务器设置", "en": "Server settings"}`，客户端按界面语言选（缺了就用另一种）。服主写的内容（公告、页面、商品描述）是普通字符串，不翻译。

## portal.json

```json
{
  "updated_at": "2026-11-27T12:00:00Z",
  "site": {
    "name": "示例方舟",
    "description": "PvE 休闲服，每周三维护",
    "qq_group": "123456789",
    "discord": "https://discord.gg/example",
    "links": [{"title": "服务器规则", "url": "https://example.com/rules"}]
  },
  "announcements": [
    {
      "id": "k3j9x2",
      "title": "周三停服维护",
      "body": "11 月 27 日 14:00 停服更新，预计一小时。\n更新内容：……",
      "pinned": true,
      "published_at": "2026-11-26T08:00:00Z",
      "updated_at": "2026-11-26T09:30:00Z"
    }
  ],
  "servers": [
    {
      "key": "3",
      "game": "ase",
      "name": "示例方舟 | 孤岛 | 3 倍",
      "map": "TheIsland",
      "status": "online",
      "players": 12,
      "max_players": 70,
      "password": false,
      "pvp": false,
      "arkapi": true,
      "cluster_group": 1,
      "mods": [{"id": "731604991", "title": "Structures Plus (S+)"}],
      "rates": {"xp": 3, "taming": 5, "harvest": 3, "breeding": 10},
      "version": "358.24",
      "address": "203.0.113.10:7777",
      "query_port": 27015,
      "room_id": 0,
      "next_restart_at": "2026-11-28T04:00:00Z",
      "shop": true,
      "settings_page": "settings-3",
      "points": true
    }
  ],
  "pages": [
    {
      "id": "1",
      "kind": "custom",
      "title": "服务器规则",
      "blocks": [
        {"type": "heading", "text": "基本规则"},
        {"type": "list", "items": ["禁止外挂", "禁止辱骂"]},
        {"type": "paragraph", "text": "第一次警告，第二次封禁。"}
      ]
    },
    {
      "id": "settings-3",
      "kind": "settings",
      "title": {"zh": "服务器设置", "en": "Server settings"},
      "server_key": "3",
      "blocks": [
        {"type": "heading", "text": {"zh": "倍率", "en": "Rates"}},
        {"type": "kv", "items": [{"label": {"zh": "经验", "en": "XP"}, "value": 3}, {"label": {"zh": "PvP", "en": "PvP"}, "value": false}]}
      ]
    }
  ]
}
```

`site`：服主在 Commander 里填写的站点资料。`name` 为空时玩家站显示第一台服务器的名称。`links` 只接受 `http`、`https` 地址，玩家站显示时再检查一次。

`announcements`：按 `published_at` 从新到旧，玩家站把 `pinned` 的放在最前面。`id` 在一个 Commander 里唯一，不会变；`body` 里的换行原样显示。

`servers`：开启了玩家站的服务器，按 Commander 里的顺序。

| 字段 | 说明 |
|---|---|
| `key` | 服务器在 Commander 里的 ID，也是商店文件名 |
| `game` | `ase` / `asa` |
| `status` | `online`（可以连接）、`starting`（启动中、下载中、加载中）、`offline` |
| `players` | 在线人数，`status` 不是 `online` 时为 0 |
| `cluster_group` | 同一 Commander 里集群的编号（同一 ClusterID 编号相同，从 1 开始），不属于集群为 0。ClusterID 本身不写进快照 |
| `mods` | 按加载顺序，`id` 是创意工坊 ID（ASE）或 CurseForge 项目 ID（ASA），`title` 是创意工坊上的名称（ASA 和查不到时为空，客户端显示 ID） |
| `rates` | 经验、驯服、采集、繁殖倍率，没有设置时为 1 |
| `version` | 游戏版本，不知道时为空 |
| `address` | 玩家直接连接的地址 `IP:游戏端口`，只有发布到收录站的服务器才有，否则为空 |
| `query_port` | ASE 的查询端口（加进 Steam 收藏用），ASA 和没有 `address` 时为 0 |
| `room_id` | 这台服务器所在的 P2P 联机房号，不在联机房里为 0 |
| `next_restart_at` | 下一次定时重启或停服的时间，没有时为 `null` |
| `shop` | 有没有商店文件 |
| `settings_page` | 这台服务器的「服务器设置」页（`pages` 里的 `id`） |
| `points` | 登录后能不能看自己在这台服务器商店里的点数（有 `players/<key>.json`） |

## 页面（pages）

通用页面：服主写的规则、新手指南等（`kind: custom`，按服主排的顺序），加上每台服务器自动生成的「服务器设置」页（`kind: settings`，`server_key` 是哪台服务器）。以后新增的内容也尽量做成页面，客户端不用改。

页面由块组成，客户端只认下面四种，**不认识的块跳过**：

| `type` | 字段 | 显示 |
|---|---|---|
| `heading` | `text` | 小标题 |
| `paragraph` | `text` | 段落，保留换行 |
| `list` | `items`：文本数组 | 列表 |
| `kv` | `items`：`[{"label": 文本, "value": 字符串、数字或布尔值}]` | 名称和值的表格，布尔值显示成开 / 关 |

`title`、`text`、`label`、列表项都可以是字符串或两种语言的文本。服主写的页面内容在 Commander 里是纯文本：「# 」开头的行变成 `heading`，「- 」开头的连续行变成 `list`，其余的连续行是一个 `paragraph`，空行分隔。

## identity.key

玩家站的身份密钥，Commander 第一次开启玩家站时生成，文件只给玩家站容器的用户读。玩家站用它：

- 向联机助手证明自己（`GET /api/v1/identity`），arkfgf.com 签发的登录令牌只对这个身份（公钥的 SHA-256）有效
- 派生签会话用的密钥，密钥换了以后所有会话作废

## players/&lt;key&gt;.json

```json
{
  "updated_at": "2026-11-27T12:00:00Z",
  "players": {
    "76561198000000001": {"points": 120, "total_spent": 30, "kits": {"starter": 1}}
  }
}
```

Commander 从这台服务器 ArkShop 的 SQLite 库里读出来（复制一份再读，不锁住插件），库文件变了才重新读。只有 ASE、ArkShop 用自己的 SQLite 库（没有开 MySQL、没有改库路径）时才有。`kits` 是每个礼包还能领的次数。这些文件不公开，玩家站只把登录玩家自己的那一条返回给他。

## shops/&lt;key&gt;.json

从这台服务器的 ArkShop `config.json` 整理出来，**只包含给玩家看的字段**。`Mysql` 整段和 `General` 里的设置一律不写；`command` 类商品和礼包里的命令只写 `DisplayAs`，命令本身不写（可能是以管理员身份执行的命令）；没有 `DisplayAs` 时不写这一条。

```json
{
  "updated_at": "2026-11-27T12:00:00Z",
  "commands": {"buy": "/buy", "kit": "/kit", "buy_kit": "/buykit", "points": "/points", "shop": "/shop", "sell": "/sell"},
  "items": [
    {
      "id": "ingots100",
      "type": "item",
      "description": "Metal Ingot (100x)",
      "price": 15,
      "min_level": 0,
      "max_level": 0,
      "permissions": [],
      "items": [{"blueprint": "Blueprint'/Game/PrimalEarth/CoreBlueprints/Resources/PrimalItemResource_MetalIngot.PrimalItemResource_MetalIngot'", "amount": 100, "quality": 0, "blueprint_only": false}]
    },
    {
      "id": "carno",
      "type": "dino",
      "description": "Male Carnotaurus",
      "price": 50,
      "min_level": 0,
      "max_level": 0,
      "permissions": [],
      "dinos": [{"blueprint": "Blueprint'/Game/PrimalEarth/Dinos/Carno/Carno_Character_BP.Carno_Character_BP'", "level": 15, "neutered": true, "gender": "male", "saddle_blueprint": "Blueprint'/Game/PrimalEarth/CoreBlueprints/Items/Armor/Saddles/PrimalItemArmor_CarnoSaddle.PrimalItemArmor_CarnoSaddle'"}]
    },
    {"id": "crate25", "type": "beacon", "description": "SupplyCrate Lvl 25", "price": 100, "min_level": 0, "max_level": 0, "permissions": [], "beacon": "SupplyCrate_Level25_Double_C"},
    {"id": "exp1000", "type": "experience", "description": "1000 points of experience", "price": 55, "min_level": 0, "max_level": 0, "permissions": [], "experience": {"amount": 1000, "to_dino": false}},
    {"id": "tekengram", "type": "unlockengram", "description": "Engram TEK Boots & Gloves", "price": 20, "min_level": 0, "max_level": 0, "permissions": [], "engrams": ["Blueprint'/Game/PrimalEarth/CoreBlueprints/Items/Armor/TEK/PrimalItemArmor_TekBoots.PrimalItemArmor_TekBoots'"]},
    {"id": "allengrams", "type": "command", "description": "All engrams", "price": 1000, "min_level": 0, "max_level": 0, "permissions": [], "commands": ["Gives you all engram unlocks"]}
  ],
  "kits": [
    {
      "name": "starter",
      "description": "Starter kit (carno with saddle)",
      "price": 90,
      "default_amount": 2,
      "min_level": 0,
      "max_level": 0,
      "permissions": [],
      "only_from_spawn": true,
      "items": [{"blueprint": "…", "amount": 1, "quality": 0, "blueprint_only": false}],
      "dinos": [{"blueprint": "…", "level": 10, "neutered": false, "gender": "", "saddle_blueprint": ""}],
      "commands": []
    }
  ],
  "sell": [
    {"id": "metal", "description": "100x stone", "price": 10, "amount": 100, "blueprint": "…"}
  ]
}
```

- `commands`：ArkShop `Messages` 里的聊天命令名（`BuyCmd`、`KitCmd`、`BuyKitCmd`、`PointsCmd`、`ShopCmd`、`SellCmd`），服主可能改成别的名字。没有配置的用括号里的默认值：`/buy`、`/kit`、`/buykit`、`/points`、`/shop`、`/sell`
- 顺序和 `config.json` 里的一致（`ShopItems`、`Kits`、`SellItems` 是 JSON 对象，Commander 按原文的键顺序读取）
- `items[].type`：`item`、`dino`、`beacon`、`experience`、`unlockengram`、`command`。只填对应类型的字段（`items` / `dinos` / `beacon` / `experience` / `engrams` / `commands`），其余省略。不认识的类型照写 `type`，不写内容
- `min_level`、`max_level` 为 0 表示不限制；`permissions` 是可以购买的权限组（ArkShop 的 `Permissions`，逗号分隔拆开），空数组表示所有人
- `kits[].price` 为 `null` 时只能用免费次数领取（`/kit`），不能用点数购买（ArkShop 的 `Price` 为 -1 或没有设置）；`default_amount` 是每个玩家的免费次数
- `dinos[].gender`：`male`、`female`、`random`，没有设置为空
- 蓝图路径原样写入，玩家站从里面取出类名显示（`PrimalItemResource_MetalIngot` 显示为 `MetalIngot`）。商品的名称以服主写的 `description` 为主
- 命令格式（玩家站据此生成可复制的命令）：商品 `{buy} <id> <数量>`，买礼包 `{buy_kit} <name> <数量>`，领礼包 `{kit} <name>`，回收 `{sell} <id> <数量>`
