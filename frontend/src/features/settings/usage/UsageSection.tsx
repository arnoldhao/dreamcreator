import * as React from "react";
import { Activity, BarChart3, Coins, Database, HelpCircle, RefreshCw, Zap } from "lucide-react";
import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from "recharts";

import { useI18n } from "@/shared/i18n";
import { useProviders } from "@/shared/query/providers";
import { useUsageCost, useUsageStatus } from "@/shared/query/usage";
import { Button } from "@/shared/ui/button";
import { CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from "@/shared/ui/chart";
import {
  DASHBOARD_CHART_SURFACE_CLASS,
  DASHBOARD_EMPTY_PANEL_CLASS,
  MetricCard,
  PanelCard,
} from "@/shared/ui/dashboard";
import { Select } from "@/shared/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";

type RangeOption = {
  id: string;
  labelKey: string;
  fallbackLabel: string;
  window: string;
};

type SourceOption = {
  id: string;
  labelKey: string;
  fallbackLabel: string;
};

type UsageTab = "models" | "cost";

type ActivityRange = {
  startDay: Date;
  endDay: Date;
  startAt: string;
  endAt: string;
  timezoneOffsetMinutes: number;
};

type ActivityDayCell = {
  key: string;
  date: Date;
  future: boolean;
  requests: number;
  units: number;
  costMicros: number;
  level: number;
};

type ActivityWeek = {
  key: string;
  days: ActivityDayCell[];
};

type ActivityMonthLabel = {
  key: string;
  label: string;
  startIndex: number;
  span: number;
};

const activityHeatmapWeeks = 53;
const activityHeatmapMinWidthPx = 920;

const rangeOptions: RangeOption[] = [
  { id: "1h", labelKey: "settings.usage.range.option.1h", fallbackLabel: "Last hour", window: "1h" },
  {
    id: "24h",
    labelKey: "settings.usage.range.option.24h",
    fallbackLabel: "Last 24 hours",
    window: "24h",
  },
  { id: "7d", labelKey: "settings.usage.range.option.7d", fallbackLabel: "Last 7 days", window: "7d" },
  {
    id: "30d",
    labelKey: "settings.usage.range.option.30d",
    fallbackLabel: "Last 30 days",
    window: "30d",
  },
  { id: "all", labelKey: "settings.usage.range.option.all", fallbackLabel: "All time", window: "all" },
];

const sourceOptions: SourceOption[] = [
  { id: "all", labelKey: "settings.usage.source.option.all", fallbackLabel: "All sources" },
  { id: "dialogue", labelKey: "settings.usage.source.option.dialogue", fallbackLabel: "Dialogue" },
  { id: "relay", labelKey: "settings.usage.source.option.relay", fallbackLabel: "Relay" },
  { id: "one-shot", labelKey: "settings.usage.source.option.oneShot", fallbackLabel: "One-shot" },
];

const formatCost = (micros: number) => {
  if (!Number.isFinite(micros)) {
    return "$0.0000";
  }
  return `$${(micros / 1_000_000).toFixed(4)}`;
};

const formatCostChartValue = (micros: number) => {
  if (!Number.isFinite(micros) || micros <= 0) {
    return 0;
  }
  return Number((micros / 1_000_000).toFixed(4));
};

const formatTokenUnits = (value: number) => {
  if (!Number.isFinite(value) || value <= 0) {
    return "0";
  }
  if (value >= 1_000_000_000) {
    return `${Math.round(value / 1_000_000_000)}B`;
  }
  if (value >= 1_000_000) {
    return `${Math.round(value / 1_000_000)}M`;
  }
  if (value >= 1_000) {
    return `${Math.round(value / 1_000)}K`;
  }
  return String(Math.round(value));
};

const formatPercentage = (value: number) => {
  if (!Number.isFinite(value) || value <= 0) {
    return "0%";
  }
  const percentage = value * 100;
  if (percentage > 0 && percentage < 0.1) {
    return "<0.1%";
  }
  return `${percentage >= 10 ? percentage.toFixed(0) : percentage.toFixed(1)}%`;
};

const formatTokenExact = (value: number) => {
  if (!Number.isFinite(value) || value <= 0) {
    return "0";
  }
  return Math.round(value).toLocaleString();
};

const formatProviderModel = (providerLabel?: string, modelName?: string, fallback = "Unknown") => {
  const provider = (providerLabel ?? "").trim();
  const model = (modelName ?? "").trim();
  if (provider && model) {
    return `${provider} / ${model}`;
  }
  if (model) {
    return model;
  }
  if (provider) {
    return provider;
  }
  return fallback;
};

const resolveDisplayTokens = (inputTokens: number, cachedInputTokens: number) =>
  Math.max(inputTokens - cachedInputTokens, 0);

const resolveDisplayOutputTokens = (outputTokens: number, reasoningTokens: number) =>
  Math.max(outputTokens - reasoningTokens, 0);

const shortenLabel = (value: string, maxLength = 24) => {
  const trimmed = value.trim();
  if (trimmed.length <= maxLength) {
    return trimmed;
  }
  return `${trimmed.slice(0, maxLength - 3)}...`;
};

const chartColor = (index: number) => `hsl(var(--chart-${(index % 5) + 1}))`;

const startOfLocalDay = (date: Date) => new Date(date.getFullYear(), date.getMonth(), date.getDate());

const addLocalDays = (date: Date, days: number) => {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return startOfLocalDay(result);
};

const formatActivityDateKey = (date: Date) => {
  const year = String(date.getFullYear());
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const buildActivityRange = (now = new Date()): ActivityRange => {
  const endDay = startOfLocalDay(now);
  const currentWeekStart = addLocalDays(endDay, -endDay.getDay());
  const startDay = addLocalDays(currentWeekStart, -(activityHeatmapWeeks - 1) * 7);
  return {
    startDay,
    endDay,
    startAt: startDay.toISOString(),
    endAt: addLocalDays(endDay, 1).toISOString(),
    timezoneOffsetMinutes: now.getTimezoneOffset(),
  };
};

const resolveActivityLevel = (requests: number, maxRequests: number) => {
  if (!Number.isFinite(requests) || requests <= 0 || maxRequests <= 0) {
    return 0;
  }
  return Math.max(1, Math.min(4, Math.ceil((requests / maxRequests) * 4)));
};

const activityCellStyle = (level: number, future: boolean): React.CSSProperties => {
  if (future) {
    return {
      backgroundColor: "hsl(var(--muted) / 0.08)",
      borderColor: "hsl(var(--border) / 0.32)",
    };
  }
  if (level <= 0) {
    return {
      backgroundColor: "hsl(var(--muted) / 0.18)",
      borderColor: "hsl(var(--border) / 0.5)",
    };
  }
  const alpha = [0, 0.24, 0.42, 0.62, 0.82][level] ?? 0.24;
  return {
    backgroundColor: `hsl(var(--chart-2) / ${alpha})`,
    borderColor: `hsl(var(--chart-2) / ${Math.min(alpha + 0.12, 0.95)})`,
  };
};

function UsageCardTitleWithTooltip({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex items-center gap-1.5">
      <CardTitle className="text-sm font-medium">{title}</CardTitle>
      <TooltipProvider delayDuration={100}>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="inline-flex h-5 w-5 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
              aria-label={description}
            >
              <HelpCircle className="h-4 w-4" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom" className="max-w-[260px]">
            {description}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}

export function UsageSection() {
  const { t, language } = useI18n();
  const { data: providers = [] } = useProviders();
  const [rangeId, setRangeId] = React.useState<string>("24h");
  const [sourceId, setSourceId] = React.useState<string>("all");
  const [tab, setTab] = React.useState<UsageTab>("models");
  const [isManualRefreshing, setIsManualRefreshing] = React.useState(false);
  const [activityRange] = React.useState<ActivityRange>(() => buildActivityRange());
  const activityHeatmapScrollRef = React.useRef<HTMLDivElement>(null);
  const activeRange = rangeOptions.find((option) => option.id === rangeId) ?? rangeOptions[0];
  const requestSource = sourceId === "all" ? undefined : sourceId;

  const modelUsage = useUsageStatus({
    window: activeRange.window,
    category: "tokens",
    requestSource,
    groupBy: ["providerId", "modelName"],
  });
  const costUsage = useUsageCost({
    window: activeRange.window,
    requestSource,
    groupBy: ["providerId", "modelName", "category", "costBasis"],
  });
  const sourceUsage = useUsageStatus({
    window: activeRange.window,
    category: "tokens",
    requestSource,
    groupBy: ["requestSource"],
  });
  const activityUsage = useUsageStatus({
    startAt: activityRange.startAt,
    endAt: activityRange.endAt,
    category: "tokens",
    requestSource,
    timezoneOffsetMinutes: activityRange.timezoneOffsetMinutes,
    groupBy: ["day"],
  });

  const modelTotals = modelUsage.data?.totals;
  const totalCostMicros = costUsage.data?.totalCostMicros ?? 0;
  const modelRows = modelUsage.data?.buckets ?? [];
  const costRows = costUsage.data?.lines ?? [];
  const sourceRows = sourceUsage.data?.buckets ?? [];
  const activityRows = activityUsage.data?.buckets ?? [];
  const tokenCostMicros = modelTotals?.costMicros ?? 0;
  const inputTokens = modelTotals?.inputTokens ?? 0;
  const cachedInputTokens = modelTotals?.cachedInputTokens ?? 0;
  const cachedTokenRate = inputTokens > 0 ? cachedInputTokens / inputTokens : 0;
  const totalCostRequests = costRows.reduce((total, row) => total + row.requests, 0);
  const avgCostPerRequestMicros = totalCostRequests > 0 ? totalCostMicros / totalCostRequests : 0;
  const isRefreshing =
    isManualRefreshing || modelUsage.isFetching || costUsage.isFetching || sourceUsage.isFetching || activityUsage.isFetching;
  const refreshUsage = React.useCallback(async () => {
    if (isManualRefreshing) {
      return;
    }
    const refreshStartAt = Date.now();
    setIsManualRefreshing(true);
    try {
      await new Promise<void>((resolve) => {
        if (typeof requestAnimationFrame === "function") {
          requestAnimationFrame(() => resolve());
          return;
        }
        window.setTimeout(resolve, 16);
      });
      await Promise.allSettled([modelUsage.refetch(), costUsage.refetch(), sourceUsage.refetch(), activityUsage.refetch()]);
    } finally {
      const elapsed = Date.now() - refreshStartAt;
      const minSpinDurationMs = 350;
      if (elapsed < minSpinDurationMs) {
        await new Promise<void>((resolve) => window.setTimeout(resolve, minSpinDurationMs - elapsed));
      }
      setIsManualRefreshing(false);
    }
  }, [activityUsage, costUsage, isManualRefreshing, modelUsage, sourceUsage]);

  const hasError = modelUsage.isError || costUsage.isError || sourceUsage.isError || activityUsage.isError;
  const errorMessage = modelUsage.error ?? costUsage.error ?? sourceUsage.error ?? activityUsage.error;
  const providerNameById = React.useMemo(() => {
    const result = new Map<string, string>();
    for (const provider of providers) {
      const id = provider.id.trim().toLowerCase();
      const name = provider.name.trim();
      if (!id || !name) {
        continue;
      }
      result.set(id, name);
    }
    return result;
  }, [providers]);
  const resolveProviderLabel = React.useCallback(
    (providerId?: string) => {
      const normalized = (providerId ?? "").trim();
      if (!normalized) {
        return "";
      }
      return providerNameById.get(normalized.toLowerCase()) ?? normalized;
    },
    [providerNameById]
  );

  const formatCategory = React.useCallback(
    (category?: string) => {
      switch ((category ?? "").trim()) {
        case "tokens":
          return t("settings.usage.category.tokens");
        case "context_tokens":
          return t("settings.usage.category.contextTokens");
        case "tts":
          return t("settings.usage.category.tts");
        default:
          return category || t("settings.usage.labels.unknown");
      }
    },
    [t]
  );

  const formatCostBasis = React.useCallback(
    (basis?: string) => {
      switch ((basis ?? "").trim()) {
        case "estimated":
          return t("settings.usage.costBasis.estimated");
        case "reconciled":
          return t("settings.usage.costBasis.reconciled");
        default:
          return basis || t("settings.usage.labels.unknown");
      }
    },
    [t]
  );

  const formatSource = React.useCallback(
    (source?: string) => {
      switch ((source ?? "").trim()) {
        case "dialogue":
          return t("settings.usage.source.option.dialogue");
        case "relay":
          return t("settings.usage.source.option.relay");
        case "one-shot":
          return t("settings.usage.source.option.oneShot");
        case "unknown":
          return t("settings.usage.source.option.unknown");
        default:
          return source || t("settings.usage.labels.unknown");
      }
    },
    [t]
  );

  const modelTokenChartData = React.useMemo(
    () =>
      modelRows.slice(0, 8).map((row) => {
        const label = formatProviderModel(resolveProviderLabel(row.providerId), row.modelName, t("settings.usage.labels.unknown"));
        return {
          name: shortenLabel(label, 28),
          fullName: label,
          input: resolveDisplayTokens(row.inputTokens, row.cachedInputTokens),
          cached: row.cachedInputTokens,
          output: resolveDisplayOutputTokens(row.outputTokens, row.reasoningTokens),
          reasoning: row.reasoningTokens,
        };
      }),
    [modelRows, resolveProviderLabel, t]
  );

  const modelCostChartData = React.useMemo(() => {
    const buckets = new Map<string, { name: string; fullName: string; requests: number; costMicros: number }>();
    for (const row of costRows) {
      const label = formatProviderModel(resolveProviderLabel(row.providerId), row.modelName, t("settings.usage.labels.unknown"));
      const key = `${row.providerId ?? ""}::${row.modelName ?? ""}`;
      const current = buckets.get(key) ?? {
        name: shortenLabel(label, 28),
        fullName: label,
        requests: 0,
        costMicros: 0,
      };
      current.requests += row.requests;
      current.costMicros += row.costMicros;
      buckets.set(key, current);
    }
    return Array.from(buckets.values())
      .sort((left, right) => right.costMicros - left.costMicros || right.requests - left.requests)
      .slice(0, 8)
      .map((row) => ({
        name: row.name,
        fullName: row.fullName,
        cost: formatCostChartValue(row.costMicros),
        requests: row.requests,
      }));
  }, [costRows, resolveProviderLabel, t]);

  const sourceChartData = React.useMemo(
    () =>
      sourceRows.map((row, index) => ({
        name: formatSource(row.requestSource),
        tokens: row.units,
        requests: row.requests,
        fill: chartColor(index),
      })),
    [formatSource, sourceRows]
  );

  const tokenChartConfig = React.useMemo(
    () => ({
      input: { label: t("settings.usage.chart.input"), color: chartColor(0) },
      cached: { label: t("settings.usage.chart.cached"), color: chartColor(1) },
      output: { label: t("settings.usage.chart.output"), color: chartColor(2) },
      reasoning: { label: t("settings.usage.chart.reasoning"), color: chartColor(3) },
    }),
    [t]
  );
  const costChartConfig = React.useMemo(
    () => ({
      cost: { label: t("settings.usage.summary.cost"), color: chartColor(4) },
    }),
    [t]
  );
  const sourceChartConfig = React.useMemo(
    () => ({
      tokens: { label: t("settings.usage.summary.units"), color: chartColor(0) },
    }),
    [t]
  );
  const activityBucketByDay = React.useMemo(() => {
    const result = new Map<string, { requests: number; units: number; costMicros: number }>();
    for (const row of activityRows) {
      const day = (row.bucketStart ?? row.key).trim().slice(0, 10);
      if (!day) {
        continue;
      }
      result.set(day, {
        requests: row.requests,
        units: row.units,
        costMicros: row.costMicros,
      });
    }
    return result;
  }, [activityRows]);
  const activityMaxRequests = React.useMemo(
    () => Math.max(0, ...Array.from(activityBucketByDay.values()).map((row) => row.requests)),
    [activityBucketByDay]
  );
  const activityWeeks = React.useMemo<ActivityWeek[]>(() => {
    return Array.from({ length: activityHeatmapWeeks }, (_, weekIndex) => {
      const weekStart = addLocalDays(activityRange.startDay, weekIndex * 7);
      return {
        key: formatActivityDateKey(weekStart),
        days: Array.from({ length: 7 }, (_, dayIndex) => {
          const date = addLocalDays(weekStart, dayIndex);
          const key = formatActivityDateKey(date);
          const bucket = activityBucketByDay.get(key);
          const future = date.getTime() > activityRange.endDay.getTime();
          const requests = future ? 0 : bucket?.requests ?? 0;
          return {
            key,
            date,
            future,
            requests,
            units: future ? 0 : bucket?.units ?? 0,
            costMicros: future ? 0 : bucket?.costMicros ?? 0,
            level: resolveActivityLevel(requests, activityMaxRequests),
          };
        }),
      };
    });
  }, [activityBucketByDay, activityMaxRequests, activityRange.endDay, activityRange.startDay]);
  const activityHeatmapGridStyle = React.useMemo<React.CSSProperties>(
    () => ({
      gridTemplateColumns: `repeat(${activityWeeks.length}, minmax(0, 1fr))`,
    }),
    [activityWeeks.length]
  );
  const formatActivityMonth = React.useCallback(
    (date: Date) =>
      new Intl.DateTimeFormat(language, {
        month: "short",
      }).format(date),
    [language]
  );
  const activityMonthLabels = React.useMemo<ActivityMonthLabel[]>(() => {
    const labelStarts = activityWeeks.flatMap((week, index) => {
      const monthStart = week.days.find((day) => day.date.getDate() === 1) ?? (index === 0 ? week.days[0] : undefined);
      if (!monthStart) {
        return [];
      }
      return [
        {
          key: `${formatActivityDateKey(monthStart.date)}-${index}`,
          label: formatActivityMonth(monthStart.date),
          startIndex: index,
        },
      ];
    });
    return labelStarts.map((label, index) => ({
      ...label,
      span: Math.max(1, (labelStarts[index + 1]?.startIndex ?? activityWeeks.length) - label.startIndex),
    }));
  }, [activityWeeks, formatActivityMonth]);
  const formatActivityDate = React.useCallback(
    (date: Date) =>
      new Intl.DateTimeFormat(language, {
        month: "short",
        day: "numeric",
        year: "numeric",
      }).format(date),
    [language]
  );
  const activityRangeLabel = React.useMemo(
    () => `${formatActivityDate(activityRange.startDay)} - ${formatActivityDate(activityRange.endDay)}`,
    [activityRange.endDay, activityRange.startDay, formatActivityDate]
  );
  React.useEffect(() => {
    const element = activityHeatmapScrollRef.current;
    if (!element || activityUsage.isLoading) {
      return;
    }
    const frame = window.requestAnimationFrame(() => {
      element.scrollLeft = element.scrollWidth;
    });
    return () => window.cancelAnimationFrame(frame);
  }, [activityUsage.isLoading, activityWeeks.length]);
  const activityHeatmapPanel = (
    <PanelCard tone="solid" className="min-w-0">
      <CardHeader size="compact" className="pb-2">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <UsageCardTitleWithTooltip
            title={t("settings.usage.chart.activityHeatmap")}
            description={t("settings.usage.chart.activityHeatmapDescription")}
          />
          <span className="text-xs text-muted-foreground">{activityRangeLabel}</span>
        </div>
      </CardHeader>
      <CardContent size="compact" className="pt-0">
        <div className="rounded-lg border border-border/70 bg-background/60 p-3">
          {activityUsage.isLoading ? (
            <div className={DASHBOARD_EMPTY_PANEL_CLASS}>{t("settings.usage.loading")}</div>
          ) : (
            <TooltipProvider delayDuration={120}>
              <div ref={activityHeatmapScrollRef} className="min-w-0 overflow-x-auto pb-1">
                <div className="w-full" style={{ minWidth: activityHeatmapMinWidthPx }}>
                  <div
                    className="grid gap-1 pb-1 pl-9 text-[10px] leading-3 text-muted-foreground"
                    style={activityHeatmapGridStyle}
                  >
                    {activityMonthLabels.map((label) => (
                      <span
                        key={label.key}
                        className="min-w-0 truncate"
                        style={{ gridColumn: `${label.startIndex + 1} / span ${label.span}`, gridRow: 1 }}
                      >
                        {label.label}
                      </span>
                    ))}
                  </div>
                  <div className="flex gap-2">
                    <div className="grid w-7 shrink-0 grid-rows-7 gap-1 text-[10px] leading-3 text-muted-foreground">
                      <span />
                      <span>{t("settings.usage.activity.weekday.mon")}</span>
                      <span />
                      <span>{t("settings.usage.activity.weekday.wed")}</span>
                      <span />
                      <span>{t("settings.usage.activity.weekday.fri")}</span>
                      <span />
                    </div>
                    <div className="grid flex-1 gap-1" style={activityHeatmapGridStyle}>
                      {activityWeeks.map((week) => (
                        <div key={week.key} className="grid grid-rows-7 gap-1">
                          {week.days.map((day) => (
                            <Tooltip key={day.key}>
                              <TooltipTrigger asChild>
                                <div
                                  className="aspect-square w-full rounded-[3px] border"
                                  style={activityCellStyle(day.level, day.future)}
                                  aria-label={`${formatActivityDate(day.date)}: ${day.requests} ${t(
                                    "settings.usage.summary.requests"
                                  )}`}
                                />
                              </TooltipTrigger>
                              <TooltipContent side="bottom">
                                {day.future
                                  ? `${formatActivityDate(day.date)}: ${t("settings.usage.activity.future")}`
                                  : `${formatActivityDate(day.date)}: ${day.requests} ${t(
                                      "settings.usage.summary.requests"
                                    )} / ${formatTokenExact(day.units)} ${t(
                                      "settings.usage.summary.units"
                                    )} / ${formatCost(day.costMicros)}`}
                              </TooltipContent>
                            </Tooltip>
                          ))}
                        </div>
                      ))}
                    </div>
                  </div>
                  <div className="mt-3 flex items-center justify-end gap-1 text-[10px] text-muted-foreground">
                    <span>{t("settings.usage.activity.less")}</span>
                    {[0, 1, 2, 3, 4].map((level) => (
                      <span
                        key={level}
                        className="h-3 w-3 rounded-[3px] border"
                        style={activityCellStyle(level, false)}
                      />
                    ))}
                    <span>{t("settings.usage.activity.more")}</span>
                  </div>
                </div>
              </div>
            </TooltipProvider>
          )}
        </div>
      </CardContent>
    </PanelCard>
  );

  return (
    <div className="space-y-6">
      <PanelCard tone="solid">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-1.5">
                <CardTitle className="text-sm font-medium leading-none tracking-normal">
                  {t("settings.usage.overview.title")}
                </CardTitle>
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        className="inline-flex h-5 w-5 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                        aria-label={t("settings.usage.overview.description")}
                      >
                        <HelpCircle className="h-4 w-4" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="bottom">
                      {t("settings.usage.overview.description")}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </div>
            <div className="flex items-center gap-2 self-start sm:self-auto">
              <Select
                value={sourceId}
                onChange={(event) => setSourceId(event.target.value)}
                className="w-36"
              >
                {sourceOptions.map((option) => (
                  <option key={option.id} value={option.id}>
                    {t(option.labelKey)}
                  </option>
                ))}
              </Select>
              <Select
                value={rangeId}
                onChange={(event) => setRangeId(event.target.value)}
                className="w-40"
              >
                {rangeOptions.map((option) => (
                  <option key={option.id} value={option.id}>
                    {t(option.labelKey)}
                  </option>
                ))}
              </Select>
              <Button
                type="button"
                size="compactIcon"
                variant="outline"
                onClick={refreshUsage}
                disabled={isRefreshing}
                aria-label={t("settings.usage.actions.refresh")}
                title={t("settings.usage.actions.refresh")}
              >
                <RefreshCw className={isRefreshing ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent size="compact" className="pt-0">
          <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3 2xl:grid-cols-6">
            <MetricCard
              title={t("settings.usage.summary.requests")}
              value={(modelTotals?.requests ?? 0).toLocaleString()}
              description={t("settings.usage.metric.requestsHint")}
              icon={Activity}
              minHeightClassName="min-h-[104px]"
            />
            <MetricCard
              title={t("settings.usage.summary.units")}
              value={formatTokenUnits(modelTotals?.units ?? 0)}
              description={formatTokenExact(modelTotals?.units ?? 0)}
              icon={BarChart3}
              minHeightClassName="min-h-[104px]"
            />
            <MetricCard
              title={t("settings.usage.summary.promptTokens")}
              value={formatTokenUnits(inputTokens)}
              description={t("settings.usage.metric.inputHint")}
              icon={Database}
              minHeightClassName="min-h-[104px]"
            />
            <MetricCard
              title={t("settings.usage.summary.completionTokens")}
              value={formatTokenUnits(modelTotals?.outputTokens ?? 0)}
              description={t("settings.usage.metric.outputHint")}
              icon={Zap}
              minHeightClassName="min-h-[104px]"
            />
            <MetricCard
              title={t("settings.usage.summary.cost")}
              value={formatCost(totalCostMicros)}
              description={`${t("settings.usage.metric.avgCost")} ${formatCost(avgCostPerRequestMicros)}`}
              icon={Coins}
              minHeightClassName="min-h-[104px]"
            />
            <MetricCard
              title={t("settings.usage.summary.cachedTokens")}
              value={formatTokenUnits(cachedInputTokens)}
              description={formatPercentage(cachedTokenRate)}
              icon={Database}
              minHeightClassName="min-h-[104px]"
            />
          </div>
        </CardContent>
      </PanelCard>

      {activityHeatmapPanel}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
        <PanelCard tone="solid" className="min-w-0">
          <CardHeader size="compact" className="pb-2">
            <UsageCardTitleWithTooltip
              title={t("settings.usage.chart.tokensByModel")}
              description={t("settings.usage.chart.tokensByModelDescription")}
            />
          </CardHeader>
          <CardContent size="compact" className="pt-0">
            <div className={DASHBOARD_CHART_SURFACE_CLASS}>
              {modelTokenChartData.length === 0 ? (
                <div className={DASHBOARD_EMPTY_PANEL_CLASS}>{t("settings.usage.breakdown.empty")}</div>
              ) : (
                <ChartContainer config={tokenChartConfig} className="h-[280px]">
                  <BarChart data={modelTokenChartData} layout="vertical" margin={{ left: 8, right: 12, top: 8, bottom: 8 }}>
                    <CartesianGrid horizontal={false} strokeDasharray="3 3" />
                    <XAxis type="number" tickFormatter={formatTokenUnits} tickLine={false} axisLine={false} />
                    <YAxis
                      type="category"
                      dataKey="name"
                      width={128}
                      tickLine={false}
                      axisLine={false}
                      tick={{ fontSize: 11 }}
                    />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar dataKey="input" stackId="tokens" fill="var(--color-input)" radius={[4, 0, 0, 4]} />
                    <Bar dataKey="cached" stackId="tokens" fill="var(--color-cached)" />
                    <Bar dataKey="output" stackId="tokens" fill="var(--color-output)" />
                    <Bar dataKey="reasoning" stackId="tokens" fill="var(--color-reasoning)" radius={[0, 4, 4, 0]} />
                  </BarChart>
                </ChartContainer>
              )}
            </div>
          </CardContent>
        </PanelCard>

        <div className="grid min-w-0 gap-4 md:grid-cols-2 xl:grid-cols-1">
          <PanelCard tone="solid" className="min-w-0">
            <CardHeader size="compact" className="pb-2">
              <UsageCardTitleWithTooltip
                title={t("settings.usage.chart.costByModel")}
                description={t("settings.usage.chart.costByModelDescription")}
              />
            </CardHeader>
            <CardContent size="compact" className="pt-0">
              <div className={DASHBOARD_CHART_SURFACE_CLASS}>
                {modelCostChartData.length === 0 ? (
                  <div className={DASHBOARD_EMPTY_PANEL_CLASS}>{t("settings.usage.breakdown.empty")}</div>
                ) : (
                  <ChartContainer config={costChartConfig} className="h-[220px]">
                    <BarChart data={modelCostChartData} layout="vertical" margin={{ left: 8, right: 12, top: 8, bottom: 8 }}>
                      <CartesianGrid horizontal={false} strokeDasharray="3 3" />
                      <XAxis type="number" tickFormatter={(value) => `$${value}`} tickLine={false} axisLine={false} />
                      <YAxis
                        type="category"
                        dataKey="name"
                        width={112}
                        tickLine={false}
                        axisLine={false}
                        tick={{ fontSize: 11 }}
                      />
                      <ChartTooltip content={<ChartTooltipContent />} />
                      <Bar dataKey="cost" fill="var(--color-cost)" radius={4} />
                    </BarChart>
                  </ChartContainer>
                )}
              </div>
            </CardContent>
          </PanelCard>

          <PanelCard tone="solid" className="min-w-0">
            <CardHeader size="compact" className="pb-2">
              <UsageCardTitleWithTooltip
                title={t("settings.usage.chart.sourceBreakdown")}
                description={t("settings.usage.chart.sourceBreakdownDescription")}
              />
            </CardHeader>
            <CardContent size="compact" className="pt-0">
              <div className={DASHBOARD_CHART_SURFACE_CLASS}>
                {sourceChartData.length === 0 ? (
                  <div className={DASHBOARD_EMPTY_PANEL_CLASS}>{t("settings.usage.breakdown.empty")}</div>
                ) : (
                  <ChartContainer config={sourceChartConfig} className="h-[220px]">
                    <BarChart data={sourceChartData} margin={{ left: 8, right: 12, top: 8, bottom: 8 }}>
                      <CartesianGrid vertical={false} strokeDasharray="3 3" />
                      <XAxis dataKey="name" tickLine={false} axisLine={false} tick={{ fontSize: 11 }} />
                      <YAxis tickFormatter={formatTokenUnits} tickLine={false} axisLine={false} width={42} />
                      <ChartTooltip content={<ChartTooltipContent />} />
                      <Bar dataKey="tokens" radius={4}>
                        {sourceChartData.map((entry) => (
                          <Cell key={entry.name} fill={entry.fill} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ChartContainer>
                )}
              </div>
            </CardContent>
          </PanelCard>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(value) => setTab(value as UsageTab)}>
        <div className="flex justify-center">
          <TabsList className="w-fit">
            <TabsTrigger value="models">{t("settings.usage.tabs.models")}</TabsTrigger>
            <TabsTrigger value="cost">{t("settings.usage.tabs.cost")}</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="models" className="mt-3">
          <TooltipProvider delayDuration={120}>
            <div className="rounded-lg bg-card outline outline-1 outline-border">
              <div className="overflow-x-auto p-2">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("settings.usage.labels.providerModel")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.requests")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.units")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.promptTokens")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.cachedTokens")}</TableHead>
                      <TableHead className="text-right">
                        {t("settings.usage.summary.completionTokens")}
                      </TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.reasoningTokens")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.cost")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.labels.share")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {modelUsage.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={9} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.loading")}
                        </TableCell>
                      </TableRow>
                    ) : modelRows.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={9} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.breakdown.empty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      modelRows.map((row) => {
                        const modelLabel = formatProviderModel(
                          resolveProviderLabel(row.providerId),
                          row.modelName,
                          t("settings.usage.labels.unknown")
                        );
                        return (
                          <TableRow key={row.key}>
                            <TableCell className="max-w-0">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="block truncate">{modelLabel}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">{modelLabel}</TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">{row.requests}</TableCell>
                            <TableCell className="text-right tabular-nums">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>{formatTokenUnits(row.units)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">{formatTokenExact(row.units)}</TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>{formatTokenUnits(row.inputTokens)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">{formatTokenExact(row.inputTokens)}</TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>{formatTokenUnits(row.cachedInputTokens)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">
                                  {formatTokenExact(row.cachedInputTokens)}
                                </TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>{formatTokenUnits(row.outputTokens)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">
                                  {formatTokenExact(row.outputTokens)}
                                </TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>{formatTokenUnits(row.reasoningTokens)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">
                                  {formatTokenExact(row.reasoningTokens)}
                                </TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell className="text-right tabular-nums">{formatCost(row.costMicros)}</TableCell>
                            <TableCell className="text-right tabular-nums">
                              {formatPercentage(tokenCostMicros > 0 ? row.costMicros / tokenCostMicros : 0)}
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          </TooltipProvider>
        </TabsContent>

        <TabsContent value="cost" className="mt-3">
          <TooltipProvider delayDuration={120}>
            <div className="rounded-lg bg-card outline outline-1 outline-border">
              <div className="overflow-x-auto p-2">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("settings.usage.labels.providerModel")}</TableHead>
                      <TableHead>{t("settings.usage.labels.category")}</TableHead>
                      <TableHead>{t("settings.usage.labels.costBasis")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.requests")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.cost")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.labels.share")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {costUsage.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.loading")}
                        </TableCell>
                      </TableRow>
                    ) : costRows.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.breakdown.empty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      costRows.map((row, index) => {
                        const modelLabel = formatProviderModel(
                          resolveProviderLabel(row.providerId),
                          row.modelName,
                          t("settings.usage.labels.unknown")
                        );
                        return (
                          <TableRow key={`${row.providerId ?? ""}:${row.modelName ?? ""}:${row.category ?? ""}:${index}`}>
                            <TableCell className="max-w-0">
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="block truncate">{modelLabel}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">{modelLabel}</TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell>{formatCategory(row.category)}</TableCell>
                            <TableCell>{formatCostBasis(row.costBasis)}</TableCell>
                            <TableCell className="text-right tabular-nums">{row.requests}</TableCell>
                            <TableCell className="text-right tabular-nums">{formatCost(row.costMicros)}</TableCell>
                            <TableCell className="text-right tabular-nums">
                              {formatPercentage(totalCostMicros > 0 ? row.costMicros / totalCostMicros : 0)}
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          </TooltipProvider>
        </TabsContent>
      </Tabs>

      {hasError ? (
        <div className="text-xs text-destructive">
          {t("settings.usage.error")} {String(errorMessage ?? "")}
        </div>
      ) : null}
    </div>
  );
}
