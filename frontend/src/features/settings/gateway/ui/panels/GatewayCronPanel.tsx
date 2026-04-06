import * as React from "react";

import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import {
  CRON_SESSION_RETENTION_OPTIONS,
  controlClassName,
  panelClassName,
  parseCronRunLogMaxMegabytes,
  rowClassName,
  rowLabelClassName,
} from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayCronPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
  onCronEnabledChange: (value: boolean) => void;
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

export function GatewayCronPanel({
  t,
  gateway,
  isDisabled,
  updateGateway,
  onCronEnabledChange,
}: GatewayCronPanelProps) {
  const sessionRetention = (gateway?.cron.sessionRetention ?? "").trim();
  const runLogMaxMegabytes = parseCronRunLogMaxMegabytes(gateway?.cron.runLog.maxBytes);
  const hasCustomSessionRetention =
    sessionRetention !== "" &&
    !CRON_SESSION_RETENTION_OPTIONS.includes(
      sessionRetention as (typeof CRON_SESSION_RETENTION_OPTIONS)[number]
    );

  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.cron.enabled"))}
      <Switch
        checked={gateway?.cron.enabled ?? false}
        disabled={isDisabled}
        onCheckedChange={onCronEnabledChange}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.cron.maxConcurrentRuns"))}
      <Input
        type="number"
        value={gateway?.cron.maxConcurrentRuns ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ cron: { maxConcurrentRuns: Number(event.target.value) || 0 } })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.cron.sessionRetention"))}
      <Select
        value={gateway?.cron.sessionRetention ?? ""}
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) => updateGateway({ cron: { sessionRetention: event.target.value } })}
      >
        {CRON_SESSION_RETENTION_OPTIONS.map((value) => (
          <option key={value} value={value}>
            {t(`settings.gateway.detailsPanel.cron.sessionRetentionOptions.${value}`)}
          </option>
        ))}
        {hasCustomSessionRetention ? <option value={sessionRetention}>{sessionRetention}</option> : null}
      </Select>
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.cron.runLogMaxBytes"))}
      <Input
        type="number"
        value={runLogMaxMegabytes}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({
            cron: { runLog: { maxBytes: `${Math.max(0, Number(event.target.value) || 0)}mb` } },
          })
        }
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.cron.runLogKeepLines"))}
      <Input
        type="number"
        value={gateway?.cron.runLog.keepLines ?? 0}
        size="compact"
        className={controlClassName}
        disabled={isDisabled}
        onChange={(event) =>
          updateGateway({ cron: { runLog: { keepLines: Number(event.target.value) || 0 } } })
        }
      />
    </div>,
  ]);
}
