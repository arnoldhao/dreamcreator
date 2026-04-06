import * as React from "react";
import { HelpCircle } from "lucide-react";

import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import { controlClassName, panelClassName, rowClassName, rowLabelClassName } from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayContextPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  contextTab: "guard" | "compaction" | "memory";
  onContextTabChange: (value: "guard" | "compaction" | "memory") => void;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
}

const renderRowLabel = (label: string, description?: string) => (
  <div className="flex min-w-0 flex-1 items-center gap-1.5">
    <span className={rowLabelClassName} title={label}>
      {label}
    </span>
    {description ? (
      <TooltipProvider delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
              aria-label={label}
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top" align="start">
            {description}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ) : null}
  </div>
);

const renderRows = (rows: React.ReactNode[]) => (
  <div className={panelClassName}>
    {rows.map((row, index) => (
      <React.Fragment key={index}>
        {row}
        {index < rows.length - 1 ? <Separator /> : null}
      </React.Fragment>
    ))}
  </div>
);

export function GatewayContextPanel({
  t,
  gateway,
  isDisabled,
  contextTab,
  onContextTabChange,
  updateGateway,
}: GatewayContextPanelProps) {
  return (
    <Tabs value={contextTab} onValueChange={(value) => onContextTabChange(value as typeof contextTab)}>
      <div className="flex justify-center">
        <TabsList className="w-fit">
          <TabsTrigger value="guard" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.contextTabs.guard")}
          </TabsTrigger>
          <TabsTrigger value="compaction" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.contextTabs.compaction")}
          </TabsTrigger>
          <TabsTrigger value="memory" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.contextTabs.memory")}
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="guard">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderRows([
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.contextWarn"),
                  t("settings.gateway.detailsPanel.runtime.contextWarnHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.contextWindow.warnTokens ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { contextWindow: { warnTokens: Number(event.target.value) || 0 } },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.contextHard"),
                  t("settings.gateway.detailsPanel.runtime.contextHardHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.contextWindow.hardTokens ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { contextWindow: { hardTokens: Number(event.target.value) || 0 } },
                    })
                  }
                />
              </div>,
            ])}
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="compaction">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderRows([
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMode"),
                  t("settings.gateway.detailsPanel.runtime.compactionModeHelp")
                )}
                <Select
                  value={gateway?.runtime.compaction.mode || "default"}
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({ runtime: { compaction: { mode: event.target.value } } })
                  }
                >
                  <option value="default">
                    {t("settings.gateway.detailsPanel.runtime.compactionModeOptions.default")}
                  </option>
                  <option value="safeguard">
                    {t("settings.gateway.detailsPanel.runtime.compactionModeOptions.safeguard")}
                  </option>
                </Select>
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionReserveTokens"),
                  t("settings.gateway.detailsPanel.runtime.compactionReserveTokensHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.compaction.reserveTokens ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { compaction: { reserveTokens: Number(event.target.value) || 0 } },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionKeepRecent"),
                  t("settings.gateway.detailsPanel.runtime.compactionKeepRecentHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.compaction.keepRecentTokens ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { compaction: { keepRecentTokens: Number(event.target.value) || 0 } },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionReserveFloor"),
                  t("settings.gateway.detailsPanel.runtime.compactionReserveFloorHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.compaction.reserveTokensFloor ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: {
                        compaction: { reserveTokensFloor: Number(event.target.value) || 0 },
                      },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMaxHistoryShare"),
                  t("settings.gateway.detailsPanel.runtime.compactionMaxHistoryShareHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.compaction.maxHistoryShare ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  step="0.1"
                  onChange={(event) =>
                    updateGateway({
                      runtime: { compaction: { maxHistoryShare: Number(event.target.value) || 0 } },
                    })
                  }
                />
              </div>,
            ])}
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="memory">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderRows([
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushEnabled"),
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushEnabledHelp")
                )}
                <Switch
                  checked={gateway?.runtime.compaction.memoryFlush?.enabled ?? false}
                  disabled={isDisabled}
                  onCheckedChange={(value) =>
                    updateGateway({ runtime: { compaction: { memoryFlush: { enabled: value } } } })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushSoft"),
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushSoftHelp")
                )}
                <Input
                  type="number"
                  value={gateway?.runtime.compaction.memoryFlush?.softThresholdTokens ?? 0}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: {
                        compaction: {
                          memoryFlush: { softThresholdTokens: Number(event.target.value) || 0 },
                        },
                      },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushPrompt"),
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushPromptHelp")
                )}
                <Input
                  value={gateway?.runtime.compaction.memoryFlush?.prompt ?? ""}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { compaction: { memoryFlush: { prompt: event.target.value } } },
                    })
                  }
                />
              </div>,
              <div className={rowClassName}>
                {renderRowLabel(
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushSystem"),
                  t("settings.gateway.detailsPanel.runtime.compactionMemoryFlushSystemHelp")
                )}
                <Input
                  value={gateway?.runtime.compaction.memoryFlush?.systemPrompt ?? ""}
                  size="compact"
                  className={controlClassName}
                  disabled={isDisabled}
                  onChange={(event) =>
                    updateGateway({
                      runtime: { compaction: { memoryFlush: { systemPrompt: event.target.value } } },
                    })
                  }
                />
              </div>,
            ])}
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
