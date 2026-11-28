import { useEffect, useState } from "react";
import { useLiveStatus, useMe, usePortal } from "./api";
import { Announcements } from "./Announcements";
import { MeView } from "./Me";
import { PageLinks, PageView } from "./Page";
import { Servers } from "./Servers";
import { ShopView } from "./Shop";
import { safeURL } from "./format";
import { setLang, useLang, useT } from "./i18n";
import type { Portal } from "./types";
import { Chip, SectionTitle, sortAnnouncements } from "./ui";

// 路由放在 hash 里（#/、#/news、#/shop/3、#/page/<id>、#/me），刷新和分享链接都不需要服务端配合
type Route =
  | { page: "home" }
  | { page: "news" }
  | { page: "shop"; key: string | null }
  | { page: "page"; id: string }
  | { page: "me"; error: string };

function parseRoute(hash: string): Route {
  const [path, query = ""] = hash.replace(/^#\/?/, "").split("?");
  const parts = path.split("/");
  if (parts[0] === "news") return { page: "news" };
  if (parts[0] === "shop") return { page: "shop", key: /^\d+$/.test(parts[1] ?? "") ? parts[1] : null };
  if (parts[0] === "page" && parts[1]) return { page: "page", id: decodeURIComponent(parts[1]) };
  if (parts[0] === "me") return { page: "me", error: new URLSearchParams(query).get("error") ?? "" };
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
  const t = useT();
  const portal = usePortal();
  const me = useMe();
  const route = useRoute();
  const live = useLiveStatus();
  const p = portal.data;

  const siteName = p?.site.name || p?.servers[0]?.name || t("portal");
  useEffect(() => {
    document.title = siteName;
  }, [siteName]);

  return (
    <div className="mx-auto flex min-h-screen max-w-6xl flex-col px-4 pb-10 sm:px-6">
      <Header portal={p} siteName={siteName} route={route} live={live} loggedIn={!!me.data?.user} />
      <main className="mt-6 flex-1">
        {!p ? (
          <Placeholder loading={portal.loading} message={portal.error?.message} notReady={portal.error?.code === "not_ready"} />
        ) : route.page === "news" ? (
          <Announcements items={p.announcements} />
        ) : route.page === "shop" ? (
          <ShopView servers={p.servers} selected={route.key} me={me.data} />
        ) : route.page === "page" ? (
          <PageView portal={p} id={route.id} />
        ) : route.page === "me" ? (
          <MeView portal={p} me={me} reload={me.reload} error={route.error} />
        ) : (
          <Home portal={p} />
        )}
      </main>
      <footer className="mt-10 text-center text-xs text-muted">
        <a className="hover:text-primary" href="https://github.com/TBro1998/ArkServerCommander" target="_blank" rel="noreferrer noopener">
          {t("poweredBy", { name: "ArkServerCommander" })}
        </a>
      </footer>
    </div>
  );
}

function Header({ portal, siteName, route, live, loggedIn }: { portal: Portal | null; siteName: string; route: Route; live: boolean; loggedIn: boolean }) {
  const t = useT();
  const lang = useLang();
  const site = portal?.site;
  const hasShop = portal?.servers.some((s) => s.shop) ?? false;
  const tabs: { href: string; label: string; active: boolean; show: boolean }[] = [
    { href: "#/", label: t("tabHome"), active: route.page === "home" || route.page === "page", show: true },
    { href: "#/news", label: t("tabNews"), active: route.page === "news", show: true },
    { href: "#/shop", label: t("tabShop"), active: route.page === "shop", show: hasShop },
    { href: "#/me", label: t("tabMe") + (loggedIn ? " ●" : ""), active: route.page === "me", show: true },
  ];
  return (
    <header className="pt-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-wide break-words sm:text-3xl">{siteName}</h1>
          {site?.description && <p className="mt-2 max-w-3xl whitespace-pre-line text-sm text-muted">{site.description}</p>}
        </div>
        <div className="flex items-center gap-3">
          <button
            className="rounded border border-line px-2 py-0.5 text-xs text-muted hover:border-primary hover:text-primary"
            onClick={() => setLang(lang === "zh" ? "en" : "zh")}
            aria-label="Language"
          >
            {lang === "zh" ? "English" : "中文"}
          </button>
          <span className={`flex items-center gap-1.5 text-xs ${live ? "text-ok" : "text-muted"}`} title={live ? t("liveHint") : t("reconnectingHint")}>
            <span className={`size-2 rounded-full ${live ? "bg-ok" : "bg-muted"}`} />
            {live ? t("live") : t("reconnecting")}
          </span>
        </div>
      </div>
      {site && <Contacts site={site} />}
      <nav className="mt-6 flex gap-1 border-b border-line">
        {tabs
          .filter((x) => x.show)
          .map((x) => (
            <a
              key={x.href}
              href={x.href}
              className={`-mb-px border-b-2 px-4 py-2 text-sm transition-colors ${
                x.active ? "border-primary text-primary" : "border-transparent text-muted hover:text-text"
              }`}
            >
              {x.label}
            </a>
          ))}
      </nav>
    </header>
  );
}

function Contacts({ site }: { site: Portal["site"] }) {
  const t = useT();
  const discord = site.discord ? safeURL(site.discord) : null;
  const links = site.links.map((l) => ({ ...l, url: safeURL(l.url) })).filter((l) => l.url);
  if (!site.qq_group && !discord && links.length === 0) return null;
  return (
    <div className="mt-3 flex flex-wrap gap-2 text-sm">
      {site.qq_group && <Chip>{t("qqGroup", { qq: site.qq_group })}</Chip>}
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
  const t = useT();
  const latest = sortAnnouncements(portal.announcements).slice(0, 3);
  const hasPages = (portal.pages ?? []).some((p) => p.kind === "custom");
  return (
    <div className="space-y-8">
      {latest.length > 0 && (
        <section>
          <SectionTitle title={t("latestNews")} more={portal.announcements.length > latest.length ? { href: "#/news", label: t("allNews") } : undefined} />
          <Announcements items={latest} compact />
        </section>
      )}
      {hasPages && (
        <section>
          <SectionTitle title={t("pages")} />
          <PageLinks portal={portal} />
        </section>
      )}
      <section>
        <SectionTitle title={t("servers")} />
        <Servers servers={portal.servers} />
      </section>
    </div>
  );
}

function Placeholder({ loading, message, notReady }: { loading: boolean; message?: string; notReady: boolean }) {
  const t = useT();
  return (
    <div className="panel hud px-6 py-16 text-center text-muted">{loading ? t("loading") : notReady ? t("notReady") : message || t("loadFailed")}</div>
  );
}
