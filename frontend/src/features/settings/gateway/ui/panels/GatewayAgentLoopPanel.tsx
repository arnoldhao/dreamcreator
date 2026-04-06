import * as React from "react";
import { HelpCircle } from "lucide-react";

import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import { controlClassName, panelClassName, rowClassName, rowLabelClassName } from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayAgentLoopPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
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

export function GatewayAgentLoopPanel({ t, gateway, isDisabled, updateGateway }: GatewayAgentLoopPanelProps) {
  const toolLoop = gateway?.runtime.toolLoopDetection;
  const toolLoopCritical = toolLoop?.criticalThreshold ?? toolLoop?.abortThreshold ?? 0;
  const toolLoopGlobal = toolLoop?.globalCircuitBreakerThreshold ?? 0;
  const toolLoopHistory = toolLoop?.historySize ?? toolLoop?.windowSize ?? 0;
  const toolLoopDetectors = toolLoop?.detectors;

  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.enabled"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.enabledHelp")
      )}
      <Switch
        checked={gateway?.runtime.toolLoopDetection.enabled ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) =>
          updateGateway({ runtime: { toolLoopDetection: { enabled: value } } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.warn"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.warnHelp")
      )}
      <Input
        type="number"
        value={gateway?.runtime.toolLoopDetection.warnThreshold ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            runtime: { toolLoopDetection: { warnThreshold: Number(event.target.value) || 0 } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.critical"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.criticalHelp")
      )}
      <Input
        type="number"
        value={toolLoopCritical}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            runtime: {
              toolLoopDetection: {
                criticalThreshold: Number(event.target.value) || 0,
                abortThreshold: Number(event.target.value) || 0,
              },
            },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.global"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.globalHelp")
      )}
      <Input
        type="number"
        value={toolLoopGlobal}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            runtime: {
              toolLoopDetection: { globalCircuitBreakerThreshold: Number(event.target.value) || 0 },
            },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.history"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.historyHelp")
      )}
      <Input
        type="number"
        value={toolLoopHistory}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            runtime: {
              toolLoopDetection: {
                historySize: Number(event.target.value) || 0,
                windowSize: Number(event.target.value) || 0,
              },
            },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorGeneric"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorGenericHelp")
      )}
      <Switch
        checked={toolLoopDetectors?.genericRepeat ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) =>
          updateGateway({
            runtime: { toolLoopDetection: { detectors: { genericRepeat: value } } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorPoll"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorPollHelp")
      )}
      <Switch
        checked={toolLoopDetectors?.knownPollNoProgress ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) =>
          updateGateway({
            runtime: { toolLoopDetection: { detectors: { knownPollNoProgress: value } } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorPingPong"),
        t("settings.gateway.detailsPanel.runtime.toolLoop.detectorPingPongHelp")
      )}
      <Switch
        checked={toolLoopDetectors?.pingPong ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) =>
          updateGateway({
            runtime: { toolLoopDetection: { detectors: { pingPong: value } } },
          })
        }
      />
    </div>,
  ]);
}
