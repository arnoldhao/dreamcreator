import { RefreshCw } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";

import { cn } from "@/lib/utils";
import type { MemorySummary } from "@/shared/query/memory";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";

interface SummaryTabProps {
  t: (key: string) => string;
  language: string;
  summary?: MemorySummary;
  isLoading: boolean;
  onRefresh: () => void;
}

interface DashboardItemModel {
  key: string;
  label: string;
  value: string;
  title?: string;
  dense?: boolean;
  clampValueToTwoLines?: boolean;
}

interface SummarySectionCardProps {
  title: string;
  items: DashboardItemModel[];
  isLoading: boolean;
  onRefresh: () => void;
  refreshLabel: string;
}

function formatSummaryTime(value: string | undefined, language: string, justNowLabel: string): string {
  const normalized = value?.trim() ?? "";
  if (!normalized) {
    return "";
  }
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) {
    return normalized;
  }
  const diffSeconds = Math.round((parsed.getTime() - Date.now()) / 1000);
  const absSeconds = Math.abs(diffSeconds);
  if (absSeconds < 5) {
    return justNowLabel;
  }
  const rtf = (() => {
    try {
      return new Intl.RelativeTimeFormat(language, { numeric: "always", style: "short" });
    } catch {
      return new Intl.RelativeTimeFormat(language, { numeric: "always" });
    }
  })();
  if (absSeconds < 60) {
    return rtf.format(diffSeconds, "second");
  }
  const diffMinutes = Math.round(diffSeconds / 60);
  if (Math.abs(diffMinutes) < 60) {
    return rtf.format(diffMinutes, "minute");
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (Math.abs(diffHours) < 24) {
    return rtf.format(diffHours, "hour");
  }
  const diffDays = Math.round(diffHours / 24);
  if (Math.abs(diffDays) < 30) {
    return rtf.format(diffDays, "day");
  }
  const diffMonths = Math.round(diffDays / 30);
  if (Math.abs(diffMonths) < 12) {
    return rtf.format(diffMonths, "month");
  }
  const diffYears = Math.round(diffMonths / 12);
  return rtf.format(diffYears, "year");
}

function normalizeCountMap(value: Record<string, number> | undefined): Record<string, number> {
  if (!value || typeof value !== "object") {
    return {};
  }
  const result: Record<string, number> = {};
  for (const [key, rawCount] of Object.entries(value)) {
    const normalizedKey = key.trim();
    if (!normalizedKey) {
      continue;
    }
    const count = Number(rawCount);
    result[normalizedKey] = Number.isFinite(count) ? count : 0;
  }
  return result;
}

function formatBytes(value: number | undefined): string {
  const size = Number(value ?? 0);
  if (!Number.isFinite(size) || size <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let normalized = size;
  let unitIndex = 0;
  while (normalized >= 1024 && unitIndex < units.length - 1) {
    normalized /= 1024;
    unitIndex += 1;
  }
  const digits = normalized >= 100 || unitIndex === 0 ? 0 : 1;
  return `${normalized.toFixed(digits)} ${units[unitIndex]}`;
}

const fixedCategoryOrder = ["preference", "fact", "decision", "entity", "reflection", "other"] as const;
const baseColumnsPerRow = 6;
const itemMinWidthRem = 8.5;
const gridGapRem = 0.75;
const gridGapPx = 12;

function distributeRowsEvenly<T>(items: T[], maxColumns: number): T[][] {
  if (items.length === 0) {
    return [];
  }
  const normalizedMaxColumns = Math.max(1, Math.floor(maxColumns));
  const rows = Math.ceil(items.length / normalizedMaxColumns);
  if (rows <= 1) {
    return [items];
  }
  const baseRowSize = Math.floor(items.length / rows);
  let remainder = items.length % rows;
  const result: T[][] = [];
  let index = 0;
  for (let row = 0; row < rows; row += 1) {
    const currentRowSize = baseRowSize + (remainder > 0 ? 1 : 0);
    result.push(items.slice(index, index + currentRowSize));
    index += currentRowSize;
    if (remainder > 0) {
      remainder -= 1;
    }
  }
  return result;
}

function buildRowGridStyle(columns: number): CSSProperties {
  const normalizedColumns = Math.max(1, Math.floor(columns));
  const gapRem = (normalizedColumns - 1) * gridGapRem;
  return {
    gridTemplateColumns: `repeat(${normalizedColumns},minmax(min(${itemMinWidthRem}rem,calc((100% - ${gapRem}rem)/${normalizedColumns})),1fr))`,
  };
}

function resolveAdaptiveMaxColumns(containerWidth: number): number {
  if (containerWidth <= 0) {
    return baseColumnsPerRow;
  }
  const rootFontSize =
    typeof window === "undefined"
      ? 16
      : Number.parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
  const itemMinWidthPx = itemMinWidthRem * rootFontSize;
  const adaptiveColumns = Math.max(1, Math.floor((containerWidth + gridGapPx) / (itemMinWidthPx + gridGapPx)));
  return Math.max(baseColumnsPerRow, adaptiveColumns);
}

function SummarySectionCard({ title, items, isLoading, onRefresh, refreshLabel }: SummarySectionCardProps) {
  const gridHostRef = useRef<HTMLDivElement | null>(null);
  const [maxColumns, setMaxColumns] = useState(baseColumnsPerRow);

  useEffect(() => {
    const element = gridHostRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }
    const update = () => {
      const nextColumns = resolveAdaptiveMaxColumns(element.clientWidth);
      setMaxColumns((current) => (current === nextColumns ? current : nextColumns));
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const rows = useMemo(() => distributeRowsEvenly(items, maxColumns), [items, maxColumns]);
  const sharedColumns = useMemo(() => {
    let maxRowSize = 1;
    for (const row of rows) {
      if (row.length > maxRowSize) {
        maxRowSize = row.length;
      }
    }
    return maxRowSize;
  }, [rows]);

  return (
    <Card className="w-full border bg-card">
      <CardHeader size="compact" className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-sm font-medium leading-none tracking-normal">{title}</CardTitle>
          <Button
            type="button"
            size="compactIcon"
            variant="outline"
            onClick={onRefresh}
            disabled={isLoading}
            aria-label={refreshLabel}
            title={refreshLabel}
          >
            <RefreshCw className={cn("h-4 w-4", isLoading && "animate-spin")} />
          </Button>
        </div>
      </CardHeader>
      <CardContent size="compact" className="pt-0">
        <div ref={gridHostRef} className="space-y-3">
          {rows.map((row, rowIndex) => (
            <div key={`${title}-${rowIndex}`} className="grid gap-3" style={buildRowGridStyle(sharedColumns)}>
              {row.map((item) => (
                <Card key={item.key}>
                  <CardHeader size="compact" className="flex min-h-[5.25rem] flex-col pb-1">
                    <CardDescription className="truncate" title={item.label}>
                      {item.label}
                    </CardDescription>
                    <CardTitle
                      className={cn(
                        "mt-auto break-words tracking-tight",
                        item.dense ? "text-sm font-medium leading-5" : "text-2xl tabular-nums",
                        item.clampValueToTwoLines &&
                          "overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:2]"
                      )}
                      title={item.title}
                    >
                      {item.value}
                    </CardTitle>
                  </CardHeader>
                </Card>
              ))}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export function SummaryTab({ t, language, summary, isLoading, onRefresh }: SummaryTabProps) {
  const categoryCounts = normalizeCountMap(summary?.categoryCounts);
  const channelCounts = normalizeCountMap(summary?.channelCounts);
  const accountCounts = normalizeCountMap(summary?.accountCounts);
  const scopeCounts = normalizeCountMap(summary?.scopeCounts);
  const noDataText = t("settings.memory.summary.noData");
  const justNowLabel = t("common.justNow");

  const allCategories = [
    ...fixedCategoryOrder,
    ...Object.keys(categoryCounts)
      .filter((item) => !fixedCategoryOrder.includes(item as (typeof fixedCategoryOrder)[number]))
      .sort(),
  ];

  const channelTotal = summary?.channelCount ?? Object.keys(channelCounts).length;
  const accountTotal = summary?.accountCount ?? Object.keys(accountCounts).length;
  const scopeTotal = Object.keys(scopeCounts).length;
  const updatedAtValue = formatSummaryTime(summary?.lastUpdatedAt, language, justNowLabel) || noDataText;

  const metrics: DashboardItemModel[] = [
    {
      key: "total",
      label: t("settings.memory.summary.total"),
      value: String(summary?.totalMemories ?? 0),
    },
    {
      key: "assistants",
      label: t("settings.memory.summary.assistants"),
      value: String(summary?.assistantCount ?? 0),
    },
    {
      key: "threads",
      label: t("settings.memory.summary.threads"),
      value: String(summary?.threadCount ?? 0),
    },
    {
      key: "users",
      label: t("settings.memory.summary.users"),
      value: String(summary?.userCount ?? 0),
    },
    {
      key: "groups",
      label: t("settings.memory.summary.groups"),
      value: String(summary?.groupCount ?? 0),
    },
    {
      key: "channels",
      label: t("settings.memory.summary.channels"),
      value: String(channelTotal),
    },
    {
      key: "accounts",
      label: t("settings.memory.summary.accounts"),
      value: String(accountTotal),
    },
    {
      key: "scopes",
      label: t("settings.memory.summary.scopes"),
      value: String(scopeTotal),
    },
    {
      key: "updatedAt",
      label: t("settings.memory.summary.updatedAt"),
      value: updatedAtValue,
      title: updatedAtValue,
      dense: true,
      clampValueToTwoLines: true,
    },
  ];

  const categoryItems: DashboardItemModel[] = allCategories.map((category) => ({
    key: `category-${category}`,
    label: t(`settings.memory.entries.categoryOption.${category}`),
    value: String(categoryCounts[category] ?? 0),
  }));

  const storageItems: DashboardItemModel[] = [
    {
      key: "storage-total",
      label: t("settings.memory.summary.storageTotal"),
      value: formatBytes(summary?.storage?.totalBytes),
    },
    {
      key: "storage-collections",
      label: t("settings.memory.summary.storageCollections"),
      value: formatBytes(summary?.storage?.collectionsBytes),
    },
    {
      key: "storage-chunks",
      label: t("settings.memory.summary.storageChunks"),
      value: formatBytes(summary?.storage?.chunksBytes),
    },
    {
      key: "storage-summary-files",
      label: t("settings.memory.summary.storageSummaryFiles"),
      value: formatBytes(summary?.storage?.assistantSummaryBytes),
    },
    {
      key: "storage-avatar-cache",
      label: t("settings.memory.summary.storageAvatarCache"),
      value: formatBytes(summary?.storage?.avatarCacheBytes),
    },
  ];

  const refreshLabel = t("common.refresh");

  return (
    <div className="space-y-3">
      <SummarySectionCard
        title={t("settings.memory.summary.title")}
        items={metrics}
        isLoading={isLoading}
        onRefresh={onRefresh}
        refreshLabel={refreshLabel}
      />
      <SummarySectionCard
        title={t("settings.memory.summary.categories")}
        items={categoryItems}
        isLoading={isLoading}
        onRefresh={onRefresh}
        refreshLabel={refreshLabel}
      />
      <SummarySectionCard
        title={t("settings.memory.summary.fileStorage")}
        items={storageItems}
        isLoading={isLoading}
        onRefresh={onRefresh}
        refreshLabel={refreshLabel}
      />
    </div>
  );
}
