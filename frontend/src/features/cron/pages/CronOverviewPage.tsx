import * as React from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Activity, AlarmClock, Clock3, Timer } from "lucide-react";

import type { CronRunRecord } from "@/shared/store/cron";
import { CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from "@/shared/ui/chart";
import {
  DASHBOARD_CHART_SURFACE_CLASS,
  DASHBOARD_EMPTY_PANEL_CLASS,
  DASHBOARD_RECORD_ITEM_CLASS,
  MetricCard,
  PanelCard,
} from "@/shared/ui/dashboard";
import { Select } from "@/shared/ui/select";
import { cn } from "@/lib/utils";

type OverviewCardItem = {
  id: string;
  label: string;
  value: string;
};

type OverviewChartPoint = {
  label: string;
  success: number;
  failed: number;
};

type GranularityOption = {
  value: string;
  label: string;
};

const resolveOverviewCardIcon = (cardID: string) => {
  switch (cardID) {
    case "total-crons":
      return Clock3;
    case "total-executions":
      return Activity;
    case "avg-duration":
      return Timer;
    case "next-wake":
      return AlarmClock;
    default:
      return Clock3;
  }
};

const resolveOverviewCardValueClass = (value: string) => {
  const length = Array.from(value ?? "").length;
  if (length > 30) {
    return "text-sm sm:text-base";
  }
  if (length > 22) {
    return "text-base sm:text-lg";
  }
  if (length > 14) {
    return "text-lg sm:text-xl";
  }
  return "text-2xl";
};

type CronOverviewPageProps = {
  cards: OverviewCardItem[];
  chartData: OverviewChartPoint[];
  chartTitle: string;
  chartSuccessLabel: string;
  chartFailedLabel: string;
  chartGranularity: string;
  chartGranularityOptions: GranularityOption[];
  onChartGranularityChange: (value: string) => void;
  recentTitle: string;
  recentRuns: CronRunRecord[];
  resolveRunTitle: (run: CronRunRecord) => string;
  resolveRunFrequency: (run: CronRunRecord) => string;
  renderRunStatusIcon: (run: CronRunRecord) => React.ReactNode;
  formatStartedAt: (value?: string) => string;
  emptyRecentText: string;
};

const RECENT_ROW_HEIGHT = 56;
const RECENT_HEADER_HEIGHT = 56;

export function CronOverviewPage({
  cards,
  chartData,
  chartTitle,
  chartSuccessLabel,
  chartFailedLabel,
  chartGranularity,
  chartGranularityOptions,
  onChartGranularityChange,
  recentTitle,
  recentRuns,
  resolveRunTitle,
  resolveRunFrequency,
  renderRunStatusIcon,
  formatStartedAt,
  emptyRecentText,
}: CronOverviewPageProps) {
  const recentViewportRef = React.useRef<HTMLDivElement | null>(null);
  const [recentViewportHeight, setRecentViewportHeight] = React.useState(0);

  React.useEffect(() => {
    const node = recentViewportRef.current;
    if (!node || typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      const nextHeight = entry?.contentRect?.height ?? 0;
      setRecentViewportHeight(nextHeight);
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  const recentVisibleCount = React.useMemo(() => {
    const available = Math.max(RECENT_ROW_HEIGHT, recentViewportHeight - RECENT_HEADER_HEIGHT);
    return Math.max(1, Math.floor(available / RECENT_ROW_HEIGHT));
  }, [recentViewportHeight]);

  const visibleRecentRuns = React.useMemo(() => recentRuns.slice(0, recentVisibleCount), [recentRuns, recentVisibleCount]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => {
          return (
            <MetricCard
              key={card.id}
              title={card.label}
              value={card.value}
              icon={resolveOverviewCardIcon(card.id)}
              valueClassName={resolveOverviewCardValueClass(card.value)}
            />
          );
        })}
      </div>

      <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[minmax(0,2fr)_minmax(300px,1fr)]">
        <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col">
          <CardHeader className="pb-2">
            <div className="flex items-start justify-between gap-3">
              <CardTitle className="text-sm font-semibold">{chartTitle}</CardTitle>
              <Select
                className="w-[132px]"
                value={chartGranularity}
                onChange={(event) => onChartGranularityChange(event.target.value)}
              >
                {chartGranularityOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </div>
          </CardHeader>
          <CardContent className="min-h-0 flex-1 pb-3">
            <div className={DASHBOARD_CHART_SURFACE_CLASS}>
              <ChartContainer
                config={{
                  success: { label: chartSuccessLabel, color: "hsl(var(--primary))" },
                  failed: { label: chartFailedLabel, color: "hsl(var(--primary) / 0.55)" },
                }}
                className="h-full w-full"
              >
                <BarChart data={chartData} margin={{ left: 8, right: 8, top: 24, bottom: 4 }}>
                  <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border/70" />
                  <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={10} fontSize={11} minTickGap={20} />
                  <YAxis allowDecimals={false} tickLine={false} axisLine={false} fontSize={11} width={32} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <ChartLegend align="right" verticalAlign="top" content={<ChartLegendContent />} />
                  <Bar
                    dataKey="success"
                    stackId="executions"
                    fill="var(--color-success)"
                    radius={[0, 0, 4, 4]}
                  />
                  <Bar
                    dataKey="failed"
                    stackId="executions"
                    fill="var(--color-failed)"
                    radius={[4, 4, 0, 0]}
                  />
                </BarChart>
              </ChartContainer>
            </div>
          </CardContent>
        </PanelCard>

        <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">{recentTitle}</CardTitle>
          </CardHeader>
          <CardContent ref={recentViewportRef} className="min-h-0 flex-1 overflow-hidden pb-3">
            {visibleRecentRuns.length === 0 ? (
              <div className={DASHBOARD_EMPTY_PANEL_CLASS}>
                {emptyRecentText}
              </div>
            ) : (
              <div className="space-y-2 overflow-hidden">
                {visibleRecentRuns.map((run) => (
                  <div
                    key={run.runId}
                    className={DASHBOARD_RECORD_ITEM_CLASS}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate text-xs font-medium">{resolveRunTitle(run)}</span>
                      {renderRunStatusIcon(run)}
                    </div>
                    <div className={cn("mt-1 flex items-center justify-between gap-2 text-[11px] text-muted-foreground")}>
                      <span className="truncate">{resolveRunFrequency(run)}</span>
                      <span className="shrink-0 text-right">{formatStartedAt(run.startedAt)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </PanelCard>
      </div>
    </div>
  );
}
