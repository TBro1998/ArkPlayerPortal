// 各页面共用的小组件
import type { ReactNode } from "react";

export function Chip({ children, link, tone }: { children: ReactNode; link?: boolean; tone?: "ok" | "amber" | "bad" }) {
  const color =
    tone === "ok"
      ? "text-ok border-ok/30"
      : tone === "amber"
        ? "text-amber border-amber/30"
        : tone === "bad"
          ? "text-bad border-bad/30"
          : "text-text border-line";
  return (
    <span className={`inline-flex items-center rounded-md border bg-panel-2 px-2 py-0.5 text-xs ${color} ${link ? "hover:border-primary hover:text-primary" : ""}`}>
      {children}
    </span>
  );
}

export function SectionTitle({ title, more }: { title: string; more?: { href: string; label: string } }) {
  return (
    <div className="mb-3 flex items-baseline justify-between">
      <h2 className="text-sm font-semibold uppercase tracking-widest text-primary">{title}</h2>
      {more && (
        <a href={more.href} className="text-xs text-muted hover:text-primary">
          {more.label} →
        </a>
      )}
    </div>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="panel px-6 py-10 text-center text-sm text-muted">{children}</div>;
}

// 置顶的在前，其余从新到旧
export function sortAnnouncements<T extends { pinned: boolean; published_at: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => Number(b.pinned) - Number(a.pinned) || b.published_at.localeCompare(a.published_at));
}
