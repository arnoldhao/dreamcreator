import * as React from "react";
import { Events } from "@wailsio/runtime";
import { Check, ChevronsUpDown, X } from "lucide-react";

import { cn } from "@/lib/utils";
import type { Assistant } from "@/shared/store/assistant";
import { useI18n } from "@/shared/i18n";
import { useGatewayTools } from "@/shared/query/tools";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import { useSkillsCatalog } from "@/shared/query/skills";
import { useMemorySummary } from "@/shared/query/memory";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { parseModelRef, resolveGatewayCoreReadiness } from "@/features/setup/readiness";

import { Assistant3DAvatar } from "./Assistant3DAvatar";
import { AssistantEmojiPicker } from "./AssistantEmojiPicker";
import type { AssistantParameterTab } from "./AssistantParametersPanel";
import type { GatewayCharacterTab } from "./GatewayCharacterPanel";

interface GatewayStatusPanelProps {
  assistant: Assistant;
  assistants: Assistant[];
  onSelectAssistant: (id: string) => void;
  onOpenAssistant: (tab?: AssistantParameterTab) => void;
  onOpenCharacter: (tab?: GatewayCharacterTab) => void;
}

interface GatewayStatusMetricItem {
  key: string;
  label: string;
  value: string;
  title?: string;
  dense?: boolean;
}

interface GatewayReadinessItem {
  key: string;
  label: string;
  readyLabel: string;
  badgeLabel: string;
  action: () => void;
  actionLabel: string;
}

const baseColumnsPerRow = 3;
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

function buildRowGridStyle(columns: number): React.CSSProperties {
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

export function GatewayStatusPanel({
  assistant,
  assistants,
  onSelectAssistant,
  onOpenAssistant,
}: GatewayStatusPanelProps) {
  const { t } = useI18n();
  const gridHostRef = React.useRef<HTMLDivElement | null>(null);
  const [maxColumns, setMaxColumns] = React.useState(baseColumnsPerRow);
  const { data: gatewayTools = [] } = useGatewayTools();
  const { data: skillsCatalog = [] } = useSkillsCatalog();
  const { data: providersWithModels = [], isLoading: providersLoading } = useEnabledProvidersWithModels();
  const assistantMemorySummary = useMemorySummary(assistant.id);

  const gatewayReadiness = React.useMemo(
    () =>
      resolveGatewayCoreReadiness({
        assistant,
        providersWithModels,
        checking: providersLoading,
        includeGatewayDisabled: false,
        requireAssistantEnabled: false,
      }),
    [assistant, providersLoading, providersWithModels]
  );
  const missing = React.useMemo(
    () => gatewayReadiness.issues.filter((item) => item === "providers.models" || item === "model.agent.primary"),
    [gatewayReadiness.issues]
  );
  const missingSet = React.useMemo(() => new Set<string>(missing), [missing]);
  const ready = missing.length === 0;

  const readinessItems = React.useMemo<GatewayReadinessItem[]>(
    () => [
      {
        key: "providers.models",
        badgeLabel: t("app.settings.title.provider"),
        label: t("settings.gateway.readiness.providers"),
        readyLabel: t("settings.gateway.readiness.providersReady"),
        action: () => Events.Emit("settings:navigate", "provider"),
        actionLabel: t("settings.gateway.readiness.openProviders"),
      },
      {
        key: "model.agent.primary",
        badgeLabel: t("settings.gateway.statusCard.agentModel"),
        label: t("settings.gateway.readiness.agentModel"),
        readyLabel: t("settings.gateway.readiness.agentModelReady"),
        action: () => onOpenAssistant("models"),
        actionLabel: t("settings.gateway.readiness.openModels"),
      },
    ],
    [onOpenAssistant, t]
  );
  const agentModelLabel = React.useMemo(() => {
    const parsed = parseModelRef(assistant.model?.agent?.primary ?? "");
    const providerId = parsed.providerId.trim();
    const modelName = parsed.modelName.trim();
    if (!providerId) {
      return modelName;
    }
    const providerGroup = providersWithModels.find(
      (item) => item.provider.id.trim().toLowerCase() === providerId.toLowerCase()
    );
    const providerLabel = providerGroup?.provider.name?.trim() || providerId;
    const modelLabel =
      providerGroup?.models.find((item) => item.name.trim().toLowerCase() === modelName.toLowerCase())?.displayName?.trim() ||
      modelName;
    return [providerLabel, modelLabel].filter(Boolean).join(" / ");
  }, [assistant.model?.agent?.primary, providersWithModels]);

  const totalTools = gatewayTools.length || (assistant.tools?.items?.length ?? 0);
  const enabledTools =
    (assistant.tools?.items?.length ?? 0) === 0
      ? totalTools
      : (assistant.tools?.items ?? []).filter((item) => item.enabled).length;
  const totalSkills = skillsCatalog.length;
  const skillsInjectionEnabled = (assistant.skills?.mode?.trim().toLowerCase() ?? "on") !== "off";
  const enabledSkills = skillsInjectionEnabled ? totalSkills : 0;
  const memoryCount = assistantMemorySummary.data?.totalMemories ?? 0;
  const metricItems = React.useMemo<GatewayStatusMetricItem[]>(
    () => [
      {
        key: "tools",
        label: t("settings.gateway.statusCard.tools"),
        value: `${enabledTools}/${totalTools}`,
      },
      {
        key: "skills",
        label: t("settings.gateway.statusCard.skills"),
        value: `${enabledSkills}/${totalSkills}`,
      },
      {
        key: "memories",
        label: t("settings.gateway.statusCard.memories"),
        value: String(memoryCount),
      },
      {
        key: "agentModel",
        label: t("settings.gateway.statusCard.agentModel"),
        value: agentModelLabel || "-",
        title: agentModelLabel || "-",
        dense: true,
      },
    ],
    [agentModelLabel, enabledSkills, enabledTools, memoryCount, t, totalSkills, totalTools]
  );

  React.useEffect(() => {
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

  const rows = React.useMemo(() => distributeRowsEvenly(metricItems, maxColumns), [maxColumns, metricItems]);
  const sharedColumns = React.useMemo(() => {
    let maxRowSize = 1;
    for (const row of rows) {
      if (row.length > maxRowSize) {
        maxRowSize = row.length;
      }
    }
    return maxRowSize;
  }, [rows]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <Card className="flex min-h-0 flex-1 flex-col border-0 bg-transparent shadow-none">
        <CardContent className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 px-[var(--app-sidebar-padding)] pb-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)] text-center">
          <div className="w-full max-w-[min(260px,40vh)]">
            <Assistant3DAvatar assistant={assistant} className="aspect-square w-full" iconClassName="h-6 w-6" />
          </div>
          <div className="flex flex-col items-center gap-2">
            <div className="flex items-center gap-2">
              <AssistantEmojiPicker assistant={assistant} emojiClassName="text-2xl" />
              <span className="text-lg font-semibold uppercase tracking-wide">
                {assistant.identity?.name}
              </span>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    size="compactIcon"
                    className="h-7 w-7 rounded-full"
                    aria-label={t("settings.gateway.action.switch")}
                  >
                    <ChevronsUpDown className="h-3.5 w-3.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  {assistants.map((item) => {
                    const emoji = item.identity?.emoji?.trim() || "🙂";
                    return (
                      <DropdownMenuCheckboxItem
                        key={item.id}
                        checked={item.id === assistant.id}
                        onSelect={(event) => {
                          event.preventDefault();
                          if (item.id !== assistant.id) {
                            onSelectAssistant(item.id);
                          }
                        }}
                        className="gap-2"
                      >
                        <span className="text-base">{emoji}</span>
                        <span className="min-w-0 flex-1 truncate">{item.identity?.name}</span>
                      </DropdownMenuCheckboxItem>
                    );
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <div className="text-xs text-muted-foreground">
              {assistant.identity?.creature || t("settings.gateway.emptyDescription")}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="w-full border bg-card">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <CardTitle className="text-sm font-medium leading-none tracking-normal">
              {ready
                ? t("settings.gateway.readiness.readyTitle")
                : t("settings.gateway.readiness.incompleteTitle")}
            </CardTitle>
            <div className="flex flex-wrap justify-end gap-2">
              {readinessItems.map((item) => {
                const isReady = !missingSet.has(item.key);
                return (
                  <Badge
                    key={item.key}
                    variant="outline"
                    className={cn(
                      "gap-1.5 border-transparent bg-transparent px-1 py-0.5 text-foreground shadow-none"
                    )}
                    title={item.badgeLabel}
                  >
                    <span
                      className={cn(
                        "inline-flex h-5 w-5 items-center justify-center rounded-full",
                        isReady
                          ? "bg-emerald-500/15 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400"
                          : "bg-destructive/15 text-destructive"
                      )}
                    >
                      {isReady ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                    </span>
                    <span className="text-xs text-foreground">{item.badgeLabel}</span>
                  </Badge>
                );
              })}
            </div>
          </div>
        </CardHeader>
        <CardContent size="compact" className="pt-0">
          {ready ? (
            <div ref={gridHostRef} className="space-y-3">
              {rows.map((row, rowIndex) => (
                <div key={`gateway-status-row-${rowIndex}`} className="grid gap-3" style={buildRowGridStyle(sharedColumns)}>
                  {row.map((item) => (
                    <Card key={item.key}>
                      <CardHeader size="compact" className="flex min-h-[5.25rem] flex-col pb-1">
                        <CardDescription className="truncate" title={item.label}>
                          {item.label}
                        </CardDescription>
                        {item.key === "agentModel" ? (
                          <TooltipProvider delayDuration={0}>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <CardTitle className="mt-auto truncate text-sm font-medium leading-5 tracking-normal">
                                  {item.value}
                                </CardTitle>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="max-w-[28rem] break-words">
                                {item.title || item.value}
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          <CardTitle
                            className={
                              item.dense
                                ? "mt-auto break-words text-sm font-medium leading-5 tracking-normal"
                                : "mt-auto break-words text-2xl tabular-nums tracking-tight"
                            }
                            title={item.title}
                          >
                            {item.value}
                          </CardTitle>
                        )}
                      </CardHeader>
                    </Card>
                  ))}
                </div>
              ))}
            </div>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              {readinessItems.map((item) => {
                const isReady = !missingSet.has(item.key);
                const statusLabel = isReady ? item.readyLabel : item.label;
                return (
                  <Card key={item.key}>
                    <CardHeader size="compact" className="pb-1">
                      <CardTitle className="flex items-center gap-2 text-sm font-medium leading-5 tracking-normal">
                        <span
                          className={cn(
                            "inline-flex h-5 w-5 items-center justify-center rounded-full",
                            isReady
                              ? "bg-emerald-500/15 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400"
                              : "bg-destructive/15 text-destructive"
                          )}
                        >
                          {isReady ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                        </span>
                        <span className="truncate" title={statusLabel}>
                          {statusLabel}
                        </span>
                      </CardTitle>
                    </CardHeader>
                    <CardContent size="compact" className="pt-0">
                      <Button type="button" size="compact" variant="outline" onClick={item.action}>
                        {item.actionLabel}
                      </Button>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
