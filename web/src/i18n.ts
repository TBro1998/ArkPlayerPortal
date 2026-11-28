// 界面语言：中文和英文。默认按浏览器语言，玩家切换后记在 localStorage。
// 服主写的内容（公告、页面、商品描述）不翻译；快照里的 {"zh","en"} 文本按界面语言选
import { useSyncExternalStore } from "react";

export type Lang = "zh" | "en";

const STORAGE_KEY = "portal-lang";

function initial(): Lang {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "zh" || saved === "en") return saved;
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

let current: Lang = initial();
const listeners = new Set<() => void>();

export function setLang(l: Lang) {
  current = l;
  localStorage.setItem(STORAGE_KEY, l);
  document.documentElement.lang = l === "zh" ? "zh-CN" : "en";
  listeners.forEach((f) => f());
}
document.documentElement.lang = current === "zh" ? "zh-CN" : "en";

export function useLang(): Lang {
  return useSyncExternalStore(
    (f) => {
      listeners.add(f);
      return () => listeners.delete(f);
    },
    () => current,
  );
}

const zh = {
  portal: "玩家站",
  live: "实时",
  liveHint: "内容会自动更新",
  reconnecting: "连接中",
  reconnectingHint: "正在重新连接",
  tabHome: "概览",
  tabNews: "公告",
  tabShop: "商店",
  tabMe: "我的",
  latestNews: "最新公告",
  allNews: "全部公告",
  servers: "服务器",
  pages: "服务器指南",
  loading: "加载中…",
  notReady: "服主还没有发布玩家站的内容",
  loadFailed: "加载失败",
  unreachable: "连不上玩家站",
  poweredBy: "由 {name} 提供",
  qqGroup: "QQ 群 {qq}",
  noNews: "还没有公告",
  pinned: "置顶",
  updatedAt: "（{time} 更新）",
  noServers: "没有服务器",
  online: "在线",
  starting: "启动中",
  offline: "离线",
  password: "有密码",
  cluster: "集群 {n}",
  shopLink: "商店 →",
  settingsLink: "服务器设置 →",
  rateXP: "经验",
  rateTaming: "驯服",
  rateHarvest: "采集",
  rateBreeding: "繁殖",
  address: "地址",
  copy: "复制",
  copyCommand: "复制命令",
  copied: "已复制",
  copyFailed: "复制失败",
  connectAsa: "在游戏主菜单按 Tab 打开控制台，输入 {cmd}",
  connectAse: "在 Steam「查看 → 游戏服务器 → 收藏」里添加这个地址，再从游戏里的「加入方舟 → 收藏」进服",
  linkRoom: "P2P 联机房 {id}",
  linkRoomHint: "用 ARKFGF 联机助手加入这个房间进服",
  nextRestart: "下次定时重启：{time}",
  mods: "Mod（{n}）",
  pageNotFound: "页面不存在",
  back: "← 返回",
  on: "开",
  off: "关",
  noShops: "没有服务器开放商店",
  noShop: "这台服务器没有商店",
  shopIntro: "在游戏里按 Enter 打开聊天框输入命令购买，点数不够或不满足条件时游戏里会提示。查看自己的点数：",
  shopUpdated: "更新于 {time}",
  myPoints: "我的点数：{points}",
  myPointsLogin: "登录后可以看自己的点数",
  items: "商品",
  kits: "礼包",
  sell: "回收",
  allTypes: "全部类型",
  search: "搜索名称、编号",
  noMatchItems: "没有符合条件的商品",
  noItems: "没有商品",
  noMatchKits: "没有符合条件的礼包",
  noMatchSell: "没有符合条件的回收项",
  points: "{n} 点",
  free: "免费",
  pointsGain: "+{n} 点",
  buy: "购买",
  claim: "领取",
  sellAction: "出售",
  amount: "数量",
  itemId: "编号 {id}",
  kitName: "名称 {name}",
  kitFree: "每人免费 {n} 次",
  kitLeft: "你还能领 {n} 次",
  levelRange: "等级 {min}–{max}",
  levelMin: "等级 ≥ {min}",
  levelMax: "等级 ≤ {max}",
  onlyFor: "仅限 {groups}",
  onlyFromSpawn: "只能在刚出生时领取",
  bp: "（图纸）",
  quality: "，品质 {q}",
  level: "{name} {level} 级",
  neutered: "已绝育",
  withSaddle: "带 {name}",
  crate: "补给箱 {name}",
  exp: "{n} 经验",
  expDino: "（给骑乘的恐龙）",
  unlock: "解锁 {name}",
  give: "交出 {name} × {n}",
  male: "公",
  female: "母",
  randomGender: "随机性别",
  typeItem: "物品",
  typeDino: "恐龙",
  typeBeacon: "补给箱",
  typeExperience: "经验",
  typeUnlockengram: "解锁图纸",
  typeCommand: "特殊",
  kitBadge: "礼包",
  sellBadge: "回收",
  meTitle: "我的",
  loginSteam: "用 Steam 登录",
  loginHint: "用 Steam 登录后可以看自己在各台服务器商店里的点数和礼包剩余次数。玩家站只拿到你的 SteamID、昵称和头像，拿不到密码。",
  logout: "退出登录",
  loggedInAs: "已登录：{name}",
  noData: "还没有你的数据：在开了商店的服务器上进过一次游戏后才会有",
  totalSpent: "已花 {n} 点",
  kitsLeft: "礼包剩余",
  loginError: "登录失败：{msg}",
  pointsUnsupported: "这台服务器的商店不支持查看点数",
};

type Dict = typeof zh;

const en: Dict = {
  portal: "Player portal",
  live: "Live",
  liveHint: "Content updates automatically",
  reconnecting: "Connecting",
  reconnectingHint: "Reconnecting",
  tabHome: "Overview",
  tabNews: "News",
  tabShop: "Store",
  tabMe: "Me",
  latestNews: "Latest news",
  allNews: "All news",
  servers: "Servers",
  pages: "Guides",
  loading: "Loading…",
  notReady: "The owner has not published the portal yet",
  loadFailed: "Failed to load",
  unreachable: "Cannot reach the portal",
  poweredBy: "Powered by {name}",
  qqGroup: "QQ group {qq}",
  noNews: "No announcements yet",
  pinned: "Pinned",
  updatedAt: " (updated {time})",
  noServers: "No servers",
  online: "Online",
  starting: "Starting",
  offline: "Offline",
  password: "Password",
  cluster: "Cluster {n}",
  shopLink: "Store →",
  settingsLink: "Server settings →",
  rateXP: "XP",
  rateTaming: "Taming",
  rateHarvest: "Harvest",
  rateBreeding: "Breeding",
  address: "Address",
  copy: "Copy",
  copyCommand: "Copy command",
  copied: "Copied",
  copyFailed: "Copy failed",
  connectAsa: "In the main menu press Tab to open the console and enter {cmd}",
  connectAse: "Add this address under Steam → View → Game Servers → Favorites, then join from Join ARK → Favorites in game",
  linkRoom: "P2P room {id}",
  linkRoomHint: "Join this room with ARKFGF Link to play",
  nextRestart: "Next scheduled restart: {time}",
  mods: "Mods ({n})",
  pageNotFound: "Page not found",
  back: "← Back",
  on: "On",
  off: "Off",
  noShops: "No server has a store",
  noShop: "This server has no store",
  shopIntro: "Press Enter in game to open chat and type the command to buy. The game tells you if you lack points or don't meet a requirement. Check your points:",
  shopUpdated: "Updated {time}",
  myPoints: "My points: {points}",
  myPointsLogin: "Sign in to see your points",
  items: "Items",
  kits: "Kits",
  sell: "Sell",
  allTypes: "All types",
  search: "Search name or ID",
  noMatchItems: "No matching items",
  noItems: "No items",
  noMatchKits: "No matching kits",
  noMatchSell: "No matching entries",
  points: "{n} pts",
  free: "Free",
  pointsGain: "+{n} pts",
  buy: "Buy",
  claim: "Claim",
  sellAction: "Sell",
  amount: "Amount",
  itemId: "ID {id}",
  kitName: "Name {name}",
  kitFree: "{n} free per player",
  kitLeft: "{n} left for you",
  levelRange: "Level {min}–{max}",
  levelMin: "Level ≥ {min}",
  levelMax: "Level ≤ {max}",
  onlyFor: "Only for {groups}",
  onlyFromSpawn: "Only right after spawning",
  bp: " (blueprint)",
  quality: ", quality {q}",
  level: "{name} level {level}",
  neutered: "neutered",
  withSaddle: "with {name}",
  crate: "Supply crate {name}",
  exp: "{n} XP",
  expDino: " (to ridden dino)",
  unlock: "Unlock {name}",
  give: "Give {name} × {n}",
  male: "male",
  female: "female",
  randomGender: "random gender",
  typeItem: "Item",
  typeDino: "Dino",
  typeBeacon: "Crate",
  typeExperience: "XP",
  typeUnlockengram: "Engram",
  typeCommand: "Special",
  kitBadge: "Kit",
  sellBadge: "Sell",
  meTitle: "Me",
  loginSteam: "Sign in with Steam",
  loginHint: "Sign in with Steam to see your points and remaining kits on each server's store. The portal only gets your SteamID, name and avatar, never your password.",
  logout: "Sign out",
  loggedInAs: "Signed in as {name}",
  noData: "No data for you yet: it appears after you have joined a server with a store",
  totalSpent: "{n} pts spent",
  kitsLeft: "Kits left",
  loginError: "Sign-in failed: {msg}",
  pointsUnsupported: "This store does not support viewing points",
};

const dicts: Record<Lang, Dict> = { zh, en };

export type TKey = keyof Dict;

export function useT() {
  const lang = useLang();
  return (key: TKey, vars?: Record<string, string | number>) => {
    let s: string = dicts[lang][key];
    if (vars) for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, String(v));
    return s;
  };
}

// pick 快照里的文本：字符串原样，{"zh","en"} 按界面语言选
export function pick(text: unknown, lang: Lang): string {
  if (typeof text === "string") return text;
  if (text && typeof text === "object") {
    const t = text as Record<string, unknown>;
    const v = t[lang] ?? t.zh ?? t.en;
    return typeof v === "string" ? v : "";
  }
  return "";
}

export function locale(lang: Lang) {
  return lang === "zh" ? "zh-CN" : "en";
}
