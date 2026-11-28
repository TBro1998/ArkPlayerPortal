// 通用页面：只认 heading、paragraph、list、kv 四种块，不认识的块跳过。所有文本只当纯文本显示
import { pick, useLang, useT } from "./i18n";
import type { Block, Page, Portal, Text } from "./types";
import { Empty } from "./ui";

export function pageTitle(page: Page, portal: Portal, lang: "zh" | "en"): string {
  const title = pick(page.title, lang);
  if (page.server_key) {
    const server = portal.servers.find((s) => s.key === page.server_key);
    if (server) return `${title} · ${server.name}`;
  }
  return title;
}

export function PageView({ portal, id }: { portal: Portal; id: string }) {
  const t = useT();
  const lang = useLang();
  const page = (portal.pages ?? []).find((p) => p.id === id);
  if (!page) return <Empty>{t("pageNotFound")}</Empty>;
  return (
    <article className="panel hud space-y-3 px-6 py-5">
      <a href="#/" className="text-xs text-muted hover:text-primary">
        {t("back")}
      </a>
      <h2 className="text-xl font-semibold break-words">{pageTitle(page, portal, lang)}</h2>
      {page.blocks.map((b, i) => (
        <BlockView key={i} block={b} />
      ))}
    </article>
  );
}

function BlockView({ block }: { block: Block }) {
  const t = useT();
  const lang = useLang();
  const text = (v: unknown) => pick(v as Text, lang);
  switch (block.type) {
    case "heading":
      return <h3 className="pt-2 text-sm font-semibold uppercase tracking-widest text-primary">{text(block.text)}</h3>;
    case "paragraph":
      return <p className="whitespace-pre-line break-words text-sm leading-relaxed text-text/85">{text(block.text)}</p>;
    case "list":
      return (
        <ul className="list-disc space-y-1 pl-5 text-sm text-text/85">
          {(block.items ?? []).map((it, i) => (
            <li key={i} className="break-words">
              {text(it)}
            </li>
          ))}
        </ul>
      );
    case "kv":
      return (
        <dl className="grid gap-x-6 gap-y-1.5 text-sm sm:grid-cols-2">
          {((block.items ?? []) as { label: Text; value: unknown }[]).map((it, i) => (
            <div key={i} className="flex justify-between gap-4 border-b border-line/60 py-1">
              <dt className="text-muted">{text(it.label)}</dt>
              <dd className="text-right num">{kvValue(it.value, t)}</dd>
            </div>
          ))}
        </dl>
      );
    default:
      return null;
  }
}

function kvValue(v: unknown, t: ReturnType<typeof useT>): string {
  if (typeof v === "boolean") return v ? t("on") : t("off");
  if (typeof v === "number") return Number.isInteger(v) ? String(v) : String(Math.round(v * 1000) / 1000);
  if (typeof v === "string") return v;
  return "";
}

// PageLinks 首页的页面入口（服主写的页面；服务器设置页从服务器卡片进入）
export function PageLinks({ portal }: { portal: Portal }) {
  const lang = useLang();
  const pages = (portal.pages ?? []).filter((p) => p.kind === "custom");
  if (pages.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-2">
      {pages.map((p) => (
        <a key={p.id} href={`#/page/${encodeURIComponent(p.id)}`} className="panel px-4 py-2.5 text-sm hover:border-primary hover:text-primary">
          {pick(p.title, lang)}
        </a>
      ))}
    </div>
  );
}
