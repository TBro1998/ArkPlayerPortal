// 快照的结构，和 docs/snapshot.md 一致

export type Game = "ase" | "asa";

export interface Portal {
  updated_at: string;
  site: Site;
  announcements: Announcement[];
  servers: Server[];
  pages: Page[];
}

// 通用页面：文本是字符串或 {"zh","en"}，块只有这几种（客户端不认识的块跳过）
export type Text = string | { zh?: string; en?: string };
export type Block =
  | { type: "heading"; text: Text }
  | { type: "paragraph"; text: Text }
  | { type: "list"; items: Text[] }
  | { type: "kv"; items: { label: Text; value: string | number | boolean }[] }
  | { type: string; text?: Text; items?: unknown[] };

export interface Page {
  id: string;
  kind: "custom" | "settings" | string;
  title: Text;
  server_key?: string;
  blocks: Block[];
}

export interface PlayerData {
  points: number;
  total_spent: number;
  kits?: Record<string, number>;
}

export interface Me {
  user: { steam_id: string; name: string; avatar: string } | null;
  steam_login: boolean;
  link_login: boolean;
  data?: Record<string, PlayerData>;
}

export interface Site {
  name: string;
  description: string;
  qq_group: string;
  discord: string;
  links: { title: string; url: string }[];
}

export interface Announcement {
  id: string;
  title: string;
  body: string;
  pinned: boolean;
  published_at: string;
  updated_at: string;
}

export interface Server {
  key: string;
  game: Game;
  name: string;
  map: string;
  status: "online" | "starting" | "offline";
  players: number;
  max_players: number;
  password: boolean;
  pvp: boolean;
  arkapi: boolean;
  cluster_group: number;
  mods: { id: string; title: string }[];
  rates: { xp: number; taming: number; harvest: number; breeding: number };
  version: string;
  address: string;
  query_port: number;
  room_id: number;
  next_restart_at: string | null;
  shop: boolean;
  settings_page: string;
  points: boolean;
}

export interface Shop {
  updated_at: string;
  commands: { buy: string; kit: string; buy_kit: string; points: string; shop: string; sell: string };
  items: ShopItem[];
  kits: Kit[];
  sell: SellItem[];
}

export interface ItemEntry {
  blueprint: string;
  amount: number;
  quality: number;
  blueprint_only: boolean;
}

export interface DinoEntry {
  blueprint: string;
  level: number;
  neutered: boolean;
  gender: "" | "male" | "female" | "random";
  saddle_blueprint: string;
}

export interface Requirement {
  price: number | null;
  min_level: number;
  max_level: number;
  permissions: string[];
}

export interface ShopItem extends Requirement {
  id: string;
  type: string;
  description: string;
  price: number;
  items?: ItemEntry[];
  dinos?: DinoEntry[];
  beacon?: string;
  experience?: { amount: number; to_dino: boolean };
  engrams?: string[];
  commands?: string[];
}

export interface Kit extends Requirement {
  name: string;
  description: string;
  default_amount: number;
  only_from_spawn: boolean;
  items: ItemEntry[];
  dinos: DinoEntry[];
  commands: string[];
}

export interface SellItem {
  id: string;
  description: string;
  price: number;
  amount: number;
  blueprint: string;
}
