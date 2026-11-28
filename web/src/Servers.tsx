import { useState } from "react";
import { Chip, Empty } from "./ui";
import { copyText, formatRate, formatTime, modURL } from "./format";
import { useLang, useT } from "./i18n";
import type { Server } from "./types";

const statusTone: Record<Server["status"], "ok" | "amber" | undefined> = { online: "ok", starting: "amber", offline: undefined };

export function Servers({ servers }: { servers: Server[] }) {
  const t = useT();
  if (servers.length === 0) return <Empty>{t("noServers")}</Empty>;
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {servers.map((s) => (
        <ServerCard key={s.key} server={s} />
      ))}
    </div>
  );
}

function ServerCard({ server: s }: { server: Server }) {
  const t = useT();
  const lang = useLang();
  const [showMods, setShowMods] = useState(false);
  const status = s.status in statusTone ? s.status : "offline";
  return (
    <div className="panel hud flex flex-col gap-3 px-5 py-4">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="font-medium break-words">{s.name}</h3>
          <p className="mt-0.5 text-sm text-muted">
            {s.map}
            {s.version && <span> · v{s.version}</span>}
          </p>
        </div>
        <div className="text-right">
          <Chip tone={statusTone[status]}>{t(status)}</Chip>
          {s.status === "online" && (
            <p className="mt-1 text-sm num">
              <span className="text-primary">{s.players}</span>
              <span className="text-muted"> / {s.max_players}</span>
            </p>
          )}
        </div>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Chip>{s.game === "asa" ? "ASA" : "ASE"}</Chip>
        <Chip>{s.pvp ? "PvP" : "PvE"}</Chip>
        {s.password && <Chip tone="amber">{t("password")}</Chip>}
        {s.cluster_group > 0 && <Chip>{t("cluster", { n: s.cluster_group })}</Chip>}
        {s.arkapi && <Chip>{s.game === "asa" ? "AsaApi" : "ArkApi"}</Chip>}
        {s.settings_page && (
          <a href={`#/page/${encodeURIComponent(s.settings_page)}`}>
            <Chip link>{t("settingsLink")}</Chip>
          </a>
        )}
        {s.shop && (
          <a href={`#/shop/${s.key}`}>
            <Chip link>{t("shopLink")}</Chip>
          </a>
        )}
      </div>

      <dl className="grid grid-cols-4 gap-2 text-center text-xs">
        {(
          [
            [t("rateXP"), s.rates.xp],
            [t("rateTaming"), s.rates.taming],
            [t("rateHarvest"), s.rates.harvest],
            [t("rateBreeding"), s.rates.breeding],
          ] as const
        ).map(([label, v]) => (
          <div key={label} className="rounded-md bg-panel-2 py-1.5">
            <dt className="text-muted">{label}</dt>
            <dd className="mt-0.5 text-sm num">{formatRate(v)}</dd>
          </div>
        ))}
      </dl>

      <Connect server={s} />

      {s.next_restart_at && <p className="text-xs text-muted">{t("nextRestart", { time: formatTime(s.next_restart_at, lang) })}</p>}

      {s.mods.length > 0 && (
        <div className="text-xs">
          <button className="text-muted hover:text-primary" onClick={() => setShowMods((v) => !v)}>
            {t("mods", { n: s.mods.length })} {showMods ? "▴" : "▾"}
          </button>
          {showMods && (
            <ol className="mt-2 flex flex-wrap gap-1.5">
              {s.mods.map((m) => (
                <li key={m.id}>
                  <a href={modURL(s.game, m.id)} target="_blank" rel="noreferrer noopener" title={m.id}>
                    <Chip link>{m.title || m.id}</Chip>
                  </a>
                </li>
              ))}
            </ol>
          )}
        </div>
      )}
    </div>
  );
}

function Connect({ server: s }: { server: Server }) {
  const t = useT();
  if (s.address) {
    // ASE 在 Steam 收藏里添加的是查询端口，ASA 在游戏控制台里 open 游戏端口
    const host = s.address.slice(0, s.address.lastIndexOf(":"));
    const shown = s.game === "ase" && s.query_port > 0 ? `${host}:${s.query_port}` : s.address;
    const cmd = `open ${s.address}`;
    return (
      <div className="rounded-md border border-line bg-bg/60 px-3 py-2 text-sm">
        <div className="flex items-center gap-2">
          <span className="text-muted">{t("address")}</span>
          <code className="font-mono num">{shown}</code>
          <CopyButton text={s.game === "asa" ? cmd : shown} label={s.game === "asa" ? t("copyCommand") : t("copy")} />
        </div>
        <p className="mt-1 text-xs text-muted">{s.game === "asa" ? t("connectAsa", { cmd }) : t("connectAse")}</p>
      </div>
    );
  }
  if (s.room_id > 0) {
    return (
      <div className="rounded-md border border-line bg-bg/60 px-3 py-2 text-sm">
        <span className="text-muted num">{t("linkRoom", { id: s.room_id })}</span>
        <p className="mt-1 text-xs text-muted">{t("linkRoomHint")}</p>
      </div>
    );
  }
  return null;
}

export function CopyButton({ text, label }: { text: string; label?: string }) {
  const t = useT();
  const [state, setState] = useState<"" | "ok" | "fail">("");
  return (
    <button
      className={`rounded border px-2 py-0.5 text-xs transition-colors ${
        state === "ok" ? "border-ok/40 text-ok" : state === "fail" ? "border-bad/40 text-bad" : "border-line text-muted hover:border-primary hover:text-primary"
      }`}
      onClick={async () => {
        setState((await copyText(text)) ? "ok" : "fail");
        setTimeout(() => setState(""), 1500);
      }}
    >
      {state === "ok" ? t("copied") : state === "fail" ? t("copyFailed") : (label ?? t("copy"))}
    </button>
  );
}
