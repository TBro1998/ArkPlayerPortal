// 显示用的小工具

// blueprintName 从蓝图路径取出类名，去掉常见前缀：
// Blueprint'/Game/.../PrimalItemResource_MetalIngot.PrimalItemResource_MetalIngot' → MetalIngot
// Blueprint'/Game/.../Carno_Character_BP.Carno_Character_BP' → Carno
export function blueprintName(bp: string): string {
  let s = bp.trim().replace(/^[A-Za-z]*'/, "").replace(/'$/, "");
  s = s.split("/").pop() ?? s;
  s = s.split(".").pop() ?? s;
  s = s.replace(/_C$/, "");
  s = s.replace(/^PrimalItem(Resource|Armor|Consumable|Skin|Structure|Ammo|Dye|CustomFood|Costume|Artifact|Trophy)?_/, "");
  s = s.replace(/^Weapon(?=[A-Z])/, "");
  s = s.replace(/_Character_BP.*$/, "").replace(/_BP$/, "");
  return s || bp;
}

export function formatTime(iso: string | null | undefined, lang: "zh" | "en" = "zh"): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(lang === "zh" ? "zh-CN" : "en", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

export function formatRate(n: number): string {
  return `${Number.isInteger(n) ? n : n.toFixed(1)}×`;
}

// safeURL 只接受 http、https 链接
export function safeURL(url: string): string | null {
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:" ? u.href : null;
  } catch {
    return null;
  }
}

export function modURL(game: "ase" | "asa", id: string): string {
  return game === "asa"
    ? `https://www.curseforge.com/projects/${encodeURIComponent(id)}`
    : `https://steamcommunity.com/sharedfiles/filedetails/?id=${encodeURIComponent(id)}`;
}

// copyText 玩家站通常是 http://IP:端口，不是安全上下文，navigator.clipboard 不可用，退回 execCommand
export async function copyText(text: string): Promise<boolean> {
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // 继续用旧办法
    }
  }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  let ok = false;
  try {
    ok = document.execCommand("copy");
  } catch {
    ok = false;
  }
  document.body.removeChild(ta);
  return ok;
}
