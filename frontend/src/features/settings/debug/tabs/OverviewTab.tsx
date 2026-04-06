import { Fragment, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { RefreshCw, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { TabsContent } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { cn } from "@/lib/utils";

import type { OverviewTabProps } from "../types";

interface GatewayStatusMetricItem {
  key: string;
  label: string;
  value: string;
}

const gatewayBaseColumnsPerRow = 5;
const gatewayItemMinWidthRem = 8.5;
const gatewayGridGapRem = 0.75;
const gatewayGridGapPx = 12;

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
  const gapRem = (normalizedColumns - 1) * gatewayGridGapRem;
  return {
    gridTemplateColumns: `repeat(${normalizedColumns},minmax(min(${gatewayItemMinWidthRem}rem,calc((100% - ${gapRem}rem)/${normalizedColumns})),1fr))`,
  };
}

function resolveAdaptiveMaxColumns(containerWidth: number): number {
  if (containerWidth <= 0) {
    return gatewayBaseColumnsPerRow;
  }
  const rootFontSize =
    typeof window === "undefined"
      ? 16
      : Number.parseFloat(window.getComputedStyle(document.documentElement).fontSize) || 16;
  const itemMinWidthPx = gatewayItemMinWidthRem * rootFontSize;
  const adaptiveColumns = Math.max(
    1,
    Math.floor((containerWidth + gatewayGridGapPx) / (itemMinWidthPx + gatewayGridGapPx))
  );
  return Math.max(gatewayBaseColumnsPerRow, adaptiveColumns);
}

export function OverviewTab({
  t,
  refreshOverview,
  gatewayStatus,
  health,
  url,
  statusLabel,
  status,
  metrics,
  selectedTopic,
  setSelectedTopic,
  topicOptions,
  visibleMessages,
  messages,
  clearMessages,
}: OverviewTabProps) {
  const [expandedMessageKey, setExpandedMessageKey] = useState<string | null>(null);
  const gatewayGridHostRef = useRef<HTMLDivElement | null>(null);
  const [gatewayMaxColumns, setGatewayMaxColumns] = useState(gatewayBaseColumnsPerRow);
  const healthOverall = (health?.overall ?? "").trim();
  const healthBadgeClass = cn(
    "rounded-full px-2 py-0.5 text-xs",
    !health
      ? "bg-muted text-muted-foreground"
      : healthOverall.toLowerCase() === "healthy"
        ? "bg-emerald-100 text-emerald-800"
        : healthOverall.toLowerCase() === "degraded"
          ? "bg-amber-100 text-amber-800"
          : healthOverall.toLowerCase() === "unhealthy"
            ? "bg-destructive/15 text-destructive"
            : "bg-muted text-muted-foreground"
  );
  const totalMessageCount = useMemo(
    () => Object.values(messages).reduce((sum, list) => sum + (Array.isArray(list) ? list.length : 0), 0),
    [messages]
  );
  const gatewayStatusItems = useMemo<GatewayStatusMetricItem[]>(
    () => [
      {
        key: "activeSessions",
        label: t("settings.debug.status.activeSessions"),
        value: gatewayStatus?.activeSessions != null ? String(gatewayStatus.activeSessions) : "-",
      },
      {
        key: "activeRuns",
        label: t("settings.debug.status.activeRuns"),
        value: gatewayStatus?.activeRuns != null ? String(gatewayStatus.activeRuns) : "-",
      },
      {
        key: "queueDepth",
        label: t("settings.debug.status.queueDepth"),
        value: gatewayStatus?.queueDepth != null ? String(gatewayStatus.queueDepth) : "-",
      },
      {
        key: "connectedNodes",
        label: t("settings.debug.status.connectedNodes"),
        value: gatewayStatus?.connectedNodes != null ? String(gatewayStatus.connectedNodes) : "-",
      },
      {
        key: "uptime",
        label: t("settings.debug.status.uptime"),
        value: gatewayStatus ? `${gatewayStatus.uptimeSec}s` : "-",
      },
    ],
    [gatewayStatus, t]
  );
  const gatewayStatusRows = useMemo(
    () => distributeRowsEvenly(gatewayStatusItems, gatewayMaxColumns),
    [gatewayMaxColumns, gatewayStatusItems]
  );
  const gatewaySharedColumns = useMemo(() => {
    let maxRowSize = 1;
    for (const row of gatewayStatusRows) {
      if (row.length > maxRowSize) {
        maxRowSize = row.length;
      }
    }
    return maxRowSize;
  }, [gatewayStatusRows]);

  useEffect(() => {
    const element = gatewayGridHostRef.current;
    if (!element || typeof ResizeObserver === "undefined") {
      return;
    }
    const update = () => {
      const nextColumns = resolveAdaptiveMaxColumns(element.clientWidth);
      setGatewayMaxColumns((current) => (current === nextColumns ? current : nextColumns));
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const toggleMessagePayload = (key: string) => {
    setExpandedMessageKey((current) => (current === key ? null : key));
  };

  const formatPayload = (payload: unknown) => {
    if (payload == null) {
      return t("settings.debug.realtime.noPayload");
    }
    try {
      return JSON.stringify(payload, null, 2);
    } catch {
      return String(payload);
    }
  };

  return (
    <TabsContent value="overview" className="mt-0 space-y-3">
      <Card className="w-full border bg-card">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <CardTitle className="text-sm font-medium leading-none tracking-normal">
              {t("settings.debug.status.title")}
            </CardTitle>
            <div className="flex items-center gap-2">
              <span className="font-mono text-[11px] text-muted-foreground">
                {t("settings.debug.diagnostics.health")}
              </span>
              <span className={healthBadgeClass}>{health ? healthOverall || "-" : t("common.loading")}</span>
              <Button
                variant="outline"
                size="compactIcon"
                onClick={refreshOverview}
                aria-label={t("common.refresh")}
                title={t("common.refresh")}
              >
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent size="compact" className="pt-0">
          <div ref={gatewayGridHostRef} className="space-y-3">
            {gatewayStatusRows.map((row, rowIndex) => (
              <div key={`gateway-overview-row-${rowIndex}`} className="grid gap-3" style={buildRowGridStyle(gatewaySharedColumns)}>
                {row.map((item) => (
                  <Card key={item.key}>
                    <CardHeader size="compact" className="flex min-h-[5.25rem] flex-col pb-1">
                      <CardDescription className="truncate" title={item.label}>
                        {item.label}
                      </CardDescription>
                      <CardTitle className="mt-auto break-words text-2xl tabular-nums tracking-tight">{item.value}</CardTitle>
                    </CardHeader>
                  </Card>
                ))}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card className="w-full border bg-card">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <CardTitle className="text-sm font-medium leading-none tracking-normal">
              {t("settings.debug.message.realtime.websocket")}
            </CardTitle>
            <div className="flex items-center gap-2">
              <span className="font-mono text-[11px] text-muted-foreground">
                {url || t("settings.debug.message.realtime.urlEmpty")}
              </span>
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-xs",
                  status === "connected"
                    ? "bg-emerald-100 text-emerald-800"
                    : status === "connecting"
                      ? "bg-amber-100 text-amber-800"
                      : "bg-muted text-muted-foreground"
                )}
              >
                {statusLabel}
              </span>
            </div>
          </div>
        </CardHeader>

        <CardContent size="compact" className="space-y-2 pt-0">
          <div className="grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-md border border-border/60 bg-muted/20 p-2">
            {t("settings.debug.realtime.metrics.reconnects")}: {metrics.reconnects}
          </div>
          <div className="rounded-md border border-border/60 bg-muted/20 p-2">
            {t("settings.debug.realtime.metrics.replay")}: {metrics.replayEvents}
          </div>
          <div className="rounded-md border border-border/60 bg-muted/20 p-2">
            {t("settings.debug.realtime.metrics.resync")}: {metrics.resyncRequired}
          </div>
          <div className="rounded-md border border-border/60 bg-muted/20 p-2">
            {t("settings.debug.realtime.metrics.duplicateDrop")}: {metrics.duplicateDrops}
          </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2 text-xs">
              <div className="font-medium text-muted-foreground">{t("settings.debug.message.realtime.topics")}</div>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">
                  {t("settings.debug.realtime.messagesCount")}: {visibleMessages.length}/{totalMessageCount}
                </span>
                <Select
                  value={selectedTopic}
                  onChange={(event) => setSelectedTopic(event.target.value)}
                  className="min-w-[220px]"
                >
                  {topicOptions.map((topic, index) => {
                    const count = topic === "all" ? visibleMessages.length : messages[topic]?.length ?? 0;
                    return (
                      <option key={`${topic || "topic"}-${index}`} value={topic}>
                        {topic} ({count})
                      </option>
                    );
                  })}
                </Select>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button size="compactIcon" variant="destructive" onClick={clearMessages}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent side="bottom">{t("settings.debug.message.realtime.clearAll")}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            </div>
            <div className="rounded-lg bg-card outline outline-1 outline-border">
              <div className="p-2">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "20%" }} />
                    <col style={{ width: "40%" }} />
                    <col style={{ width: "20%" }} />
                    <col style={{ width: "20%" }} />
                  </colgroup>
                  <TableHeader className="app-table-dense-head [&_tr]:border-b">
                    <TableRow>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.realtime.table.time")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.realtime.table.topic")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.realtime.table.type")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.realtime.table.seq")}</TableHead>
                    </TableRow>
                  </TableHeader>
                </Table>
              </div>

              <div className="max-h-64 overflow-y-auto overflow-x-hidden">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "20%" }} />
                    <col style={{ width: "40%" }} />
                    <col style={{ width: "20%" }} />
                    <col style={{ width: "20%" }} />
                  </colgroup>
                  <TableBody>
                    {visibleMessages.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="px-2 py-1 text-muted-foreground">
                          {t("settings.debug.message.realtime.logEmpty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      visibleMessages.slice(0, 120).map((entry, index) => {
                        const rowKey = entry.id || `${entry.topic}-${entry.ts}-${index}`;
                        const isExpanded = expandedMessageKey === rowKey;
                        const seqText = typeof entry.seq === "number" ? String(entry.seq) : "-";
                        const replaySuffix = entry.replay ? ` · ${t("settings.debug.realtime.table.replay")}` : "";
                        return (
                          <Fragment key={rowKey}>
                            <TableRow
                              className="cursor-pointer"
                              onClick={() => toggleMessagePayload(rowKey)}
                              title={t("settings.debug.realtime.table.viewPayload")}
                            >
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px] text-muted-foreground" title={new Date(entry.ts).toLocaleString()}>
                                {new Date(entry.ts).toLocaleTimeString()}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={entry.topic}>
                                {entry.topic}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={entry.type || "-"}>
                                {entry.type || "-"}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={`${seqText}${replaySuffix}`}>
                                {seqText}
                                {replaySuffix}
                              </TableCell>
                            </TableRow>
                            {isExpanded ? (
                              <TableRow>
                                <TableCell colSpan={4} className="border-0 px-2 pb-2">
                                  <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted/20 p-2 text-[11px] text-foreground">
                                    {formatPayload(entry.payload)}
                                  </pre>
                                </TableCell>
                              </TableRow>
                            ) : null}
                          </Fragment>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </TabsContent>
  );
}
