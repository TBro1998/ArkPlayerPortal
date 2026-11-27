import { useEffect, useState } from "react";
import { usePortal, useLiveStatus } from "./api";
import { Announcements } from "./Announcements";
import { Servers } from "./Servers";
import { ShopView } from "./Shop";
import { safeURL } from "./format";
import type { Portal } from "./types";
import { Chip, SectionTitle, sortAnnouncements } from "./ui";

// 路由放在 hash 里（#/、#/news、#/shop/3），刷新和分享链接都不需要服务端配合
type Route = { page: "home" } | { page: "news" } | { page: "shop"; key: string | null };

function parseRoute(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/");
  if (parts[0] === "news") return { page: "news" };
  if (parts[0] === "shop") return { page: "shop", key: /^\d+$/.test(parts[1] ?? "") ? parts[1] : null };
  return { page: "home" };
}

function useRoute(): Route {
  const [route, setRoute] = useState(() => parseRoute(location.hash));
  useEffect(() => {
    const on = () => setRoute(parseRoute(location.hash));
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);
  return route;
}

export default function App() {
  const portal = usePortal();
  const route = useRoute();
  const live = useLiveStatus();
  const p = portal.data;

  const siteName = p?.site.name || p?.servers[0]?.name || "玩家站";
  useEffect(() => {
    document.title = siteName;
  }, [siteName]);

  return (
    <div className="mx-auto flex min-h-screen max-w-6xl flex-col px-4 pb-10 sm:px-6">
      <Header portal={p} siteName={siteName} route={route} live={live} />
      <main className="mt-6 flex-1">
        {!p ? (
          <Placeholder loading={portal.loading} message={portal.error?.message} notReady={portal.error?.code === "not_ready"} />
        ) : route.page === "news" ? (
          <Announcements items={p.announcements} />
        ) : route.page === "shop" ? (
          <ShopView servers={p.servers} selected={route.key} />
        ) : (
          <Home portal={p} />
        )}
      </main>
      <footer className="mt-10 text-center text-xs text-muted">
        由 <a className="hover:text-primary" href="https://github.com/TBro1998/ArkServerCommander" target="_blank" rel="noreferrer noopener">ArkServerCommander</a> 提供
      </footer>
    </div>
  );
}

function Header({ portal, siteName, route, live }: { portal: Portal | null; siteName: string; route: Route; live: boolean }) {
  const site = portal?.site;
  const hasShop = portal?.servers.some((s) => s.shop) ?? false;
  const tabs: { href: string; label: string; active: boolean; show: boolean }[] = [
    { href: "#/", label: "概览", active: route.page === "home", show: true },
    { href: "#/news", label: "公告", active: route.page === "news", show: true },
    { href: "#/shop", label: "商店", active: route.page === "shop", show: hasShop },
  ];
  return (
    <header className="pt-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-wide break-words sm:text-3xl">{siteName}</h1>
          {site?.description && <p className="mt-2 max-w-3xl whitespace-pre-line text-sm text-muted">{site.description}</p>}
        </div>
        <span className={`flex items-center gap-1.5 text-xs ${live ? "text-ok" : "text-muted"}`} title={live ? "内容会自动更新" : "正在重新连接"}>
          <span className={`size-2 rounded-full ${live ? "bg-ok" : "bg-muted"}`} />
          {live ? "实时" : "连接中"}
        </span>
      </div>
      {site && <Contacts site={site} />}
      <nav className="mt-6 flex gap-1 border-b border-line">
        {tabs
          .filter((t) => t.show)
          .map((t) => (
            <a
              key={t.href}
              href={t.href}
              className={`-mb-px border-b-2 px-4 py-2 text-sm transition-colors ${
                t.active ? "border-primary text-primary" : "border-transparent text-muted hover:text-text"
              }`}
            >
              {t.label}
            </a>
          ))}
      </nav>
    </header>
  );
}

function Contacts({ site }: { site: Portal["site"] }) {
  const discord = site.discord ? safeURL(site.discord) : null;
  const links = site.links.map((l) => ({ ...l, url: safeURL(l.url) })).filter((l) => l.url);
  if (!site.qq_group && !discord && links.length === 0) return null;
  return (
    <div className="mt-3 flex flex-wrap gap-2 text-sm">
      {site.qq_group && <Chip>QQ 群 {site.qq_group}</Chip>}
      {discord && (
        <a href={discord} target="_blank" rel="noreferrer noopener">
          <Chip link>Discord</Chip>
        </a>
      )}
      {links.map((l) => (
        <a key={l.url} href={l.url!} target="_blank" rel="noreferrer noopener">
          <Chip link>{l.title || l.url}</Chip>
        </a>
      ))}
    </div>
  );
}

function Home({ portal }: { portal: Portal }) {
  const latest = sortAnnouncements(portal.announcements).slice(0, 3);
  return (
    <div className="space-y-8">
      {latest.length > 0 && (
        <section>
          <SectionTitle title="最新公告" more={portal.announcements.length > latest.length ? { href: "#/news", label: "全部公告" } : undefined} />
          <Announcements items={latest} compact />
        </section>
      )}
      <section>
        <SectionTitle title="服务器" />
        <Servers servers={portal.servers} />
      </section>
    </div>
  );
}


function Placeholder({ loading, message, notReady }: { loading: boolean; message?: string; notReady: boolean }) {
  return (
    <div className="panel hud px-6 py-16 text-center text-muted">
      {loading ? "加载中…" : notReady ? "服主还没有发布玩家站的内容" : message || "加载失败"}
    </div>
  );
}
