import * as React from "react";
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  List,
  RotateCcw,
  SlidersHorizontal,
  Trash2,
} from "lucide-react";

import { Skeleton } from "@/shared/ui/skeleton";
import { canAdoptIncomingDraftSnapshot, shouldSkipDraftPersist } from "@/app/settings/draftSync";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useSettings, useUpdateSettings } from "@/shared/query/settings";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";

import { isRecord } from "../utils/calls-utils";
import {
  buildNextSettingsTools,
  DEFAULT_SKILLS_AUDIT_RETENTION_DAYS,
  parseSkillsAudit,
  parseSkillsAuditConfig,
  resolveSkillsSettingsState,
  SKILLS_AUDIT_RETENTION_DAY_OPTIONS,
  toAuditConfigPayload,
  type SkillsAuditConfig,
  type SkillsAuditRecord,
} from "../utils/skills-settings-utils";
import {
  SKILLS_LINE_TABS_LIST_CLASS,
  SKILLS_LINE_TABS_TRIGGER_CLASS,
  SKILLS_PAGE_CARD_CLASS,
  SKILLS_SELECT_TEXT_CLASS,
} from "./skills-page-styles";

const AUDIT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const;
const AUDIT_PRIMARY_COL_WIDTH_CLASS = "w-[16.666%]";
const AUDIT_SECONDARY_COL_WIDTH_CLASS = "w-[12.5%]";

type SkillsAuditViewTab = "data" | "config";

function cloneAuditConfig(config: SkillsAuditConfig): SkillsAuditConfig {
  return {
    hideUiOperationRecords: config.hideUiOperationRecords !== false,
    retentionDays: normalizeAuditRetentionDays(config.retentionDays),
  };
}

function normalizeAuditRetentionDays(value: number): number {
  if (!Number.isFinite(value)) {
    return DEFAULT_SKILLS_AUDIT_RETENTION_DAYS;
  }
  const normalized = Math.floor(value);
  if (normalized <= 0) {
    return DEFAULT_SKILLS_AUDIT_RETENTION_DAYS;
  }
  if (normalized > 365) {
    return 365;
  }
  return normalized;
}

function formatAuditTime(value: string | undefined, locale?: string): string {
  const trimmed = (value ?? "").trim();
  if (!trimmed) {
    return "-";
  }
  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) {
    return trimmed;
  }
  return parsed.toLocaleString(locale || undefined);
}

function formatAuditRelativeTime(
  value: string | undefined,
  locale?: string,
  justNowLabel = "just now"
): string {
  const trimmed = (value ?? "").trim();
  if (!trimmed) {
    return "-";
  }
  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) {
    return trimmed;
  }
  const now = new Date();
  const diffSeconds = Math.round((parsed.getTime() - now.getTime()) / 1000);
  const absSeconds = Math.abs(diffSeconds);
  if (absSeconds < 5) {
    return justNowLabel;
  }
  const units: Array<{ unit: Intl.RelativeTimeFormatUnit; seconds: number }> = [
    { unit: "year", seconds: 60 * 60 * 24 * 365 },
    { unit: "month", seconds: 60 * 60 * 24 * 30 },
    { unit: "week", seconds: 60 * 60 * 24 * 7 },
    { unit: "day", seconds: 60 * 60 * 24 },
    { unit: "hour", seconds: 60 * 60 },
    { unit: "minute", seconds: 60 },
    { unit: "second", seconds: 1 },
  ];
  const formatter = new Intl.RelativeTimeFormat(locale || undefined, { numeric: "auto" });
  for (const { unit, seconds } of units) {
    if (absSeconds >= seconds || unit === "second") {
      return formatter.format(Math.round(diffSeconds / seconds), unit);
    }
  }
  return formatter.format(diffSeconds, "second");
}

function resolveOutcomeDotClass(ok?: boolean): string {
  if (ok === true) {
    return "bg-emerald-500";
  }
  if (ok === false) {
    return "bg-rose-500";
  }
  return "bg-muted-foreground/50";
}

function resolveAuditSource(record: SkillsAuditRecord): string {
  const source = (record.source ?? "").trim().toLowerCase();
  if (source === "web" || source === "tool_call") {
    return source;
  }
  if (source === "toolcall" || source === "tool-call") {
    return "tool_call";
  }
  if (source === "ui" || source === "frontend") {
    return "web";
  }
  return "unknown";
}

function normalizeAuditInlineText(value: string | undefined): string {
  return (value ?? "").replace(/\s+/g, " ").trim();
}

function resolveAuditRowKey(record: SkillsAuditRecord, index: number): string {
  const timestamp = (record.timestamp ?? "").trim() || "none";
  const action = (record.action ?? "").trim() || "none";
  const skill = (record.skill ?? "").trim() || "none";
  return `${timestamp}-${action}-${skill}-${index}`;
}

function formatTemplate(template: string, values: Record<string, number>): string {
  return Object.entries(values).reduce(
    (output, [key, value]) => output.replace(new RegExp(`\\{${key}\\}`, "g"), String(value)),
    template
  );
}

function AdaptiveAuditTimeValue({
  value,
  language,
  justNowLabel,
}: {
  value?: string;
  language?: string;
  justNowLabel: string;
}) {
  const containerRef = React.useRef<HTMLSpanElement | null>(null);
  const preciseMeasureRef = React.useRef<HTMLSpanElement | null>(null);
  const [useRelativeText, setUseRelativeText] = React.useState(false);

  const preciseText = React.useMemo(() => formatAuditTime(value, language), [language, value]);
  const relativeText = React.useMemo(
    () => formatAuditRelativeTime(value, language, justNowLabel),
    [justNowLabel, language, value]
  );

  const syncOverflowState = React.useCallback(() => {
    const container = containerRef.current;
    const measure = preciseMeasureRef.current;
    if (!container || !measure) {
      return;
    }
    setUseRelativeText(measure.scrollWidth - container.clientWidth > 1);
  }, []);

  React.useLayoutEffect(() => {
    syncOverflowState();
    const container = containerRef.current;
    if (!container || typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(() => {
      syncOverflowState();
    });
    observer.observe(container);
    return () => observer.disconnect();
  }, [syncOverflowState, preciseText]);

  return (
    <span
      ref={containerRef}
      className="relative block min-w-0 flex-1 overflow-hidden whitespace-nowrap text-ellipsis font-mono text-xs text-muted-foreground"
      title={preciseText}
    >
      {useRelativeText ? relativeText : preciseText}
      <span
        ref={preciseMeasureRef}
        className="pointer-events-none absolute invisible whitespace-nowrap font-mono text-xs"
      >
        {preciseText}
      </span>
    </span>
  );
}

export function SkillsAuditTab() {
  const { t, language } = useI18n();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();

  const [viewTab, setViewTab] = React.useState<SkillsAuditViewTab>("data");

  const settingsToolsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.tools;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.tools]);
  const settingsSkillsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.skills;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.skills]);
  const { toolsConfig, skillsConfig } = React.useMemo(
    () => resolveSkillsSettingsState(settingsToolsRaw, settingsSkillsRaw),
    [settingsSkillsRaw, settingsToolsRaw]
  );
  const records = React.useMemo(() => parseSkillsAudit(skillsConfig), [skillsConfig]);
  const auditConfig = React.useMemo(() => parseSkillsAuditConfig(skillsConfig), [skillsConfig]);

  const [actionFilter, setActionFilter] = React.useState("all");
  const [skillFilter, setSkillFilter] = React.useState("all");
  const [resultFilter, setResultFilter] = React.useState<"all" | "ok" | "error">("all");
  const [page, setPage] = React.useState(1);
  const [rowsPerPage, setRowsPerPage] = React.useState<number>(20);
  const [expandedRows, setExpandedRows] = React.useState<Record<string, boolean>>({});
  const [configDraft, setConfigDraft] = React.useState<SkillsAuditConfig>(() => cloneAuditConfig(auditConfig));
  const lastPersistedConfigSignatureRef = React.useRef("");
  const pendingConfigSignatureRef = React.useRef("");

  const normalizedConfigDraft = React.useMemo(
    () => ({
      hideUiOperationRecords: configDraft.hideUiOperationRecords !== false,
      retentionDays: normalizeAuditRetentionDays(configDraft.retentionDays),
    }),
    [configDraft]
  );
  const currentConfigSignature = React.useMemo(() => JSON.stringify(auditConfig), [auditConfig]);
  const draftConfigSignature = React.useMemo(() => JSON.stringify(normalizedConfigDraft), [normalizedConfigDraft]);
  const previousConfigSignatureRef = React.useRef(currentConfigSignature);

  React.useEffect(() => {
    const previousConfigSignature = previousConfigSignatureRef.current;
    previousConfigSignatureRef.current = currentConfigSignature;
    const canAdoptRemote = canAdoptIncomingDraftSnapshot({
      draftSignature: draftConfigSignature,
      currentRemoteSignature: currentConfigSignature,
      previousRemoteSignature: previousConfigSignature,
      lastPersistedSignature: lastPersistedConfigSignatureRef.current,
    });
    if (!canAdoptRemote) {
      return;
    }
    setConfigDraft(cloneAuditConfig(auditConfig));
  }, [auditConfig, currentConfigSignature, draftConfigSignature]);

  const configDirty = React.useMemo(
    () => currentConfigSignature !== draftConfigSignature,
    [currentConfigSignature, draftConfigSignature]
  );

  const persistConfig = React.useCallback(() => {
    const submittedSignature = draftConfigSignature;
    pendingConfigSignatureRef.current = submittedSignature;
    const nextSkillsConfig: Record<string, unknown> = {
      ...skillsConfig,
      auditConfig: toAuditConfigPayload(normalizedConfigDraft),
    };
    const nextToolsConfig: Record<string, unknown> = {
      ...toolsConfig,
    };
    const nextTools = buildNextSettingsTools(nextToolsConfig);
    updateSettings.mutate(
      {
        tools: nextTools,
        skills: nextSkillsConfig,
      },
      {
        onSuccess: () => {
          lastPersistedConfigSignatureRef.current = submittedSignature;
          if (pendingConfigSignatureRef.current === submittedSignature) {
            pendingConfigSignatureRef.current = "";
          }
        },
        onError: (error) => {
          if (pendingConfigSignatureRef.current === submittedSignature) {
            pendingConfigSignatureRef.current = "";
          }
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.calls.skills.audit.config.saveFailed"),
            description: error instanceof Error ? error.message : String(error ?? ""),
          });
        },
      }
    );
  }, [
    draftConfigSignature,
    normalizedConfigDraft,
    skillsConfig,
    t,
    toolsConfig,
    updateSettings,
  ]);

  React.useEffect(() => {
    if (!configDirty) {
      return;
    }
    if (
      shouldSkipDraftPersist({
        draftSignature: draftConfigSignature,
        lastPersistedSignature: lastPersistedConfigSignatureRef.current,
        pendingSubmittedSignature: pendingConfigSignatureRef.current,
      })
    ) {
      return;
    }
    const timer = window.setTimeout(() => {
      if (updateSettings.isPending) {
        return;
      }
      persistConfig();
    }, 500);
    return () => window.clearTimeout(timer);
  }, [configDirty, draftConfigSignature, persistConfig, updateSettings.isPending]);

  React.useEffect(() => {
    if (!configDirty && currentConfigSignature === draftConfigSignature) {
      lastPersistedConfigSignatureRef.current = draftConfigSignature;
      if (pendingConfigSignatureRef.current === draftConfigSignature) {
        pendingConfigSignatureRef.current = "";
      }
    }
  }, [configDirty, currentConfigSignature, draftConfigSignature]);

  const clearAuditLogsNow = React.useCallback(() => {
    const nextSkillsConfig: Record<string, unknown> = {
      ...skillsConfig,
      audit: [],
      auditConfig: toAuditConfigPayload(normalizedConfigDraft),
    };
    const nextToolsConfig: Record<string, unknown> = {
      ...toolsConfig,
    };
    const nextTools = buildNextSettingsTools(nextToolsConfig);
    updateSettings.mutate(
      {
        tools: nextTools,
        skills: nextSkillsConfig,
      },
      {
        onSuccess: () => {
          messageBus.publishToast({
            intent: "success",
            title: t("settings.calls.skills.audit.config.clearDone"),
          });
        },
        onError: (error) => {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.calls.skills.audit.config.clearFailed"),
            description: error instanceof Error ? error.message : String(error ?? ""),
          });
        },
      }
    );
  }, [normalizedConfigDraft, skillsConfig, t, toolsConfig, updateSettings]);

  const clearAuditLogs = React.useCallback(() => {
    if (updateSettings.isPending || records.length === 0) {
      return;
    }
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.calls.skills.audit.config.clearDialogTitle"),
      description: t("settings.calls.skills.audit.config.clearDialogDescription"),
      confirmLabel: t("settings.calls.skills.audit.config.clearNow"),
      cancelLabel: t("common.cancel"),
      onConfirm: () => {
        clearAuditLogsNow();
      },
    });
  }, [clearAuditLogsNow, records.length, t, updateSettings.isPending]);

  const visibleRecords = React.useMemo(() => {
    if (!normalizedConfigDraft.hideUiOperationRecords) {
      return records;
    }
    return records.filter((item) => resolveAuditSource(item) !== "web");
  }, [normalizedConfigDraft.hideUiOperationRecords, records]);

  const actionOptions = React.useMemo(() => {
    const values = Array.from(
      new Set(
        visibleRecords
          .map((item) => (item.action ?? "").trim())
          .filter((value) => value !== "")
      )
    );
    values.sort((left, right) => left.localeCompare(right));
    return values;
  }, [visibleRecords]);

  const skillOptions = React.useMemo(() => {
    const values = Array.from(
      new Set(
        visibleRecords
          .map((item) => (item.skill ?? "").trim())
          .filter((value) => value !== "")
      )
    );
    values.sort((left, right) => left.localeCompare(right));
    return values;
  }, [visibleRecords]);

  const filtered = React.useMemo(() => {
    const actionQuery = actionFilter.trim().toLowerCase();
    const skillQuery = skillFilter.trim().toLowerCase();
    return visibleRecords.filter((item) => {
      if (actionQuery !== "all" && (item.action ?? "").trim().toLowerCase() !== actionQuery) {
        return false;
      }
      if (skillQuery !== "all" && (item.skill ?? "").trim().toLowerCase() !== skillQuery) {
        return false;
      }
      if (resultFilter === "ok" && item.ok !== true) {
        return false;
      }
      if (resultFilter === "error" && item.ok !== false) {
        return false;
      }
      return true;
    });
  }, [actionFilter, resultFilter, skillFilter, visibleRecords]);

  const filteredCount = filtered.length;
  const totalPages = Math.max(1, Math.ceil(filteredCount / rowsPerPage));

  React.useEffect(() => {
    setPage(1);
  }, [actionFilter, skillFilter, resultFilter, normalizedConfigDraft.hideUiOperationRecords]);

  React.useEffect(() => {
    setPage((previous) => Math.min(previous, totalPages));
  }, [totalPages]);

  const pagedRecords = React.useMemo(() => {
    const start = (page - 1) * rowsPerPage;
    return filtered.slice(start, start + rowsPerPage).map((item, offset) => ({
      item,
      index: start + offset,
    }));
  }, [filtered, page, rowsPerPage]);

  const toggleRowExpanded = React.useCallback((rowKey: string) => {
    setExpandedRows((previous) => ({
      ...previous,
      [rowKey]: !previous[rowKey],
    }));
  }, []);

  if (settingsQuery.isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-72 w-full" />
      </div>
    );
  }

  if (settingsQuery.isError) {
    return (
      <Card className={SKILLS_PAGE_CARD_CLASS}>
        <CardContent size="compact" className="flex items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            {t("settings.calls.skills.audit.loadFailed")}
          </p>
          <Button variant="outline" size="compact" onClick={() => void settingsQuery.refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            {t("common.retry")}
          </Button>
        </CardContent>
      </Card>
    );
  }

  const totalRecordsText = formatTemplate(
    t("settings.calls.skills.audit.totalRecords"),
    { count: visibleRecords.length }
  );
  const filteredRecordsText = formatTemplate(
    t("settings.calls.skills.audit.filteredRecords"),
    { count: filteredCount }
  );
  const rowsPerPageTemplate = t("settings.calls.skills.audit.rowsPerPage");
  const pageText = formatTemplate(t("settings.calls.skills.audit.pageOf"), {
    page,
    total: totalPages,
  });
  const justNowLabel = t("library.time.justNow");

  return (
    <Tabs
      value={viewTab}
      onValueChange={(value) => setViewTab(value as SkillsAuditViewTab)}
      className="flex min-h-0 flex-1 flex-col gap-3"
    >
      <div className="flex items-center gap-3">
        <TabsList className={`shrink-0 ${SKILLS_LINE_TABS_LIST_CLASS}`}>
          <TabsTrigger value="data" className={SKILLS_LINE_TABS_TRIGGER_CLASS}>
            <List className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.audit.view.data")}</span>
          </TabsTrigger>
          <TabsTrigger value="config" className={SKILLS_LINE_TABS_TRIGGER_CLASS}>
            <SlidersHorizontal className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.audit.view.config")}</span>
          </TabsTrigger>
        </TabsList>

        {viewTab === "data" ? (
          <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
            <Select
              value={actionFilter}
              onChange={(event) => setActionFilter(event.target.value)}
              className={`w-44 ${SKILLS_SELECT_TEXT_CLASS}`}
            >
              <option value="all">{t("settings.calls.skills.audit.actionAll")}</option>
              {actionOptions.map((action) => (
                <option key={action} value={action}>
                  {action}
                </option>
              ))}
            </Select>
            <Select
              value={skillFilter}
              onChange={(event) => setSkillFilter(event.target.value)}
              className={`w-44 ${SKILLS_SELECT_TEXT_CLASS}`}
            >
              <option value="all">{t("settings.calls.skills.audit.skillAll")}</option>
              {skillOptions.map((skill) => (
                <option key={skill} value={skill}>
                  {skill}
                </option>
              ))}
            </Select>
            <Select
              value={resultFilter}
              onChange={(event) => setResultFilter(event.target.value as "all" | "ok" | "error")}
              className={`w-40 ${SKILLS_SELECT_TEXT_CLASS}`}
            >
              <option value="all">{t("settings.calls.skills.audit.result.all")}</option>
              <option value="ok">{t("settings.calls.skills.audit.result.ok")}</option>
              <option value="error">{t("settings.calls.skills.audit.result.error")}</option>
            </Select>
          </div>
        ) : null}
      </div>

      {viewTab === "data" ? (
        <>
          <div className={SKILLS_PAGE_CARD_CLASS + " border min-h-0 flex-1 overflow-hidden"}>
            <div className="h-full overflow-auto">
              <Table className="text-xs min-w-full table-fixed">
                <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
                  <TableRow>
                    <TableHead className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.time")}
                    </TableHead>
                    <TableHead className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.action")}
                    </TableHead>
                    <TableHead className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.group")}
                    </TableHead>
                    <TableHead className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.skill")}
                    </TableHead>
                    <TableHead className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.source")}
                    </TableHead>
                    <TableHead className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.resultLabel")}
                    </TableHead>
                    <TableHead className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} whitespace-nowrap tracking-wide`}>
                      {t("settings.calls.skills.audit.error")}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody className="select-none">
                  {pagedRecords.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="py-12 text-center text-sm text-muted-foreground">
                        {visibleRecords.length === 0
                          ? t("settings.calls.skills.audit.emptyDescription")
                          : t("settings.calls.skills.audit.emptyFiltered")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    pagedRecords.map(({ item, index }) => {
                      const rowKey = resolveAuditRowKey(item, index);
                      const expanded = expandedRows[rowKey] === true;
                      const errorText = normalizeAuditInlineText(item.error) || "-";
                      const detailLine = [
                        `${t("settings.calls.skills.audit.tool")}: ${item.tool || "-"}`,
                        `${t("settings.calls.skills.audit.action")}: ${item.action || "-"}`,
                        `${t("settings.calls.skills.audit.group")}: ${item.group || "-"}`,
                        `${t("settings.calls.skills.audit.skill")}: ${item.skill || "-"}`,
                        `${t("settings.calls.skills.audit.source")}: ${resolveAuditSource(item)}`,
                        `${t("settings.calls.skills.audit.assistantId")}: ${item.assistantId || "-"}`,
                        `${t("settings.calls.skills.audit.providerId")}: ${item.providerId || "-"}`,
                        `${t("settings.calls.skills.audit.error")}: ${errorText}`,
                      ].join("  |  ");
                      return (
                        <React.Fragment key={rowKey}>
                          <TableRow
                            className="cursor-pointer odd:bg-muted/[0.14] transition-colors hover:bg-muted/40"
                            onClick={() => toggleRowExpanded(rowKey)}
                          >
                            <TableCell className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs text-muted-foreground`}>
                              <div className="flex min-w-0 items-center gap-1.5">
                                {expanded ? (
                                  <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                                ) : (
                                  <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                                )}
                                <AdaptiveAuditTimeValue
                                  value={item.timestamp}
                                  language={language}
                                  justNowLabel={justNowLabel}
                                />
                              </div>
                            </TableCell>
                            <TableCell className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} align-middle overflow-hidden font-mono text-xs`}>
                              <span className="block truncate whitespace-nowrap" title={item.action || "-"}>
                                {item.action || "-"}
                              </span>
                            </TableCell>
                            <TableCell className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs`}>
                              <span className="block truncate whitespace-nowrap" title={item.group ?? "-"}>
                                {item.group ?? "-"}
                              </span>
                            </TableCell>
                            <TableCell className={`${AUDIT_PRIMARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs`}>
                              <span className="block truncate whitespace-nowrap" title={item.skill ?? "-"}>
                                {item.skill ?? "-"}
                              </span>
                            </TableCell>
                            <TableCell className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs text-muted-foreground`}>
                              <span className="block truncate whitespace-nowrap" title={resolveAuditSource(item)}>
                                {resolveAuditSource(item)}
                              </span>
                            </TableCell>
                            <TableCell className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs`}>
                              <Badge
                                variant="ghost"
                                className="inline-flex max-w-full items-center gap-1.5 truncate whitespace-nowrap px-2 text-xs font-medium"
                              >
                                <span className={`h-1.5 w-1.5 rounded-full ${resolveOutcomeDotClass(item.ok)}`} aria-hidden="true" />
                                <span className="truncate whitespace-nowrap text-foreground">
                                  {item.ok === true
                                    ? t("settings.calls.skills.audit.outcome.ok")
                                    : item.ok === false
                                      ? t("settings.calls.skills.audit.outcome.error")
                                      : t("settings.calls.skills.audit.outcome.unknown")}
                                </span>
                              </Badge>
                            </TableCell>
                            <TableCell className={`${AUDIT_SECONDARY_COL_WIDTH_CLASS} align-middle overflow-hidden text-xs text-muted-foreground`}>
                              <div className="flex min-w-0 items-center gap-2 overflow-hidden">
                                {item.errorCode ? (
                                  <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px]">{item.errorCode}</span>
                                ) : null}
                                <span className="block min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
                                  {errorText}
                                </span>
                              </div>
                            </TableCell>
                          </TableRow>
                          {expanded ? (
                            <TableRow className="bg-muted/20">
                              <TableCell colSpan={7} className="py-2">
                                <div className="whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground">
                                  {detailLine}
                                </div>
                              </TableCell>
                            </TableRow>
                          ) : null}
                        </React.Fragment>
                      );
                    })
                  )}
                </TableBody>
              </Table>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
            <div className="text-xs text-muted-foreground">{totalRecordsText}</div>
            <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
              <div className="text-xs text-muted-foreground">{filteredRecordsText}</div>
              <Select
                className={`w-[112px] ${SKILLS_SELECT_TEXT_CLASS}`}
                value={String(rowsPerPage)}
                onChange={(event) => {
                  const next = Number(event.target.value);
                  if (Number.isFinite(next) && next > 0) {
                    setRowsPerPage(next);
                    setPage(1);
                  }
                }}
              >
                {AUDIT_PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>
                    {formatTemplate(rowsPerPageTemplate, { count: size })}
                  </option>
                ))}
              </Select>
              <div className="text-xs text-muted-foreground">{pageText}</div>
              <Button
                variant="outline"
                size="compactIcon"
                onClick={() => setPage(1)}
                disabled={page <= 1}
                aria-label={t("settings.calls.skills.audit.firstPage")}
              >
                <ChevronsLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="compactIcon"
                onClick={() => setPage((previous) => Math.max(1, previous - 1))}
                disabled={page <= 1}
                aria-label={t("settings.calls.skills.audit.prevPage")}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="compactIcon"
                onClick={() => setPage((previous) => Math.min(totalPages, previous + 1))}
                disabled={page >= totalPages}
                aria-label={t("settings.calls.skills.audit.nextPage")}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="compactIcon"
                onClick={() => setPage(totalPages)}
                disabled={page >= totalPages}
                aria-label={t("settings.calls.skills.audit.lastPage")}
              >
                <ChevronsRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </>
      ) : (
        <Card className={SKILLS_PAGE_CARD_CLASS}>
          <CardContent size="compact" className="p-0">
            <div className="divide-y divide-border/70">
              <div className="flex items-center justify-between gap-4 px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  {t("settings.calls.skills.audit.config.hideUiOperationRecords")}
                </span>
                <div className="flex items-center">
                  <Switch
                    checked={normalizedConfigDraft.hideUiOperationRecords}
                    onCheckedChange={(checked) =>
                      setConfigDraft((previous) => ({
                        ...previous,
                        hideUiOperationRecords: checked === true,
                      }))
                    }
                    disabled={updateSettings.isPending}
                  />
                </div>
              </div>
              <div className="flex items-center justify-between gap-4 px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  {t("settings.calls.skills.audit.config.retentionDays")}
                </span>
                <Select
                  value={String(normalizedConfigDraft.retentionDays)}
                  onChange={(event) =>
                    setConfigDraft((previous) => ({
                      ...previous,
                      retentionDays: normalizeAuditRetentionDays(Number(event.target.value)),
                    }))
                  }
                  disabled={updateSettings.isPending}
                  className={`${SKILLS_SELECT_TEXT_CLASS} w-44`}
                >
                  {SKILLS_AUDIT_RETENTION_DAY_OPTIONS.map((days) => (
                    <option key={days} value={String(days)}>
                      {formatTemplate(t("settings.calls.skills.audit.config.days"), { count: days })}
                    </option>
                  ))}
                </Select>
              </div>
              <div className="flex items-center justify-between gap-4 px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  {t("settings.calls.skills.audit.config.clearLogs")}
                </span>
                <Button
                  variant="outline"
                  size="compact"
                  onClick={clearAuditLogs}
                  disabled={updateSettings.isPending || records.length === 0}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t("settings.calls.skills.audit.config.clearNow")}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </Tabs>
  );
}
