// 快照的结构，和 docs/snapshot.md 一致

export type Game = "ase" | "asa";

export interface Portal {
  updated_at: string;
  site: Site;
  announcements: Announcement[];
  servers: Server[];
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
  mods: string[];
  rates: { xp: number; taming: number; harvest: number; breeding: number };
  version: string;
  address: string;
  query_port: number;
  room_id: number;
  next_restart_at: string | null;
  shop: boolean;
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
