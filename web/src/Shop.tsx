import { useMemo, useState, type ReactNode } from "react";
import { useShop } from "./api";
import { blueprintName, formatTime } from "./format";
import { useLang, useT, type TKey } from "./i18n";
import { CopyButton } from "./Servers";
import type { DinoEntry, ItemEntry, Kit, Me, PlayerData, Requirement, SellItem, Server, Shop, ShopItem } from "./types";
import { Chip, Empty } from "./ui";

type Tab = "items" | "kits" | "sell";
type T = ReturnType<typeof useT>;

const typeKey: Record<string, TKey> = {
  item: "typeItem",
  dino: "typeDino",
  beacon: "typeBeacon",
  experience: "typeExperience",
  unlockengram: "typeUnlockengram",
  command: "typeCommand",
};
const genderKey: Record<string, TKey> = { male: "male", female: "female", random: "randomGender" };

const typeName = (t: T, type: string) => (typeKey[type] ? t(typeKey[type]) : type);

export function ShopView({ servers, selected, me }: { servers: Server[]; selected: string | null; me: Me | null }) {
  const t = useT();
  const withShop = servers.filter((s) => s.shop);
  const key = selected && withShop.some((s) => s.key === selected) ? selected : (withShop[0]?.key ?? null);
  const shop = useShop(key);

  if (withShop.length === 0) return <Empty>{t("noShops")}</Empty>;
  const server = withShop.find((s) => s.key === key)!;
  const mine = me?.user && server.points ? me.data?.[server.key] : undefined;

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
        <ShopBody shop={shop.data} server={server} mine={mine} loggedIn={!!me?.user} />
      ) : (
        <Empty>{shop.loading ? t("loading") : shop.error?.code === "shop_not_found" ? t("noShop") : shop.error?.message || t("loadFailed")}</Empty>
      )}
    </div>
  );
}

function ShopBody({ shop, server, mine, loggedIn }: { shop: Shop; server: Server; mine?: PlayerData; loggedIn: boolean }) {
  const t = useT();
  const lang = useLang();
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
    { id: "items", label: t("items"), count: shop.items.length },
    { id: "kits", label: t("kits"), count: shop.kits.length },
    { id: "sell", label: t("sell"), count: shop.sell.length },
  ];

  return (
    <div className="space-y-4">
      <div className="panel hud flex flex-wrap items-start gap-4 px-5 py-4 text-sm">
        <div className="min-w-0 flex-1">
          <p>
            {t("shopIntro")}
            <code className="mx-1 font-mono text-primary">{shop.commands.points}</code>
          </p>
          <p className="mt-1 text-xs text-muted">
            {server.name}
            {shop.updated_at && <> · {t("shopUpdated", { time: formatTime(shop.updated_at, lang) })}</>}
          </p>
        </div>
        {server.points &&
          (mine ? (
            <p className="text-lg text-amber num">{t("myPoints", { points: mine.points })}</p>
          ) : !loggedIn ? (
            <a href="#/me" className="text-xs text-muted hover:text-primary">
              {t("myPointsLogin")} →
            </a>
          ) : null)}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex gap-1 rounded-md border border-line bg-panel p-1">
          {tabs
            .filter((x) => x.count > 0 || x.id === "items")
            .map((x) => (
              <button
                key={x.id}
                onClick={() => setTab(x.id)}
                className={`rounded px-3 py-1 text-sm ${tab === x.id ? "bg-primary/15 text-primary" : "text-muted hover:text-text"}`}
              >
                {x.label}
                <span className="ml-1 text-xs opacity-70 num">{x.count}</span>
              </button>
            ))}
        </div>
        {tab === "items" && types.length > 1 && (
          <select value={type} onChange={(e) => setType(e.target.value)} className="rounded-md border border-line bg-panel px-2 py-1.5 text-sm text-text">
            <option value="">{t("allTypes")}</option>
            {types.map((x) => (
              <option key={x} value={x}>
                {typeName(t, x)}
              </option>
            ))}
          </select>
        )}
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("search")}
          className="min-w-0 flex-1 rounded-md border border-line bg-panel px-3 py-1.5 text-sm placeholder:text-muted focus:border-primary focus:outline-none sm:max-w-xs"
        />
      </div>

      {tab === "items" &&
        (items.length ? (
          <List>
            {items.map((i) => (
              <ItemRow key={i.id} item={i} shop={shop} />
            ))}
          </List>
        ) : (
          <Empty>{q || type ? t("noMatchItems") : t("noItems")}</Empty>
        ))}
      {tab === "kits" &&
        (kits.length ? (
          <List>
            {kits.map((k) => (
              <KitRow key={k.name} kit={k} shop={shop} left={mine?.kits?.[k.name]} />
            ))}
          </List>
        ) : (
          <Empty>{t("noMatchKits")}</Empty>
        ))}
      {tab === "sell" &&
        (sell.length ? (
          <List>
            {sell.map((s) => (
              <SellRow key={s.id} item={s} shop={shop} />
            ))}
          </List>
        ) : (
          <Empty>{t("noMatchSell")}</Empty>
        ))}
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

type Cmd = { label: string; build: (n: number) => string; amount: boolean };

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
  commands: Cmd[];
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
  const t = useT();
  const parts: ReactNode[] = [];
  if (req.min_level > 0 || req.max_level > 0) {
    parts.push(
      <Chip key="lv" tone="amber">
        {req.min_level > 0 && req.max_level > 0
          ? t("levelRange", { min: req.min_level, max: req.max_level })
          : req.min_level > 0
            ? t("levelMin", { min: req.min_level })
            : t("levelMax", { max: req.max_level })}
      </Chip>,
    );
  }
  if (req.permissions.length > 0) {
    parts.push(
      <Chip key="perm" tone="amber">
        {t("onlyFor", { groups: req.permissions.join(", ") })}
      </Chip>,
    );
  }
  if (req.only_from_spawn) {
    parts.push(
      <Chip key="spawn" tone="amber">
        {t("onlyFromSpawn")}
      </Chip>,
    );
  }
  return parts.length ? <div className="flex flex-wrap gap-1.5">{parts}</div> : null;
}

function CommandLine({ label, build, amount }: Cmd) {
  const t = useT();
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
          aria-label={t("amount")}
          onChange={(e) => setN(Math.min(999, Math.max(1, Math.floor(Number(e.target.value)) || 1)))}
          className="w-14 rounded border border-line bg-panel px-1.5 py-0.5 text-sm num"
        />
      )}
      <CopyButton text={text} />
    </div>
  );
}

function itemLine(t: T, e: ItemEntry): string {
  return `${blueprintName(e.blueprint)} × ${e.amount}${e.blueprint_only ? t("bp") : ""}${e.quality > 0 ? t("quality", { q: e.quality }) : ""}`;
}

function dinoLine(t: T, d: DinoEntry): string {
  const extra = [d.gender && genderKey[d.gender] && t(genderKey[d.gender]), d.neutered && t("neutered"), d.saddle_blueprint && t("withSaddle", { name: blueprintName(d.saddle_blueprint) })].filter(Boolean);
  return t("level", { name: blueprintName(d.blueprint), level: d.level }) + (extra.length ? ` (${extra.join(", ")})` : "");
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

function ItemRow({ item: i, shop }: { item: ShopItem; shop: Shop }) {
  const t = useT();
  const lines: string[] = [];
  (i.items ?? []).forEach((e) => lines.push(itemLine(t, e)));
  (i.dinos ?? []).forEach((d) => lines.push(dinoLine(t, d)));
  if (i.beacon) lines.push(t("crate", { name: i.beacon }));
  if (i.experience) lines.push(t("exp", { n: i.experience.amount }) + (i.experience.to_dino ? t("expDino") : ""));
  (i.engrams ?? []).forEach((bp) => lines.push(t("unlock", { name: blueprintName(bp) })));
  (i.commands ?? []).forEach((c) => lines.push(c));
  return (
    <Row
      title={i.description || i.id}
      badge={typeName(t, i.type)}
      price={t("points", { n: i.price })}
      req={i}
      commands={[{ label: t("buy"), amount: true, build: (n) => `${shop.commands.buy} ${i.id} ${n}` }]}
    >
      <Contents lines={lines} />
      <span className="opacity-60">{t("itemId", { id: i.id })}</span>
    </Row>
  );
}

function KitRow({ kit: k, shop, left }: { kit: Kit; shop: Shop; left?: number }) {
  const t = useT();
  const commands: Cmd[] = [];
  if (k.default_amount > 0) commands.push({ label: t("claim"), amount: false, build: () => `${shop.commands.kit} ${k.name}` });
  if (k.price !== null) commands.push({ label: t("buy"), amount: true, build: (n) => `${shop.commands.buy_kit} ${k.name} ${n}` });
  const lines = [...k.items.map((e) => itemLine(t, e)), ...k.dinos.map((d) => dinoLine(t, d)), ...k.commands];
  return (
    <Row title={k.description || k.name} badge={t("kitBadge")} price={k.price !== null ? t("points", { n: k.price }) : <span className="text-ok">{t("free")}</span>} req={k} commands={commands}>
      <Contents lines={lines} />
      <span className="opacity-60">
        {t("kitName", { name: k.name })}
        {k.default_amount > 0 && <> · {t("kitFree", { n: k.default_amount })}</>}
        {left !== undefined && <> · {t("kitLeft", { n: left })}</>}
      </span>
    </Row>
  );
}

function SellRow({ item: s, shop }: { item: SellItem; shop: Shop }) {
  const t = useT();
  return (
    <Row
      title={s.description || s.id}
      badge={t("sellBadge")}
      price={<span className="text-ok">{t("pointsGain", { n: s.price })}</span>}
      commands={[{ label: t("sellAction"), amount: true, build: (n) => `${shop.commands.sell} ${s.id} ${n}` }]}
    >
      <Contents lines={[t("give", { name: blueprintName(s.blueprint), n: s.amount })]} />
      <span className="opacity-60">{t("itemId", { id: s.id })}</span>
    </Row>
  );
}
