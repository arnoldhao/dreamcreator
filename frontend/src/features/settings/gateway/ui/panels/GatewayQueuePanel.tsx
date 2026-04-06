import * as React from "react";

import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import { controlClassName, panelClassName, rowClassName, rowLabelClassName } from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayQueuePanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
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

export function GatewayQueuePanel({ t, gateway, isDisabled, updateGateway }: GatewayQueuePanelProps) {
  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.queue.globalConcurrency"))}
      <Input
        type="number"
        value={gateway?.queue.globalConcurrency ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ queue: { globalConcurrency: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.queue.sessionConcurrency"))}
      <Input
        type="number"
        value={gateway?.queue.sessionConcurrency ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ queue: { sessionConcurrency: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.queue.laneMain"))}
      <Input
        type="number"
        value={gateway?.queue.lanes?.main ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            queue: { lanes: { main: Number(event.target.value) || 0 } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.queue.laneSubagent"))}
      <Input
        type="number"
        value={gateway?.queue.lanes?.subagent ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            queue: { lanes: { subagent: Number(event.target.value) || 0 } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.queue.laneCron"))}
      <Input
        type="number"
        value={gateway?.queue.lanes?.cron ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            queue: { lanes: { cron: Number(event.target.value) || 0 } },
          })
        }
      />
    </div>,
  ]);
}
