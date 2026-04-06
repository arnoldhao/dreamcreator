import * as React from "react";
import { HelpCircle } from "lucide-react";

import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import {
  controlClassName,
  normalizeExecPermissionMode,
  panelClassName,
  rowClassName,
  rowLabelClassName,
  type ExecPermissionMode,
} from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayCorePanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  execPermissionMode: ExecPermissionMode;
  permissionModeOptions: Array<{ value: ExecPermissionMode; label: string }>;
  onExecPermissionModeChange: (value: ExecPermissionMode) => void;
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

export function GatewayCorePanel({
  t,
  gateway,
  isDisabled,
  execPermissionMode,
  permissionModeOptions,
  onExecPermissionModeChange,
  updateGateway,
}: GatewayCorePanelProps) {
  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.gateway.controlPlane"),
        t("settings.gateway.detailsPanel.gateway.controlPlaneHelp")
      )}
      <Switch
        checked={gateway?.controlPlaneEnabled ?? false}
        disabled={isDisabled || (gateway?.controlPlaneEnabled ?? false)}
        onCheckedChange={(value) => updateGateway({ controlPlaneEnabled: value })}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.gateway.permission"),
        t("settings.gateway.detailsPanel.gateway.permissionHelp")
      )}
      <Select
        value={execPermissionMode}
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          onExecPermissionModeChange(normalizeExecPermissionMode(event.target.value))
        }
      >
        {permissionModeOptions.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </Select>
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.gateway.sandbox"),
        t("settings.gateway.detailsPanel.gateway.sandboxHelp")
      )}
      <Switch
        checked={gateway?.sandboxEnabled ?? true}
        disabled={isDisabled}
        onCheckedChange={(value) => updateGateway({ sandboxEnabled: value })}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.runtime.maxSteps"),
        t("settings.gateway.detailsPanel.runtime.maxStepsHelp")
      )}
      <Input
        type="number"
        value={gateway?.runtime.maxSteps ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ runtime: { maxSteps: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.channelHealth.minutes"),
        t("settings.gateway.detailsPanel.channelHealth.minutesHelp")
      )}
      <Input
        type="number"
        value={gateway?.channelHealthCheckMinutes ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ channelHealthCheckMinutes: Number(event.target.value) || 0 })
        }
      />
    </div>,
  ]);
}
