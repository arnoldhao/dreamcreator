import * as React from "react";
import { HelpCircle, Loader2, Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { resolveNoticeTitle, type Notice } from "@/shared/contracts/notice";
import { Input } from "@/shared/ui/input";
import type { HeartbeatLastStatus } from "@/shared/query/heartbeat";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import {
  controlClassName,
  formatHeartbeatEvery,
  panelClassName,
  parseHeartbeatEveryMinutes,
  rowClassName,
  rowLabelClassName,
  type HeartbeatSpecDraft,
  type HeartbeatSpecItemDraft,
} from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayHeartbeatPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  heartbeatSessionOptions: Array<{ value: string; label: string }>;
  heartbeatTab: "general" | "runtime" | "checklist";
  onHeartbeatTabChange: (value: "general" | "runtime" | "checklist") => void;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
  heartbeatSpecDraft: HeartbeatSpecDraft;
  setHeartbeatSpecDraft: React.Dispatch<React.SetStateAction<HeartbeatSpecDraft>>;
  heartbeatSpecVersion: number;
  heartbeatSpecUpdatedAt: string;
  heartbeatSpecSaving: boolean;
  heartbeatTriggerPending: boolean;
  heartbeatTriggerFeedback: { intent: "success" | "warning" | "info"; message: string } | null;
  latestHeartbeatStatus: HeartbeatLastStatus | null;
  latestHeartbeatNotice: Notice | null;
  onOpenNotificationCenter: () => void;
  onUpdateHeartbeatSpecItem: (index: number, patch: Partial<HeartbeatSpecItemDraft>) => void;
  onTriggerHeartbeat: () => void | Promise<void>;
  onClearHeartbeatSpec: () => void;
  onSaveHeartbeatSpec: () => void;
}

const heartbeatEveryOptions = ["5m", "15m", "30m", "60m", "120m", "360m", "720m", "1440m"] as const;
const activeHourOptions = Array.from({ length: 12 }, (_, index) => String(index + 1).padStart(2, "0"));
const activeMinuteOptions = Array.from({ length: 60 }, (_, index) => String(index).padStart(2, "0"));
const defaultActiveStart = "00:00";
const defaultActiveEnd = "23:59";

type TimeMeridiem = "am" | "pm";

type ActiveTimeParts = {
  meridiem: TimeMeridiem;
  hour12: string;
  minute: string;
};

const clampTwoDigits = (value: number, min: number, max: number) =>
  String(Math.min(max, Math.max(min, value))).padStart(2, "0");

const parseActiveTimeParts = (value: string, fallback: string): ActiveTimeParts => {
  const trimmed = value.trim();
  const source = /^(\d{1,2}):(\d{2})$/.test(trimmed) ? trimmed : fallback;
  const matched = source.match(/^(\d{1,2}):(\d{2})$/);
  if (!matched) {
    return { meridiem: "am", hour12: "12", minute: "00" };
  }
  let hour24 = Number(matched[1]);
  let minute = Number(matched[2]);
  if (!Number.isFinite(hour24)) {
    hour24 = 0;
  }
  if (!Number.isFinite(minute)) {
    minute = 0;
  }
  hour24 = Math.min(23, Math.max(0, hour24));
  minute = Math.min(59, Math.max(0, minute));
  const meridiem: TimeMeridiem = hour24 >= 12 ? "pm" : "am";
  const normalizedHour12 = hour24 % 12 === 0 ? 12 : hour24 % 12;
  return {
    meridiem,
    hour12: String(normalizedHour12).padStart(2, "0"),
    minute: clampTwoDigits(minute, 0, 59),
  };
};

const composeActiveTime = (parts: ActiveTimeParts) => {
  const normalizedHour12 = Number(parts.hour12);
  const normalizedMinute = Number(parts.minute);
  const safeHour12 = Number.isFinite(normalizedHour12) ? Math.min(12, Math.max(1, normalizedHour12)) : 12;
  const safeMinute = Number.isFinite(normalizedMinute) ? Math.min(59, Math.max(0, normalizedMinute)) : 0;
  const hour24 = safeHour12 % 12 + (parts.meridiem === "pm" ? 12 : 0);
  return `${String(hour24).padStart(2, "0")}:${String(safeMinute).padStart(2, "0")}`;
};

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

export function GatewayHeartbeatPanel({
  t,
  gateway,
  isDisabled,
  heartbeatSessionOptions,
  heartbeatTab,
  onHeartbeatTabChange,
  updateGateway,
  heartbeatSpecDraft,
  setHeartbeatSpecDraft,
  heartbeatSpecVersion,
  heartbeatSpecUpdatedAt,
  heartbeatSpecSaving,
  heartbeatTriggerPending,
  heartbeatTriggerFeedback,
  latestHeartbeatStatus,
  latestHeartbeatNotice,
  onOpenNotificationCenter,
  onUpdateHeartbeatSpecItem,
  onTriggerHeartbeat,
  onClearHeartbeatSpec,
  onSaveHeartbeatSpec,
}: GatewayHeartbeatPanelProps) {
  const heartbeatEvery = formatHeartbeatEvery(gateway?.heartbeat.periodic);
  const heartbeatEveryValue = React.useMemo(() => {
    const normalized = heartbeatEvery.trim();
    return heartbeatEveryOptions.includes(normalized as (typeof heartbeatEveryOptions)[number])
      ? normalized
      : "30m";
  }, [heartbeatEvery]);
  const latestHeartbeatStatusLabel = React.useMemo(() => {
    switch ((latestHeartbeatStatus?.status ?? "").trim().toLowerCase()) {
      case "ok-token":
      case "ok-empty":
        return t("settings.gateway.detailsPanel.heartbeat.lastStatus.ok");
      case "sent":
        return t("settings.gateway.detailsPanel.heartbeat.lastStatus.alert");
      case "failed":
        return t("settings.gateway.detailsPanel.heartbeat.lastStatus.failed");
      case "skipped":
        return t("settings.gateway.detailsPanel.heartbeat.lastStatus.skipped");
      default:
        return t("settings.gateway.detailsPanel.heartbeat.lastStatus.idle");
    }
  }, [latestHeartbeatStatus?.status, t]);
  const latestHeartbeatStatusTime = latestHeartbeatStatus?.createdAt?.trim() || "-";
  const latestHeartbeatNoticeTitle = latestHeartbeatNotice
    ? resolveNoticeTitle(latestHeartbeatNotice, t)
    : t("settings.gateway.detailsPanel.heartbeat.lastNotice.empty");
  const latestHeartbeatNoticeTime =
    latestHeartbeatNotice?.lastOccurredAt?.trim() ||
    latestHeartbeatNotice?.createdAt?.trim() ||
    "-";

  const runSessionValue = gateway?.heartbeat.runSession ?? "";
  const hasRunSessionOption = heartbeatSessionOptions.some(
    (option) => option.value === runSessionValue
  );
  const runtimeFieldDisabled = !gateway;
  const activeStartValue = gateway?.heartbeat.activeHours.start?.trim() || defaultActiveStart;
  const activeEndValue = gateway?.heartbeat.activeHours.end?.trim() || defaultActiveEnd;
  const [activeStartParts, setActiveStartParts] = React.useState<ActiveTimeParts>(() =>
    parseActiveTimeParts(activeStartValue, defaultActiveStart)
  );
  const [activeEndParts, setActiveEndParts] = React.useState<ActiveTimeParts>(() =>
    parseActiveTimeParts(activeEndValue, defaultActiveEnd)
  );
  React.useEffect(() => {
    setActiveStartParts(parseActiveTimeParts(activeStartValue, defaultActiveStart));
  }, [activeStartValue]);
  React.useEffect(() => {
    setActiveEndParts(parseActiveTimeParts(activeEndValue, defaultActiveEnd));
  }, [activeEndValue]);

  const updateActiveStartParts = React.useCallback(
    (patch: Partial<ActiveTimeParts>) => {
      setActiveStartParts((previous) => {
        const next = { ...previous, ...patch };
        updateGateway({ heartbeat: { activeHours: { start: composeActiveTime(next) } } });
        return next;
      });
    },
    [updateGateway]
  );
  const updateActiveEndParts = React.useCallback(
    (patch: Partial<ActiveTimeParts>) => {
      setActiveEndParts((previous) => {
        const next = { ...previous, ...patch };
        updateGateway({ heartbeat: { activeHours: { end: composeActiveTime(next) } } });
        return next;
      });
    },
    [updateGateway]
  );

  const systemTimezone = React.useMemo(() => {
    try {
      return Intl.DateTimeFormat().resolvedOptions().timeZone?.trim() ?? "";
    } catch {
      return "";
    }
  }, []);
  const timezoneOptions = React.useMemo(() => {
    const supportedValuesOf = (Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supportedValuesOf !== "function") {
      return [] as string[];
    }
    try {
      return supportedValuesOf("timeZone").slice().sort((left, right) => left.localeCompare(right));
    } catch {
      return [] as string[];
    }
  }, []);
  const activeTimezoneValue = gateway?.heartbeat.activeHours.timezone ?? "";
  const autoTimezoneBootstrappedRef = React.useRef(false);
  React.useEffect(() => {
    if (autoTimezoneBootstrappedRef.current || !gateway) {
      return;
    }
    const normalizedCurrent = activeTimezoneValue.trim();
    if (normalizedCurrent !== "") {
      autoTimezoneBootstrappedRef.current = true;
      return;
    }
    const normalizedSystemTimezone = systemTimezone.trim();
    if (normalizedSystemTimezone === "") {
      return;
    }
    autoTimezoneBootstrappedRef.current = true;
    updateGateway({ heartbeat: { activeHours: { timezone: normalizedSystemTimezone } } });
  }, [activeTimezoneValue, gateway, systemTimezone, updateGateway]);
  const heartbeatTimezoneOptions = React.useMemo(() => {
    const current = activeTimezoneValue.trim();
    if (!current) {
      const fallback = systemTimezone.trim();
      if (fallback !== "" && !timezoneOptions.includes(fallback)) {
        return [fallback, ...timezoneOptions];
      }
      return timezoneOptions;
    }
    if (timezoneOptions.includes(current)) {
      return timezoneOptions;
    }
    return [current, ...timezoneOptions];
  }, [activeTimezoneValue, systemTimezone, timezoneOptions]);

  const renderHeartbeatGeneralPanel = () =>
    renderRows([
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.heartbeat.enabled"),
          t("settings.gateway.detailsPanel.heartbeat.enabledHelp")
        )}
        <Switch
          checked={gateway?.heartbeat.enabled ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) => updateGateway({ heartbeat: { enabled: value } })}
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.heartbeat.periodicEnabled"),
          t("settings.gateway.detailsPanel.heartbeat.periodicEnabledHelp")
        )}
        <Switch
          checked={(gateway?.heartbeat.enabled ?? false) && (gateway?.heartbeat.periodic.enabled ?? false)}
          disabled={isDisabled}
          onCheckedChange={(value) => {
            if (value) {
              updateGateway({ heartbeat: { enabled: true, periodic: { enabled: true } } });
              return;
            }
            updateGateway({ heartbeat: { periodic: { enabled: false } } });
          }}
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.heartbeat.every"),
          t("settings.gateway.detailsPanel.heartbeat.everyHelp")
        )}
        <Select
          value={heartbeatEveryValue}
          className={controlClassName}
          disabled={isDisabled || !(gateway?.heartbeat.periodic.enabled ?? false)}
          onChange={(event) => {
            const value = event.target.value;
            updateGateway({
              heartbeat: {
                periodic: {
                  every: value,
                },
                every: value,
                everyMinutes: parseHeartbeatEveryMinutes(value),
              },
            });
          }}
        >
          {heartbeatEveryOptions.map((value) => (
            <option key={`heartbeat-every-${value}`} value={value}>
              {value}
            </option>
          ))}
        </Select>
      </div>,
      <div className="rounded-xl border border-border/70 bg-background/60 px-4 py-3">
        <div className="flex items-center justify-between gap-3">
          <div className="space-y-1">
            <div className="text-sm font-medium text-foreground">
              {t("settings.gateway.detailsPanel.heartbeat.notificationCenter")}
            </div>
            <div className="text-xs text-muted-foreground">
              {t("settings.gateway.detailsPanel.heartbeat.notificationCenterHelp")}
            </div>
          </div>
          <Button type="button" variant="secondary" size="compact" onClick={onOpenNotificationCenter}>
            {t("settings.gateway.detailsPanel.heartbeat.openNotifications")}
          </Button>
        </div>
        <div className="mt-3 grid gap-3 text-xs text-muted-foreground sm:grid-cols-2">
          <div className="space-y-1">
            <div className="font-medium text-foreground">
              {t("settings.gateway.detailsPanel.heartbeat.lastStatus.label")}
            </div>
            <div>{latestHeartbeatStatusLabel}</div>
            <div>{latestHeartbeatStatusTime}</div>
          </div>
          <div className="space-y-1">
            <div className="font-medium text-foreground">
              {t("settings.gateway.detailsPanel.heartbeat.lastNotice.label")}
            </div>
            <div>{latestHeartbeatNoticeTitle}</div>
            <div>{latestHeartbeatNoticeTime}</div>
          </div>
        </div>
      </div>,
    ]);

  const renderHeartbeatRuntimePanel = () =>
    renderRows([
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.heartbeat.promptAppend"),
          t("settings.gateway.detailsPanel.heartbeat.promptAppendHelp")
        )}
        <Input
          value={gateway?.heartbeat.promptAppend ?? ""}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) => updateGateway({ heartbeat: { promptAppend: event.target.value } })}
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(t("settings.gateway.detailsPanel.heartbeat.includeReasoning"))}
        <Switch
          checked={gateway?.heartbeat.includeReasoning ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) => updateGateway({ heartbeat: { includeReasoning: value } })}
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(t("settings.gateway.detailsPanel.heartbeat.suppressToolErrorWarnings"))}
        <Switch
          checked={gateway?.heartbeat.suppressToolErrorWarnings ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) =>
            updateGateway({ heartbeat: { suppressToolErrorWarnings: value } })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(t("settings.gateway.detailsPanel.heartbeat.activeStart"))}
        <div className="flex items-center justify-end gap-1.5">
          <Select
            value={activeStartParts.meridiem}
            className="h-8 w-[86px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) =>
              updateActiveStartParts({ meridiem: event.target.value === "pm" ? "pm" : "am" })
            }
          >
            <option value="am">{t("settings.gateway.detailsPanel.heartbeat.timeMeridiemAm")}</option>
            <option value="pm">{t("settings.gateway.detailsPanel.heartbeat.timeMeridiemPm")}</option>
          </Select>
          <Select
            value={activeStartParts.hour12}
            className="h-8 w-[72px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) => updateActiveStartParts({ hour12: event.target.value })}
          >
            {activeHourOptions.map((hour) => (
              <option key={`active-start-hour-${hour}`} value={hour}>
                {hour}
              </option>
            ))}
          </Select>
          <Select
            value={activeStartParts.minute}
            className="h-8 w-[72px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) => updateActiveStartParts({ minute: event.target.value })}
          >
            {activeMinuteOptions.map((minute) => (
              <option key={`active-start-minute-${minute}`} value={minute}>
                {minute}
              </option>
            ))}
          </Select>
        </div>
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(t("settings.gateway.detailsPanel.heartbeat.activeEnd"))}
        <div className="flex items-center justify-end gap-1.5">
          <Select
            value={activeEndParts.meridiem}
            className="h-8 w-[86px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) =>
              updateActiveEndParts({ meridiem: event.target.value === "pm" ? "pm" : "am" })
            }
          >
            <option value="am">{t("settings.gateway.detailsPanel.heartbeat.timeMeridiemAm")}</option>
            <option value="pm">{t("settings.gateway.detailsPanel.heartbeat.timeMeridiemPm")}</option>
          </Select>
          <Select
            value={activeEndParts.hour12}
            className="h-8 w-[72px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) => updateActiveEndParts({ hour12: event.target.value })}
          >
            {activeHourOptions.map((hour) => (
              <option key={`active-end-hour-${hour}`} value={hour}>
                {hour}
              </option>
            ))}
          </Select>
          <Select
            value={activeEndParts.minute}
            className="h-8 w-[72px] text-xs"
            disabled={runtimeFieldDisabled}
            onChange={(event) => updateActiveEndParts({ minute: event.target.value })}
          >
            {activeMinuteOptions.map((minute) => (
              <option key={`active-end-minute-${minute}`} value={minute}>
                {minute}
              </option>
            ))}
          </Select>
        </div>
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(t("settings.gateway.detailsPanel.heartbeat.activeTimezone"))}
        {heartbeatTimezoneOptions.length > 0 ? (
          <Select
            value={activeTimezoneValue || systemTimezone}
            className={controlClassName}
            disabled={runtimeFieldDisabled}
            onChange={(event) =>
              updateGateway({ heartbeat: { activeHours: { timezone: event.target.value.trim() } } })
            }
          >
            <option value="">{t("settings.gateway.detailsPanel.heartbeat.activeTimezonePlaceholder")}</option>
            {heartbeatTimezoneOptions.map((timezone) => (
              <option key={timezone} value={timezone}>
                {timezone}
              </option>
            ))}
          </Select>
        ) : (
          <Input
            value={activeTimezoneValue}
            size="compact"
            className={controlClassName}
            disabled={runtimeFieldDisabled}
            onChange={(event) =>
              updateGateway({ heartbeat: { activeHours: { timezone: event.target.value } } })
            }
          />
        )}
      </div>,
    ]);

  const renderChecklistPanel = () => (
    <div className="space-y-4">
      <div className="rounded-xl border border-border/70 bg-background/60 px-4 py-3">
        <div className={rowClassName}>
          {renderRowLabel(
            t("settings.gateway.detailsPanel.heartbeat.runSession"),
            t("settings.gateway.detailsPanel.heartbeat.runSessionHelp")
          )}
          <Select
            value={runSessionValue}
            className={controlClassName}
            disabled={isDisabled}
            onChange={(event) =>
              updateGateway({ heartbeat: { runSession: event.target.value.trim() } })
            }
          >
            <option value="">
              {t("settings.gateway.detailsPanel.heartbeat.runSessionSelectPlaceholder")}
            </option>
            {heartbeatSessionOptions.map((option) => (
              <option
                key={
                  option.value.trim() === ""
                    ? "__heartbeat_run_session_empty__"
                    : `heartbeat-session-${option.value}`
                }
                value={option.value}
              >
                {option.label}
              </option>
            ))}
            {runSessionValue.trim() !== "" && !hasRunSessionOption ? (
              <option value={runSessionValue}>{runSessionValue}</option>
            ) : null}
          </Select>
        </div>

        <Separator className="my-3" />

        <div className="flex items-center justify-between gap-3">
          <div className="text-sm font-medium text-foreground">
            {t("settings.gateway.detailsPanel.heartbeat.triggerLabel")}
          </div>
          <Button
            type="button"
            variant="secondary"
            size="compact"
            disabled={heartbeatTriggerPending}
            onClick={onTriggerHeartbeat}
          >
            {heartbeatTriggerPending ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : null}
            {t("settings.gateway.detailsPanel.heartbeat.trigger")}
          </Button>
        </div>
        {heartbeatTriggerFeedback ? (
          <div
            className={`mt-3 rounded-md border px-3 py-2 text-xs ${
              heartbeatTriggerFeedback.intent === "success"
                ? "border-emerald-300/60 text-emerald-700 dark:text-emerald-300"
                : heartbeatTriggerFeedback.intent === "info"
                  ? "border-blue-300/60 text-blue-700 dark:text-blue-300"
                  : "border-amber-300/60 text-amber-700 dark:text-amber-300"
            }`}
          >
            {heartbeatTriggerFeedback.message}
          </div>
        ) : null}
      </div>

      <Card className="border-border/70">
        <CardContent className="space-y-4 p-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-2">
              <div className="text-xs font-medium text-muted-foreground">
                {t("settings.gateway.detailsPanel.heartbeat.spec.items")}
              </div>
              <Button
                type="button"
                variant="secondary"
                size="compact"
                disabled={isDisabled || heartbeatSpecSaving}
                onClick={() =>
                  setHeartbeatSpecDraft((previous) => ({
                    ...previous,
                    items: [
                      ...previous.items,
                      {
                        id: `item-${Date.now()}`,
                        text: "",
                        done: false,
                        priority: "",
                      },
                    ],
                  }))
                }
              >
                <Plus className="mr-1.5 h-4 w-4" />
                {t("settings.gateway.detailsPanel.heartbeat.spec.addItem")}
              </Button>
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("settings.gateway.detailsPanel.heartbeat.spec.item")}</TableHead>
                  <TableHead>{t("settings.gateway.detailsPanel.heartbeat.spec.done")}</TableHead>
                  <TableHead>{t("settings.gateway.detailsPanel.heartbeat.spec.priority")}</TableHead>
                  <TableHead className="w-[48px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {heartbeatSpecDraft.items.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-xs text-muted-foreground">
                      {t("settings.gateway.detailsPanel.heartbeat.spec.itemsEmpty")}
                    </TableCell>
                  </TableRow>
                ) : (
                  heartbeatSpecDraft.items.map((item, index) => (
                    <TableRow key={item.id || index}>
                      <TableCell>
                        <Input
                          value={item.text}
                          size="compact"
                          disabled={isDisabled || heartbeatSpecSaving}
                          onChange={(event) =>
                            onUpdateHeartbeatSpecItem(index, { text: event.target.value })
                          }
                        />
                      </TableCell>
                      <TableCell className="w-[84px]">
                        <Switch
                          checked={item.done}
                          disabled={isDisabled || heartbeatSpecSaving}
                          onCheckedChange={(value) => onUpdateHeartbeatSpecItem(index, { done: value })}
                        />
                      </TableCell>
                      <TableCell className="w-[160px]">
                        <Input
                          value={item.priority}
                          size="compact"
                          disabled={isDisabled || heartbeatSpecSaving}
                          onChange={(event) =>
                            onUpdateHeartbeatSpecItem(index, { priority: event.target.value })
                          }
                        />
                      </TableCell>
                      <TableCell className="w-[48px]">
                        <Button
                          type="button"
                          variant="ghost"
                          size="compactIcon"
                          disabled={isDisabled || heartbeatSpecSaving}
                          onClick={() =>
                            setHeartbeatSpecDraft((previous) => ({
                              ...previous,
                              items: previous.items.filter((_, itemIndex) => itemIndex !== index),
                            }))
                          }
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          <div className="space-y-2">
            <div className="text-xs font-medium text-muted-foreground">
              {t("settings.gateway.detailsPanel.heartbeat.spec.notes")}
            </div>
            <Input
              value={heartbeatSpecDraft.notes}
              size="compact"
              disabled={isDisabled || heartbeatSpecSaving}
              onChange={(event) =>
                setHeartbeatSpecDraft((previous) => ({ ...previous, notes: event.target.value }))
              }
            />
          </div>

          <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span>
                {t("settings.gateway.detailsPanel.heartbeat.spec.version")}: {heartbeatSpecVersion}
              </span>
              <span>
                {t("settings.gateway.detailsPanel.heartbeat.spec.updatedAt")}: {heartbeatSpecUpdatedAt || "-"}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                size="compact"
                disabled={isDisabled || heartbeatSpecSaving}
                onClick={onClearHeartbeatSpec}
              >
                {t("settings.gateway.detailsPanel.heartbeat.spec.clear")}
              </Button>
              <Button
                type="button"
                variant="secondary"
                size="compact"
                disabled={isDisabled || heartbeatSpecSaving}
                onClick={onSaveHeartbeatSpec}
              >
                {heartbeatSpecSaving ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : null}
                {t("settings.gateway.detailsPanel.heartbeat.spec.save")}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );

  return (
    <Tabs
      value={heartbeatTab}
      onValueChange={(value) => onHeartbeatTabChange(value as typeof heartbeatTab)}
      className="space-y-4"
    >
      <div className="flex justify-center">
        <TabsList className="w-fit">
          <TabsTrigger value="general">{t("settings.gateway.detailsPanel.heartbeatTabs.general")}</TabsTrigger>
          <TabsTrigger value="checklist">{t("settings.gateway.detailsPanel.heartbeatTabs.checklist")}</TabsTrigger>
          <TabsTrigger value="runtime">{t("settings.gateway.detailsPanel.heartbeatTabs.runtime")}</TabsTrigger>
        </TabsList>
      </div>

      <TabsContent value="general" className="mt-0">
        {renderHeartbeatGeneralPanel()}
      </TabsContent>
      <TabsContent value="checklist" className="mt-0">
        {renderChecklistPanel()}
      </TabsContent>
      <TabsContent value="runtime" className="mt-0">
        {renderHeartbeatRuntimePanel()}
      </TabsContent>
    </Tabs>
  );
}
