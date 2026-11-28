import { useCallback, useEffect, useRef, useState } from "react";
import type { Me, Portal, Shop } from "./types";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    let code = "", message = `请求失败（${res.status}）`;
    try {
      const body = await res.json();
      code = body.code ?? "";
      message = body.error ?? message;
    } catch {
      // 不是 JSON（例如反向代理返回的错误页）
    }
    throw new ApiError(res.status, code, message);
  }
  return res.json() as Promise<T>;
}

export const fetchPortal = () => getJSON<Portal>("/api/v1/portal");
export const fetchShop = (key: string) => getJSON<Shop>(`/api/v1/shops/${encodeURIComponent(key)}`);

type Listener = (kind: "portal" | "shop", key?: string) => void;

// 全站共用一个 SSE 连接
const listeners = new Set<Listener>();
let source: EventSource | null = null;
let connected = false;
const statusListeners = new Set<(c: boolean) => void>();

function setConnected(c: boolean) {
  connected = c;
  statusListeners.forEach((l) => l(c));
}

function ensureSource() {
  if (source) return;
  source = new EventSource("/api/v1/events");
  // 连上（包括断线重连）时可能错过了通知，让所有订阅者重新取一次
  source.onopen = () => {
    setConnected(true);
    listeners.forEach((l) => l("portal"));
    listeners.forEach((l) => l("shop"));
  };
  source.onerror = () => setConnected(false);
  source.addEventListener("portal", () => listeners.forEach((l) => l("portal")));
  source.addEventListener("shop", (e) => {
    let key: string | undefined;
    try {
      key = JSON.parse((e as MessageEvent).data).key;
    } catch {
      return;
    }
    listeners.forEach((l) => l("shop", key));
  });
}

function subscribe(l: Listener) {
  ensureSource();
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function useLiveStatus(): boolean {
  const [c, setC] = useState(connected);
  useEffect(() => {
    statusListeners.add(setC);
    return () => {
      statusListeners.delete(setC);
    };
  }, []);
  return c;
}

export interface Remote<T> {
  data: T | null;
  error: ApiError | null;
  loading: boolean;
}

// useLive 先取一次，之后收到对应的通知时重新取
function useLive<T>(
  load: (() => Promise<T>) | null,
  match: (kind: "portal" | "shop", key?: string) => boolean,
): Remote<T> {
  const [state, setState] = useState<Remote<T>>({ data: null, error: null, loading: load !== null });
  const loadRef = useRef(load);
  const matchRef = useRef(match);
  const seq = useRef(0);

  useEffect(() => {
    loadRef.current = load;
    matchRef.current = match;
  });

  const refresh = useCallback(() => {
    const fn = loadRef.current;
    if (!fn) return;
    const n = ++seq.current;
    fn().then(
      (data) => n === seq.current && setState({ data, error: null, loading: false }),
      (err) =>
        n === seq.current &&
        setState((s) => ({
          data: err instanceof ApiError && err.status === 404 ? null : s.data,
          error: err instanceof ApiError ? err : new ApiError(0, "network", "连不上玩家站"),
          loading: false,
        })),
    );
  }, []);

  useEffect(() => {
    if (!load) return;
    refresh();
    return subscribe((kind, key) => {
      if (matchRef.current(kind, key)) refresh();
    });
    // load 变化（换了服务器）时重新订阅
  }, [load, refresh]);

  return state;
}

export function usePortal(): Remote<Portal> {
  return useLive(fetchPortal, (kind) => kind === "portal");
}

const shopLoaders = new Map<string, () => Promise<Shop>>();
function shopLoader(key: string) {
  let l = shopLoaders.get(key);
  if (!l) {
    l = () => fetchShop(key);
    shopLoaders.set(key, l);
  }
  return l;
}

export function useShop(key: string | null): Remote<Shop> {
  return useLive(key ? shopLoader(key) : null, (kind, k) => kind === "shop" && (k === undefined || k === key));
}

export const fetchMe = () => getJSON<Me>("/api/v1/me");

export async function logout() {
  await fetch("/api/v1/logout", { method: "POST" });
}

// useMe 当前登录的玩家和他的数据，每 30 秒刷新一次（点数文件没有推送通知）
export function useMe(): Remote<Me> & { reload: () => void } {
  const [state, setState] = useState<Remote<Me>>({ data: null, error: null, loading: true });
  const seq = useRef(0);
  const reload = useCallback(() => {
    const n = ++seq.current;
    fetchMe().then(
      (data) => n === seq.current && setState({ data, error: null, loading: false }),
      (err) => n === seq.current && setState((s) => ({ ...s, error: err instanceof ApiError ? err : null, loading: false })),
    );
  }, []);
  useEffect(() => {
    reload();
    const t = setInterval(() => !document.hidden && reload(), 30000);
    return () => clearInterval(t);
  }, [reload]);
  return { ...state, reload };
}
