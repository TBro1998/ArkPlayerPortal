import { useMemo, useState, type ReactNode } from "react";
import { useShop } from "./api";
import { blueprintName, formatTime } from "./format";
import { CopyButton } from "./Servers";
import type { DinoEntry, ItemEntry, Kit, Requirement, SellItem, Server, Shop, ShopItem } from "./types";
import { Chip, Empty } from "./ui";

type Tab = "items" | "kits" | "sell";

const typeLabel: Record<string, string> = {
  item: "物品",
  dino: "恐龙",
  beacon: "补给箱",
  experience: "经验",
  unlockengram: "解锁图纸",
  command: "特殊",
};

const genderLabel: Record<string, string> = { male: "公", female: "母", random: "随机性别" };

export function ShopView({ servers, selected }: { servers: Server[]; selected: string | null }) {
  const withShop = servers.filter((s) => s.shop);
  const key = selected && withShop.some((s) => s.key === selected) ? selected : (withShop[0]?.key ?? null);
  const shop = useShop(key);

  if (withShop.length === 0) return <Empty>没有服务器开放商店</Empty>;
  const server = withShop.find((s) => s.key === key)!;

  return (
    <div className="space-y-4">
      {withShop.length > 1 && (
        <div className="flex flex-wrap gap-2">
          {withShop.map((s) => (
            <a
              key={s.key}
              href={`#/shop/${s.key}`}
              className={`rounded-md border px-3 py-1.5 text-sm transition-colors ${
                s.key === key ? "border-primary bg-primary/10 text-primary" : "border-line bg-panel text-muted hover:text-text"
              }`}
            >
              {s.name}
            </a>
          ))}
        </div>
      )}
      {shop.data ? (
        <ShopBody shop={shop.data} server={server} />
      ) : (
        <Empty>{shop.loading ? "加载中…" : shop.error?.code === "shop_not_found" ? "这台服务器没有商店" : shop.error?.message || "加载失败"}</Empty>
      )}
    </div>
  );
}

function ShopBody({ shop, server }: { shop: Shop; server: Server }) {
  const [tab, setTab] = useState<Tab>("items");
  const [query, setQuery] = useState("");
  const [type, setType] = useState("");
  const q = query.trim().toLowerCase();

  const types = useMemo(() => [...new Set(shop.items.map((i) => i.type))], [shop.items]);
  const items = useMemo(
    () => shop.items.filter((i) => (!type || i.type === type) && (!q || matches(q, i.id, i.description, ...contentNames(i)))),
    [shop.items, type, q],
  );
  const kits = useMemo(
    () => shop.kits.filter((k) => !q || matches(q, k.name, k.description, ...k.items.map((e) => blueprintName(e.blueprint)), ...k.dinos.map((d) => blueprintName(d.blueprint)))),
    [shop.kits, q],
  );
  const sell = useMemo(() => shop.sell.filter((s) => !q || matches(q, s.id, s.description, blueprintName(s.blueprint))), [shop.sell, q]);

  const tabs: { id: Tab; label: string; count: number }[] = [
    { id: "items", label: "商品", count: shop.items.length },
    { id: "kits", label: "礼包", count: shop.kits.length },
    { id: "sell", label: "回收", count: shop.sell.length },
  ];

  return (
    <div className="space-y-4">
      <div className="panel hud px-5 py-4 text-sm">
        <p>
          在游戏里按 Enter 打开聊天框输入命令购买，点数不够或不满足条件时游戏里会提示。查看自己的点数：
          <code className="mx-1 font-mono text-primary">{shop.commands.points}</code>
        </p>
        <p className="mt-1 text-xs text-muted">
          {server.name}
          {shop.updated_at && <> · 更新于 {formatTime(shop.updated_at)}</>}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex gap-1 rounded-md border border-line bg-panel p-1">
          {tabs
            .filter((t) => t.count > 0 || t.id === "items")
            .map((t) => (
              <button
                key={t.id}
                onClick={() => setTab(t.id)}
                className={`rounded px-3 py-1 text-sm ${tab === t.id ? "bg-primary/15 text-primary" : "text-muted hover:text-text"}`}
              >
                {t.label}
                <span className="ml-1 text-xs opacity-70 num">{t.count}</span>
              </button>
            ))}
        </div>
        {tab === "items" && types.length > 1 && (
          <select
            value={type}
            onChange={(e) => setType(e.target.value)}
            className="rounded-md border border-line bg-panel px-2 py-1.5 text-sm text-text"
          >
            <option value="">全部类型</option>
            {types.map((t) => (
              <option key={t} value={t}>
                {typeLabel[t] ?? t}
              </option>
            ))}
          </select>
        )}
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索名称、编号"
          className="min-w-0 flex-1 rounded-md border border-line bg-panel px-3 py-1.5 text-sm placeholder:text-muted focus:border-primary focus:outline-none sm:max-w-xs"
        />
      </div>

      {tab === "items" &&
        (items.length ? (
          <List>{items.map((i) => <ItemRow key={i.id} item={i} shop={shop} />)}</List>
        ) : (
          <Empty>{q || type ? "没有符合条件的商品" : "没有商品"}</Empty>
        ))}
      {tab === "kits" && (kits.length ? <List>{kits.map((k) => <KitRow key={k.name} kit={k} shop={shop} />)}</List> : <Empty>没有符合条件的礼包</Empty>)}
      {tab === "sell" && (sell.length ? <List>{sell.map((s) => <SellRow key={s.id} item={s} shop={shop} />)}</List> : <Empty>没有符合条件的回收项</Empty>)}
    </div>
  );
}

function matches(q: string, ...fields: string[]) {
  return fields.some((f) => f.toLowerCase().includes(q));
}

function contentNames(i: ShopItem): string[] {
  return [
    ...(i.items ?? []).map((e) => blueprintName(e.blueprint)),
    ...(i.dinos ?? []).map((d) => blueprintName(d.blueprint)),
    ...(i.engrams ?? []).map(blueprintName),
    ...(i.commands ?? []),
    i.beacon ?? "",
  ];
}

function List({ children }: { children: ReactNode }) {
  return <ul className="grid gap-3 lg:grid-cols-2">{children}</ul>;
}

function Row({
  title,
  badge,
  price,
  req,
  children,
  commands,
}: {
  title: string;
  badge?: string;
  price: ReactNode;
  req?: Requirement & { only_from_spawn?: boolean };
  children?: ReactNode;
  commands: { label: string; build: (n: number) => string; amount: boolean }[];
}) {
  return (
    <li className="panel flex flex-col gap-2 px-4 py-3">
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            {badge && <span className="rounded bg-panel-2 px-1.5 py-0.5 text-[11px] text-muted">{badge}</span>}
            <h3 className="font-medium break-words">{title}</h3>
          </div>
        </div>
        <div className="shrink-0 text-right text-sm text-amber num">{price}</div>
      </div>
      {children && <div className="text-xs leading-relaxed text-muted">{children}</div>}
      {req && <Requirements req={req} />}
      <div className="mt-auto space-y-1.5 pt-1">
        {commands.map((c) => (
          <CommandLine key={c.label} {...c} />
        ))}
      </div>
    </li>
  );
}

function Requirements({ req }: { req: Requirement & { only_from_spawn?: boolean } }) {
  const parts: ReactNode[] = [];
  if (req.min_level > 0 || req.max_level > 0) {
    parts.push(
      <Chip key="lv" tone="amber">
        {req.min_level > 0 && req.max_level > 0
          ? `等级 ${req.min_level}–${req.max_level}`
          : req.min_level > 0
            ? `等级 ≥ ${req.min_level}`
            : `等级 ≤ ${req.max_level}`}
      </Chip>,
    );
  }
  if (req.permissions.length > 0) {
    parts.push(
      <Chip key="perm" tone="amber">
        仅限 {req.permissions.join("、")}
      </Chip>,
    );
  }
  if (req.only_from_spawn) {
    parts.push(
      <Chip key="spawn" tone="amber">
        只能在刚出生时领取
      </Chip>,
    );
  }
  return parts.length ? <div className="flex flex-wrap gap-1.5">{parts}</div> : null;
}

function CommandLine({ label, build, amount }: { label: string; build: (n: number) => string; amount: boolean }) {
  const [n, setN] = useState(1);
  const text = build(n);
  return (
    <div className="flex items-center gap-2 rounded-md border border-line bg-bg/60 px-2 py-1.5">
      <span className="shrink-0 text-xs text-muted">{label}</span>
      <code className="min-w-0 flex-1 truncate font-mono text-sm" title={text}>
        {text}
      </code>
      {amount && (
        <input
          type="number"
          min={1}
          max={999}
          value={n}
          aria-label="数量"
          onChange={(e) => setN(Math.min(999, Math.max(1, Math.floor(Number(e.target.value)) || 1)))}
          className="w-14 rounded border border-line bg-panel px-1.5 py-0.5 text-sm num"
        />
      )}
      <CopyButton text={text} />
    </div>
  );
}

function ItemRow({ item: i, shop }: { item: ShopItem; shop: Shop }) {
  return (
    <Row
      title={i.description || i.id}
      badge={typeLabel[i.type] ?? i.type}
      price={<>{i.price} 点</>}
      req={i}
      commands={[{ label: "购买", amount: true, build: (n) => `${shop.commands.buy} ${i.id} ${n}` }]}
    >
      <ItemContents item={i} />
      <span className="opacity-60">编号 {i.id}</span>
    </Row>
  );
}

function ItemContents({ item: i }: { item: ShopItem }) {
  const lines: string[] = [];
  (i.items ?? []).forEach((e) => lines.push(itemLine(e)));
  (i.dinos ?? []).forEach((d) => lines.push(dinoLine(d)));
  if (i.beacon) lines.push(`补给箱 ${i.beacon}`);
  if (i.experience) lines.push(`${i.experience.amount} 经验${i.experience.to_dino ? "（给骑乘的恐龙）" : ""}`);
  (i.engrams ?? []).forEach((bp) => lines.push(`解锁 ${blueprintName(bp)}`));
  (i.commands ?? []).forEach((c) => lines.push(c));
  return <Contents lines={lines} />;
}

function Contents({ lines }: { lines: string[] }) {
  if (lines.length === 0) return null;
  return (
    <ul className="mb-1 space-y-0.5">
      {lines.map((l, idx) => (
        <li key={idx} className="break-words">
          · {l}
        </li>
      ))}
    </ul>
  );
}

function itemLine(e: ItemEntry): string {
  return `${blueprintName(e.blueprint)} × ${e.amount}${e.blueprint_only ? "（图纸）" : ""}${e.quality > 0 ? `，品质 ${e.quality}` : ""}`;
}

function dinoLine(d: DinoEntry): string {
  const extra = [d.gender && genderLabel[d.gender], d.neutered && "已绝育", d.saddle_blueprint && `带 ${blueprintName(d.saddle_blueprint)}`].filter(Boolean);
  return `${blueprintName(d.blueprint)} ${d.level} 级${extra.length ? `（${extra.join("，")}）` : ""}`;
}

function KitRow({ kit: k, shop }: { kit: Kit; shop: Shop }) {
  const commands = [];
  if (k.default_amount > 0) commands.push({ label: "领取", amount: false, build: () => `${shop.commands.kit} ${k.name}` });
  if (k.price !== null) commands.push({ label: "购买", amount: true, build: (n: number) => `${shop.commands.buy_kit} ${k.name} ${n}` });
  const lines = [...k.items.map(itemLine), ...k.dinos.map(dinoLine), ...k.commands];
  return (
    <Row
      title={k.description || k.name}
      badge="礼包"
      price={k.price !== null ? <>{k.price} 点</> : <span className="text-ok">免费</span>}
      req={k}
      commands={commands}
    >
      <Contents lines={lines} />
      <span className="opacity-60">
        名称 {k.name}
        {k.default_amount > 0 && <> · 每人免费 {k.default_amount} 次</>}
      </span>
    </Row>
  );
}

function SellRow({ item: s, shop }: { item: SellItem; shop: Shop }) {
  return (
    <Row
      title={s.description || s.id}
      badge="回收"
      price={<span className="text-ok">+{s.price} 点</span>}
      commands={[{ label: "出售", amount: true, build: (n) => `${shop.commands.sell} ${s.id} ${n}` }]}
    >
      <Contents lines={[`交出 ${blueprintName(s.blueprint)} × ${s.amount}`]} />
      <span className="opacity-60">编号 {s.id}</span>
    </Row>
  );
}
