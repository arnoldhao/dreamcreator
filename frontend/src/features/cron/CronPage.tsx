import * as React from "react";
import {
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Circle,
  History,
  LayoutGrid,
  List,
  ListChecks,
  Loader2,
  MinusCircle,
  Play,
  Plus,
  RefreshCcw,
  Search,
  SlidersHorizontal,
  Trash2,
  XCircle,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { subscribeGatewayEvents, type GatewayEvent } from "@/shared/realtime";
import {
  useAddCronJob,
  useCronJobs,
  useCronRunDetail,
  useCronRuns,
  useCronStatus,
  useRemoveCronJob,
  useRunCronJob,
  useUpdateCronJob,
} from "@/shared/query/cron";
import { useAssistants } from "@/shared/query/assistant";
import { useChannels } from "@/shared/query/channels";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import { useChatRuntimeStore } from "@/shared/store/chat-runtime";
import type {
  CronCreateRequest,
  CronDelivery,
  CronDeliveryMode,
  CronJob,
  CronPatchRequest,
  CronPayloadKind,
  CronRunEvent,
  CronRunRecord,
  CronScheduleType,
  CronSessionTarget,
  CronUpdateRequest,
  CronWakeMode,
} from "@/shared/store/cron";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  DASHBOARD_CONTROL_GROUP_CLASS,
  DASHBOARD_FIELD_SURFACE_CLASS,
  DASHBOARD_PANEL_SURFACE_CLASS,
  DASHBOARD_SOFT_SURFACE_CLASS,
  PanelCard,
} from "@/shared/ui/dashboard";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";

import { type CronTab, useCronViewStore } from "./model/viewStore";
import { CronOverviewPage } from "./pages/CronOverviewPage";
import { CronListPage } from "./pages/CronListPage";
import { CronExecutionRecordPage } from "./pages/CronExecutionRecordPage";

type EveryUnit = "minutes" | "hours" | "days";
type StaggerUnit = "seconds" | "minutes";
type DialogMode = "create" | "edit" | "view";
type JobEnabledFilter = "" | "enabled" | "disabled";

type JobDraft = {
  id: string;
  name: string;
  description: string;
  assistantId: string;
  enabled: boolean;
  deleteAfterRun: boolean;
  scheduleKind: CronScheduleType;
  scheduleAt: string;
  everyAmount: string;
  everyUnit: EveryUnit;
  cronExpr: string;
  cronTz: string;
  scheduleExact: boolean;
  staggerAmount: string;
  staggerUnit: StaggerUnit;
  sessionTarget: CronSessionTarget;
  wakeMode: CronWakeMode;
  payloadKind: CronPayloadKind;
  payloadText: string;
  payloadModel: string;
  payloadThinking: string;
  timeoutSeconds: string;
  deliveryMode: CronDeliveryMode;
  deliveryChannel: string;
  deliveryTo: string;
  deliveryBestEffort: boolean;
  deliveryFailureMode: "" | "announce" | "webhook";
  deliveryFailureChannel: string;
  deliveryFailureTo: string;
  deliveryFailureAccountId: string;
  createdAt?: string;
  updatedAt?: string;
};

type JobsColumnID =
  | "id"
  | "name"
  | "description"
  | "schedule"
  | "nextRun"
  | "runningAt"
  | "payload"
  | "assistantId"
  | "sessionTarget"
  | "wakeMode"
  | "delivery"
  | "sourceChannel"
  | "enabled"
  | "deleteAfterRun"
  | "lastRun"
  | "status"
  | "lastDuration"
  | "consecutiveErrors"
  | "scheduleErrors"
  | "lastDeliveryStatus"
  | "lastDeliveryError"
  | "lastError"
  | "createdAt"
  | "updatedAt"
  | "ops";

type RunsColumnID =
  | "runId"
  | "started"
  | "job"
  | "status"
  | "stage"
  | "duration"
  | "ended"
  | "deliveryStatus"
  | "model"
  | "provider"
  | "sessionKey"
  | "usage"
  | "summary"
  | "deliveryError"
  | "error";

type ColumnOption = {
  id: string;
  label: string;
};

type LifecycleStepID = "trigger" | "waiting" | "delivery" | "terminal";
type LifecycleStepState = "pending" | "success" | "failed";

type LifecycleStepSpec = {
  id: LifecycleStepID;
  labelKey: string;
  fallback: string;
  stages: string[];
  pendingStages: string[];
  successStages: string[];
  failedStages: string[];
};

const isLifecycleStepID = (value: string): value is LifecycleStepID =>
  value === "trigger" || value === "waiting" || value === "delivery" || value === "terminal";

type TranslateFn = (key: string) => string;

const CRON_EMPTY_VALUE = "-";

const EMPTY_JOB_DRAFT: JobDraft = {
  id: "",
  name: "",
  description: "",
  assistantId: "",
  enabled: true,
  deleteAfterRun: false,
  scheduleKind: "every",
  scheduleAt: "",
  everyAmount: "30",
  everyUnit: "minutes",
  cronExpr: "0 7 * * *",
  cronTz: "",
  scheduleExact: false,
  staggerAmount: "",
  staggerUnit: "seconds",
  sessionTarget: "main",
  wakeMode: "now",
  payloadKind: "systemEvent",
  payloadText: "",
  payloadModel: "",
  payloadThinking: "",
  timeoutSeconds: "",
  deliveryMode: "announce",
  deliveryChannel: "default",
  deliveryTo: "",
  deliveryBestEffort: false,
  deliveryFailureMode: "",
  deliveryFailureChannel: "default",
  deliveryFailureTo: "",
  deliveryFailureAccountId: "",
};

const RUN_STATUS_OPTIONS = ["", "running", "completed", "failed", "ok", "error", "skipped"];
const PAGINATION_PAGE_SIZE_OPTIONS = [10, 20, 30, 50];
const CRON_SELECT_TEXT_CLASS = "!text-xs [&>option]:text-xs";
const OVERVIEW_GRANULARITY_MS: Record<string, number> = {
  "1m": 60 * 1000,
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "1d": 24 * 60 * 60 * 1000,
};

const JOBS_DEFAULT_VISIBILITY: Record<JobsColumnID, boolean> = {
  id: false,
  name: true,
  description: false,
  schedule: true,
  nextRun: true,
  runningAt: false,
  payload: true,
  assistantId: false,
  sessionTarget: false,
  wakeMode: false,
  delivery: false,
  sourceChannel: false,
  enabled: true,
  deleteAfterRun: false,
  lastRun: true,
  status: true,
  lastDuration: false,
  consecutiveErrors: false,
  scheduleErrors: false,
  lastDeliveryStatus: false,
  lastDeliveryError: false,
  lastError: false,
  createdAt: false,
  updatedAt: false,
  ops: true,
};

const RUNS_DEFAULT_VISIBILITY: Record<RunsColumnID, boolean> = {
  runId: false,
  started: true,
  job: true,
  status: true,
  stage: true,
  duration: true,
  ended: true,
  deliveryStatus: true,
  model: false,
  provider: false,
  sessionKey: false,
  usage: false,
  summary: true,
  deliveryError: false,
  error: false,
};

const RUN_LIFECYCLE_STEPS: LifecycleStepSpec[] = [
  {
    id: "trigger",
    labelKey: "cron.runs.lifecycleStep.trigger",
    fallback: "Trigger",
    stages: ["started", "action_accepted", "action_failed", "running"],
    pendingStages: ["started", "running"],
    successStages: ["action_accepted"],
    failedStages: ["action_failed"],
  },
  {
    id: "waiting",
    labelKey: "cron.runs.lifecycleStep.waiting",
    fallback: "Awaiting",
    stages: ["delivery_pending", "heartbeat_received", "heartbeat_timeout"],
    pendingStages: ["delivery_pending"],
    successStages: ["heartbeat_received"],
    failedStages: ["heartbeat_timeout"],
  },
  {
    id: "delivery",
    labelKey: "cron.runs.lifecycleStep.delivery",
    fallback: "Delivery",
    stages: ["delivery_attempted", "delivery_delivered", "delivery_failed"],
    pendingStages: ["delivery_attempted"],
    successStages: ["delivery_delivered"],
    failedStages: ["delivery_failed"],
  },
  {
    id: "terminal",
    labelKey: "cron.runs.lifecycleStep.terminal",
    fallback: "Terminal",
    stages: ["completed", "failed"],
    pendingStages: [],
    successStages: ["completed"],
    failedStages: ["failed"],
  },
];

const formatTemplate = (template: string, values: Record<string, string | number>) =>
  template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = values[key];
    if (value === undefined || value === null) {
      return match;
    }
    return String(value);
  });

const formatCronErrorMessage = (error: unknown): string => {
  if (error instanceof Error) {
    return error.message || String(error);
  }
  if (typeof error === "string") {
    return error;
  }
  if (error && typeof error === "object") {
    const top = error as { message?: unknown; error?: unknown };
    if (typeof top.message === "string" && top.message.trim()) {
      return top.message;
    }
    if (typeof top.error === "string" && top.error.trim()) {
      return top.error;
    }
    if (top.error && typeof top.error === "object") {
      const nested = top.error as { message?: unknown; error?: unknown };
      if (typeof nested.message === "string" && nested.message.trim()) {
        return nested.message;
      }
      if (typeof nested.error === "string" && nested.error.trim()) {
        return nested.error;
      }
    }
    try {
      return JSON.stringify(error);
    } catch {
      return String(error);
    }
  }
  return String(error);
};

const toLocalDatetimeInput = (value?: string) => {
  if (!value) {
    return "";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  if (parsed.getUTCFullYear() <= 1) {
    return "";
  }
  const year = parsed.getFullYear();
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  const hours = String(parsed.getHours()).padStart(2, "0");
  const minutes = String(parsed.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
};

const formatDateTime = (value?: string) => {
  if (!value) {
    return CRON_EMPTY_VALUE;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  if (parsed.getUTCFullYear() <= 1) {
    return CRON_EMPTY_VALUE;
  }
  return parsed.toLocaleString();
};

const formatRelativeDateTime = (value: string | undefined, language: string): string => {
  if (!value) {
    return CRON_EMPTY_VALUE;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  if (parsed.getUTCFullYear() <= 1) {
    return CRON_EMPTY_VALUE;
  }

  const diffMs = parsed.getTime() - Date.now();
  const absMs = Math.abs(diffMs);
  const minuteMs = 60 * 1000;
  const hourMs = 60 * minuteMs;
  const dayMs = 24 * hourMs;
  const monthMs = 30 * dayMs;
  const yearMs = 365 * dayMs;

  const rtf = new Intl.RelativeTimeFormat(language, { numeric: "auto" });
  if (absMs < minuteMs) {
    return rtf.format(0, "second");
  }
  if (absMs < hourMs) {
    return rtf.format(Math.round(diffMs / minuteMs), "minute");
  }
  if (absMs < dayMs) {
    return rtf.format(Math.round(diffMs / hourMs), "hour");
  }
  if (absMs < monthMs) {
    return rtf.format(Math.round(diffMs / dayMs), "day");
  }
  if (absMs < yearMs) {
    return rtf.format(Math.round(diffMs / monthMs), "month");
  }
  return rtf.format(Math.round(diffMs / yearMs), "year");
};

const normalizeEnumToken = (value?: string): string => {
  const normalized = (value ?? "").trim().toLowerCase();
  if (!normalized) {
    return "";
  }
  return normalized.replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, "");
};

const statusDotClass = (status: string): string => {
  const normalized = status.trim().toLowerCase();
  if (normalized === "completed" || normalized === "ok") {
    return "bg-emerald-500";
  }
  if (normalized === "running") {
    return "bg-amber-500";
  }
  if (normalized === "failed" || normalized === "error") {
    return "bg-rose-500";
  }
  if (normalized === "skipped") {
    return "bg-slate-400";
  }
  return "bg-muted-foreground";
};

const deliveryDotClass = (status: string): string => {
  const normalized = status.trim().toLowerCase();
  if (normalized === "delivered" || normalized === "ok") {
    return "bg-emerald-500";
  }
  if (normalized === "failed" || normalized === "error" || normalized === "not-delivered") {
    return "bg-rose-500";
  }
  if (normalized === "pending" || normalized === "unknown") {
    return "bg-amber-500";
  }
  if (normalized === "not-requested") {
    return "bg-slate-400";
  }
  return "bg-muted-foreground";
};

const resolveRunStatusLabel = (status: string | undefined, t: TranslateFn): string => {
  const raw = (status ?? "").trim();
  if (!raw) {
    return CRON_EMPTY_VALUE;
  }
  const key = normalizeEnumToken(raw);
  if (!key) {
    return raw;
  }
  return t(`cron.runs.statusValue.${key}`);
};

const resolveRunStageValueLabel = (stage: string | undefined, t: TranslateFn): string => {
  const raw = (stage ?? "").trim();
  if (!raw) {
    return CRON_EMPTY_VALUE;
  }
  const key = normalizeEnumToken(raw);
  if (!key) {
    return raw;
  }
  return t(`cron.runs.stageValue.${key}`);
};

const resolveRunDeliveryStatusLabel = (status: string | undefined, t: TranslateFn): string => {
  const raw = (status ?? "").trim();
  if (!raw) {
    return CRON_EMPTY_VALUE;
  }
  const key = normalizeEnumToken(raw);
  if (!key) {
    return raw;
  }
  return t(`cron.runs.deliveryStatusValue.${key}`);
};

const resolveRunDurationSeconds = (run: CronRunRecord): number | null => {
  if (!run.startedAt || !run.endedAt) {
    return null;
  }
  const started = new Date(run.startedAt).getTime();
  const ended = new Date(run.endedAt).getTime();
  if (!Number.isFinite(started) || !Number.isFinite(ended) || ended < started) {
    return null;
  }
  return Math.max(0, Math.round((ended - started) / 1000));
};

const formatRunDurationText = (run: CronRunRecord, t: TranslateFn): string => {
  const seconds = resolveRunDurationSeconds(run);
  if (seconds === null) {
    return CRON_EMPTY_VALUE;
  }
  return formatTemplate(t("cron.runs.durationSeconds"), { count: seconds });
};

const formatDurationMsShort = (value?: number): string => {
  if (!value || value <= 0) {
    return CRON_EMPTY_VALUE;
  }
  const totalSeconds = Math.max(1, Math.round(value / 1000));
  if (totalSeconds < 60) {
    return `${totalSeconds}s`;
  }
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes < 60) {
    return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
};

type RunDateValueRenderOptions = {
  adaptiveWhenOverflow?: boolean;
  language?: string;
};

function AdaptiveRunDateValue({
  value,
  language,
}: {
  value?: string;
  language: string;
}) {
  const containerRef = React.useRef<HTMLSpanElement | null>(null);
  const preciseMeasureRef = React.useRef<HTMLSpanElement | null>(null);
  const [isCompressed, setIsCompressed] = React.useState(false);

  const preciseText = React.useMemo(() => formatDateTime(value), [value]);
  const relativeText = React.useMemo(() => formatRelativeDateTime(value, language), [value, language]);

  const syncCompressionState = React.useCallback(() => {
    const container = containerRef.current;
    const measure = preciseMeasureRef.current;
    if (!container || !measure) {
      return;
    }
    const availableWidth = container.clientWidth;
    const requiredWidth = measure.scrollWidth;
    setIsCompressed(requiredWidth - availableWidth > 1);
  }, []);

  React.useLayoutEffect(() => {
    syncCompressionState();
    const container = containerRef.current;
    if (!container || typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(() => {
      syncCompressionState();
    });
    observer.observe(container);
    return () => observer.disconnect();
  }, [syncCompressionState, preciseText]);

  return (
    <span
      ref={containerRef}
      className="relative block w-full overflow-hidden select-none whitespace-nowrap font-mono text-xs text-muted-foreground"
      title={preciseText}
    >
      {isCompressed ? relativeText : preciseText}
      <span ref={preciseMeasureRef} className="pointer-events-none absolute invisible whitespace-nowrap font-mono text-xs">
        {preciseText}
      </span>
    </span>
  );
}

const renderRunDateValue = (value?: string, options?: RunDateValueRenderOptions) => {
  if (options?.adaptiveWhenOverflow && options.language?.trim()) {
    return <AdaptiveRunDateValue value={value} language={options.language} />;
  }
  return (
    <span className="select-none whitespace-nowrap font-mono text-xs text-muted-foreground">
      {formatDateTime(value)}
    </span>
  );
};

const renderRunCodeValue = (value?: string) => (
  <Badge
    variant="outline"
    className="max-w-full select-none truncate px-2 py-0.5 font-mono text-[11px] font-normal"
  >
    {(value ?? "").trim() || CRON_EMPTY_VALUE}
  </Badge>
);

const renderRunTextValue = (
  value: string | undefined,
  opts?: { tone?: "default" | "muted" | "destructive" }
) => {
  const tone = opts?.tone ?? "default";
  return (
    <span
      className={cn(
        "block w-full select-none truncate whitespace-nowrap text-xs",
        tone === "muted" && "text-muted-foreground",
        tone === "destructive" && "text-destructive",
        tone === "default" && "text-foreground"
      )}
    >
      {(value ?? "").trim() || CRON_EMPTY_VALUE}
    </span>
  );
};

const renderRunPlainTextValue = (value: string | undefined) => (
  <span className="block w-full select-none truncate whitespace-nowrap text-xs text-foreground">
    {(value ?? "").trim() || CRON_EMPTY_VALUE}
  </span>
);

const renderRunStatusValue = (status: string | undefined, t: TranslateFn) => {
  const raw = (status ?? "").trim();
  const label = resolveRunStatusLabel(status, t);
  return (
    <Badge
      variant="outline"
      className="inline-flex select-none items-center gap-1.5 border-border/70 bg-transparent text-xs font-normal text-foreground"
    >
      <span className={cn("h-2 w-2 shrink-0 rounded-full", statusDotClass(raw))} />
      {label}
    </Badge>
  );
};

const renderRunStatusIconValue = (status: string | undefined) => {
  const normalized = normalizeEnumToken(status);
  let icon: React.ReactNode = <Circle className="h-3.5 w-3.5 text-muted-foreground" />;

  if (normalized === "completed" || normalized === "ok") {
    icon = <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />;
  } else if (normalized === "running") {
    icon = <Loader2 className="h-3.5 w-3.5 animate-spin text-amber-500" />;
  } else if (normalized === "failed" || normalized === "error") {
    icon = <XCircle className="h-3.5 w-3.5 text-rose-500" />;
  } else if (normalized === "skipped") {
    icon = <MinusCircle className="h-3.5 w-3.5 text-slate-400" />;
  }

  return <span className="inline-flex h-5 w-5 select-none items-center justify-center">{icon}</span>;
};

const renderRunDeliveryStatusValue = (status: string | undefined, t: TranslateFn) => {
  const raw = (status ?? "").trim();
  const label = resolveRunDeliveryStatusLabel(status, t);
  return (
    <span className="inline-flex max-w-full select-none items-center gap-1.5 truncate whitespace-nowrap text-xs text-muted-foreground">
      <span className={cn("h-2 w-2 shrink-0 rounded-full", deliveryDotClass(raw))} />
      {label}
    </span>
  );
};

const renderRunStageValue = (run: CronRunRecord, t: TranslateFn) => {
  const stageValue = runStageLabel(run);
  return (
    <span className="block w-full select-none truncate whitespace-nowrap text-xs text-muted-foreground">
      {resolveRunStageValueLabel(stageValue, t)}
    </span>
  );
};

const renderRunDurationValue = (run: CronRunRecord, t: TranslateFn) => {
  const seconds = resolveRunDurationSeconds(run);
  if (seconds === null) {
    return renderRunCodeValue(undefined);
  }
  return (
    <Badge variant="secondary" className="select-none font-mono text-[11px] font-normal">
      {formatTemplate(t("cron.runs.durationSeconds"), { count: seconds })}
    </Badge>
  );
};

const summarizeUsage = (usageJson?: string): string => {
  const raw = (usageJson ?? "").trim();
  if (!raw) {
    return CRON_EMPTY_VALUE;
  }
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const totalTokens = Number(parsed.total_tokens ?? parsed.totalTokens ?? 0);
    const inputTokens = Number(parsed.input_tokens ?? parsed.inputTokens ?? 0);
    const outputTokens = Number(parsed.output_tokens ?? parsed.outputTokens ?? 0);
    if (Number.isFinite(totalTokens) && totalTokens > 0) {
      return `${Math.floor(totalTokens)} tok`;
    }
    if ((Number.isFinite(inputTokens) && inputTokens > 0) || (Number.isFinite(outputTokens) && outputTokens > 0)) {
      return `${Math.max(0, Math.floor(inputTokens))}/${Math.max(0, Math.floor(outputTokens))}`;
    }
    return raw;
  } catch {
    return raw;
  }
};

const runStageLabel = (run: CronRunRecord): string => {
  const stage = run.latestStage?.trim();
  if (stage) {
    return stage;
  }
  const deliveryStatus = run.deliveryStatus?.trim().toLowerCase();
  if (deliveryStatus === "pending") {
    return "delivery_pending";
  }
  if (deliveryStatus === "delivered" || deliveryStatus === "ok") {
    return "delivery_delivered";
  }
  if (deliveryStatus === "failed") {
    return "delivery_failed";
  }
  const status = run.status?.trim().toLowerCase();
  if (status === "running") {
    return "running";
  }
  if (status === "completed" || status === "ok") {
    return "completed";
  }
  if (status === "failed" || status === "error") {
    return "failed";
  }
  return status || CRON_EMPTY_VALUE;
};

const resolveLifecycleStepByStage = (stage: string | undefined): LifecycleStepID | "" => {
  const token = normalizeEnumToken(stage);
  switch (token) {
    case "started":
    case "action_accepted":
    case "action_failed":
    case "running":
      return "trigger";
    case "delivery_pending":
    case "heartbeat_received":
    case "heartbeat_timeout":
      return "waiting";
    case "delivery_attempted":
    case "delivery_delivered":
    case "delivery_failed":
      return "delivery";
    case "completed":
    case "failed":
      return "terminal";
    default:
      return "";
  }
};

const isRunFailureStatus = (status: string | undefined): boolean => {
  const token = normalizeEnumToken(status);
  return token === "failed" || token === "error";
};

const isRunTerminalStatus = (status: string | undefined): boolean => {
  const token = normalizeEnumToken(status);
  return token === "completed" || token === "ok" || token === "failed" || token === "error" || token === "skipped";
};

const isRunWaitingDelivery = (run: CronRunRecord | null): boolean => {
  if (!run) {
    return false;
  }
  const status = normalizeEnumToken(run.status);
  const deliveryStatus = normalizeEnumToken(run.deliveryStatus);
  return status === "running" || deliveryStatus === "pending";
};

const resolveGatewayEventRunID = (event: GatewayEvent): string => {
  const direct = (event.runId ?? "").trim();
  if (direct) {
    return direct;
  }
  const payload = event.payload;
  if (!payload || typeof payload !== "object") {
    return "";
  }
  const fromPayload = (payload as { runId?: unknown }).runId;
  if (typeof fromPayload !== "string") {
    return "";
  }
  return fromPayload.trim();
};

const formatEveryInterval = (everyMs: number, t: TranslateFn): string => {
  if (everyMs <= 0) {
    return CRON_EMPTY_VALUE;
  }
  const day = 24 * 60 * 60 * 1000;
  const hour = 60 * 60 * 1000;
  const minute = 60 * 1000;
  const second = 1000;

  let amount = everyMs;
  let unitKey: "days" | "hours" | "minutes" | "seconds" | "milliseconds" = "milliseconds";
  if (everyMs % day === 0) {
    amount = everyMs / day;
    unitKey = "days";
  } else if (everyMs % hour === 0) {
    amount = everyMs / hour;
    unitKey = "hours";
  } else if (everyMs % minute === 0) {
    amount = everyMs / minute;
    unitKey = "minutes";
  } else if (everyMs % second === 0) {
    amount = everyMs / second;
    unitKey = "seconds";
  }

  const unit = t(`cron.scheduleValue.unit.${unitKey}`);
  return formatTemplate(t("cron.scheduleValue.interval"), {
    count: amount,
    unit,
  });
};

const formatSchedule = (job: CronJob, t: TranslateFn) => {
  const kind = job.schedule.kind;
  if (kind === "every") {
    const everyMs = job.schedule.everyMs ?? 0;
    const interval = formatEveryInterval(everyMs, t);
    if (interval === CRON_EMPTY_VALUE) {
      return CRON_EMPTY_VALUE;
    }
    return formatTemplate(t("cron.scheduleValue.every"), { interval });
  }
  if (kind === "at") {
    return formatTemplate(t("cron.scheduleValue.at"), {
      value: formatDateTime(job.schedule.at),
    });
  }
  const expr = job.schedule.expr ?? "";
  return expr.trim() || CRON_EMPTY_VALUE;
};

const formatPayloadSummary = (job: CronJob, t: (key: string) => string) => {
  const kind = job.payload?.kind ?? "systemEvent";
  if (kind === "agentTurn") {
    return t("cron.payload.agentTurn");
  }
  return t("cron.payload.systemEvent");
};

const normalizeSourceChannel = (sourceChannel?: string) => {
  const normalized = (sourceChannel ?? "").trim().toLowerCase();
  if (!normalized) {
    return "";
  }
  if (normalized === "aui") {
    return "app";
  }
  return normalized;
};

const formatDeliveryChannelLabel = (
  channel: string | undefined,
  sourceChannel: string | undefined,
  t: (key: string) => string
) => {
  const normalized = channel?.trim() || "default";
  if (normalized === "default") {
    const resolved = normalizeSourceChannel(sourceChannel);
    if (resolved) {
      return resolved;
    }
    return t("cron.form.deliveryChannelDefault");
  }
  return normalized;
};

const formatDeliverySummary = (
  delivery: CronDelivery | undefined,
  sourceChannel: string | undefined,
  t: (key: string) => string
) => {
  if (!delivery) {
    return "none";
  }
  const rawMode = (delivery.mode ?? "").trim().toLowerCase();
  if (rawMode === "none") {
    return "none";
  }
  if (rawMode === "webhook") {
    return delivery.to ? `webhook (${delivery.to})` : "webhook";
  }
  const channel = formatDeliveryChannelLabel(delivery.channel, sourceChannel, t);
  const failureMode = (delivery.failureDestination?.mode ?? "").trim().toLowerCase();
  const failureText = failureMode ? `, failure:${failureMode}` : "";
  if (delivery.to?.trim()) {
    return `announce (${channel} -> ${delivery.to.trim()}${failureText})`;
  }
  return `announce (${channel}${failureText})`;
};

const normalizeFailureMode = (value?: string): JobDraft["deliveryFailureMode"] => {
  const normalized = (value ?? "").trim().toLowerCase();
  if (normalized === "announce" || normalized === "webhook") {
    return normalized;
  }
  return "";
};

const normalizeAnnounceChannelValue = (value: string): "default" | "app" | "telegram" | undefined => {
  const normalized = value.trim().toLowerCase();
  if (normalized === "default" || normalized === "app" || normalized === "telegram") {
    return normalized;
  }
  return undefined;
};

const parseEveryDraft = (everyMs: number): Pick<JobDraft, "everyAmount" | "everyUnit"> => {
  const day = 24 * 60 * 60 * 1000;
  const hour = 60 * 60 * 1000;
  const minute = 60 * 1000;
  if (everyMs > 0 && everyMs % day === 0) {
    return { everyAmount: String(everyMs / day), everyUnit: "days" };
  }
  if (everyMs > 0 && everyMs % hour === 0) {
    return { everyAmount: String(everyMs / hour), everyUnit: "hours" };
  }
  const minutes = Math.max(1, Math.ceil(everyMs / minute));
  return { everyAmount: String(minutes), everyUnit: "minutes" };
};

const parseStaggerDraft = (staggerMs?: number): Pick<JobDraft, "scheduleExact" | "staggerAmount" | "staggerUnit"> => {
  if (staggerMs === 0) {
    return { scheduleExact: true, staggerAmount: "", staggerUnit: "seconds" };
  }
  if (!staggerMs || staggerMs < 0) {
    return { scheduleExact: false, staggerAmount: "", staggerUnit: "seconds" };
  }
  if (staggerMs % (60 * 1000) === 0) {
    return {
      scheduleExact: false,
      staggerAmount: String(Math.max(1, staggerMs / (60 * 1000))),
      staggerUnit: "minutes",
    };
  }
  return {
    scheduleExact: false,
    staggerAmount: String(Math.max(1, Math.ceil(staggerMs / 1000))),
    staggerUnit: "seconds",
  };
};

const toDraft = (job?: CronJob | null): JobDraft => {
  if (!job) {
    return { ...EMPTY_JOB_DRAFT };
  }

  const scheduleKind = job.schedule.kind as CronScheduleType;
  const payloadKind = job.payload.kind as CronPayloadKind;
  const everyMs = job.schedule.everyMs ?? 0;
  const every = parseEveryDraft(everyMs);
  const stagger = parseStaggerDraft(job.schedule.staggerMs);
  const rawDeliveryMode = (job.delivery?.mode ?? "none").trim().toLowerCase();
  const normalizedDeliveryMode: CronDeliveryMode =
    rawDeliveryMode === "webhook" ? "webhook" : rawDeliveryMode === "none" ? "none" : "announce";
  const failureMode = normalizeFailureMode(job.delivery?.failureDestination?.mode);

  return {
    id: job.id,
    name: job.name.trim(),
    description: job.description?.trim() || "",
    assistantId: job.assistantId.trim(),
    enabled: job.enabled,
    deleteAfterRun: Boolean(job.deleteAfterRun),
    scheduleKind,
    scheduleAt: toLocalDatetimeInput(job.schedule.at),
    everyAmount: every.everyAmount,
    everyUnit: every.everyUnit,
    cronExpr: job.schedule.expr || "",
    cronTz: job.schedule.tz || "",
    scheduleExact: stagger.scheduleExact,
    staggerAmount: stagger.staggerAmount,
    staggerUnit: stagger.staggerUnit,
    sessionTarget: job.sessionTarget,
    wakeMode: job.wakeMode,
    payloadKind,
    payloadText: payloadKind === "agentTurn" ? job.payload.message || "" : job.payload.text || "",
    payloadModel: job.payload.model || "",
    payloadThinking: job.payload.thinking || "",
    timeoutSeconds:
      payloadKind === "agentTurn" && typeof job.payload.timeoutSeconds === "number"
        ? String(job.payload.timeoutSeconds)
        : "",
    deliveryMode: normalizedDeliveryMode,
    deliveryChannel: job.delivery?.channel || "default",
    deliveryTo: job.delivery?.to || "",
    deliveryBestEffort: Boolean(job.delivery?.bestEffort),
    deliveryFailureMode: failureMode,
    deliveryFailureChannel: job.delivery?.failureDestination?.channel || "default",
    deliveryFailureTo: job.delivery?.failureDestination?.to || "",
    deliveryFailureAccountId: job.delivery?.failureDestination?.accountId || "",
    createdAt: job.createdAt,
    updatedAt: job.updatedAt,
  };
};

type CronRequestParts = Omit<CronCreateRequest, "id" | "assistantId" | "name" | "description" | "enabled" | "deleteAfterRun">;

const toRequestParts = (draft: JobDraft): CronRequestParts => {
  const everyAmount = Number(draft.everyAmount);
  const everyUnitMs =
    draft.everyUnit === "days" ? 24 * 60 * 60 * 1000 : draft.everyUnit === "hours" ? 60 * 60 * 1000 : 60 * 1000;
  const everyMs = Number.isFinite(everyAmount) && everyAmount > 0 ? Math.round(everyAmount * everyUnitMs) : 0;

  const staggerAmount = Number(draft.staggerAmount);
  const staggerUnitMs = draft.staggerUnit === "minutes" ? 60 * 1000 : 1000;
  const staggerMs =
    draft.scheduleExact
      ? 0
      : Number.isFinite(staggerAmount) && staggerAmount > 0
          ? Math.round(staggerAmount * staggerUnitMs)
          : undefined;

  const atISO = draft.scheduleAt.trim() ? new Date(draft.scheduleAt).toISOString() : "";

  const schedule =
    draft.scheduleKind === "every"
      ? {
          kind: "every" as const,
          everyMs,
        }
      : draft.scheduleKind === "at"
          ? {
              kind: "at" as const,
              at: atISO,
            }
          : {
          kind: "cron" as const,
          expr: draft.cronExpr.trim(),
          tz: draft.cronTz.trim() || undefined,
          staggerMs,
        };

  const payload =
    draft.payloadKind === "agentTurn"
      ? {
          kind: "agentTurn" as const,
          message: draft.payloadText.trim(),
          model: draft.payloadModel.trim() || undefined,
          thinking: draft.payloadThinking.trim() || undefined,
          timeoutSeconds: (() => {
            const parsedTimeout = Number(draft.timeoutSeconds);
            if (!draft.timeoutSeconds.trim() || !Number.isFinite(parsedTimeout) || parsedTimeout <= 0) {
              return undefined;
            }
            return Math.max(1, Math.floor(parsedTimeout));
          })(),
        }
      : {
          kind: "systemEvent" as const,
          text: draft.payloadText.trim(),
        };

  let delivery: CronDelivery | undefined;
  if (draft.deliveryMode === "none") {
    delivery = {
      mode: "none",
    };
  } else {
    delivery = {
      mode: draft.deliveryMode,
      channel: draft.deliveryMode === "announce" ? draft.deliveryChannel.trim() || "default" : undefined,
      to: draft.deliveryTo.trim() || undefined,
      bestEffort: draft.deliveryBestEffort,
    };
    if (draft.deliveryFailureMode) {
      delivery.failureDestination = {
        mode: draft.deliveryFailureMode,
        channel:
          draft.deliveryFailureMode === "announce"
            ? normalizeAnnounceChannelValue(draft.deliveryFailureChannel) || "default"
            : undefined,
        to: draft.deliveryFailureTo.trim() || undefined,
        accountId: draft.deliveryFailureAccountId.trim() || undefined,
      };
    }
  }

  return {
    schedule,
    sessionTarget: draft.sessionTarget,
    wakeMode: draft.wakeMode,
    payload,
    delivery,
    sessionKey: draft.sessionTarget === "main" ? "cron/main" : "cron/isolated",
  };
};

const toCreatePayload = (draft: JobDraft): CronCreateRequest => {
  const request = toRequestParts(draft);
  return {
    id: draft.id.trim() || undefined,
    assistantId: draft.assistantId.trim(),
    name: draft.name.trim(),
    description: draft.description.trim() || undefined,
    enabled: draft.enabled,
    deleteAfterRun: draft.deleteAfterRun,
    ...request,
  };
};

const toUpdatePayload = (draft: JobDraft, current: CronJob): CronUpdateRequest => {
  const request = toRequestParts(draft);
  const patch: CronPatchRequest = {
    assistantId: draft.assistantId.trim(),
    name: draft.name.trim(),
    description: draft.description.trim() || undefined,
    enabled: draft.enabled,
    deleteAfterRun: draft.deleteAfterRun,
    ...request,
  };
  return {
    id: current.id,
    patch,
  };
};

const validateDraft = (draft: JobDraft, t: (key: string) => string) => {
  if (!draft.name.trim()) {
    return t("cron.form.nameRequired");
  }
  if (draft.scheduleKind === "at") {
    if (!draft.scheduleAt.trim()) {
      return t("cron.form.atRequired");
    }
    if (Number.isNaN(new Date(draft.scheduleAt).getTime())) {
      return t("cron.form.atInvalid");
    }
  }
  if (draft.scheduleKind === "every") {
    const amount = Number(draft.everyAmount);
    if (!Number.isFinite(amount) || amount <= 0) {
      return t("cron.form.everyRequired");
    }
  }
  if (draft.scheduleKind === "cron") {
    if (!draft.cronExpr.trim()) {
      return t("cron.form.cronExprRequired");
    }
    if (!draft.scheduleExact && draft.staggerAmount.trim()) {
      const stagger = Number(draft.staggerAmount);
      if (!Number.isFinite(stagger) || stagger <= 0) {
        return t("cron.form.staggerInvalid");
      }
    }
  }
  if (!draft.payloadText.trim()) {
    return draft.payloadKind === "systemEvent"
      ? t("cron.form.payloadTextRequired")
      : t("cron.form.payloadMessageRequired");
  }
  if (draft.payloadKind === "agentTurn" && draft.timeoutSeconds.trim()) {
    const timeout = Number(draft.timeoutSeconds);
    if (!Number.isFinite(timeout) || timeout <= 0 || Math.floor(timeout) < 1) {
      return t("cron.form.timeoutInvalid");
    }
  }
  if (draft.sessionTarget === "main" && draft.payloadKind !== "systemEvent") {
    return t("cron.form.mainTargetRequiresSystemEvent");
  }
  if (draft.sessionTarget === "isolated" && draft.payloadKind !== "agentTurn") {
    return t("cron.form.isolatedTargetRequiresAgentTurn");
  }
  if (draft.deliveryMode === "webhook") {
    const target = draft.deliveryTo.trim();
    if (!target) {
      return t("cron.form.deliveryToRequired");
    }
    if (!/^https?:\/\//i.test(target)) {
      return t("cron.form.deliveryToInvalid");
    }
  }
  if (draft.deliveryMode !== "none" && draft.deliveryFailureMode === "announce") {
    const channel = draft.deliveryFailureChannel.trim().toLowerCase();
    if (channel && channel !== "default" && channel !== "app" && channel !== "telegram") {
      return t("cron.form.deliveryFailureChannelInvalid");
    }
  }
  if (draft.deliveryMode !== "none" && draft.deliveryFailureMode === "webhook") {
    const target = draft.deliveryFailureTo.trim();
    if (!target) {
      return t("cron.form.deliveryFailureToRequired");
    }
    if (!/^https?:\/\//i.test(target)) {
      return t("cron.form.deliveryFailureToInvalid");
    }
  }
  return "";
};

function renderSectionLabel(label: string) {
  return <label className="text-xs text-muted-foreground">{label}</label>;
}

export function CronPage() {
  const { t, language } = useI18n();
  const gatewayAssistantId = useChatRuntimeStore((state) => state.assistantId);
  const rawActiveTab = useCronViewStore((state) => state.activeTab);
  const jobsViewMode = useCronViewStore((state) => state.jobsViewMode);
  const runsViewMode = useCronViewStore((state) => state.runsViewMode);
  const jobsRowsPerPage = useCronViewStore((state) => state.jobsRowsPerPage);
  const runsRowsPerPage = useCronViewStore((state) => state.runsRowsPerPage);
  const columnVisibility = useCronViewStore((state) => state.columnVisibility);
  const setActiveTab = useCronViewStore((state) => state.setActiveTab);
  const setJobsViewMode = useCronViewStore((state) => state.setJobsViewMode);
  const setRunsViewMode = useCronViewStore((state) => state.setRunsViewMode);
  const setJobsRowsPerPage = useCronViewStore((state) => state.setJobsRowsPerPage);
  const setRunsRowsPerPage = useCronViewStore((state) => state.setRunsRowsPerPage);
  const setColumnVisibility = useCronViewStore((state) => state.setColumnVisibility);

  const [jobSearchQuery, setJobSearchQuery] = React.useState("");
  const [jobEnabledFilter, setJobEnabledFilter] = React.useState<JobEnabledFilter>("");
  const [jobLastRunStatusFilter, setJobLastRunStatusFilter] = React.useState("");
  const [jobSelectionMode, setJobSelectionMode] = React.useState(false);
  const [selectedJobSelection, setSelectedJobSelection] = React.useState<Record<string, boolean>>({});
  const [batchDeleteJobsPending, setBatchDeleteJobsPending] = React.useState(false);
  const [runSearchQuery, setRunSearchQuery] = React.useState("");
  const [jobFilter, setJobFilter] = React.useState<string>("");
  const [runStatusFilter, setRunStatusFilter] = React.useState<string>("");
  const [jobsPage, setJobsPage] = React.useState(1);
  const [runsPage, setRunsPage] = React.useState(1);

  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [dialogSaving, setDialogSaving] = React.useState(false);
  const [dialogMode, setDialogMode] = React.useState<DialogMode>("create");
  const [editingJob, setEditingJob] = React.useState<CronJob | null>(null);
  const [draft, setDraft] = React.useState<JobDraft>({ ...EMPTY_JOB_DRAFT });
  const [runDetailOpen, setRunDetailOpen] = React.useState(false);
  const [selectedRunID, setSelectedRunID] = React.useState<string>("");
  const [selectedLifecycleStepID, setSelectedLifecycleStepID] = React.useState<LifecycleStepID>("trigger");
  const [expandedLifecycleEvents, setExpandedLifecycleEvents] = React.useState<Record<string, boolean>>({});
  const [chartGranularity, setChartGranularity] = React.useState("1h");
  const lastLifecycleRunIDRef = React.useRef<string>("");
  const refreshAnimationTimeoutRef = React.useRef<number | null>(null);
  const autoRefreshDebounceRef = React.useRef<number | null>(null);
  const [refreshAnimating, setRefreshAnimating] = React.useState(false);
  const isDialogView = dialogMode === "view";
  const activeTab: CronTab =
    rawActiveTab === "overview" || rawActiveTab === "list" || rawActiveTab === "records"
      ? rawActiveTab
      : "overview";

  React.useEffect(() => {
    if (rawActiveTab !== activeTab) {
      setActiveTab(activeTab);
    }
  }, [rawActiveTab, activeTab, setActiveTab]);

  const statusQuery = useCronStatus();
  const jobsQuery = useCronJobs({ includeDisabled: true });
  const assistantsQuery = useAssistants(true);
  const channelsQuery = useChannels();
  const providersQuery = useEnabledProvidersWithModels();
  const runsQuery = useCronRuns({
    scope: jobFilter ? "job" : "all",
    id: jobFilter || undefined,
    statuses: runStatusFilter ? [runStatusFilter] : undefined,
    query: runSearchQuery.trim() || undefined,
    limit: runsRowsPerPage,
    offset: (runsPage - 1) * runsRowsPerPage,
  });
  const overviewRunsQuery = useCronRuns({
    scope: "all",
    limit: 600,
    offset: 0,
    sortDir: "desc",
  });
  const runDetailQuery = useCronRunDetail(
    {
      runId: selectedRunID,
      eventsLimit: 300,
    },
    {
      enabled: runDetailOpen && selectedRunID.trim().length > 0,
    }
  );

  const addJob = useAddCronJob();
  const updateJob = useUpdateCronJob();
  const removeJob = useRemoveCronJob();
  const runJob = useRunCronJob();

  const jobs = jobsQuery.data ?? [];
  const runs = runsQuery.data?.items ?? [];
  const selectedRunDetail = runDetailQuery.data;
  const selectedRunFallback = React.useMemo(
    () => runs.find((item) => item.runId === selectedRunID),
    [runs, selectedRunID]
  );
  const runDetailRecord = selectedRunDetail?.run ?? selectedRunFallback ?? null;
  const runDetailEvents = React.useMemo(() => {
    const items = [...(selectedRunDetail?.events ?? [])];
    items.sort((left, right) => {
      const leftTime = new Date(left.createdAt).getTime();
      const rightTime = new Date(right.createdAt).getTime();
      if (Number.isFinite(leftTime) && Number.isFinite(rightTime)) {
        return leftTime - rightTime;
      }
      return String(left.createdAt).localeCompare(String(right.createdAt));
    });
    return items;
  }, [selectedRunDetail]);
  const runDetailSessionKey = runDetailRecord?.sessionKey?.trim() || CRON_EMPTY_VALUE;
  const runDetailStageLabel = runDetailRecord ? resolveRunStageValueLabel(runStageLabel(runDetailRecord), t) : CRON_EMPTY_VALUE;
  const runDetailDurationText = runDetailRecord ? formatRunDurationText(runDetailRecord, t) : CRON_EMPTY_VALUE;
  const runDetailIsTerminal = isRunTerminalStatus(runDetailRecord?.status);
  const runDetailIsLive = !runDetailIsTerminal && isRunWaitingDelivery(runDetailRecord);
  const runDetailLifecycleStatusText = runDetailIsLive
    ? t("cron.runs.detailLive")
    : runDetailQuery.isFetching
    ? t("cron.runs.detailRefreshing")
    : t("cron.runs.detailStable");
  const runDetailEventsByStep = React.useMemo<Record<LifecycleStepID, CronRunEvent[]>>(() => {
    const grouped: Record<LifecycleStepID, CronRunEvent[]> = {
      trigger: [],
      waiting: [],
      delivery: [],
      terminal: [],
    };
    for (const event of runDetailEvents) {
      const stepID = resolveLifecycleStepByStage(event.stage);
      if (!stepID) {
        continue;
      }
      grouped[stepID].push(event);
    }
    return grouped;
  }, [runDetailEvents]);
  const runDetailActiveLifecycleStepID = React.useMemo<LifecycleStepID>(() => {
    if (runDetailRecord) {
      const stageStep = resolveLifecycleStepByStage(runStageLabel(runDetailRecord));
      if (stageStep) {
        return stageStep;
      }
    }
    if (runDetailEventsByStep.terminal.length > 0 || runDetailIsTerminal) {
      return "terminal";
    }
    if (runDetailEventsByStep.delivery.length > 0) {
      return "delivery";
    }
    if (runDetailEventsByStep.waiting.length > 0 || runDetailIsLive) {
      return "waiting";
    }
    if (runDetailEventsByStep.trigger.length > 0) {
      return "trigger";
    }
    return "trigger";
  }, [runDetailEventsByStep, runDetailIsLive, runDetailIsTerminal, runDetailRecord]);
  const lifecycleSteps = React.useMemo<
    Array<LifecycleStepSpec & { events: CronRunEvent[]; latestEvent: CronRunEvent | null; state: LifecycleStepState }>
  >(() => {
    const activeIndex = RUN_LIFECYCLE_STEPS.findIndex((item) => item.id === runDetailActiveLifecycleStepID);
    const terminalFailed = isRunFailureStatus(runDetailRecord?.status) || isRunFailureStatus(runDetailRecord?.deliveryStatus);
    return RUN_LIFECYCLE_STEPS.map((step, index) => {
      const events = runDetailEventsByStep[step.id];
      const latestEvent = events.length > 0 ? events[events.length - 1] : null;
      const latestStage = normalizeEnumToken(latestEvent?.stage);
      const stageTokens = events.map((event) => normalizeEnumToken(event.stage)).filter(Boolean);
      const hasFailedStage = stageTokens.some((token) => step.failedStages.includes(token));
      const hasSuccessStage = stageTokens.some((token) => step.successStages.includes(token));

      let state: LifecycleStepState = "pending";

      if (step.id === "terminal" && runDetailIsTerminal) {
        state = terminalFailed ? "failed" : "success";
      } else if (hasFailedStage || step.failedStages.includes(latestStage)) {
        state = "failed";
      } else if (hasSuccessStage || step.successStages.includes(latestStage)) {
        state = "success";
      } else if (index < activeIndex) {
        state = "success";
      } else {
        state = "pending";
      }

      return {
        ...step,
        events,
        latestEvent,
        state,
      };
    });
  }, [runDetailActiveLifecycleStepID, runDetailEventsByStep, runDetailIsTerminal, runDetailRecord]);
  const selectedLifecycleStep =
    lifecycleSteps.find((step) => step.id === selectedLifecycleStepID) ?? lifecycleSteps[0] ?? null;
  const runsTotal = runsQuery.data?.total ?? 0;

  const timezoneOptions = React.useMemo(() => {
    const supportedValuesOf = (Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supportedValuesOf !== "function") {
      return [] as string[];
    }
    try {
      return supportedValuesOf("timeZone").slice().sort((a, b) => a.localeCompare(b));
    } catch {
      return [] as string[];
    }
  }, []);

  const cronTimezoneOptions = React.useMemo(() => {
    const current = draft.cronTz.trim();
    if (!current) {
      return timezoneOptions;
    }
    if (timezoneOptions.includes(current)) {
      return timezoneOptions;
    }
    return [current, ...timezoneOptions];
  }, [draft.cronTz, timezoneOptions]);

  const deliveryChannelOptions = React.useMemo(() => {
    const options: Array<{ value: string; label: string }> = [
      {
        value: "default",
        label: t("cron.form.deliveryChannelDefault"),
      },
      {
        value: "app",
        label: t("cron.form.deliveryChannelApp"),
      },
    ];
    const seen = new Set<string>(["default", "app"]);
    const sortedChannels = (channelsQuery.data ?? [])
      .slice()
      .sort((a, b) => (a.displayName || a.channelId).localeCompare(b.displayName || b.channelId));
    for (const channel of sortedChannels) {
      const channelId = channel.channelId.trim();
      if (!channelId || seen.has(channelId)) {
        continue;
      }
      seen.add(channelId);
      const displayName = channel.displayName?.trim() || channelId;
      options.push({
        value: channelId,
        label: displayName === channelId ? channelId : `${displayName} (${channelId})`,
      });
    }
    const current = draft.deliveryChannel.trim();
    if (current && !seen.has(current)) {
      options.push({
        value: current,
        label: `${t("cron.form.deliveryChannelCustom")} / ${current}`,
      });
    }
    return options;
  }, [channelsQuery.data, draft.deliveryChannel, t]);

  const payloadModelOptions = React.useMemo(() => {
    const options: Array<{ value: string; label: string }> = [
      {
        value: "",
        label: t("cron.form.modelAuto"),
      },
    ];
    const seen = new Set<string>();
    for (const providerWithModels of providersQuery.data ?? []) {
      const providerID = providerWithModels.provider.id.trim();
      if (!providerID) {
        continue;
      }
      const providerLabel = providerWithModels.provider.name?.trim() || providerID;
      for (const model of providerWithModels.models ?? []) {
        if (!model.enabled) {
          continue;
        }
        const modelName = model.name.trim();
        if (!modelName) {
          continue;
        }
        const value = `${providerID}/${modelName}`;
        if (seen.has(value)) {
          continue;
        }
        seen.add(value);
        options.push({
          value,
          label: `${providerLabel} / ${model.displayName?.trim() || modelName}`,
        });
      }
    }
    const currentValue = draft.payloadModel.trim();
    if (currentValue && !seen.has(currentValue)) {
      options.push({
        value: currentValue,
        label: `${t("cron.form.modelCustom")} / ${currentValue}`,
      });
    }
    return options;
  }, [draft.payloadModel, providersQuery.data, t]);

  const payloadThinkingOptions = React.useMemo(() => {
    const options: Array<{ value: string; label: string }> = [
      {
        value: "",
        label: t("cron.form.thinkingAuto"),
      },
      { value: "off", label: t("cron.form.thinkingOption.off") },
      { value: "minimal", label: t("cron.form.thinkingOption.minimal") },
      { value: "low", label: t("cron.form.thinkingOption.low") },
      { value: "medium", label: t("cron.form.thinkingOption.medium") },
      { value: "high", label: t("cron.form.thinkingOption.high") },
      { value: "xhigh", label: t("cron.form.thinkingOption.xhigh") },
    ];
    const currentValue = draft.payloadThinking.trim();
    if (currentValue && !options.some((item) => item.value === currentValue)) {
      options.push({
        value: currentValue,
        label: `${t("cron.form.thinkingCustom")} / ${currentValue}`,
      });
    }
    return options;
  }, [draft.payloadThinking, t]);
  const createAssistantId = React.useMemo(() => {
    const fromGateway = gatewayAssistantId.trim();
    if (fromGateway) {
      return fromGateway;
    }
    const assistants = assistantsQuery.data ?? [];
    if (assistants.length === 0) {
      return "";
    }
    return assistants.find((assistant) => assistant.isDefault)?.id?.trim() || assistants[0]?.id?.trim() || "";
  }, [assistantsQuery.data, gatewayAssistantId]);

  const filteredJobs = React.useMemo(() => {
    const query = jobSearchQuery.trim().toLowerCase();
    const normalizedLastRunStatusFilter = normalizeEnumToken(jobLastRunStatusFilter);
    return jobs.filter((job) => {
      if (jobEnabledFilter === "enabled" && !job.enabled) {
        return false;
      }
      if (jobEnabledFilter === "disabled" && job.enabled) {
        return false;
      }
      if (normalizedLastRunStatusFilter) {
        const normalizedLastRunStatus = normalizeEnumToken(job.state?.lastRunStatus);
        if (normalizedLastRunStatus !== normalizedLastRunStatusFilter) {
          return false;
        }
      }
      if (!query) {
        return true;
      }
      const searchText = [
        job.id,
        job.name,
        job.description,
        job.assistantId,
        job.sourceChannel,
        job.state?.lastError,
        job.state?.lastDeliveryError,
        formatSchedule(job, t),
        formatPayloadSummary(job, t),
      ]
        .map((item) => String(item ?? "").toLowerCase())
        .join(" ");
      return searchText.includes(query);
    });
  }, [jobEnabledFilter, jobLastRunStatusFilter, jobSearchQuery, jobs, t]);
  const jobsTotalPages = Math.max(1, Math.ceil(filteredJobs.length / jobsRowsPerPage));
  const runsTotalPages = Math.max(1, Math.ceil(runsTotal / runsRowsPerPage));

  const pagedJobs = React.useMemo(() => {
    const start = (jobsPage - 1) * jobsRowsPerPage;
    return filteredJobs.slice(start, start + jobsRowsPerPage);
  }, [filteredJobs, jobsPage, jobsRowsPerPage]);
  const selectedJobIDs = React.useMemo(
    () => Object.keys(selectedJobSelection).filter((jobID) => selectedJobSelection[jobID]),
    [selectedJobSelection]
  );
  const selectedJobCount = selectedJobIDs.length;
  const pagedJobIDs = React.useMemo(
    () => pagedJobs.map((job) => job.id.trim()).filter(Boolean),
    [pagedJobs]
  );
  const selectedPagedJobCount = React.useMemo(
    () => pagedJobIDs.filter((jobID) => selectedJobSelection[jobID]).length,
    [pagedJobIDs, selectedJobSelection]
  );
  const areAllPagedJobsSelected = pagedJobIDs.length > 0 && selectedPagedJobCount === pagedJobIDs.length;
  const areSomePagedJobsSelected = selectedPagedJobCount > 0 && !areAllPagedJobsSelected;
  const jobFilterCount =
    (jobSearchQuery.trim().length > 0 ? 1 : 0) +
    (jobEnabledFilter ? 1 : 0) +
    (jobLastRunStatusFilter ? 1 : 0);
  const runFilterCount =
    (runSearchQuery.trim().length > 0 ? 1 : 0) +
    (jobFilter ? 1 : 0) +
    (runStatusFilter ? 1 : 0);

  const jobNameByID = React.useMemo(() => {
    const map = new Map<string, string>();
    jobs.forEach((job) => {
      const id = job.id.trim();
      if (!id) {
        return;
      }
      map.set(id, job.name.trim() || id);
    });
    return map;
  }, [jobs]);
  const jobScheduleByID = React.useMemo(() => {
    const map = new Map<string, string>();
    jobs.forEach((job) => {
      const id = job.id.trim();
      if (!id) {
        return;
      }
      const scheduleText = formatSchedule(job, t);
      map.set(id, scheduleText.trim() || CRON_EMPTY_VALUE);
    });
    return map;
  }, [jobs, t]);
  const assistantNameByID = React.useMemo(() => {
    const map = new Map<string, string>();
    for (const assistant of assistantsQuery.data ?? []) {
      const id = assistant.id.trim();
      if (!id) {
        continue;
      }
      const name = assistant.identity?.name?.trim() || id;
      map.set(id, name);
    }
    return map;
  }, [assistantsQuery.data]);
  const assistantWatermarkName = React.useMemo(() => {
    const currentID = draft.assistantId.trim();
    if (!currentID) {
      return CRON_EMPTY_VALUE;
    }
    return assistantNameByID.get(currentID) || currentID;
  }, [assistantNameByID, draft.assistantId]);

  React.useEffect(() => {
    if (jobsPage > jobsTotalPages) {
      setJobsPage(jobsTotalPages);
    }
  }, [jobsPage, jobsTotalPages]);

  React.useEffect(() => {
    if (runsPage > runsTotalPages) {
      setRunsPage(runsTotalPages);
    }
  }, [runsPage, runsTotalPages]);

  React.useEffect(() => {
    setJobsPage(1);
  }, [jobEnabledFilter, jobLastRunStatusFilter, jobSearchQuery, jobsRowsPerPage]);

  React.useEffect(() => {
    setRunsPage(1);
  }, [jobFilter, runSearchQuery, runStatusFilter, runsRowsPerPage]);

  React.useEffect(() => {
    if (activeTab !== "list") {
      setJobSelectionMode(false);
      setSelectedJobSelection({});
    }
  }, [activeTab]);

  React.useEffect(() => {
    if (!jobSelectionMode) {
      return;
    }
    if (jobsViewMode !== "table") {
      setJobsViewMode("table");
    }
  }, [jobSelectionMode, jobsViewMode, setJobsViewMode]);

  React.useEffect(() => {
    if (!jobSelectionMode) {
      return;
    }
    const availableJobIDs = new Set(filteredJobs.map((job) => job.id.trim()).filter(Boolean));
    setSelectedJobSelection((current) => {
      const next: Record<string, boolean> = {};
      let changed = false;
      for (const [jobID, selected] of Object.entries(current)) {
        if (!selected) {
          changed = true;
          continue;
        }
        if (!availableJobIDs.has(jobID)) {
          changed = true;
          continue;
        }
        next[jobID] = true;
      }
      if (!changed && Object.keys(next).length === Object.keys(current).length) {
        return current;
      }
      return next;
    });
  }, [filteredJobs, jobSelectionMode]);

  React.useEffect(() => {
    const runID = selectedRunID.trim();
    if (!runDetailOpen || !runID) {
      return;
    }
    const unsubscribe = subscribeGatewayEvents((event) => {
      const eventName = (event.event ?? "").trim().toLowerCase();
      if (eventName !== "cron.rundetail" && eventName !== "cron.runs") {
        return;
      }
      const eventRunID = resolveGatewayEventRunID(event);
      if (eventRunID && eventRunID !== runID) {
        return;
      }
      void runDetailQuery.refetch();
      void runsQuery.refetch();
    });
    return () => {
      unsubscribe();
    };
  }, [runDetailOpen, runDetailQuery.refetch, runsQuery.refetch, selectedRunID]);

  React.useEffect(() => {
    const runID = selectedRunID.trim();
    if (!runDetailOpen || !runID) {
      return;
    }
    if (lastLifecycleRunIDRef.current === runID) {
      return;
    }
    lastLifecycleRunIDRef.current = runID;
    setSelectedLifecycleStepID(runDetailActiveLifecycleStepID);
    setExpandedLifecycleEvents({});
  }, [runDetailActiveLifecycleStepID, runDetailOpen, selectedRunID]);

  React.useEffect(() => {
    if (!runDetailOpen) {
      return;
    }
    const current = lifecycleSteps.find((item) => item.id === selectedLifecycleStepID);
    const active = lifecycleSteps.find((item) => item.id === runDetailActiveLifecycleStepID);
    if (!current || !active || current.id === active.id) {
      return;
    }
    if (current.events.length > 0) {
      return;
    }
    if (active.events.length > 0 || runDetailIsLive) {
      setSelectedLifecycleStepID(active.id);
    }
  }, [lifecycleSteps, runDetailActiveLifecycleStepID, runDetailIsLive, runDetailOpen, selectedLifecycleStepID]);

  React.useEffect(() => {
    if (runDetailOpen) {
      return;
    }
    lastLifecycleRunIDRef.current = "";
    setSelectedLifecycleStepID("trigger");
    setExpandedLifecycleEvents({});
  }, [runDetailOpen]);

  const jobsVisibility = React.useMemo(
    () => ({ ...JOBS_DEFAULT_VISIBILITY, ...(columnVisibility.list ?? {}) }),
    [columnVisibility.list]
  );
  const runsVisibility = React.useMemo(
    () => ({ ...RUNS_DEFAULT_VISIBILITY, ...(columnVisibility.records ?? {}) }),
    [columnVisibility.records]
  );

  const visibleJobsColumns = React.useMemo(() => {
    const base: Array<{ id: JobsColumnID; label: string; className?: string; render: (job: CronJob) => React.ReactNode }> = [
      {
        id: "id",
        label: t("cron.table.id"),
        className: "min-w-[160px]",
        render: (job) => renderRunCodeValue(job.id),
      },
      {
        id: "name",
        label: t("cron.table.name"),
        className: "min-w-[160px]",
        render: (job) => renderRunPlainTextValue(job.name.trim() || job.id),
      },
      {
        id: "description",
        label: t("cron.table.description"),
        className: "min-w-[200px]",
        render: (job) => renderRunTextValue(job.description, { tone: "muted" }),
      },
      {
        id: "schedule",
        label: t("cron.table.schedule"),
        className: "min-w-[156px]",
        render: (job) => renderRunTextValue(formatSchedule(job, t), { tone: "muted" }),
      },
      {
        id: "nextRun",
        label: t("cron.table.nextRun"),
        className: "min-w-[200px]",
        render: (job) =>
          renderRunDateValue(
            job.state?.nextRunAtMs && job.state.nextRunAtMs > 0
              ? new Date(job.state.nextRunAtMs).toISOString()
              : undefined,
            { adaptiveWhenOverflow: true, language }
          ),
      },
      {
        id: "runningAt",
        label: t("cron.table.runningAt"),
        className: "min-w-[200px]",
        render: (job) =>
          renderRunDateValue(
            job.state?.runningAtMs && job.state.runningAtMs > 0
              ? new Date(job.state.runningAtMs).toISOString()
              : undefined,
            { adaptiveWhenOverflow: true, language }
          ),
      },
      {
        id: "payload",
        label: t("cron.table.payload"),
        className: "min-w-[132px]",
        render: (job) => renderRunTextValue(formatPayloadSummary(job, t), { tone: "muted" }),
      },
      {
        id: "assistantId",
        label: t("cron.table.assistantId"),
        className: "min-w-[128px]",
        render: (job) => renderRunPlainTextValue(assistantNameByID.get(job.assistantId.trim()) || job.assistantId),
      },
      {
        id: "sessionTarget",
        label: t("cron.table.sessionTarget"),
        className: "min-w-[92px]",
        render: (job) =>
          renderRunTextValue(
            job.sessionTarget === "main"
              ? t("cron.form.sessionTargetOptions.main")
              : t("cron.form.sessionTargetOptions.isolated"),
            { tone: "muted" }
          ),
      },
      {
        id: "wakeMode",
        label: t("cron.table.wakeMode"),
        className: "min-w-[110px]",
        render: (job) =>
          renderRunTextValue(
            job.wakeMode === "now"
              ? t("cron.form.wakeModeOptions.now")
              : t("cron.form.wakeModeOptions.nextHeartbeat"),
            { tone: "muted" }
          ),
      },
      {
        id: "delivery",
        label: t("cron.table.delivery"),
        className: "min-w-[180px]",
        render: (job) => renderRunTextValue(formatDeliverySummary(job.delivery, job.sourceChannel, t), { tone: "muted" }),
      },
      {
        id: "sourceChannel",
        label: t("cron.table.sourceChannel"),
        className: "min-w-[112px]",
        render: (job) => renderRunTextValue(job.sourceChannel?.trim() || "default", { tone: "muted" }),
      },
      {
        id: "deleteAfterRun",
        label: t("cron.table.deleteAfterRun"),
        className: "min-w-[112px]",
        render: (job) =>
          renderRunTextValue(
            job.deleteAfterRun ? t("common.enabled") : t("common.disabled"),
            { tone: "muted" }
          ),
      },
      {
        id: "lastRun",
        label: t("cron.table.lastRun"),
        className: "min-w-[200px]",
        render: (job) =>
          renderRunDateValue(
            job.state?.lastRunAtMs && job.state.lastRunAtMs > 0
              ? new Date(job.state.lastRunAtMs).toISOString()
              : undefined,
            { adaptiveWhenOverflow: true, language }
          ),
      },
      {
        id: "status",
        label: t("cron.table.status"),
        className: "min-w-[72px]",
        render: (job) => renderRunStatusValue(job.state?.lastRunStatus, t),
      },
      {
        id: "lastDuration",
        label: t("cron.table.lastDuration"),
        className: "min-w-[96px]",
        render: (job) => renderRunTextValue(formatDurationMsShort(job.state?.lastDurationMs), { tone: "muted" }),
      },
      {
        id: "consecutiveErrors",
        label: t("cron.table.consecutiveErrors"),
        className: "min-w-[104px]",
        render: (job) =>
          renderRunTextValue(
            String(Math.max(0, Number(job.state?.consecutiveErrors ?? 0))),
            { tone: "muted" }
          ),
      },
      {
        id: "scheduleErrors",
        label: t("cron.table.scheduleErrors"),
        className: "min-w-[100px]",
        render: (job) =>
          renderRunTextValue(
            String(Math.max(0, Number(job.state?.scheduleErrorCount ?? 0))),
            { tone: "muted" }
          ),
      },
      {
        id: "lastDeliveryStatus",
        label: t("cron.table.lastDeliveryStatus"),
        className: "min-w-[108px]",
        render: (job) => renderRunDeliveryStatusValue(job.state?.lastDeliveryStatus, t),
      },
      {
        id: "lastDeliveryError",
        label: t("cron.table.lastDeliveryError"),
        className: "min-w-[240px]",
        render: (job) =>
          renderRunTextValue(job.state?.lastDeliveryError, {
            tone: job.state?.lastDeliveryError?.trim() ? "destructive" : "muted",
          }),
      },
      {
        id: "lastError",
        label: t("cron.table.lastError"),
        className: "min-w-[240px]",
        render: (job) =>
          renderRunTextValue(job.state?.lastError, {
            tone: job.state?.lastError?.trim() ? "destructive" : "muted",
          }),
      },
      {
        id: "createdAt",
        label: t("cron.table.createdAt"),
        className: "min-w-[200px]",
        render: (job) => renderRunDateValue(job.createdAt, { adaptiveWhenOverflow: true, language }),
      },
      {
        id: "updatedAt",
        label: t("cron.table.updatedAt"),
        className: "min-w-[200px]",
        render: (job) => renderRunDateValue(job.updatedAt, { adaptiveWhenOverflow: true, language }),
      },
      {
        id: "enabled",
        label: t("cron.table.enabled"),
        className: "w-[72px] min-w-[72px] text-center",
        render: (job) => (
          <div className="flex items-center justify-center" onClick={(event) => event.stopPropagation()}>
            <Switch
              checked={job.enabled}
              onCheckedChange={(enabled) => void handleToggleEnabled(job, enabled)}
            />
          </div>
        ),
      },
      {
        id: "ops",
        label: t("cron.table.ops"),
        className: "w-[84px] min-w-[84px] text-right",
        render: (job) => renderJobActions(job),
      },
    ];

    return base.filter((column) => jobsVisibility[column.id]);
  }, [assistantNameByID, jobsVisibility, language, t]);

  const visibleRunsColumns = React.useMemo(() => {
    const base: Array<{ id: RunsColumnID; label: string; className?: string; render: (run: CronRunRecord) => React.ReactNode }> = [
      {
        id: "runId",
        label: t("cron.runs.runId"),
        className: "min-w-[160px]",
        render: (run) => renderRunCodeValue(run.runId),
      },
      {
        id: "started",
        label: t("cron.runs.time"),
        className: "min-w-[200px]",
        render: (run) => renderRunDateValue(run.startedAt, { adaptiveWhenOverflow: true, language }),
      },
      {
        id: "job",
        label: t("cron.runs.job"),
        className: "min-w-[152px]",
        render: (run) => renderRunPlainTextValue(jobNameByID.get(run.jobId) || run.jobName || run.jobId),
      },
      {
        id: "status",
        label: t("cron.runs.status"),
        className: "min-w-[72px]",
        render: (run) => renderRunStatusValue(run.status, t),
      },
      {
        id: "stage",
        label: t("cron.runs.stage"),
        className: "min-w-[112px]",
        render: (run) => renderRunStageValue(run, t),
      },
      {
        id: "duration",
        label: t("cron.runs.duration"),
        className: "min-w-[74px]",
        render: (run) => renderRunDurationValue(run, t),
      },
      {
        id: "ended",
        label: t("cron.runs.ended"),
        className: "min-w-[200px]",
        render: (run) => renderRunDateValue(run.endedAt, { adaptiveWhenOverflow: true, language }),
      },
      {
        id: "deliveryStatus",
        label: t("cron.runs.deliveryStatus"),
        className: "min-w-[84px]",
        render: (run) => renderRunDeliveryStatusValue(run.deliveryStatus, t),
      },
      {
        id: "model",
        label: t("cron.runs.model"),
        className: "min-w-[120px]",
        render: (run) => renderRunCodeValue(run.model),
      },
      {
        id: "provider",
        label: t("cron.runs.provider"),
        className: "min-w-[100px]",
        render: (run) => renderRunCodeValue(run.provider),
      },
      {
        id: "sessionKey",
        label: t("cron.runs.sessionKey"),
        className: "min-w-[152px]",
        render: (run) => renderRunCodeValue(run.sessionKey),
      },
      {
        id: "usage",
        label: t("cron.runs.usage"),
        className: "min-w-[96px]",
        render: (run) => renderRunTextValue(summarizeUsage(run.usageJson), { tone: "muted" }),
      },
      {
        id: "summary",
        label: t("cron.runs.summary"),
        className: "min-w-[240px]",
        render: (run) => renderRunTextValue(run.summary, { tone: "muted" }),
      },
      {
        id: "deliveryError",
        label: t("cron.runs.deliveryError"),
        className: "min-w-[240px]",
        render: (run) =>
          renderRunTextValue(run.deliveryError, {
            tone: run.deliveryError?.trim() ? "destructive" : "muted",
          }),
      },
      {
        id: "error",
        label: t("cron.runs.error"),
        className: "min-w-[220px]",
        render: (run) =>
          renderRunTextValue(run.error, {
            tone: run.error?.trim() ? "destructive" : "muted",
          }),
      },
    ];
    return base.filter((column) => runsVisibility[column.id]);
  }, [jobNameByID, language, runsVisibility, t]);

  const jobsColumnOptions = React.useMemo<ColumnOption[]>(
    () =>
      (Object.keys(JOBS_DEFAULT_VISIBILITY) as JobsColumnID[])
        .filter((id) => id !== "ops")
        .map((id) => ({
          id,
          label: t(`cron.columns.${id}`),
        })),
    [t]
  );

  const runsColumnOptions = React.useMemo<ColumnOption[]>(
    () =>
      (Object.keys(RUNS_DEFAULT_VISIBILITY) as RunsColumnID[]).map((id) => ({
        id,
        label: t(`cron.columns.${id}`),
      })),
    [t]
  );

  const playRefreshAnimation = React.useCallback(() => {
    if (refreshAnimationTimeoutRef.current !== null) {
      window.clearTimeout(refreshAnimationTimeoutRef.current);
    }
    setRefreshAnimating(true);
    refreshAnimationTimeoutRef.current = window.setTimeout(() => {
      setRefreshAnimating(false);
      refreshAnimationTimeoutRef.current = null;
    }, 800);
  }, []);

  React.useEffect(
    () => () => {
      if (refreshAnimationTimeoutRef.current !== null) {
        window.clearTimeout(refreshAnimationTimeoutRef.current);
      }
      if (autoRefreshDebounceRef.current !== null) {
        window.clearTimeout(autoRefreshDebounceRef.current);
      }
    },
    []
  );

  const refreshByTab = React.useCallback(
    (tab: CronTab) => {
      if (tab === "overview") {
        void statusQuery.refetch();
        void jobsQuery.refetch();
        void overviewRunsQuery.refetch();
        return;
      }
      if (tab === "list") {
        void statusQuery.refetch();
        void jobsQuery.refetch();
        return;
      }
      void runsQuery.refetch();
    },
    [jobsQuery, overviewRunsQuery, runsQuery, statusQuery]
  );

  const refresh = React.useCallback(() => {
    playRefreshAnimation();
    refreshByTab(activeTab);
  }, [activeTab, playRefreshAnimation, refreshByTab]);

  React.useEffect(() => {
    const unsubscribe = subscribeGatewayEvents((event) => {
      const eventName = (event.event ?? "").trim().toLowerCase();
      let shouldRefresh = false;
      if (activeTab === "overview") {
        shouldRefresh =
          eventName === "cron.status" ||
          eventName === "cron.list" ||
          eventName === "cron.runs" ||
          eventName === "cron.rundetail";
      } else if (activeTab === "list") {
        shouldRefresh = eventName === "cron.status" || eventName === "cron.list" || eventName === "cron.runs";
      } else {
        shouldRefresh =
          eventName === "cron.runs" || eventName === "cron.rundetail" || eventName === "cron.runevents";
      }
      if (!shouldRefresh) {
        return;
      }
      if (autoRefreshDebounceRef.current !== null) {
        window.clearTimeout(autoRefreshDebounceRef.current);
      }
      autoRefreshDebounceRef.current = window.setTimeout(() => {
        autoRefreshDebounceRef.current = null;
        playRefreshAnimation();
        refreshByTab(activeTab);
      }, 180);
    });
    return () => {
      unsubscribe();
      if (autoRefreshDebounceRef.current !== null) {
        window.clearTimeout(autoRefreshDebounceRef.current);
        autoRefreshDebounceRef.current = null;
      }
    };
  }, [activeTab, playRefreshAnimation, refreshByTab]);

  const toggleLifecycleEventExpanded = React.useCallback((key: string) => {
    setExpandedLifecycleEvents((prev) => ({
      ...prev,
      [key]: !prev[key],
    }));
  }, []);

  const openCreateDialog = () => {
    setDialogMode("create");
    setEditingJob(null);
    setDraft({ ...EMPTY_JOB_DRAFT, assistantId: createAssistantId });
    setDialogOpen(true);
  };

  React.useEffect(() => {
    if (!dialogOpen || dialogMode !== "create") {
      return;
    }
    const resolvedAssistantId = createAssistantId.trim();
    if (!resolvedAssistantId || draft.assistantId.trim()) {
      return;
    }
    setDraft((prev) => {
      if (prev.assistantId.trim()) {
        return prev;
      }
      return { ...prev, assistantId: resolvedAssistantId };
    });
  }, [createAssistantId, dialogMode, dialogOpen, draft.assistantId]);

  const openEditDialog = (job: CronJob) => {
    setDialogMode("edit");
    setEditingJob(job);
    setDraft(toDraft(job));
    setDialogOpen(true);
  };

  const openRunDetailDialog = (runID: string) => {
    setSelectedRunID(runID.trim());
    setRunDetailOpen(true);
  };

  const submitDialog = async () => {
    if (isDialogView) {
      setDialogOpen(false);
      return;
    }
    const errorMessage = validateDraft(draft, t);
    if (errorMessage) {
      messageBus.publishToast({
        intent: "warning",
        title: t("cron.form.invalid"),
        description: errorMessage,
      });
      return;
    }

    setDialogSaving(true);
    try {
      if (editingJob) {
        await updateJob.mutateAsync(toUpdatePayload(draft, editingJob));
      } else {
        await addJob.mutateAsync(toCreatePayload(draft));
      }
      setDialogOpen(false);
      messageBus.publishToast({
        intent: "success",
        title: editingJob ? t("cron.toast.updated") : t("cron.toast.created"),
      });
      if (activeTab === "records") {
        void runsQuery.refetch();
      }
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: editingJob
          ? t("cron.toast.updateFailed")
          : t("cron.toast.createFailed"),
        description: formatCronErrorMessage(error),
      });
    } finally {
      setDialogSaving(false);
    }
  };

  const handleToggleEnabled = async (job: CronJob, enabled: boolean) => {
    try {
      await updateJob.mutateAsync({
        id: job.id,
        patch: {
          enabled,
        },
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("cron.toast.updateFailed"),
        description: formatCronErrorMessage(error),
      });
    }
  };

  const handleRunJob = async (jobID: string) => {
    try {
      await runJob.mutateAsync({ id: jobID, mode: "force" });
      messageBus.publishToast({
        intent: "success",
        title: t("cron.toast.runQueued"),
      });
      if (activeTab === "records") {
        void runsQuery.refetch();
      }
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("cron.toast.runFailed"),
        description: formatCronErrorMessage(error),
      });
    }
  };

  const handleDeleteJob = (job: CronJob) => {
    messageBus.publishDialog({
      intent: "danger",
      title: t("cron.dialog.deleteTitle"),
      description: t("cron.dialog.deleteDescription"),
      confirmLabel: t("common.delete"),
      cancelLabel: t("common.cancel"),
      onConfirm: async () => {
        try {
          await removeJob.mutateAsync({ id: job.id });
          messageBus.publishToast({
            intent: "success",
            title: t("cron.toast.deleted"),
          });
        } catch (error) {
          messageBus.publishToast({
            intent: "warning",
            title: t("cron.toast.deleteFailed"),
            description: formatCronErrorMessage(error),
          });
        }
      },
    });
  };

  const handleJobSelectionCheckedChange = React.useCallback((jobID: string, checked: boolean) => {
    const normalizedJobID = jobID.trim();
    if (!normalizedJobID) {
      return;
    }
    setSelectedJobSelection((current) => {
      const currentlyChecked = Boolean(current[normalizedJobID]);
      if (currentlyChecked === checked) {
        return current;
      }
      const next = { ...current };
      if (checked) {
        next[normalizedJobID] = true;
      } else {
        delete next[normalizedJobID];
      }
      return next;
    });
  }, []);

  const handleSelectAllPagedJobsChange = React.useCallback(
    (checked: boolean) => {
      setSelectedJobSelection((current) => {
        const next = { ...current };
        if (checked) {
          pagedJobIDs.forEach((jobID) => {
            next[jobID] = true;
          });
        } else {
          pagedJobIDs.forEach((jobID) => {
            delete next[jobID];
          });
        }
        return next;
      });
    },
    [pagedJobIDs]
  );

  const handleEnterJobSelectionMode = React.useCallback(() => {
    setJobSelectionMode(true);
    setJobsViewMode("table");
  }, [setJobsViewMode]);

  const handleExitJobSelectionMode = React.useCallback(() => {
    setJobSelectionMode(false);
    setSelectedJobSelection({});
  }, []);

  const handleBatchDeleteJobs = React.useCallback(() => {
    if (selectedJobCount === 0 || batchDeleteJobsPending) {
      return;
    }
    messageBus.publishDialog({
      intent: "danger",
      title: t("cron.dialog.bulkDeleteTitle"),
      description: formatTemplate(t("cron.dialog.bulkDeleteDescription"), { count: selectedJobCount }),
      confirmLabel: formatTemplate(t("cron.dialog.bulkDeleteConfirm"), { count: selectedJobCount }),
      cancelLabel: t("common.cancel"),
      onConfirm: async () => {
        setBatchDeleteJobsPending(true);
        let successCount = 0;
        let failedCount = 0;
        try {
          for (const jobID of selectedJobIDs) {
            try {
              await removeJob.mutateAsync({ id: jobID });
              successCount += 1;
            } catch {
              failedCount += 1;
            }
          }
          if (failedCount === 0) {
            messageBus.publishToast({
              intent: "success",
              title: formatTemplate(t("cron.toast.bulkDeleteSuccess"), { count: successCount }),
            });
          } else if (successCount > 0) {
            messageBus.publishToast({
              intent: "warning",
              title: formatTemplate(t("cron.toast.bulkDeletePartial"), {
                success: successCount,
                failed: failedCount,
              }),
            });
          } else {
            messageBus.publishToast({
              intent: "warning",
              title: t("cron.toast.bulkDeleteFailed"),
            });
          }
        } finally {
          setBatchDeleteJobsPending(false);
          handleExitJobSelectionMode();
        }
      },
    });
  }, [
    batchDeleteJobsPending,
    handleExitJobSelectionMode,
    removeJob,
    selectedJobCount,
    selectedJobIDs,
    t,
  ]);

  const renderJobActions = (job: CronJob) => (
    <div className="flex items-center justify-end gap-1">
      <Button
        variant="ghost"
        size="compactIcon"
        onClick={(event) => {
          event.stopPropagation();
          void handleRunJob(job.id);
        }}
        aria-label={t("cron.action.run")}
      >
        <Play className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="compactIcon"
        className="text-destructive"
        onClick={(event) => {
          event.stopPropagation();
          handleDeleteJob(job);
        }}
        aria-label={t("cron.action.delete")}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );

  const renderJobsTable = () => (
    <div className={cn("min-h-0 flex-1 overflow-hidden", DASHBOARD_PANEL_SURFACE_CLASS)}>
      <div className="h-full overflow-auto">
        <Table className="min-w-full table-fixed">
          <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
            <TableRow>
              {jobSelectionMode ? (
                <TableHead className="w-[44px] min-w-[44px] text-center">
                  <CronTableSelectionCheckbox
                    checked={areAllPagedJobsSelected}
                    indeterminate={areSomePagedJobsSelected}
                    onChange={(event) => handleSelectAllPagedJobsChange(event.target.checked)}
                    aria-label={t("cron.action.selectJobs")}
                  />
                </TableHead>
              ) : null}
              {visibleJobsColumns.map((column) => (
                <TableHead
                  key={column.id}
                  className={cn("whitespace-nowrap text-xs font-semibold tracking-wide text-muted-foreground", column.className)}
                >
                  {column.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className="select-none">
            {filteredJobs.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={visibleJobsColumns.length + (jobSelectionMode ? 1 : 0)}
                  className="py-8 text-center text-sm text-muted-foreground"
                >
                  {t("cron.empty.jobs")}
                </TableCell>
              </TableRow>
            ) : (
              pagedJobs.map((job) => {
                const normalizedJobID = job.id.trim();
                const selected = Boolean(selectedJobSelection[normalizedJobID]);
                return (
                  <TableRow
                    key={job.id}
                    className={cn(
                      "cursor-pointer odd:bg-muted/[0.14] transition-colors hover:bg-muted/40",
                      selected ? "bg-muted/40" : ""
                    )}
                    onClick={() => {
                      if (jobSelectionMode) {
                        handleJobSelectionCheckedChange(normalizedJobID, !selected);
                        return;
                      }
                      openEditDialog(job);
                    }}
                  >
                    {jobSelectionMode ? (
                      <TableCell className="w-[44px] min-w-[44px] text-center" onClick={(event) => event.stopPropagation()}>
                        <CronTableSelectionCheckbox
                          checked={selected}
                          onChange={(event) => handleJobSelectionCheckedChange(normalizedJobID, event.target.checked)}
                          aria-label={t("cron.action.selectJobs")}
                        />
                      </TableCell>
                    ) : null}
                    {visibleJobsColumns.map((column) => (
                      <TableCell key={column.id} className={cn("align-middle overflow-hidden text-xs", column.className)}>
                        {column.render(job)}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );

  const renderJobsCards = () => (
    <div className={cn("min-h-0 flex-1 overflow-hidden", DASHBOARD_PANEL_SURFACE_CLASS)}>
      <div className="h-full overflow-auto p-3">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {filteredJobs.length === 0 ? (
            <PanelCard tone="inset">
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {t("cron.empty.jobs")}
              </CardContent>
            </PanelCard>
          ) : (
            pagedJobs.map((job) => (
              <PanelCard
                key={job.id}
                tone="inset"
                className="cursor-pointer transition hover:border-border"
                onClick={() => openEditDialog(job)}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="space-y-1">
                      <CardTitle className="text-sm">{job.name.trim() || job.id}</CardTitle>
                      <CardDescription className="text-xs">{formatSchedule(job, t)}</CardDescription>
                    </div>
                    <Badge variant={job.enabled ? "default" : "subtle"}>
                      {job.enabled ? t("cron.status.enabled") : t("cron.status.disabled")}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="space-y-1 text-xs text-muted-foreground">
                    <div>
                      {t("cron.table.payload")}: {formatPayloadSummary(job, t)}
                    </div>
                    <div>
                      {t("cron.table.sessionTarget")}: {job.sessionTarget || "-"}
                    </div>
                    <div>
                      {t("cron.table.lastRun")}:
                      {job.state?.lastRunAtMs && job.state.lastRunAtMs > 0
                        ? ` ${formatDateTime(new Date(job.state.lastRunAtMs).toISOString())}`
                        : " -"}
                    </div>
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <div onClick={(event) => event.stopPropagation()}>
                      <Switch
                        checked={job.enabled}
                        onCheckedChange={(enabled) => void handleToggleEnabled(job, enabled)}
                      />
                    </div>
                    {renderJobActions(job)}
                  </div>
                </CardContent>
              </PanelCard>
            ))
          )}
        </div>
      </div>
    </div>
  );

  const renderRunsTable = () => (
    <div className={cn("min-h-0 flex-1 overflow-hidden", DASHBOARD_PANEL_SURFACE_CLASS)}>
      <div className="h-full overflow-auto">
        <Table className="min-w-full table-fixed">
          <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
            <TableRow>
              {visibleRunsColumns.map((column) => (
                <TableHead
                  key={column.id}
                  className={cn("whitespace-nowrap text-xs font-semibold tracking-wide text-muted-foreground", column.className)}
                >
                  {column.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className="select-none">
            {runs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={visibleRunsColumns.length} className="py-8 text-center text-sm text-muted-foreground">
                  {t("cron.empty.runs")}
                </TableCell>
              </TableRow>
            ) : (
              runs.map((run) => (
                <TableRow
                  key={run.runId}
                  className="cursor-pointer odd:bg-muted/[0.14] transition-colors hover:bg-muted/40"
                  onClick={() => openRunDetailDialog(run.runId)}
                >
                  {visibleRunsColumns.map((column) => (
                    <TableCell key={column.id} className={cn("align-middle overflow-hidden text-xs", column.className)}>
                      {column.render(run)}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );

  const renderRunsCards = () => (
    <div className={cn("min-h-0 flex-1 overflow-hidden", DASHBOARD_PANEL_SURFACE_CLASS)}>
      <div className="h-full overflow-auto p-3">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {runs.length === 0 ? (
            <PanelCard tone="inset">
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {t("cron.empty.runs")}
              </CardContent>
            </PanelCard>
          ) : (
            runs.map((run) => (
              <PanelCard
                key={run.runId}
                tone="inset"
                className="cursor-pointer transition hover:border-border"
                onClick={() => openRunDetailDialog(run.runId)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-2">
                    <div className="space-y-1">
                      <CardTitle className="text-sm">{jobNameByID.get(run.jobId) || run.jobName || run.jobId}</CardTitle>
                      <CardDescription>{formatDateTime(run.startedAt)}</CardDescription>
                    </div>
                    {renderRunStatusValue(run.status, t)}
                  </div>
                </CardHeader>
                <CardContent className="space-y-1 text-xs text-muted-foreground">
                  <div className="truncate">{t("cron.runs.stage")}: {resolveRunStageValueLabel(runStageLabel(run), t)}</div>
                  <div className="truncate">{t("cron.runs.deliveryStatus")}: {resolveRunDeliveryStatusLabel(run.deliveryStatus, t)}</div>
                  <div className="truncate">{t("cron.runs.duration")}: {formatRunDurationText(run, t)}</div>
                  <div className="truncate">{run.summary?.trim() || CRON_EMPTY_VALUE}</div>
                </CardContent>
              </PanelCard>
            ))
          )}
        </div>
      </div>
    </div>
  );

  const renderJobsPaginationControls = () => {
    const totalText = formatTemplate(t("cron.pagination.totalJobs"), {
      count: filteredJobs.length,
    });
    const rowsPerPageTemplate = t("cron.pagination.rowsPerPage");
    const pageText = formatTemplate(t("cron.pagination.pageOf"), {
      page: jobsPage,
      total: jobsTotalPages,
    });

    return (
      <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
        <div className="text-xs text-muted-foreground">{totalText}</div>
        <div className="flex items-center gap-2">
          <Select
            className={cn("w-[112px]", CRON_SELECT_TEXT_CLASS)}
            value={String(jobsRowsPerPage)}
            onChange={(event) => {
              const next = Number(event.target.value);
              setJobsRowsPerPage(next);
              setJobsPage(1);
            }}
          >
            {PAGINATION_PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {formatTemplate(rowsPerPageTemplate, { count: size })}
              </option>
            ))}
          </Select>
          <div className="text-xs text-muted-foreground">{pageText}</div>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setJobsPage(1)}
            disabled={jobsPage <= 1}
            aria-label={t("cron.pagination.firstPage")}
          >
            <ChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setJobsPage((prev) => Math.max(1, prev - 1))}
            disabled={jobsPage <= 1}
            aria-label={t("cron.pagination.prevPage")}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setJobsPage((prev) => Math.min(jobsTotalPages, prev + 1))}
            disabled={jobsPage >= jobsTotalPages}
            aria-label={t("cron.pagination.nextPage")}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setJobsPage(jobsTotalPages)}
            disabled={jobsPage >= jobsTotalPages}
            aria-label={t("cron.pagination.lastPage")}
          >
            <ChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    );
  };

  const renderRunsPaginationControls = () => {
    const totalText = formatTemplate(t("cron.pagination.totalRuns"), {
      count: runsTotal,
    });
    const rowsPerPageTemplate = t("cron.pagination.rowsPerPage");
    const pageText = formatTemplate(t("cron.pagination.pageOf"), {
      page: runsPage,
      total: runsTotalPages,
    });

    return (
      <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
        <div className="text-xs text-muted-foreground">{totalText}</div>
        <div className="flex items-center gap-2">
          <Select
            className={cn("w-[112px]", CRON_SELECT_TEXT_CLASS)}
            value={String(runsRowsPerPage)}
            onChange={(event) => {
              const next = Number(event.target.value);
              setRunsRowsPerPage(next);
              setRunsPage(1);
            }}
          >
            {PAGINATION_PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {formatTemplate(rowsPerPageTemplate, { count: size })}
              </option>
            ))}
          </Select>
          <div className="text-xs text-muted-foreground">{pageText}</div>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setRunsPage(1)}
            disabled={runsPage <= 1}
            aria-label={t("cron.pagination.firstPage")}
          >
            <ChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setRunsPage((prev) => Math.max(1, prev - 1))}
            disabled={runsPage <= 1}
            aria-label={t("cron.pagination.prevPage")}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setRunsPage((prev) => Math.min(runsTotalPages, prev + 1))}
            disabled={runsPage >= runsTotalPages}
            aria-label={t("cron.pagination.nextPage")}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => setRunsPage(runsTotalPages)}
            disabled={runsPage >= runsTotalPages}
            aria-label={t("cron.pagination.lastPage")}
          >
            <ChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    );
  };

  const currentColumnOptions = activeTab === "list" ? jobsColumnOptions : activeTab === "records" ? runsColumnOptions : [];
  const currentVisibility = activeTab === "list" ? jobsVisibility : runsVisibility;

  const toggleColumnVisibility = (columnId: string) => {
    if (activeTab !== "list" && activeTab !== "records") {
      return;
    }
    const tabVisibility = activeTab === "list" ? jobsVisibility : runsVisibility;
    setColumnVisibility(activeTab, {
      ...tabVisibility,
      [columnId]: !(tabVisibility[columnId] ?? true),
    });
  };

  const overviewRuns = overviewRunsQuery.data?.items ?? [];
  const overviewTotalExecutions = overviewRunsQuery.data?.total ?? 0;
  const overviewAverageDurationText = React.useMemo(() => {
    const durations = overviewRuns
      .map((run) => {
        if (!run.startedAt || !run.endedAt) {
          return 0;
        }
        const start = new Date(run.startedAt).getTime();
        const end = new Date(run.endedAt).getTime();
        if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
          return 0;
        }
        return end - start;
      })
      .filter((duration) => duration > 0);
    if (durations.length === 0) {
      return CRON_EMPTY_VALUE;
    }
    const average = Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length);
    return formatDurationMsShort(average);
  }, [overviewRuns]);

  const chartGranularityOptions = React.useMemo(
    () => [
      { value: "1m", label: t("cron.overview.granularity.minute") },
      { value: "15m", label: t("cron.overview.granularity.quarterHour") },
      { value: "1h", label: t("cron.overview.granularity.hour") },
      { value: "1d", label: t("cron.overview.granularity.day") },
    ],
    [t]
  );

  const overviewChartData = React.useMemo(() => {
    const bucketSize = OVERVIEW_GRANULARITY_MS[chartGranularity] ?? OVERVIEW_GRANULARITY_MS["1h"];
    const now = Date.now();
    const bucketCount = chartGranularity === "1d" ? 14 : chartGranularity === "1h" ? 24 : 16;
    const firstBucketAt = now - (bucketCount - 1) * bucketSize;
    const buckets = Array.from({ length: bucketCount }, (_, index) => {
      const start = firstBucketAt + index * bucketSize;
      return {
        start,
        label:
          chartGranularity === "1d"
            ? new Date(start).toLocaleDateString(undefined, { month: "2-digit", day: "2-digit" })
            : new Date(start).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" }),
        success: 0,
        failed: 0,
      };
    });
    overviewRuns.forEach((run) => {
      const startAt = new Date(run.startedAt).getTime();
      if (!Number.isFinite(startAt) || startAt < firstBucketAt) {
        return;
      }
      const index = Math.floor((startAt - firstBucketAt) / bucketSize);
      if (index < 0 || index >= buckets.length) {
        return;
      }
      const status = normalizeEnumToken(run.status);
      if (status === "completed" || status === "ok") {
        buckets[index].success += 1;
      } else if (status === "failed" || status === "error") {
        buckets[index].failed += 1;
      }
    });
    return buckets.map((bucket) => ({
      label: bucket.label,
      success: bucket.success,
      failed: bucket.failed,
    }));
  }, [chartGranularity, overviewRuns]);

  const overviewCards = React.useMemo(
    () => [
      {
        id: "total-crons",
        label: t("cron.overview.cards.totalCrons"),
        value: String(jobs.length),
      },
      {
        id: "total-executions",
        label: t("cron.overview.cards.totalExecutions"),
        value: String(overviewTotalExecutions),
      },
      {
        id: "avg-duration",
        label: t("cron.overview.cards.avgDuration"),
        value: overviewAverageDurationText,
      },
      {
        id: "next-wake",
        label: t("cron.overview.cards.nextWake"),
        value: formatDateTime(statusQuery.data?.nextWakeAt),
      },
    ],
    [jobs.length, overviewAverageDurationText, overviewTotalExecutions, statusQuery.data?.nextWakeAt, t]
  );
  const overviewRecentRuns = React.useMemo(() => {
    const items = [...overviewRuns];
    items.sort((left, right) => {
      const leftTime = new Date(left.startedAt).getTime();
      const rightTime = new Date(right.startedAt).getTime();
      return rightTime - leftTime;
    });
    return items;
  }, [overviewRuns]);

  return (
    <TooltipProvider delayDuration={100} skipDelayDuration={300}>
      <div className="flex min-h-0 flex-1 flex-col gap-4">
        <div className="grid shrink-0 grid-cols-[auto_minmax(0,1fr)] items-center gap-3">
          <Tabs
            value={activeTab}
            onValueChange={(value) => setActiveTab(value as CronTab)}
            className="min-w-0 w-auto"
          >
            <TabsList>
              <TabsTrigger value="overview">
                <LayoutGrid className="h-3.5 w-3.5" />
                <span>{t("cron.tabs.overview")}</span>
              </TabsTrigger>
              <TabsTrigger value="list">
                <List className="h-3.5 w-3.5" />
                <span>{t("cron.tabs.list")}</span>
              </TabsTrigger>
              <TabsTrigger value="records">
                <History className="h-3.5 w-3.5" />
                <span>{t("cron.tabs.records")}</span>
              </TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="flex min-w-0 flex-nowrap items-center justify-end gap-2 overflow-x-auto pb-1 -mb-1">
            {activeTab === "list" ? (
              <>
                <CronJobFilterCombobox
                  searchQuery={jobSearchQuery}
                  onSearchQueryChange={setJobSearchQuery}
                  enabledFilter={jobEnabledFilter}
                  onEnabledFilterChange={setJobEnabledFilter}
                  lastRunStatusFilter={jobLastRunStatusFilter}
                  onLastRunStatusFilterChange={setJobLastRunStatusFilter}
                  onClearAll={() => {
                    setJobSearchQuery("");
                    setJobEnabledFilter("");
                    setJobLastRunStatusFilter("");
                  }}
                  filterCount={jobFilterCount}
                  t={t}
                />
                {jobSelectionMode ? (
                  <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                    <div className="inline-flex items-center gap-2 px-3 text-xs text-muted-foreground">
                      <ListChecks className="h-3.5 w-3.5" />
                      <span>
                        {formatTemplate(t("cron.action.selectedJobCount"), {
                          count: selectedJobCount,
                        })}
                      </span>
                    </div>
                    <Button
                      variant="ghost"
                      size="compact"
                      className="gap-1.5 rounded-none border-0 border-l border-border/70 text-destructive hover:text-destructive"
                      disabled={selectedJobCount === 0 || batchDeleteJobsPending}
                      onClick={handleBatchDeleteJobs}
                    >
                      <Trash2 className="h-4 w-4" />
                      {t("cron.action.delete")}
                    </Button>
                    <Button
                      variant="ghost"
                      size="compact"
                      className="gap-1.5 rounded-none border-0 border-l border-border/70"
                      onClick={handleExitJobSelectionMode}
                    >
                      {t("cron.action.cancelSelection")}
                    </Button>
                  </div>
                ) : (
                  <Button variant="outline" size="compact" className="gap-2" onClick={handleEnterJobSelectionMode}>
                    <ListChecks className="h-4 w-4" />
                    {t("cron.action.selectJobs")}
                  </Button>
                )}
              </>
            ) : null}

            {activeTab === "records" ? (
              <CronRunFilterCombobox
                searchQuery={runSearchQuery}
                onSearchQueryChange={setRunSearchQuery}
                jobFilter={jobFilter}
                onJobFilterChange={setJobFilter}
                runStatusFilter={runStatusFilter}
                onRunStatusFilterChange={setRunStatusFilter}
                jobs={jobs}
                onClearAll={() => {
                  setRunSearchQuery("");
                  setJobFilter("");
                  setRunStatusFilter("");
                }}
                filterCount={runFilterCount}
                t={t}
              />
            ) : null}

            {(activeTab === "list" || activeTab === "records") && (
              <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={
                        (activeTab === "list" ? jobsViewMode : runsViewMode) === "table"
                          ? "secondary"
                          : "ghost"
                      }
                      size="compactIcon"
                      className="rounded-none border-0"
                      onClick={() => (activeTab === "list" ? setJobsViewMode("table") : setRunsViewMode("table"))}
                      aria-label={t("cron.view.table")}
                    >
                      <List className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("cron.tooltip.tableView")}</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={
                        (activeTab === "list" ? jobsViewMode : runsViewMode) === "cards"
                          ? "secondary"
                          : "ghost"
                      }
                      size="compactIcon"
                      className="rounded-none border-l border-border/70"
                      onClick={() => (activeTab === "list" ? setJobsViewMode("cards") : setRunsViewMode("cards"))}
                      aria-label={t("cron.view.cards")}
                      disabled={activeTab === "list" && jobSelectionMode}
                    >
                      <LayoutGrid className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("cron.tooltip.cardsView")}</TooltipContent>
                </Tooltip>
              </div>
            )}

            {activeTab === "list" ? (
              <Button variant="outline" size="compact" className="gap-2" onClick={openCreateDialog}>
                <Plus className="h-4 w-4" />
                {t("cron.action.newJob")}
              </Button>
            ) : null}

            {activeTab !== "overview" ? (
              <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                <DropdownMenu>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="compactIcon"
                          className="rounded-none border-0"
                          aria-label={t("cron.action.customizeColumns")}
                        >
                          <SlidersHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                    </TooltipTrigger>
                    <TooltipContent>{t("cron.action.customizeColumns")}</TooltipContent>
                  </Tooltip>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel>{t("cron.action.visibleColumns")}</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    {currentColumnOptions.length === 0 ? (
                      <DropdownMenuItem disabled>{t("cron.action.noConfigurableColumns")}</DropdownMenuItem>
                    ) : (
                      currentColumnOptions.map((column) => (
                        <DropdownMenuCheckboxItem
                          key={column.id}
                          checked={currentVisibility[column.id] ?? true}
                          onCheckedChange={() => toggleColumnVisibility(column.id)}
                        >
                          {column.label}
                        </DropdownMenuCheckboxItem>
                      ))
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="compactIcon"
                      className="rounded-none border-0 border-l border-border/70"
                      onClick={refresh}
                      aria-label={t("cron.action.refresh")}
                    >
                      <RefreshCcw className={cn("h-4 w-4", refreshAnimating && "animate-spin")} />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("cron.action.refresh")}</TooltipContent>
                </Tooltip>
              </div>
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" size="compactIcon" onClick={refresh} aria-label={t("cron.action.refresh")}>
                    <RefreshCcw className={cn("h-4 w-4", refreshAnimating && "animate-spin")} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("cron.action.refresh")}</TooltipContent>
              </Tooltip>
            )}

          </div>
        </div>

        {activeTab === "overview" ? (
          <CronOverviewPage
            cards={overviewCards}
            chartData={overviewChartData}
            chartTitle={t("cron.overview.chart.title")}
            chartSuccessLabel={t("cron.overview.chart.series.success")}
            chartFailedLabel={t("cron.overview.chart.series.failed")}
            chartGranularity={chartGranularity}
            chartGranularityOptions={chartGranularityOptions}
            onChartGranularityChange={setChartGranularity}
            recentTitle={t("cron.overview.recent")}
            recentRuns={overviewRecentRuns}
            resolveRunTitle={(run) => {
              const name = (jobNameByID.get(run.jobId) || run.jobName || "").trim();
              return name || CRON_EMPTY_VALUE;
            }}
            resolveRunFrequency={(run) => {
              const scheduleText = jobScheduleByID.get(run.jobId);
              return (scheduleText ?? "").trim() || CRON_EMPTY_VALUE;
            }}
            renderRunStatusIcon={(run) => renderRunStatusIconValue(run.status)}
            formatStartedAt={(value) => formatRelativeDateTime(value, language)}
            emptyRecentText={t("cron.empty.runs")}
          />
        ) : null}

        {activeTab === "list" ? (
          <CronListPage
            jobsViewMode={jobsViewMode}
            renderJobsTable={renderJobsTable}
            renderJobsCards={renderJobsCards}
            renderJobsPaginationControls={renderJobsPaginationControls}
          />
        ) : null}

        {activeTab === "records" ? (
          <CronExecutionRecordPage
            runsViewMode={runsViewMode}
            renderRunsTable={renderRunsTable}
            renderRunsCards={renderRunsCards}
            renderRunsPaginationControls={renderRunsPaginationControls}
          />
        ) : null}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="pl-2">
                {dialogMode === "view"
                  ? t("cron.dialog.viewTitle")
                  : editingJob
                  ? t("cron.dialog.editTitle")
                  : t("cron.dialog.newTitle")}
              </DialogTitle>
            </DialogHeader>

            <fieldset disabled={isDialogView} className="grid gap-4">
              <PanelCard tone="solid">
                <CardHeader size="compact" className="pb-2">
                  <div className="flex items-start justify-between gap-3">
                    <CardTitle className="text-sm">{t("cron.form.section.basic")}</CardTitle>
                    <div className="max-w-[58%] min-w-[220px] text-right">
                      <div className="inline-flex max-w-full items-center justify-end gap-1 text-[11px] text-muted-foreground/70">
                        <span className="shrink-0 tracking-[0.12em]">
                          {t("cron.table.assistantId")}:
                        </span>
                        <span className="truncate">
                          {assistantWatermarkName}
                        </span>
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent size="compact" className="space-y-2">
                  <div className="grid gap-3 py-1 md:grid-cols-2">
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-xs text-muted-foreground">{t("cron.form.enabled")}</span>
                      <Switch
                        checked={draft.enabled}
                        onCheckedChange={(enabled) => setDraft((prev) => ({ ...prev, enabled }))}
                      />
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-xs text-muted-foreground">
                        {t("cron.form.deleteAfterRun")}
                      </span>
                      <Switch
                        checked={draft.deleteAfterRun}
                        onCheckedChange={(deleteAfterRun) => setDraft((prev) => ({ ...prev, deleteAfterRun }))}
                      />
                    </div>
                  </div>

                  <Separator className="bg-border/70" />

                  <div className="grid gap-3 py-1 md:grid-cols-2">
                    <div className="flex items-center justify-between gap-3">
                      <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.name")}</span>
                      <Input
                        value={draft.name}
                        onChange={(event) => setDraft((prev) => ({ ...prev, name: event.target.value }))}
                        dir="ltr"
                        size="compact"
                        className="ml-auto w-[68%] text-left"
                      />
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.description")}</span>
                      <Input
                        value={draft.description}
                        onChange={(event) => setDraft((prev) => ({ ...prev, description: event.target.value }))}
                        dir="ltr"
                        size="compact"
                        className="ml-auto w-[68%] text-left"
                      />
                    </div>
                  </div>

                  <Separator className="bg-border/70" />

                  <div className="grid gap-3 py-1 md:grid-cols-2">
                    <div className="flex items-center justify-between gap-3">
                      <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.sessionTarget")}</span>
                      <Select
                        value={draft.sessionTarget}
                        onChange={(event) =>
                          setDraft((prev) => ({ ...prev, sessionTarget: event.target.value as CronSessionTarget }))
                        }
                        className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                      >
                        <option value="main">{t("cron.form.sessionTargetOptions.main")}</option>
                        <option value="isolated">{t("cron.form.sessionTargetOptions.isolated")}</option>
                      </Select>
                    </div>

                    <div className="flex items-center justify-between gap-3">
                      <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.wakeMode")}</span>
                      <Select
                        value={draft.wakeMode}
                        onChange={(event) =>
                          setDraft((prev) => ({ ...prev, wakeMode: event.target.value as CronWakeMode }))
                        }
                        className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                      >
                        <option value="now">{t("cron.form.wakeModeOptions.now")}</option>
                        <option value="next-heartbeat">
                          {t("cron.form.wakeModeOptions.nextHeartbeat")}
                        </option>
                      </Select>
                    </div>
                  </div>
                </CardContent>
              </PanelCard>

              <PanelCard tone="solid">
                <Tabs
                  value={draft.scheduleKind}
                  onValueChange={(value) =>
                    setDraft((prev) => ({ ...prev, scheduleKind: value as CronScheduleType }))
                  }
                >
                  <CardHeader size="compact" className="pb-2">
                    <div className="flex items-start justify-between gap-3">
                      <CardTitle className="text-sm">{t("cron.form.section.schedule")}</CardTitle>
                      <TabsList>
                        <TabsTrigger value="every">
                          {t("cron.form.scheduleTypeOptions.every")}
                        </TabsTrigger>
                        <TabsTrigger value="cron">
                          {t("cron.form.scheduleTypeOptions.cron")}
                        </TabsTrigger>
                        <TabsTrigger value="at">
                          {t("cron.form.scheduleTypeOptions.at")}
                        </TabsTrigger>
                      </TabsList>
                    </div>
                  </CardHeader>
                  <CardContent size="compact" className="space-y-2">
                    <TabsContent value="every" className="mt-0">
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.every")}</span>
                          <Input
                            value={draft.everyAmount}
                            onChange={(event) => setDraft((prev) => ({ ...prev, everyAmount: event.target.value }))}
                            size="compact"
                            className="ml-auto w-[68%] text-right"
                          />
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.everyUnit")}</span>
                          <Select
                            value={draft.everyUnit}
                            onChange={(event) =>
                              setDraft((prev) => ({ ...prev, everyUnit: event.target.value as EveryUnit }))
                            }
                            className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                          >
                            <option value="minutes">{t("cron.form.everyUnitOptions.minutes")}</option>
                            <option value="hours">{t("cron.form.everyUnitOptions.hours")}</option>
                            <option value="days">{t("cron.form.everyUnitOptions.days")}</option>
                          </Select>
                        </div>
                      </div>
                    </TabsContent>

                    <TabsContent value="at" className="mt-0">
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.at")}</span>
                          <Input
                            type="datetime-local"
                            value={draft.scheduleAt}
                            onChange={(event) => setDraft((prev) => ({ ...prev, scheduleAt: event.target.value }))}
                            size="compact"
                            className="ml-auto w-[68%] text-right"
                          />
                        </div>
                      </div>
                    </TabsContent>

                    <TabsContent value="cron" className="mt-0 space-y-2">
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {t("cron.form.cronExpr")}
                          </span>
                          <Input
                            value={draft.cronExpr}
                            onChange={(event) => setDraft((prev) => ({ ...prev, cronExpr: event.target.value }))}
                            placeholder={t("cron.form.cronExprPlaceholder")}
                            size="compact"
                            className="ml-auto w-[68%] text-right"
                          />
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.timezone")}</span>
                          {timezoneOptions.length > 0 ? (
                            <Select
                              value={draft.cronTz}
                              onChange={(event) => setDraft((prev) => ({ ...prev, cronTz: event.target.value }))}
                              className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                            >
                              <option value="">{t("cron.form.timezonePlaceholder")}</option>
                              {cronTimezoneOptions.map((timezone) => (
                                <option key={timezone} value={timezone}>
                                  {timezone}
                                </option>
                              ))}
                            </Select>
                          ) : (
                            <Input
                              value={draft.cronTz}
                              onChange={(event) => setDraft((prev) => ({ ...prev, cronTz: event.target.value }))}
                              placeholder={t("cron.form.timezonePlaceholder")}
                              size="compact"
                              className="ml-auto w-[68%] text-right"
                            />
                          )}
                        </div>
                      </div>

                      <Separator className="bg-border/70" />

                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-xs text-muted-foreground">
                            {t("cron.form.scheduleExact")}
                          </span>
                          <Switch
                            checked={draft.scheduleExact}
                            onCheckedChange={(scheduleExact) => setDraft((prev) => ({ ...prev, scheduleExact }))}
                          />
                        </div>
                        <div className="hidden md:block" />
                      </div>

                      {!draft.scheduleExact ? (
                        <>
                          <Separator className="bg-border/70" />
                          <div className="grid gap-3 md:grid-cols-2">
                            <div className="flex items-center justify-between gap-3">
                              <span className="shrink-0 text-xs text-muted-foreground">
                                {t("cron.form.staggerAmount")}
                              </span>
                              <Input
                                value={draft.staggerAmount}
                                onChange={(event) => setDraft((prev) => ({ ...prev, staggerAmount: event.target.value }))}
                                placeholder={t("cron.form.staggerAmountPlaceholder")}
                                size="compact"
                                className="ml-auto w-[68%] text-right"
                              />
                            </div>

                            <div className="flex items-center justify-between gap-3">
                              <span className="shrink-0 text-xs text-muted-foreground">
                                {t("cron.form.staggerUnit")}
                              </span>
                              <Select
                                value={draft.staggerUnit}
                                onChange={(event) =>
                                  setDraft((prev) => ({ ...prev, staggerUnit: event.target.value as StaggerUnit }))
                                }
                                className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                              >
                                <option value="seconds">{t("cron.form.staggerUnitOptions.seconds")}</option>
                                <option value="minutes">{t("cron.form.staggerUnitOptions.minutes")}</option>
                              </Select>
                            </div>
                          </div>
                        </>
                      ) : null}
                    </TabsContent>
                  </CardContent>
                </Tabs>
              </PanelCard>

              <PanelCard tone="solid">
                <Tabs
                  value={draft.payloadKind}
                  onValueChange={(value) =>
                    setDraft((prev) => ({ ...prev, payloadKind: value as CronPayloadKind }))
                  }
                >
                  <CardHeader size="compact" className="pb-2">
                    <div className="flex items-start justify-between gap-3">
                      <CardTitle className="text-sm">{t("cron.form.section.payload")}</CardTitle>
                      <TabsList>
                        <TabsTrigger value="systemEvent">
                          {t("cron.form.payloadKindOptions.systemEvent")}
                        </TabsTrigger>
                        <TabsTrigger value="agentTurn">
                          {t("cron.form.payloadKindOptions.agentTurn")}
                        </TabsTrigger>
                      </TabsList>
                    </div>
                  </CardHeader>
                  <CardContent size="compact" className="space-y-2">
                    <TabsContent value="systemEvent" className="mt-0">
                      <div className="grid gap-1.5">
                        {renderSectionLabel(t("cron.form.payloadText"))}
                        <textarea
                          value={draft.payloadText}
                          onChange={(event) => setDraft((prev) => ({ ...prev, payloadText: event.target.value }))}
                          className="min-h-[72px] rounded-md border bg-background px-3 py-2 text-sm shadow-sm outline-none transition focus-visible:ring-2 focus-visible:ring-ring"
                        />
                      </div>
                    </TabsContent>

                    <TabsContent value="agentTurn" className="mt-0 space-y-2">
                      <div className="grid gap-1.5">
                        {renderSectionLabel(t("cron.form.payloadMessage"))}
                        <textarea
                          value={draft.payloadText}
                          onChange={(event) => setDraft((prev) => ({ ...prev, payloadText: event.target.value }))}
                          className="min-h-[72px] rounded-md border bg-background px-3 py-2 text-sm shadow-sm outline-none transition focus-visible:ring-2 focus-visible:ring-ring"
                        />
                      </div>

                      <Separator className="bg-border/70" />

                      <div className="grid gap-3 md:grid-cols-3">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.model")}</span>
                          <Select
                            value={draft.payloadModel}
                            onChange={(event) => setDraft((prev) => ({ ...prev, payloadModel: event.target.value }))}
                            className={cn("ml-auto w-[58%] text-right", CRON_SELECT_TEXT_CLASS)}
                          >
                            {payloadModelOptions.map((option) => (
                              <option key={option.value || "__cron_model_auto__"} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </Select>
                        </div>

                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.thinking")}</span>
                          <Select
                            value={draft.payloadThinking}
                            onChange={(event) => setDraft((prev) => ({ ...prev, payloadThinking: event.target.value }))}
                            className={cn("ml-auto w-[58%] text-right", CRON_SELECT_TEXT_CLASS)}
                          >
                            {payloadThinkingOptions.map((option) => (
                              <option key={option.value || "__cron_thinking_auto__"} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </Select>
                        </div>

                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {t("cron.form.timeoutSeconds")}
                          </span>
                          <Input
                            value={draft.timeoutSeconds}
                            onChange={(event) => setDraft((prev) => ({ ...prev, timeoutSeconds: event.target.value }))}
                            placeholder={t("cron.form.timeoutSecondsPlaceholder")}
                            size="compact"
                            className="ml-auto w-[52%] text-right placeholder:text-xs"
                          />
                        </div>
                      </div>

                    </TabsContent>
                  </CardContent>
                </Tabs>
              </PanelCard>

              <PanelCard tone="solid">
                <Tabs
                  value={draft.deliveryMode}
                  onValueChange={(value) => {
                    const nextMode = value as CronDeliveryMode;
                    setDraft((prev) =>
                      nextMode === "none"
                        ? {
                            ...prev,
                            deliveryMode: nextMode,
                            deliveryTo: "",
                            deliveryBestEffort: false,
                            deliveryFailureMode: "",
                            deliveryFailureChannel: "default",
                            deliveryFailureTo: "",
                            deliveryFailureAccountId: "",
                          }
                        : { ...prev, deliveryMode: nextMode }
                    );
                  }}
                >
                  <CardHeader size="compact" className="pb-2">
                    <div className="flex items-start justify-between gap-3">
                      <CardTitle className="text-sm">{t("cron.form.section.delivery")}</CardTitle>
                      <TabsList>
                        <TabsTrigger value="none">
                          {t("cron.form.deliveryModeOptions.none")}
                        </TabsTrigger>
                        <TabsTrigger value="announce">
                          {t("cron.form.deliveryModeOptions.announce")}
                        </TabsTrigger>
                        <TabsTrigger value="webhook">
                          {t("cron.form.deliveryModeOptions.webhook")}
                        </TabsTrigger>
                      </TabsList>
                    </div>
                  </CardHeader>
                  <CardContent size="compact" className="space-y-2">
                    <TabsContent value="none" className="mt-0" />

                    <TabsContent value="announce" className="mt-0">
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {t("cron.form.deliveryChannel")}
                          </span>
                          <Select
                            value={draft.deliveryChannel}
                            onChange={(event) => setDraft((prev) => ({ ...prev, deliveryChannel: event.target.value }))}
                            className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                          >
                            {deliveryChannelOptions.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </Select>
                        </div>

                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">{t("cron.form.deliveryTo")}</span>
                          <Input
                            value={draft.deliveryTo}
                            onChange={(event) => setDraft((prev) => ({ ...prev, deliveryTo: event.target.value }))}
                            size="compact"
                            className="ml-auto w-[68%] text-right"
                          />
                        </div>
                      </div>
                    </TabsContent>

                    <TabsContent value="webhook" className="mt-0">
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="flex items-center justify-between gap-3">
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {t("cron.form.deliveryToWebhook")}
                          </span>
                          <Input
                            value={draft.deliveryTo}
                            onChange={(event) => setDraft((prev) => ({ ...prev, deliveryTo: event.target.value }))}
                            size="compact"
                            className="ml-auto w-[68%] text-right"
                          />
                        </div>
                      </div>
                    </TabsContent>

                    {draft.deliveryMode !== "none" ? (
                      <>
                        <Separator className="bg-border/70" />

                        <div className="grid gap-3 py-1 md:grid-cols-2">
                          <div className="flex items-center justify-between gap-3">
                            <span className="text-xs text-muted-foreground">
                              {t("cron.form.deliveryBestEffort")}
                            </span>
                            <Switch
                              checked={draft.deliveryBestEffort}
                              onCheckedChange={(value) => setDraft((prev) => ({ ...prev, deliveryBestEffort: value }))}
                            />
                          </div>
                          <div className="flex items-center justify-between gap-3">
                            <span className="shrink-0 text-xs text-muted-foreground">
                              {t("cron.form.deliveryFailureMode")}
                            </span>
                            <Select
                              value={draft.deliveryFailureMode}
                              onChange={(event) =>
                                setDraft((prev) => ({
                                  ...prev,
                                  deliveryFailureMode: event.target.value as JobDraft["deliveryFailureMode"],
                                }))
                              }
                              className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                            >
                              <option value="">{t("cron.form.deliveryFailureModeOptions.none")}</option>
                              <option value="announce">{t("cron.form.deliveryFailureModeOptions.announce")}</option>
                              <option value="webhook">{t("cron.form.deliveryFailureModeOptions.webhook")}</option>
                            </Select>
                          </div>
                        </div>

                        {draft.deliveryFailureMode === "announce" ? (
                          <>
                            <Separator className="bg-border/70" />
                            <div className="grid gap-3 md:grid-cols-2">
                              <div className="flex items-center justify-between gap-3">
                                <span className="shrink-0 text-xs text-muted-foreground">
                                  {t("cron.form.deliveryFailureChannel")}
                                </span>
                                <Select
                                  value={draft.deliveryFailureChannel}
                                  onChange={(event) =>
                                    setDraft((prev) => ({ ...prev, deliveryFailureChannel: event.target.value }))
                                  }
                                  className={cn("ml-auto w-[68%] text-right", CRON_SELECT_TEXT_CLASS)}
                                >
                                  {deliveryChannelOptions.map((option) => (
                                    <option key={option.value} value={option.value}>
                                      {option.label}
                                    </option>
                                  ))}
                                </Select>
                              </div>
                              <div className="flex items-center justify-between gap-3">
                                <span className="shrink-0 text-xs text-muted-foreground">
                                  {t("cron.form.deliveryFailureAccountId")}
                                </span>
                                <Input
                                  value={draft.deliveryFailureAccountId}
                                  onChange={(event) =>
                                    setDraft((prev) => ({ ...prev, deliveryFailureAccountId: event.target.value }))
                                  }
                                  size="compact"
                                  className="ml-auto w-[68%] text-right"
                                />
                              </div>
                            </div>
                          </>
                        ) : null}

                        {draft.deliveryFailureMode === "webhook" ? (
                          <>
                            <Separator className="bg-border/70" />
                            <div className="grid gap-3 md:grid-cols-2">
                              <div className="flex items-center justify-between gap-3">
                                <span className="shrink-0 text-xs text-muted-foreground">
                                  {t("cron.form.deliveryFailureToWebhook")}
                                </span>
                                <Input
                                  value={draft.deliveryFailureTo}
                                  onChange={(event) =>
                                    setDraft((prev) => ({ ...prev, deliveryFailureTo: event.target.value }))
                                  }
                                  size="compact"
                                  className="ml-auto w-[68%] text-right"
                                />
                              </div>
                              <div className="flex items-center justify-between gap-3">
                                <span className="shrink-0 text-xs text-muted-foreground">
                                  {t("cron.form.deliveryFailureAccountId")}
                                </span>
                                <Input
                                  value={draft.deliveryFailureAccountId}
                                  onChange={(event) =>
                                    setDraft((prev) => ({ ...prev, deliveryFailureAccountId: event.target.value }))
                                  }
                                  size="compact"
                                  className="ml-auto w-[68%] text-right"
                                />
                              </div>
                            </div>
                          </>
                        ) : null}
                      </>
                    ) : null}
                  </CardContent>
                </Tabs>
              </PanelCard>
            </fieldset>

            <DialogFooter>
              {isDialogView ? (
                <Button variant="outline" size="compact" onClick={() => setDialogOpen(false)}>
                  {t("common.close")}
                </Button>
              ) : (
                <>
                  <Button variant="outline" size="compact" onClick={() => setDialogOpen(false)}>
                    {t("common.cancel")}
                  </Button>
                  <Button size="compact" onClick={() => void submitDialog()} disabled={dialogSaving}>
                    {dialogSaving ? t("common.saving") : t("common.save")}
                  </Button>
                </>
              )}
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={runDetailOpen} onOpenChange={setRunDetailOpen}>
          <DialogContent className="grid h-[90vh] max-w-4xl grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
            <DialogHeader>
              <DialogTitle className="pl-2">{t("cron.runs.detailTitle")}</DialogTitle>
            </DialogHeader>

            <div className="flex min-h-0 flex-col gap-4">
              <PanelCard tone="solid" className="overflow-hidden">
                <CardHeader size="compact" className="pb-2">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-sm">{t("cron.runs.overview")}</CardTitle>
                    <div className="max-w-[64%] text-right">
                      <span className="block select-none truncate font-mono text-[10px] tracking-[0.1em] text-muted-foreground/35">
                        {runDetailSessionKey}
                      </span>
                    </div>
                  </div>
                </CardHeader>
                <CardContent size="compact" className="grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-3">
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.status")}</span>
                    <div className="min-w-0">{renderRunStatusValue(runDetailRecord?.status, t)}</div>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.stage")}</span>
                    <span className="min-w-0 truncate">{runDetailStageLabel}</span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.deliveryStatus")}</span>
                    <div className="min-w-0">{renderRunDeliveryStatusValue(runDetailRecord?.deliveryStatus, t)}</div>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.duration")}</span>
                    <span className="min-w-0 truncate font-mono">{runDetailDurationText}</span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.time")}</span>
                    <span className="min-w-0 truncate font-mono text-muted-foreground">
                      {formatDateTime(runDetailRecord?.startedAt)}
                    </span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.ended")}</span>
                    <span className="min-w-0 truncate font-mono text-muted-foreground">
                      {formatDateTime(runDetailRecord?.endedAt)}
                    </span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.model")}</span>
                    <span className="min-w-0 truncate">{runDetailRecord?.model?.trim() || CRON_EMPTY_VALUE}</span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2">
                    <span className="text-muted-foreground">{t("cron.runs.provider")}</span>
                    <span className="min-w-0 truncate">{runDetailRecord?.provider?.trim() || CRON_EMPTY_VALUE}</span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2 sm:col-span-2 lg:col-span-3">
                    <span className="text-muted-foreground">{t("cron.runs.summary")}</span>
                    <span className="min-w-0 truncate">{runDetailRecord?.summary?.trim() || CRON_EMPTY_VALUE}</span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2 sm:col-span-2 lg:col-span-3">
                    <span className="text-muted-foreground">{t("cron.runs.deliveryError")}</span>
                    <span className="min-w-0 truncate text-destructive">
                      {runDetailRecord?.deliveryError?.trim() || CRON_EMPTY_VALUE}
                    </span>
                  </div>
                  <div className="grid grid-cols-[86px_minmax(0,1fr)] items-center gap-2 sm:col-span-2 lg:col-span-3">
                    <span className="text-muted-foreground">{t("cron.runs.error")}</span>
                    <span className="min-w-0 truncate text-destructive">{runDetailRecord?.error?.trim() || CRON_EMPTY_VALUE}</span>
                  </div>
                </CardContent>
              </PanelCard>

              <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <CardHeader size="compact" className="pb-2">
                  <div className="flex items-center justify-between gap-2">
                    <CardTitle className="text-sm">{t("cron.runs.lifecycle")}</CardTitle>
                    <span className="truncate text-xs text-muted-foreground">{runDetailLifecycleStatusText}</span>
                  </div>
                </CardHeader>
                <CardContent size="compact" className="min-h-0 flex-1">
                  <Tabs
                    value={selectedLifecycleStep?.id ?? "trigger"}
                    onValueChange={(value) => {
                      if (isLifecycleStepID(value)) {
                        setSelectedLifecycleStepID(value);
                      }
                    }}
                    className="grid h-full min-h-0 gap-3 lg:grid-cols-[220px_minmax(0,1fr)]"
                  >
                    <TabsList
                      className={cn(
                        "h-full w-full flex-col items-stretch justify-start gap-1 overflow-y-auto p-1 pr-1",
                        DASHBOARD_SOFT_SURFACE_CLASS
                      )}
                    >
                      {lifecycleSteps.map((step) => {
                        const stepLabel = t(step.labelKey);
                        return (
                          <TabsTrigger
                            key={step.id}
                            value={step.id}
                            className="h-auto w-full justify-start gap-1.5 px-1.5 py-1.5 text-left text-xs"
                          >
                            <Circle
                              className={cn(
                                "h-3.5 w-3.5 shrink-0",
                                step.state === "success" && "fill-emerald-500 text-emerald-500",
                                step.state === "failed" && "fill-rose-500 text-rose-500",
                                step.state === "pending" && "fill-amber-400/80 text-amber-500"
                              )}
                            />
                            <div className="flex min-w-0 w-full items-center justify-between gap-2">
                              <span className="min-w-0 flex-1 truncate text-xs font-medium">{stepLabel}</span>
                              <span className="shrink-0 text-[11px] text-muted-foreground">{step.events.length}</span>
                            </div>
                          </TabsTrigger>
                        );
                      })}
                    </TabsList>

                    <div className="h-full min-h-0">
                      {selectedLifecycleStep ? (
                        <div key={selectedLifecycleStep.id} className="h-full overflow-y-auto pr-0.5">
                          {selectedLifecycleStep.events.length === 0 ? (
                            selectedLifecycleStep.id === runDetailActiveLifecycleStepID &&
                            selectedLifecycleStep.state === "pending" ? (
                              <div
                                className={cn(
                                  "flex items-center gap-2 border-dashed px-3 py-6 text-xs text-muted-foreground",
                                  DASHBOARD_FIELD_SURFACE_CLASS
                                )}
                              >
                                <Loader2 className="h-4 w-4 animate-spin" />
                                {t("cron.runs.lifecycleWaitingPush")}
                              </div>
                            ) : (
                              <div
                                className={cn(
                                  "border-dashed px-3 py-6 text-center text-xs text-muted-foreground",
                                  DASHBOARD_FIELD_SURFACE_CLASS
                                )}
                              >
                                {t("cron.runs.lifecycleStepNoEvents")}
                              </div>
                            )
                          ) : (
                            <div className="space-y-1">
                              {selectedLifecycleStep.events.map((event, index) => {
                                const eventKey = event.eventId || `${selectedLifecycleStep.id}-${event.createdAt}-${index}`;
                                const expanded = Boolean(expandedLifecycleEvents[eventKey]);
                                return (
                                  <button
                                    key={eventKey}
                                    type="button"
                                    onClick={() => toggleLifecycleEventExpanded(eventKey)}
                                    className={cn(
                                      "w-full px-2.5 py-1.5 text-left transition-colors",
                                      DASHBOARD_FIELD_SURFACE_CLASS,
                                      expanded ? "border-primary/40 bg-primary/5" : "border-border hover:bg-muted/30"
                                    )}
                                  >
                                    <div className="flex items-center justify-between gap-2">
                                      <div className="min-w-0 flex items-center gap-2">
                                        <span className="truncate text-xs font-medium">
                                          {resolveRunStageValueLabel(event.stage, t)}
                                        </span>
                                      </div>
                                      <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
                                        {formatDateTime(event.createdAt)}
                                      </span>
                                    </div>
                                    <div className="mt-0.5 grid gap-0.5 text-[11px] text-muted-foreground sm:grid-cols-2">
                                      <div className="min-w-0 truncate">
                                        {t("cron.runs.channel")}: {event.channel || CRON_EMPTY_VALUE}
                                      </div>
                                      <div className="min-w-0 truncate">
                                        {t("cron.runs.source")}: {event.source || CRON_EMPTY_VALUE}
                                      </div>
                                    </div>
                                    {expanded ? (
                                      <div className="mt-1 border-t pt-1">
                                        <pre className="max-h-44 overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-all rounded bg-muted/60 p-1 text-[11px] leading-tight">
                                          {JSON.stringify(event, null, 2)}
                                        </pre>
                                      </div>
                                    ) : null}
                                  </button>
                                );
                              })}
                            </div>
                          )}
                        </div>
                      ) : (
                        <div
                          className={cn(
                            "border-dashed px-3 py-6 text-center text-xs text-muted-foreground",
                            DASHBOARD_FIELD_SURFACE_CLASS
                          )}
                        >
                          {t("cron.runs.lifecycleStepNoEvents")}
                        </div>
                      )}
                    </div>
                  </Tabs>
                </CardContent>
              </PanelCard>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </TooltipProvider>
  );
}

const CronTableSelectionCheckbox = React.forwardRef<
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

CronTableSelectionCheckbox.displayName = "CronTableSelectionCheckbox";

function CronJobFilterCombobox(props: {
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  enabledFilter: JobEnabledFilter;
  onEnabledFilterChange: (value: JobEnabledFilter) => void;
  lastRunStatusFilter: string;
  onLastRunStatusFilterChange: (value: string) => void;
  onClearAll: () => void;
  filterCount: number;
  t: TranslateFn;
}) {
  const hasFilters = props.filterCount > 0;
  const hasSearchQuery = props.searchQuery.trim().length > 0;
  const triggerLabel = hasSearchQuery ? props.searchQuery : props.t("cron.filter.searchAndFilterJobs");
  const enabledOptions: Array<{ value: JobEnabledFilter; label: string }> = [
    { value: "", label: props.t("cron.filter.allEnableStates") },
    { value: "enabled", label: props.t("cron.filter.enabledOnly") },
    { value: "disabled", label: props.t("cron.filter.disabledOnly") },
  ];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
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
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("cron.filter.searchJobsPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>{props.t("cron.filter.enableState")}</DropdownMenuLabel>
          {enabledOptions.map((option) => (
            <DropdownMenuCheckboxItem
              key={option.value || "all"}
              checked={props.enabledFilter === option.value}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onEnabledFilterChange(option.value);
                }
              }}
            >
              {option.label}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{props.t("cron.filter.lastRunStatus")}</DropdownMenuLabel>
          {RUN_STATUS_OPTIONS.map((status) => (
            <DropdownMenuCheckboxItem
              key={status || "all"}
              checked={props.lastRunStatusFilter === status}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onLastRunStatusFilterChange(status);
                }
              }}
            >
              {status ? props.t(`cron.status.${status}`) : props.t("cron.filter.allStatus")}
            </DropdownMenuCheckboxItem>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("cron.filter.clearJobFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CronRunFilterCombobox(props: {
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  jobFilter: string;
  onJobFilterChange: (value: string) => void;
  runStatusFilter: string;
  onRunStatusFilterChange: (value: string) => void;
  jobs: CronJob[];
  onClearAll: () => void;
  filterCount: number;
  t: TranslateFn;
}) {
  const hasFilters = props.filterCount > 0;
  const hasSearchQuery = props.searchQuery.trim().length > 0;
  const triggerLabel = hasSearchQuery ? props.searchQuery : props.t("cron.filter.searchAndFilterRuns");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
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
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("cron.filter.searchRunsPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>{props.t("cron.filter.job")}</DropdownMenuLabel>
          <DropdownMenuCheckboxItem
            checked={props.jobFilter === ""}
            onSelect={(event) => event.preventDefault()}
            onCheckedChange={(checked) => {
              if (checked) {
                props.onJobFilterChange("");
              }
            }}
          >
            {props.t("cron.filter.allJobs")}
          </DropdownMenuCheckboxItem>
          {props.jobs.map((job) => (
            <DropdownMenuCheckboxItem
              key={job.id}
              checked={props.jobFilter === job.id}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onJobFilterChange(job.id);
                }
              }}
            >
              {job.name.trim() || job.id}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{props.t("cron.filter.runStatus")}</DropdownMenuLabel>
          {RUN_STATUS_OPTIONS.map((status) => (
            <DropdownMenuCheckboxItem
              key={status || "all"}
              checked={props.runStatusFilter === status}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onRunStatusFilterChange(status);
                }
              }}
            >
              {status ? props.t(`cron.status.${status}`) : props.t("cron.filter.allStatus")}
            </DropdownMenuCheckboxItem>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("cron.filter.clearRunFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
