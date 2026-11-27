import { Empty, sortAnnouncements } from "./ui";
import { formatTime } from "./format";
import type { Announcement } from "./types";

export function Announcements({ items, compact }: { items: Announcement[]; compact?: boolean }) {
  if (items.length === 0) {
    return <Empty>还没有公告</Empty>;
  }
  return (
    <div className="space-y-3">
      {sortAnnouncements(items).map((a) => (
        <article key={a.id} className={`panel px-5 py-4 ${a.pinned ? "border-primary/40" : ""}`}>
          <header className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            {a.pinned && <span className="rounded bg-primary/15 px-1.5 py-0.5 text-[11px] text-primary">置顶</span>}
            <h3 className="font-medium break-words">{a.title}</h3>
            <time className="ml-auto text-xs text-muted num" dateTime={a.published_at}>
              {formatTime(a.published_at)}
              {a.updated_at && a.updated_at !== a.published_at && <span>（{formatTime(a.updated_at)} 更新）</span>}
            </time>
          </header>
          {a.body && (
            <p className={`mt-2 whitespace-pre-line break-words text-sm leading-relaxed text-text/85 ${compact ? "line-clamp-4" : ""}`}>{a.body}</p>
          )}
        </article>
      ))}
    </div>
  );
}
