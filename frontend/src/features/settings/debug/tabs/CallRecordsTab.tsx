import * as React from "react";
import { ChevronDown, ChevronLeft, ChevronRight, ListChecks, RefreshCw, Search, Settings2, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { messageBus } from "@/shared/message";
import { useEnabledProvidersWithModels, useProviders } from "@/shared/query/providers";
import { useSettings, useUpdateSettings } from "@/shared/query/settings";
import { useClearLLMCallRecords, usePruneExpiredLLMCallRecords } from "@/shared/query/threads";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { DASHBOARD_CONTROL_GROUP_CLASS, PanelCard } from "@/shared/ui/dashboard";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@/shared/ui/sheet";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import type { GatewayCallRecordAutoCleanup, GatewayCallRecordSaveStrategy } from "@/shared/contracts/settings";

import type { CallRecordsTabProps } from "../types";

function statusClass(status: string) {
  switch (status) {
    case "completed":
      return "bg-emerald-100 text-emerald-800";
    case "cancelled":
      return "bg-amber-100 text-amber-800";
    case "error":
      return "bg-destructive/15 text-destructive";
    default:
      return "bg-muted text-muted-foreground";
  }
}

function tokenLabel(input: number, output: number, total: number) {
  if (input <= 0 && output <= 0 && total <= 0) {
    return "-";
  }
  return `${input || 0}/${output || 0}/${total || 0}`;
}

function contextTokenLabel(total: number, window: number) {
  if (total <= 0 && window <= 0) {
    return "-";
  }
  if (window > 0) {
    return `${total || 0} / ${window}`;
  }
  return `${total || 0}`;
}

function durationLabel(value: number) {
  if (!value || value <= 0) {
    return "-";
  }
  return `${value}ms`;
}

function trimmedText(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeText(value: unknown) {
  return trimmedText(value).toLowerCase();
}

function formatTemplate(template: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (result, [key, value]) => result.split(`{${key}}`).join(String(value)),
    template
  );
}

function formatJsonPayload(value: unknown) {
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return "-";
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return trimmed;
    }
  }
  if (value == null) {
    return "-";
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

type FilterOption = {
  value: string;
  label: string;
};

const ROWS_PER_PAGE_OPTIONS = [10, 20, 30, 50];
const CALL_RECORD_RETENTION_DAY_OPTIONS = [7, 14, 30, 60, 90, 180, 365];

function normalizeCallRecordSaveStrategy(value: unknown): GatewayCallRecordSaveStrategy {
  switch (trimmedText(value)) {
    case "off":
    case "errors":
    case "all":
      return trimmedText(value) as GatewayCallRecordSaveStrategy;
    default:
      return "all";
  }
}

function normalizeCallRecordAutoCleanup(value: unknown): GatewayCallRecordAutoCleanup {
  switch (trimmedText(value)) {
    case "off":
    case "on_write":
    case "hourly":
      return trimmedText(value) as GatewayCallRecordAutoCleanup;
    default:
      return "hourly";
  }
}

function normalizeCallRecordRetentionDays(value: unknown) {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric)) {
    return 30;
  }
  const rounded = Math.round(numeric);
  if (rounded <= 0) {
    return 30;
  }
  return Math.min(365, rounded);
}

const SelectionCheckbox = React.forwardRef<
  HTMLInputElement,
  Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> & { indeterminate?: boolean }
>(({ className, indeterminate = false, ...props }, forwardedRef) => {
  const innerRef = React.useRef<HTMLInputElement | null>(null);
  const setRefs = React.useCallback(
    (node: HTMLInputElement | null) => {
      innerRef.current = node;
      if (typeof forwardedRef === "function") {
        forwardedRef(node);
        return;
      }
      if (forwardedRef) {
        forwardedRef.current = node;
      }
    },
    [forwardedRef]
  );

  React.useEffect(() => {
    if (!innerRef.current) {
      return;
    }
    innerRef.current.indeterminate = indeterminate;
  }, [indeterminate]);

  return (
    <input
      {...props}
      ref={setRefs}
      type="checkbox"
      role="checkbox"
      className={cn(
        "h-4 w-4 rounded border border-border bg-background align-middle text-primary shadow-sm outline-none",
        "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      onClick={(event) => event.stopPropagation()}
    />
  );
});

SelectionCheckbox.displayName = "SelectionCheckbox";

function CallRecordFilterMenu(props: {
  t: CallRecordsTabProps["t"];
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  selectedThreadId: string;
  setSelectedThreadId: (value: string) => void;
  callSource: string;
  setCallSource: (value: string) => void;
  callStatus: string;
  setCallStatus: (value: string) => void;
  providerFilter: string;
  setProviderFilter: (value: string) => void;
  modelFilter: string;
  setModelFilter: (value: string) => void;
  runFilter: string;
  setRunFilter: (value: string) => void;
  providerOptions: FilterOption[];
  modelOptions: FilterOption[];
  runOptions: FilterOption[];
  threads: CallRecordsTabProps["threads"];
  filterCount: number;
  onClearAll: () => void;
}) {
  const hasFilters = props.filterCount > 0;
  const hasSearchQuery = props.searchQuery.trim().length > 0;
  const triggerLabel = hasSearchQuery ? props.searchQuery : props.t("settings.debug.calls.actions.searchAndFilter");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[168px] max-w-[240px] justify-between gap-2 px-2.5"
          title={triggerLabel}
        >
          <span className="flex min-w-0 items-center gap-2">
            <Search className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className={cn("min-w-0 truncate text-xs", hasSearchQuery ? "text-foreground" : "text-muted-foreground")}>
              {triggerLabel}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            {hasFilters ? (
              <Badge variant="subtle" className="h-5 px-1.5 text-[10px] font-medium">
                {props.filterCount}
              </Badge>
            ) : null}
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[320px] p-0">
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("settings.debug.calls.filters.searchPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="grid gap-2 p-2">
          <label className="grid gap-1">
            <span className="text-[11px] font-medium text-muted-foreground">
              {props.t("settings.debug.calls.table.conversation")}
            </span>
            <Select
              value={props.selectedThreadId}
              onChange={(event) => props.setSelectedThreadId(event.target.value)}
              className="w-full"
            >
              <option value="">{props.t("settings.debug.calls.filters.allThreads")}</option>
              {props.threads.map((thread) => (
                <option key={thread.id} value={thread.id}>
                  {thread.title || thread.id}
                </option>
              ))}
            </Select>
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-[11px] font-medium text-muted-foreground">
                {props.t("settings.debug.calls.table.source")}
              </span>
              <Select value={props.callSource} onChange={(event) => props.setCallSource(event.target.value)} className="w-full">
                <option value="">{props.t("settings.debug.calls.filters.allSources")}</option>
                <option value="dialogue">dialogue</option>
                <option value="relay">relay</option>
                <option value="one-shot">one-shot</option>
                <option value="memory">memory</option>
              </Select>
            </label>
            <label className="grid gap-1">
              <span className="text-[11px] font-medium text-muted-foreground">
                {props.t("settings.debug.calls.table.status")}
              </span>
              <Select value={props.callStatus} onChange={(event) => props.setCallStatus(event.target.value)} className="w-full">
                <option value="">{props.t("settings.debug.calls.filters.allStatus")}</option>
                <option value="completed">completed</option>
                <option value="error">error</option>
                <option value="cancelled">cancelled</option>
              </Select>
            </label>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-[11px] font-medium text-muted-foreground">
                {props.t("settings.debug.calls.filters.provider")}
              </span>
              <Select
                value={props.providerFilter}
                onChange={(event) => props.setProviderFilter(event.target.value)}
                className="w-full"
              >
                <option value="">{props.t("settings.debug.calls.filters.provider")}</option>
                {props.providerOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </label>
            <label className="grid gap-1">
              <span className="text-[11px] font-medium text-muted-foreground">
                {props.t("settings.debug.calls.filters.model")}
              </span>
              <Select
                value={props.modelFilter}
                onChange={(event) => props.setModelFilter(event.target.value)}
                className="w-full"
              >
                <option value="">{props.t("settings.debug.calls.filters.model")}</option>
                {props.modelOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </label>
          </div>
          <label className="grid gap-1">
            <span className="text-[11px] font-medium text-muted-foreground">
              {props.t("settings.debug.calls.table.runId")}
            </span>
            <Select
              value={props.runFilter}
              onChange={(event) => props.setRunFilter(event.target.value)}
              className="w-full"
            >
              <option value="">{props.t("settings.debug.calls.filters.runId")}</option>
              {props.runOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          </label>
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("settings.debug.calls.actions.clearFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CallRecordBatchMenu(props: {
  t: CallRecordsTabProps["t"];
  selectedCount: number;
  hasRunIds: boolean;
  hasConversationIds: boolean;
  onCopyRecordIds: () => void;
  onCopyRunIds: () => void;
  onCopyConversationIds: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="compact"
          className="gap-1.5 rounded-none border-0 border-l border-border/70"
          disabled={props.selectedCount === 0}
        >
          {props.t("settings.debug.calls.actions.batch")}
          <ChevronDown className="h-3.5 w-3.5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-[220px] p-1">
        <DropdownMenuLabel>{props.t("settings.debug.calls.actions.batch")}</DropdownMenuLabel>
        <DropdownMenuItem disabled={props.selectedCount === 0} onClick={props.onCopyRecordIds}>
          {props.t("settings.debug.calls.actions.copyRecordIds")}
        </DropdownMenuItem>
        <DropdownMenuItem disabled={props.selectedCount === 0 || !props.hasRunIds} onClick={props.onCopyRunIds}>
          {props.t("settings.debug.calls.actions.copyRunIds")}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={props.selectedCount === 0 || !props.hasConversationIds}
          onClick={props.onCopyConversationIds}
        >
          {props.t("settings.debug.calls.actions.copyConversationIds")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CallRecordSettingsMenu(props: {
  t: CallRecordsTabProps["t"];
  saveStrategy: GatewayCallRecordSaveStrategy;
  retentionDays: number;
  autoCleanup: GatewayCallRecordAutoCleanup;
  settingsPending: boolean;
  cleanupPending: boolean;
  onSaveStrategyChange: (value: GatewayCallRecordSaveStrategy) => void;
  onRetentionDaysChange: (value: number) => void;
  onAutoCleanupChange: (value: GatewayCallRecordAutoCleanup) => void;
  onPruneExpired: () => void;
  onClearAll: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="compact" className="shrink-0 gap-2">
          <Settings2 className="h-4 w-4" />
          {props.t("settings.debug.calls.config.title")}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-[18rem] max-w-[calc(100vw-2rem)] p-0">
        <div className="grid gap-2 p-3">
          <DropdownMenuLabel className="px-0 pt-0">{props.t("settings.debug.calls.config.title")}</DropdownMenuLabel>
          <label className="grid gap-1.5">
            <span className="text-[11px] font-medium text-muted-foreground">
              {props.t("settings.debug.calls.config.saveStrategy.label")}
            </span>
            <Select
              value={props.saveStrategy}
              onChange={(event) => props.onSaveStrategyChange(event.target.value as GatewayCallRecordSaveStrategy)}
              disabled={props.settingsPending}
              className="w-full"
            >
              <option value="all">{props.t("settings.debug.calls.config.saveStrategy.options.all")}</option>
              <option value="errors">{props.t("settings.debug.calls.config.saveStrategy.options.errors")}</option>
              <option value="off">{props.t("settings.debug.calls.config.saveStrategy.options.off")}</option>
            </Select>
          </label>
          <label className="grid gap-1.5">
            <span className="text-[11px] font-medium text-muted-foreground">
              {props.t("settings.debug.calls.config.retentionDays.label")}
            </span>
            <Select
              value={String(props.retentionDays)}
              onChange={(event) => props.onRetentionDaysChange(Number(event.target.value))}
              disabled={props.settingsPending}
              className="w-full"
            >
              {CALL_RECORD_RETENTION_DAY_OPTIONS.map((days) => (
                <option key={days} value={String(days)}>
                  {formatTemplate(props.t("settings.debug.calls.config.retentionDays.days"), { count: days })}
                </option>
              ))}
            </Select>
          </label>
          <label className="grid gap-1.5">
            <span className="text-[11px] font-medium text-muted-foreground">
              {props.t("settings.debug.calls.config.autoCleanup.label")}
            </span>
            <Select
              value={props.autoCleanup}
              onChange={(event) => props.onAutoCleanupChange(event.target.value as GatewayCallRecordAutoCleanup)}
              disabled={props.settingsPending}
              className="w-full"
            >
              <option value="hourly">{props.t("settings.debug.calls.config.autoCleanup.options.hourly")}</option>
              <option value="on_write">{props.t("settings.debug.calls.config.autoCleanup.options.onWrite")}</option>
              <option value="off">{props.t("settings.debug.calls.config.autoCleanup.options.off")}</option>
            </Select>
          </label>
        </div>
        <DropdownMenuSeparator />
        <div className="grid gap-2 p-3">
          <Button
            variant="outline"
            size="compact"
            className="justify-start"
            onClick={props.onPruneExpired}
            disabled={props.cleanupPending}
          >
            {props.t("settings.debug.calls.config.actions.pruneExpired")}
          </Button>
          <Button
            variant="outline"
            size="compact"
            className="justify-start text-destructive hover:text-destructive"
            onClick={props.onClearAll}
            disabled={props.cleanupPending}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {props.t("settings.debug.calls.config.actions.clearAll")}
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function CallRecordsTab({
  t,
  threads,
  optionRecords,
  selectedThreadId,
  setSelectedThreadId,
  callSource,
  setCallSource,
  callStatus,
  setCallStatus,
  providerFilter,
  setProviderFilter,
  modelFilter,
  setModelFilter,
  runFilter,
  setRunFilter,
  records,
  selectedRecord,
  setSelectedRecordId,
  isLoading,
  hasError,
  isDetailLoading,
  refresh,
  formatDateTime,
  inspectRun,
}: CallRecordsTabProps) {
  const providersQuery = useProviders();
  const enabledProvidersWithModelsQuery = useEnabledProvidersWithModels();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();
  const pruneExpiredMutation = usePruneExpiredLLMCallRecords();
  const clearMutation = useClearLLMCallRecords();
  const [searchQuery, setSearchQuery] = React.useState("");
  const [selectionMode, setSelectionMode] = React.useState(false);
  const [selectedRecordIds, setSelectedRecordIds] = React.useState<string[]>([]);
  const [rowsPerPage, setRowsPerPage] = React.useState(20);
  const [pageIndex, setPageIndex] = React.useState(0);

  const conversationTitleById = React.useMemo(() => {
    const titles = new Map<string, string>();
    for (const thread of threads) {
      titles.set(thread.id, thread.title || thread.id);
    }
    return titles;
  }, [threads]);

  const providerLabelById = React.useMemo(() => {
    const labels = new Map<string, string>();
    for (const provider of providersQuery.data ?? []) {
      const providerId = trimmedText(provider.id);
      if (!providerId) {
        continue;
      }
      labels.set(providerId, trimmedText(provider.name) || providerId);
    }
    return labels;
  }, [providersQuery.data]);

  const modelLabelByRef = React.useMemo(() => {
    const labels = new Map<string, string>();
    for (const group of enabledProvidersWithModelsQuery.data ?? []) {
      const providerId = trimmedText(group.provider.id);
      if (!providerId) {
        continue;
      }
      for (const model of group.models) {
        const displayLabel = trimmedText(model.displayName) || trimmedText(model.name) || trimmedText(model.id);
        const modelName = trimmedText(model.name);
        const modelId = trimmedText(model.id);
        if (modelName) {
          labels.set(`${providerId}::${modelName}`, displayLabel || modelName);
        }
        if (modelId) {
          labels.set(`${providerId}::${modelId}`, displayLabel || modelId);
        }
      }
    }
    return labels;
  }, [enabledProvidersWithModelsQuery.data]);

  const resolveProviderLabel = React.useCallback(
    (providerId: unknown) => {
      const normalized = trimmedText(providerId);
      if (!normalized) {
        return "-";
      }
      return providerLabelById.get(normalized) || normalized;
    },
    [providerLabelById]
  );

  const resolveModelLabel = React.useCallback(
    (providerId: unknown, modelRef: unknown) => {
      const normalizedModel = trimmedText(modelRef);
      if (!normalizedModel) {
        return "-";
      }
      const normalizedProvider = trimmedText(providerId);
      if (normalizedProvider) {
        return modelLabelByRef.get(`${normalizedProvider}::${normalizedModel}`) || normalizedModel;
      }
      return normalizedModel;
    },
    [modelLabelByRef]
  );

  const filteredRecords = React.useMemo(() => {
    const query = normalizeText(searchQuery);
    if (!query) {
      return records;
    }
    return records.filter((record) => {
      const conversationTitle = conversationTitleById.get(record.threadId) ?? "";
      const providerLabel = resolveProviderLabel(record.providerId);
      const modelLabel = resolveModelLabel(record.providerId, record.modelName);
      const haystack = [
        conversationTitle,
        record.threadId,
        record.runId,
        providerLabel,
        record.providerId,
        modelLabel,
        record.modelName,
        record.requestSource,
        record.operation,
        record.status,
        record.finishReason,
        record.errorText,
      ]
        .join("\n")
        .toLowerCase();
      return haystack.includes(query);
    });
  }, [conversationTitleById, records, resolveModelLabel, resolveProviderLabel, searchQuery]);

  React.useEffect(() => {
    if (!selectionMode) {
      setSelectedRecordIds((current) => (current.length > 0 ? [] : current));
      return;
    }
    const currentIds = new Set(records.map((record) => record.id));
    setSelectedRecordIds((current) => current.filter((id) => currentIds.has(id)));
  }, [records, selectionMode]);

  const selectedIdSet = React.useMemo(() => new Set(selectedRecordIds), [selectedRecordIds]);

  const pageCount = Math.max(1, Math.ceil(filteredRecords.length / rowsPerPage));

  React.useEffect(() => {
    setPageIndex((current) => Math.min(current, pageCount - 1));
  }, [pageCount]);

  const paginatedRecords = React.useMemo(() => {
    const start = pageIndex * rowsPerPage;
    return filteredRecords.slice(start, start + rowsPerPage);
  }, [filteredRecords, pageIndex, rowsPerPage]);

  const visibleIds = React.useMemo(() => paginatedRecords.map((record) => record.id), [paginatedRecords]);

  const allVisibleSelected =
    visibleIds.length > 0 && visibleIds.every((recordId) => selectedIdSet.has(recordId));
  const someVisibleSelected = visibleIds.some((recordId) => selectedIdSet.has(recordId)) && !allVisibleSelected;

  const selectedRows = React.useMemo(
    () => records.filter((record) => selectedIdSet.has(record.id)),
    [records, selectedIdSet]
  );

  const hasSelectedRunIds = selectedRows.some((record) => trimmedText(record.runId).length > 0);
  const hasSelectedConversationIds = selectedRows.some((record) => trimmedText(record.threadId).length > 0);

  const selectedConversationTitle = selectedRecord
    ? conversationTitleById.get(selectedRecord.threadId) ?? ""
    : "";
  const selectedProviderLabel = selectedRecord ? resolveProviderLabel(selectedRecord.providerId) : "-";
  const selectedModelLabel = selectedRecord ? resolveModelLabel(selectedRecord.providerId, selectedRecord.modelName) : "-";
  const normalizedProviderFilter = providerFilter.trim();
  const normalizedModelFilter = modelFilter.trim();
  const callRecordSettings = React.useMemo(() => {
    const raw = settingsQuery.data?.gateway.runtime.callRecords;
    return {
      saveStrategy: normalizeCallRecordSaveStrategy(raw?.saveStrategy),
      retentionDays: normalizeCallRecordRetentionDays(raw?.retentionDays),
      autoCleanup: normalizeCallRecordAutoCleanup(raw?.autoCleanup),
    };
  }, [settingsQuery.data?.gateway.runtime.callRecords]);

  const providerOptions = React.useMemo(() => {
    const labels = new Map<string, string>();
    for (const record of optionRecords) {
      const providerId = trimmedText(record.providerId);
      if (!providerId) {
        continue;
      }
      labels.set(providerId, resolveProviderLabel(providerId));
    }
    for (const provider of providersQuery.data ?? []) {
      const providerId = trimmedText(provider.id);
      if (!providerId) {
        continue;
      }
      labels.set(providerId, trimmedText(provider.name) || providerId);
    }
    if (normalizedProviderFilter && !labels.has(normalizedProviderFilter)) {
      labels.set(normalizedProviderFilter, resolveProviderLabel(normalizedProviderFilter));
    }
    return Array.from(labels.entries())
      .map(([value, label]) => ({ value, label }))
      .sort((left, right) => left.label.localeCompare(right.label, undefined, { sensitivity: "base" }));
  }, [normalizedProviderFilter, optionRecords, providersQuery.data, resolveProviderLabel]);

  const modelOptions = React.useMemo(() => {
    const labels = new Map<string, string>();
    for (const record of optionRecords) {
      const providerId = trimmedText(record.providerId);
      const modelRef = trimmedText(record.modelName);
      if (!modelRef) {
        continue;
      }
      if (normalizedProviderFilter && providerId !== normalizedProviderFilter) {
        continue;
      }
      labels.set(modelRef, resolveModelLabel(providerId, modelRef));
    }
    if (normalizedModelFilter && !labels.has(normalizedModelFilter)) {
      labels.set(normalizedModelFilter, resolveModelLabel(normalizedProviderFilter, normalizedModelFilter));
    }
    return Array.from(labels.entries())
      .map(([value, label]) => ({ value, label }))
      .sort((left, right) => left.label.localeCompare(right.label, undefined, { sensitivity: "base" }));
  }, [normalizedModelFilter, normalizedProviderFilter, optionRecords, resolveModelLabel]);

  const runOptions = React.useMemo(() => {
    const labels = new Map<string, string>();
    const sorted = [...optionRecords].sort((left, right) => right.startedAt.localeCompare(left.startedAt));
    for (const record of sorted) {
      const providerId = trimmedText(record.providerId);
      const modelRef = trimmedText(record.modelName);
      const runId = trimmedText(record.runId);
      if (!runId) {
        continue;
      }
      if (normalizedProviderFilter && providerId !== normalizedProviderFilter) {
        continue;
      }
      if (normalizedModelFilter && modelRef !== normalizedModelFilter) {
        continue;
      }
      if (!labels.has(runId)) {
        labels.set(runId, runId);
      }
    }
    const normalizedRunFilter = trimmedText(runFilter);
    if (normalizedRunFilter && !labels.has(normalizedRunFilter)) {
      labels.set(normalizedRunFilter, normalizedRunFilter);
    }
    return Array.from(labels.entries()).map(([value, label]) => ({ value, label }));
  }, [normalizedModelFilter, normalizedProviderFilter, optionRecords, runFilter]);

  const totalText = formatTemplate(t("settings.debug.calls.table.totalCalls"), {
    count: filteredRecords.length,
  });
  const rowsPerPageText = t("library.table.rowsPerPage");
  const pageText = formatTemplate(t("library.table.pageOf"), {
    page: pageIndex + 1,
    total: pageCount,
  });

  const filterCount = [
    searchQuery.trim(),
    selectedThreadId,
    callSource,
    callStatus,
    providerFilter.trim(),
    modelFilter.trim(),
    runFilter.trim(),
  ].filter(Boolean).length;

  const clearAllFilters = React.useCallback(() => {
    setSearchQuery("");
    setSelectedThreadId("");
    setCallSource("");
    setCallStatus("");
    setProviderFilter("");
    setModelFilter("");
    setRunFilter("");
  }, [
    setCallSource,
    setCallStatus,
    setModelFilter,
    setProviderFilter,
    setRunFilter,
    setSelectedThreadId,
  ]);

  const handleEnterSelectionMode = React.useCallback(() => {
    setSelectionMode(true);
  }, []);

  const handleExitSelectionMode = React.useCallback(() => {
    setSelectionMode(false);
    setSelectedRecordIds([]);
  }, []);

  const toggleRecordSelection = React.useCallback((recordId: string, checked: boolean) => {
    setSelectedRecordIds((current) => {
      if (checked) {
        if (current.includes(recordId)) {
          return current;
        }
        return [...current, recordId];
      }
      return current.filter((item) => item !== recordId);
    });
  }, []);

  const selectVisibleRows = React.useCallback(() => {
    if (visibleIds.length === 0) {
      return;
    }
    setSelectedRecordIds((current) => {
      const next = new Set(current);
      for (const recordId of visibleIds) {
        next.add(recordId);
      }
      return Array.from(next);
    });
  }, [visibleIds]);

  const toggleVisibleSelection = React.useCallback(
    (checked: boolean) => {
      if (checked) {
        selectVisibleRows();
        return;
      }
      const visibleIdSet = new Set(visibleIds);
      setSelectedRecordIds((current) => current.filter((recordId) => !visibleIdSet.has(recordId)));
    },
    [selectVisibleRows, visibleIds]
  );

  const copySelectedValues = React.useCallback(
    async (values: string[], descriptionKey: string) => {
      if (values.length === 0) {
        return;
      }
      if (typeof navigator === "undefined" || !navigator.clipboard?.writeText) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.debug.calls.toasts.clipboardUnavailable"),
        });
        return;
      }
      try {
        await navigator.clipboard.writeText(values.join("\n"));
        messageBus.publishToast({
          intent: "success",
          title: t("chat.actions.copied"),
          description: t(descriptionKey).replace("{count}", String(values.length)),
        });
      } catch (error) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.debug.calls.toasts.copyFailed"),
          description: String(error),
        });
      }
    },
    [t]
  );

  const copySelectedRecordIds = React.useCallback(() => {
    void copySelectedValues(
      selectedRows.map((record) => record.id),
      "settings.debug.calls.toasts.recordIdsCopied"
    );
  }, [copySelectedValues, selectedRows]);

  const copySelectedRunIds = React.useCallback(() => {
    const values = Array.from(new Set(selectedRows.map((record) => trimmedText(record.runId)).filter(Boolean)));
    void copySelectedValues(values, "settings.debug.calls.toasts.runIdsCopied");
  }, [copySelectedValues, selectedRows]);

  const copySelectedConversationIds = React.useCallback(() => {
    const values = Array.from(new Set(selectedRows.map((record) => trimmedText(record.threadId)).filter(Boolean)));
    void copySelectedValues(values, "settings.debug.calls.toasts.conversationIdsCopied");
  }, [copySelectedValues, selectedRows]);

  const patchCallRecordSettings = React.useCallback(
    (patch: {
      saveStrategy?: GatewayCallRecordSaveStrategy;
      retentionDays?: number;
      autoCleanup?: GatewayCallRecordAutoCleanup;
    }) => {
      updateSettings.mutate(
        {
          gateway: {
            runtime: {
              callRecords: patch,
            },
          },
        },
        {
          onError: (error) => {
            messageBus.publishToast({
              intent: "warning",
              title: t("settings.debug.calls.config.toasts.saveFailed"),
              description: String(error),
            });
          },
        }
      );
    },
    [t, updateSettings]
  );

  const handlePruneExpired = React.useCallback(() => {
    pruneExpiredMutation.mutate(undefined, {
      onSuccess: (count) => {
        messageBus.publishToast({
          intent: "success",
          title: t("settings.debug.calls.config.toasts.pruneDone"),
          description: t("settings.debug.calls.config.toasts.pruneDoneDesc").replace("{count}", String(count)),
        });
        refresh();
      },
      onError: (error) => {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.debug.calls.config.toasts.pruneFailed"),
          description: String(error),
        });
      },
    });
  }, [pruneExpiredMutation, refresh, t]);

  const handleClearAll = React.useCallback(() => {
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.debug.calls.config.confirm.clearTitle"),
      description: t("settings.debug.calls.config.confirm.clearDescription"),
      confirmLabel: t("settings.debug.calls.config.actions.clearAll"),
      cancelLabel: t("common.cancel"),
      onConfirm: () =>
        clearMutation.mutate(undefined, {
          onSuccess: (count) => {
            messageBus.publishToast({
              intent: "success",
              title: t("settings.debug.calls.config.toasts.clearDone"),
              description: t("settings.debug.calls.config.toasts.clearDoneDesc").replace("{count}", String(count)),
            });
            refresh();
          },
          onError: (error) => {
            messageBus.publishToast({
              intent: "warning",
              title: t("settings.debug.calls.config.toasts.clearFailed"),
              description: String(error),
            });
          },
        }),
    });
  }, [clearMutation, refresh, t]);

  const detailItems = selectedRecord
    ? [
        {
          label: t("settings.debug.calls.detail.startedAt"),
          value: formatDateTime(selectedRecord.startedAt),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.finishedAt"),
          value: selectedRecord.finishedAt ? formatDateTime(selectedRecord.finishedAt) : "-",
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.duration"),
          value: durationLabel(selectedRecord.durationMs),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.finishReason"),
          value: selectedRecord.finishReason || "-",
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.threadId"),
          value: selectedRecord.threadId || "-",
          className: "break-all",
        },
        {
          label: t("settings.debug.calls.detail.runId"),
          value: selectedRecord.runId || "-",
          className: "break-all",
        },
        {
          label: t("settings.debug.calls.detail.provider"),
          value: resolveProviderLabel(selectedRecord.providerId),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.model"),
          value: resolveModelLabel(selectedRecord.providerId, selectedRecord.modelName),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.source"),
          value: selectedRecord.requestSource || "-",
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.operation"),
          value: selectedRecord.operation || "-",
          className: "break-all",
        },
        {
          label: t("settings.debug.calls.detail.tokens"),
          value: tokenLabel(selectedRecord.inputTokens, selectedRecord.outputTokens, selectedRecord.totalTokens),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.contextTokens"),
          value: contextTokenLabel(selectedRecord.contextTotalTokens, selectedRecord.contextWindowTokens),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.payloadTruncated"),
          value: selectedRecord.payloadTruncated
            ? t("settings.debug.calls.detail.boolean.true")
            : t("settings.debug.calls.detail.boolean.false"),
          className: "",
        },
        {
          label: t("settings.debug.calls.detail.recordId"),
          value: selectedRecord.id,
          className: "break-all",
        },
      ]
    : [];

  return (
    <>
      <div className="flex h-full min-h-0 flex-1 flex-col gap-3">
        <div className="flex min-w-0 flex-nowrap items-center justify-end gap-2 overflow-x-auto px-0.5 py-1 -mx-0.5">
          <CallRecordFilterMenu
            t={t}
            searchQuery={searchQuery}
            onSearchQueryChange={setSearchQuery}
            selectedThreadId={selectedThreadId}
            setSelectedThreadId={setSelectedThreadId}
            callSource={callSource}
            setCallSource={setCallSource}
            callStatus={callStatus}
            setCallStatus={setCallStatus}
            providerFilter={providerFilter}
            setProviderFilter={setProviderFilter}
            modelFilter={modelFilter}
            setModelFilter={setModelFilter}
            runFilter={runFilter}
            setRunFilter={setRunFilter}
            providerOptions={providerOptions}
            modelOptions={modelOptions}
            runOptions={runOptions}
            threads={threads}
            filterCount={filterCount}
            onClearAll={clearAllFilters}
          />
          {selectionMode ? (
            <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
              <div className="inline-flex items-center gap-2 px-3 text-xs text-muted-foreground">
                <ListChecks className="h-3.5 w-3.5" />
                <span>{t("settings.debug.calls.actions.selectedCount").replace("{count}", String(selectedRows.length))}</span>
              </div>
              <CallRecordBatchMenu
                t={t}
                selectedCount={selectedRows.length}
                hasRunIds={hasSelectedRunIds}
                hasConversationIds={hasSelectedConversationIds}
                onCopyRecordIds={copySelectedRecordIds}
                onCopyRunIds={copySelectedRunIds}
                onCopyConversationIds={copySelectedConversationIds}
              />
              <Button
                variant="ghost"
                size="compact"
                className="gap-1.5 rounded-none border-0 border-l border-border/70"
                onClick={handleExitSelectionMode}
              >
                {t("settings.debug.calls.actions.cancelSelection")}
              </Button>
            </div>
          ) : (
            <Button variant="outline" size="compact" className="gap-2" onClick={handleEnterSelectionMode}>
              <ListChecks className="h-4 w-4" />
              {t("settings.debug.calls.actions.selectCalls")}
            </Button>
          )}
          <CallRecordSettingsMenu
            t={t}
            saveStrategy={callRecordSettings.saveStrategy}
            retentionDays={callRecordSettings.retentionDays}
            autoCleanup={callRecordSettings.autoCleanup}
            settingsPending={settingsQuery.isLoading || updateSettings.isPending}
            cleanupPending={pruneExpiredMutation.isPending || clearMutation.isPending}
            onSaveStrategyChange={(value) => patchCallRecordSettings({ saveStrategy: value })}
            onRetentionDaysChange={(value) =>
              patchCallRecordSettings({ retentionDays: normalizeCallRecordRetentionDays(value) })
            }
            onAutoCleanupChange={(value) => patchCallRecordSettings({ autoCleanup: value })}
            onPruneExpired={handlePruneExpired}
            onClearAll={handleClearAll}
          />
          <Button
            size="compactIcon"
            variant="outline"
            className="shrink-0"
            aria-label={t("common.refresh")}
            title={t("common.refresh")}
            onClick={refresh}
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-border/70 bg-background/70">
          <div className="relative h-full overflow-auto">
            {isLoading ? (
              <div className="flex h-full min-h-0 items-center justify-center px-4 text-xs">{t("common.loading")}</div>
            ) : hasError ? (
              <div className="flex h-full min-h-0 items-center justify-center px-4 text-xs text-destructive">
                {t("settings.debug.calls.error")}
              </div>
            ) : (
              <Table className="table-fixed min-w-[1180px]">
                <TableHeader className="app-table-dense-head sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
                  <TableRow>
                    {selectionMode ? (
                      <TableHead className="w-10">
                        <SelectionCheckbox
                          checked={allVisibleSelected}
                          indeterminate={someVisibleSelected}
                          aria-label={t("settings.debug.calls.actions.selectCalls")}
                          onChange={(event) => toggleVisibleSelection(event.currentTarget.checked)}
                        />
                      </TableHead>
                    ) : null}
                    <TableHead>{t("settings.debug.calls.table.time")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.conversation")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.runId")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.model")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.source")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.operation")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.status")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.duration")}</TableHead>
                    <TableHead>{t("settings.debug.calls.table.tokens")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedRecords.map((record) => {
                    const active = selectedRecord?.id === record.id;
                    const conversationTitle = conversationTitleById.get(record.threadId) ?? "";
                    const checked = selectedIdSet.has(record.id);

                    return (
                      <TableRow
                        key={record.id}
                        className={cn("cursor-pointer hover:bg-muted/20", active && "bg-muted/40")}
                        onClick={() => setSelectedRecordId(record.id)}
                      >
                        {selectionMode ? (
                          <TableCell className="w-10">
                            <SelectionCheckbox
                              checked={checked}
                              aria-label={record.id}
                              onChange={(event) => toggleRecordSelection(record.id, event.currentTarget.checked)}
                            />
                          </TableCell>
                        ) : null}
                        <TableCell className="font-mono text-[11px] text-muted-foreground" title={formatDateTime(record.startedAt)}>
                          {formatDateTime(record.startedAt)}
                        </TableCell>
                        <TableCell className="align-top">
                          <div className="truncate text-[11px]" title={conversationTitle || record.threadId || "-"}>
                            {conversationTitle || record.threadId || "-"}
                          </div>
                          <div className="truncate font-mono text-[10px] text-muted-foreground" title={record.threadId || "-"}>
                            {record.threadId || "-"}
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={record.runId || "-"}>
                          <div className="truncate">{record.runId || "-"}</div>
                        </TableCell>
                        <TableCell className="align-top">
                          <div
                            className="truncate text-[11px]"
                            title={resolveModelLabel(record.providerId, record.modelName)}
                          >
                            {resolveModelLabel(record.providerId, record.modelName)}
                          </div>
                          <div
                            className="truncate text-[10px] text-muted-foreground"
                            title={resolveProviderLabel(record.providerId)}
                          >
                            {resolveProviderLabel(record.providerId)}
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={record.requestSource || "-"}>
                          <div className="truncate">{record.requestSource || "-"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={record.operation || "-"}>
                          <div className="truncate">{record.operation || "-"}</div>
                        </TableCell>
                        <TableCell>
                          <span
                            className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium ${statusClass(record.status)}`}
                          >
                            {record.status || "-"}
                          </span>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]">{durationLabel(record.durationMs)}</TableCell>
                        <TableCell className="font-mono text-[11px]" title={tokenLabel(record.inputTokens, record.outputTokens, record.totalTokens)}>
                          {tokenLabel(record.inputTokens, record.outputTokens, record.totalTokens)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
            {!isLoading && !hasError && filteredRecords.length === 0 ? (
              <div className="pointer-events-none absolute inset-x-0 bottom-0 top-10 flex items-center justify-center text-sm text-muted-foreground">
                {t("library.table.noResults")}
              </div>
            ) : null}
          </div>
        </div>

        <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 text-xs">
          <div className="text-muted-foreground">{totalText}</div>
          <div className="flex items-center gap-2">
            <Select value={String(rowsPerPage)} onChange={(event) => setRowsPerPage(Number(event.target.value))}>
              {ROWS_PER_PAGE_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {formatTemplate(rowsPerPageText, { count: option })}
                </option>
              ))}
            </Select>
            <div className="text-muted-foreground">{pageText}</div>
            <Button
              variant="outline"
              size="compactIcon"
              onClick={() => setPageIndex((current) => Math.max(0, current - 1))}
              disabled={pageIndex <= 0}
              aria-label={t("library.table.prevPage")}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="compactIcon"
              onClick={() => setPageIndex((current) => Math.min(pageCount - 1, current + 1))}
              disabled={pageIndex >= pageCount - 1}
              aria-label={t("library.table.nextPage")}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      <Sheet open={Boolean(selectedRecord)} onOpenChange={(open) => (!open ? setSelectedRecordId("") : undefined)}>
        <SheetContent
          side="right"
          showCloseButton={false}
          className="flex h-full w-[420px] flex-col gap-0 overflow-hidden p-4 sm:max-w-[420px]"
        >
          <PanelCard tone="solid" className="flex min-h-0 w-full flex-1 flex-col overflow-hidden">
            <div className="border-b border-border/70 px-4 py-3">
              <div className="space-y-2">
                <div className="min-w-0">
                  <SheetTitle className="truncate text-base font-semibold text-foreground">
                    {selectedConversationTitle ||
                      (selectedModelLabel !== "-" ? selectedModelLabel : "") ||
                      t("settings.debug.calls.detail.titleFallback")}
                  </SheetTitle>
                  <SheetDescription className="sr-only">
                    {t("settings.debug.calls.detail.description")}
                  </SheetDescription>
                </div>
                {selectedRecord ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium ${statusClass(selectedRecord.status)}`}
                    >
                      {selectedRecord.status || "-"}
                    </span>
                    <span className="truncate text-[11px] text-muted-foreground">
                      {selectedProviderLabel} / {selectedModelLabel}
                    </span>
                    {selectedRecord.threadId || selectedRecord.runId ? (
                      <Button size="compact" variant="outline" onClick={() => inspectRun(selectedRecord)}>
                        {t("settings.debug.calls.detail.inspect")}
                      </Button>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
              {!selectedRecord ? (
                <div className="text-xs text-muted-foreground">{t("settings.debug.calls.detail.empty")}</div>
              ) : isDetailLoading ? (
                <div className="text-xs">{t("common.loading")}</div>
              ) : (
                <div className="space-y-3">
                  <div className="overflow-hidden rounded-md border">
                    <div className="divide-y divide-border/70">
                      {detailItems.map((item) => (
                        <div key={item.label} className="space-y-1.5 px-4 py-3">
                          <div className="text-[11px] font-medium text-muted-foreground">{item.label}</div>
                          <div className={cn("font-mono text-[11px] text-foreground", item.className)}>{item.value}</div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {selectedRecord.errorText ? (
                    <div className="overflow-hidden rounded-md border">
                      <div className="space-y-2 px-4 py-3">
                        <div className="text-[11px] font-medium text-destructive">
                          {t("settings.debug.calls.detail.error")}
                        </div>
                        <pre className="max-h-48 overflow-auto rounded-md bg-destructive/5 p-3 font-mono text-[11px] whitespace-pre-wrap break-words text-foreground">
                          {selectedRecord.errorText}
                        </pre>
                      </div>
                    </div>
                  ) : null}

                  <div className="overflow-hidden rounded-md border">
                    <div className="space-y-0 divide-y divide-border/70">
                      <div className="space-y-2 px-4 py-3">
                        <div className="text-[11px] font-medium text-muted-foreground">
                          {t("settings.debug.calls.detail.requestPayload")}
                        </div>
                        <pre className="max-h-64 overflow-auto rounded-md bg-muted/20 p-3 font-mono text-[11px] whitespace-pre-wrap break-words">
                          {formatJsonPayload(selectedRecord.requestPayloadJson)}
                        </pre>
                      </div>
                      <div className="space-y-2 px-4 py-3">
                        <div className="text-[11px] font-medium text-muted-foreground">
                          {t("settings.debug.calls.detail.responsePayload")}
                        </div>
                        <pre className="max-h-64 overflow-auto rounded-md bg-muted/20 p-3 font-mono text-[11px] whitespace-pre-wrap break-words">
                          {formatJsonPayload(selectedRecord.responsePayloadJson)}
                        </pre>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </PanelCard>
        </SheetContent>
      </Sheet>
    </>
  );
}
