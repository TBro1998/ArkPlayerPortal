// 我的：Steam 登录、自己在每台服务器商店里的点数和礼包剩余次数
import { logout, type Remote } from "./api";
import { useT } from "./i18n";
import type { Me, Portal } from "./types";
import { Empty } from "./ui";

export function MeView({ portal, me, reload, error }: { portal: Portal; me: Remote<Me>; reload: () => void; error: string }) {
  const t = useT();
  const m = me.data;
  if (!m) return <Empty>{me.loading ? t("loading") : t("loadFailed")}</Empty>;
  if (!m.user) {
    return (
      <div className="panel hud space-y-4 px-6 py-8 text-center">
        {error && <p className="text-sm text-bad">{t("loginError", { msg: error })}</p>}
        <p className="mx-auto max-w-xl text-sm text-muted">{t("loginHint")}</p>
        {m.steam_login && (
          <a href="/auth/steam/login" className="inline-block rounded-md border border-line bg-panel-2 px-5 py-2 text-sm hover:border-primary hover:text-primary">
            {t("loginSteam")}
          </a>
        )}
      </div>
    );
  }
  const shops = portal.servers.filter((s) => s.shop);
  const data = m.data ?? {};
  return (
    <div className="space-y-4">
      <div className="panel flex items-center gap-3 px-5 py-4">
        {m.user.avatar && <img src={m.user.avatar} alt="" className="size-10 rounded" referrerPolicy="no-referrer" />}
        <div className="min-w-0 flex-1">
          <p className="font-medium break-words">{t("loggedInAs", { name: m.user.name || m.user.steam_id })}</p>
          <p className="text-xs text-muted num">{m.user.steam_id}</p>
        </div>
        <button
          className="rounded border border-line px-3 py-1 text-xs text-muted hover:border-primary hover:text-primary"
          onClick={async () => {
            await logout();
            reload();
          }}
        >
          {t("logout")}
        </button>
      </div>
      {shops.length === 0 ? null : Object.keys(data).length === 0 ? (
        <Empty>{t("noData")}</Empty>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {shops.map((s) => {
            const d = data[s.key];
            return (
              <div key={s.key} className="panel px-5 py-4">
                <h3 className="font-medium break-words">{s.name}</h3>
                {!s.points ? (
                  <p className="mt-2 text-xs text-muted">{t("pointsUnsupported")}</p>
                ) : !d ? (
                  <p className="mt-2 text-xs text-muted">{t("noData")}</p>
                ) : (
                  <>
                    <p className="mt-2 text-2xl text-amber num">{t("points", { n: d.points })}</p>
                    <p className="text-xs text-muted num">{t("totalSpent", { n: d.total_spent })}</p>
                    {d.kits && Object.keys(d.kits).length > 0 && (
                      <div className="mt-3 text-xs">
                        <p className="text-muted">{t("kitsLeft")}</p>
                        <ul className="mt-1 flex flex-wrap gap-1.5">
                          {Object.entries(d.kits).map(([name, n]) => (
                            <li key={name} className="rounded bg-panel-2 px-2 py-0.5 num">
                              {name} × {n}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
