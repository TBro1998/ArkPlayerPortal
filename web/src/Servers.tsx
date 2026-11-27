import { useState } from "react";
import { Chip, Empty } from "./ui";
import { copyText, formatRate, formatTime, modURL } from "./format";
import type { Server } from "./types";

const statusLabel: Record<Server["status"], { text: string; tone: "ok" | "amber" | undefined }> = {
  online: { text: "在线", tone: "ok" },
  starting: { text: "启动中", tone: "amber" },
  offline: { text: "离线", tone: undefined },
};

export function Servers({ servers }: { servers: Server[] }) {
  if (servers.length === 0) {
    return <Empty>没有服务器</Empty>;
  }
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {servers.map((s) => (
        <ServerCard key={s.key} server={s} />
      ))}
    </div>
  );
}

function ServerCard({ server: s }: { server: Server }) {
  const st = statusLabel[s.status] ?? statusLabel.offline;
  const [showMods, setShowMods] = useState(false);
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
          <Chip tone={st.tone}>{st.text}</Chip>
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
        {s.password && <Chip tone="amber">有密码</Chip>}
        {s.cluster_group > 0 && <Chip>集群 {s.cluster_group}</Chip>}
        {s.arkapi && <Chip>{s.game === "asa" ? "AsaApi" : "ArkApi"}</Chip>}
        {s.shop && (
          <a href={`#/shop/${s.key}`}>
            <Chip link>商店 →</Chip>
          </a>
        )}
      </div>

      <dl className="grid grid-cols-4 gap-2 text-center text-xs">
        {(
          [
            ["经验", s.rates.xp],
            ["驯服", s.rates.taming],
            ["采集", s.rates.harvest],
            ["繁殖", s.rates.breeding],
          ] as const
        ).map(([label, v]) => (
          <div key={label} className="rounded-md bg-panel-2 py-1.5">
            <dt className="text-muted">{label}</dt>
            <dd className="mt-0.5 text-sm num">{formatRate(v)}</dd>
          </div>
        ))}
      </dl>

      <Connect server={s} />

      {s.next_restart_at && <p className="text-xs text-muted">下次定时重启：{formatTime(s.next_restart_at)}</p>}

      {s.mods.length > 0 && (
        <div className="text-xs">
          <button className="text-muted hover:text-primary" onClick={() => setShowMods((v) => !v)}>
            Mod（{s.mods.length}）{showMods ? "▴" : "▾"}
          </button>
          {showMods && (
            <ol className="mt-2 flex flex-wrap gap-1.5">
              {s.mods.map((id) => (
                <li key={id}>
                  <a href={modURL(s.game, id)} target="_blank" rel="noreferrer noopener">
                    <Chip link>{id}</Chip>
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
  if (s.address) {
    // ASE 在 Steam 收藏里添加的是查询端口，ASA 在游戏控制台里 open 游戏端口
    const host = s.address.slice(0, s.address.lastIndexOf(":"));
    const shown = s.game === "ase" && s.query_port > 0 ? `${host}:${s.query_port}` : s.address;
    return (
      <div className="rounded-md border border-line bg-bg/60 px-3 py-2 text-sm">
        <div className="flex items-center gap-2">
          <span className="text-muted">地址</span>
          <code className="font-mono num">{shown}</code>
          <CopyButton text={s.game === "asa" ? `open ${s.address}` : shown} label={s.game === "asa" ? "复制命令" : "复制"} />
        </div>
        <p className="mt-1 text-xs text-muted">
          {s.game === "asa" ? (
            <>
              在游戏主菜单按 Tab 打开控制台，输入 <code className="font-mono">open {s.address}</code>
            </>
          ) : (
            <>在 Steam「查看 → 游戏服务器 → 收藏」里添加这个地址，再从游戏里的「加入方舟 → 收藏」进服</>
          )}
        </p>
      </div>
    );
  }
  if (s.room_id > 0) {
    return (
      <div className="rounded-md border border-line bg-bg/60 px-3 py-2 text-sm">
        <span className="text-muted">P2P 联机房 </span>
        <span className="num">{s.room_id}</span>
        <p className="mt-1 text-xs text-muted">用 ARKFGF 联机助手加入这个房间进服</p>
      </div>
    );
  }
  return null;
}

export function CopyButton({ text, label = "复制" }: { text: string; label?: string }) {
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
      {state === "ok" ? "已复制" : state === "fail" ? "复制失败" : label}
    </button>
  );
}
