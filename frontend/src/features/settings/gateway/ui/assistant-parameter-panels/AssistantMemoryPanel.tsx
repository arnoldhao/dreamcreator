import { Events } from "@wailsio/runtime";
import { Loader2 } from "lucide-react";

import type { MemorySummary } from "@/shared/query/memory";
import { Button } from "@/shared/ui/button";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";

import { panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

interface AssistantMemoryPanelProps {
  t: Translate;
  enabled: boolean;
  summary?: MemorySummary;
  language: string;
  isLoading: boolean;
  onEnabledChange: (enabled: boolean) => void;
}

function formatRelativeTime(value: string | undefined, language: string): string {
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
  if (absSeconds < 60) {
    return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(diffSeconds, "second");
  }
  const diffMinutes = Math.round(diffSeconds / 60);
  if (Math.abs(diffMinutes) < 60) {
    return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(diffMinutes, "minute");
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (Math.abs(diffHours) < 24) {
    return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(diffHours, "hour");
  }
  const diffDays = Math.round(diffHours / 24);
  if (Math.abs(diffDays) < 30) {
    return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(diffDays, "day");
  }
  const diffMonths = Math.round(diffDays / 30);
  if (Math.abs(diffMonths) < 12) {
    return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(diffMonths, "month");
  }
  return new Intl.RelativeTimeFormat(language, { numeric: "always" }).format(Math.round(diffMonths / 12), "year");
}

export function AssistantMemoryPanel({
  t,
  enabled,
  summary,
  language,
  isLoading,
  onEnabledChange,
}: AssistantMemoryPanelProps) {
  const noData = t("settings.memory.summary.noData");
  const resolvedTotalMemories = Math.max(0, Number(summary?.totalMemories ?? 0));
  const resolvedUpdatedAt = summary?.lastUpdatedAt?.trim() || "";
  const summaryItems = [
    {
      key: "total",
      label: t("settings.memory.summary.total"),
      value: String(resolvedTotalMemories),
    },
    {
      key: "updatedAt",
      label: t("settings.memory.summary.updatedAt"),
      value: formatRelativeTime(resolvedUpdatedAt, language) || noData,
    },
  ];

  return (
    <div className={panelClassName}>
      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.memory.enabled")}
        </div>
        <Switch checked={enabled} onCheckedChange={onEnabledChange} />
      </div>

      <Separator />

      <div className="space-y-2">
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.memory.summary")}
        </div>
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>{t("settings.gateway.memory.loading")}</span>
          </div>
        ) : (
          <div className="grid gap-2 sm:grid-cols-2">
            {summaryItems.map((item) => (
              <div key={item.key} className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                <div className="truncate text-[11px] text-muted-foreground" title={item.label}>
                  {item.label}
                </div>
                <div className="truncate text-sm font-medium text-foreground" title={item.value}>
                  {item.value}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <Separator />

      <div className="flex items-center justify-end">
        <Button
          type="button"
          variant="outline"
          size="compact"
          onClick={() => Events.Emit("settings:navigate", "memory")}
        >
          {t("settings.gateway.memory.openSettings")}
        </Button>
      </div>
    </div>
  );
}
