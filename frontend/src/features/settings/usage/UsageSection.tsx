import * as React from "react";
import { HelpCircle, RefreshCw } from "lucide-react";

import { useI18n } from "@/shared/i18n";
import { useProviders } from "@/shared/query/providers";
import { useUsageCost, useUsageStatus } from "@/shared/query/usage";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
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
    return "0.0000";
  }
  return (micros / 1_000_000).toFixed(4);
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

export function UsageSection() {
  const { t } = useI18n();
  const { data: providers = [] } = useProviders();
  const [rangeId, setRangeId] = React.useState<string>("24h");
  const [sourceId, setSourceId] = React.useState<string>("all");
  const [tab, setTab] = React.useState<UsageTab>("models");
  const [isManualRefreshing, setIsManualRefreshing] = React.useState(false);
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
    groupBy: ["providerId", "modelName", "category"],
  });

  const modelTotals = modelUsage.data?.totals;
  const totalCostMicros = costUsage.data?.totalCostMicros ?? 0;
  const modelRows = modelUsage.data?.buckets ?? [];
  const costRows = costUsage.data?.lines ?? [];
  const isRefreshing = isManualRefreshing || modelUsage.isFetching || costUsage.isFetching;
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
      await Promise.allSettled([modelUsage.refetch(), costUsage.refetch()]);
    } finally {
      const elapsed = Date.now() - refreshStartAt;
      const minSpinDurationMs = 350;
      if (elapsed < minSpinDurationMs) {
        await new Promise<void>((resolve) => window.setTimeout(resolve, minSpinDurationMs - elapsed));
      }
      setIsManualRefreshing(false);
    }
  }, [costUsage, isManualRefreshing, modelUsage]);

  const hasError = modelUsage.isError || costUsage.isError;
  const errorMessage = modelUsage.error ?? costUsage.error;
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

  return (
    <div className="space-y-6">
      <Card>
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
          <div className="grid gap-3 md:grid-cols-5">
            <Card>
              <CardHeader size="compact" className="space-y-1 pb-1">
                <CardDescription>{t("settings.usage.summary.requests")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{modelTotals?.requests ?? 0}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader size="compact" className="space-y-1 pb-1">
                <CardDescription>{t("settings.usage.summary.units")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{formatTokenUnits(modelTotals?.units ?? 0)}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader size="compact" className="space-y-1 pb-1">
                <CardDescription>{t("settings.usage.summary.promptTokens")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">
                  {formatTokenUnits(modelTotals?.inputTokens ?? 0)}
                </CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader size="compact" className="space-y-1 pb-1">
                <CardDescription>{t("settings.usage.summary.completionTokens")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">
                  {formatTokenUnits(modelTotals?.outputTokens ?? 0)}
                </CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader size="compact" className="space-y-1 pb-1">
                <CardDescription>{t("settings.usage.summary.cost")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{formatCost(totalCostMicros)}</CardTitle>
              </CardHeader>
            </Card>
          </div>
        </CardContent>
      </Card>

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
                      <TableHead className="text-right">
                        {t("settings.usage.summary.completionTokens")}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {modelUsage.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.loading")}
                        </TableCell>
                      </TableRow>
                    ) : modelRows.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
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
                                  <span>{formatTokenUnits(row.outputTokens)}</span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom">
                                  {formatTokenExact(row.outputTokens)}
                                </TooltipContent>
                              </Tooltip>
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
                      <TableHead className="text-right">{t("settings.usage.summary.requests")}</TableHead>
                      <TableHead className="text-right">{t("settings.usage.summary.cost")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {costUsage.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                          {t("settings.usage.loading")}
                        </TableCell>
                      </TableRow>
                    ) : costRows.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
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
                            <TableCell>{row.category || t("settings.usage.labels.unknown")}</TableCell>
                            <TableCell className="text-right tabular-nums">{row.requests}</TableCell>
                            <TableCell className="text-right tabular-nums">{formatCost(row.costMicros)}</TableCell>
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
