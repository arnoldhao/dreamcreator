import type { MouseEvent, PropsWithChildren } from "react";
import { ChevronDown, ExternalLink, Link2 } from "lucide-react";
import { Browser } from "@wailsio/runtime";

import { cn } from "@/lib/utils";

const SOURCE_AVATAR_TONES = [
  "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  "bg-violet-500/15 text-violet-700 dark:text-violet-300",
  "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  "bg-indigo-500/15 text-indigo-700 dark:text-indigo-300",
];

const hashText = (value: string) => {
  let hash = 0;
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash << 5) - hash + value.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
};

const resolveSourceAvatarTone = (host: string) =>
  SOURCE_AVATAR_TONES[hashText(host.toLowerCase()) % SOURCE_AVATAR_TONES.length];

export const resolveSourceHost = (url: string) => {
  try {
    const host = new URL(url).hostname.replace(/^www\./, "");
    return host || url;
  } catch {
    return url;
  }
};

function SourceAvatar({ host, className, title }: { host: string; className?: string; title?: string }) {
  const normalizedHost = host.trim();
  const letter = normalizedHost.charAt(0).toUpperCase() || "\u00b7";

  return (
    <span
      className={cn(
        "inline-flex shrink-0 select-none items-center justify-center rounded-full border border-border/50 font-medium",
        resolveSourceAvatarTone(normalizedHost || "source"),
        className
      )}
      title={title}
      aria-hidden
    >
      {letter}
    </span>
  );
}

type SourcePreview = {
  key: string;
  title: string;
  host: string;
};

type SourcesOutlineProps = PropsWithChildren<{
  title: string;
  count: number;
  previews: SourcePreview[];
  open: boolean;
  onToggle: (open: boolean) => void;
  className?: string;
}>;

export function SourcesOutline({
  title,
  count,
  previews,
  open,
  onToggle,
  className,
  children,
}: SourcesOutlineProps) {
  const previewItems = previews.slice(0, 4);
  const overflowCount = Math.max(0, previews.length - previewItems.length);

  return (
    <details
      open={open}
      onToggle={(event) => onToggle(event.currentTarget.open)}
      className={cn("group min-w-0 overflow-hidden rounded-lg border border-border/60 bg-background/60 p-2.5", className)}
    >
      <summary className="flex min-w-0 cursor-pointer list-none items-center gap-1.5 text-xs text-muted-foreground [&::-webkit-details-marker]:hidden">
        <ChevronDown className="h-3.5 w-3.5 shrink-0 transition-transform duration-150 group-open:rotate-180" />
        <Link2 className="h-3.5 w-3.5 shrink-0" />
        <span className="shrink-0">{title} ({count})</span>
        <span className="ml-auto flex items-center gap-0.5">
          {previewItems.map((item, index) => (
            <SourceAvatar
              key={`${item.key}:${index}`}
              host={item.host}
              className="h-4 w-4 text-[9px]"
              title={item.title || item.host}
            />
          ))}
          {overflowCount > 0 ? (
            <span className="ml-1 text-[10px] text-muted-foreground">+{overflowCount}</span>
          ) : null}
        </span>
      </summary>
      <div className="mt-2 min-w-0 flex flex-wrap gap-1.5">{children}</div>
    </details>
  );
}

export function SourceOutline({ title, url }: { title?: string; url: string }) {
  const fallbackHost = resolveSourceHost(url);
  const label = title?.trim() || fallbackHost;
  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    if (event.defaultPrevented) {
      return;
    }
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return;
    }

    event.preventDefault();
    void Browser.OpenURL(url).catch(() => {
      if (typeof window !== "undefined") {
        window.open(url, "_blank", "noopener,noreferrer");
      }
    });
  };

  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      onClick={handleClick}
      className="group/source inline-flex min-w-0 max-w-full items-center gap-1.5 rounded-full border border-border/50 bg-background px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:border-border hover:text-foreground"
      title={url}
    >
      <SourceAvatar host={fallbackHost} className="h-3.5 w-3.5 text-[8px]" title={fallbackHost} />
      <span className="truncate">{label}</span>
      <ExternalLink className="h-3 w-3 shrink-0 opacity-70 transition-opacity group-hover/source:opacity-100" />
    </a>
  );
}
