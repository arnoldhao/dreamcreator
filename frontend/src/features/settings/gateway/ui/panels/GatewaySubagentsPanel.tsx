import * as React from "react";

import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import { controlClassName, panelClassName, rowClassName, rowLabelClassName } from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewaySubagentsPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  subagentModelOptions: Array<{ value: string; label: string }>;
  subagentThinkingOptions: Array<{ value: string; label: string }>;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
}

const renderRowLabel = (label: string) => (
  <span className={rowLabelClassName} title={label}>
    {label}
  </span>
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

export function GatewaySubagentsPanel({
  t,
  gateway,
  isDisabled,
  subagentModelOptions,
  subagentThinkingOptions,
  updateGateway,
}: GatewaySubagentsPanelProps) {
  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.subagents.maxDepth"))}
      <Input
        type="number"
        value={gateway?.subagents.maxDepth ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ subagents: { maxDepth: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.subagents.maxChildren"))}
      <Input
        type="number"
        value={gateway?.subagents.maxChildren ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ subagents: { maxChildren: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.subagents.maxConcurrent"))}
      <Input
        type="number"
        value={gateway?.subagents.maxConcurrent ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ subagents: { maxConcurrent: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.subagents.model"))}
      <Select
        value={gateway?.subagents.model ?? ""}
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) => updateGateway({ subagents: { model: event.target.value } })}
      >
        {subagentModelOptions.map((option) => (
          <option
            key={option.value === "" ? "__subagent_model_auto__" : option.value}
            value={option.value}
          >
            {option.label}
          </option>
        ))}
      </Select>
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.subagents.thinking"))}
      <Select
        value={gateway?.subagents.thinking ?? ""}
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) => updateGateway({ subagents: { thinking: event.target.value } })}
      >
        {subagentThinkingOptions.map((option) => (
          <option
            key={option.value === "" ? "__subagent_thinking_auto__" : option.value}
            value={option.value}
          >
            {option.label}
          </option>
        ))}
      </Select>
    </div>,
  ]);
}
