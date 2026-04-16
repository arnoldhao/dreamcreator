import type { ConnectionStatus, RealtimeEvent } from "@/app/ws/client";
import type { HealthSnapshotEntity, LogRecordEntity, StatusReportEntity } from "@/entities/observability";
import type { GatewayEvent } from "@/shared/realtime";
import type { LLMCallRecord, ThreadRunEvent } from "@/shared/query/threads";
import type { ChannelDebugSnapshot } from "@/shared/store/channels";

export type TranslateFn = (key: string) => string;

export type RealtimeMetrics = {
  reconnects: number;
  replayEvents: number;
  resyncRequired: number;
  duplicateDrops: number;
};

export type PromptReportShape = {
  runId?: string;
  mode?: string;
  prompt?: string;
  promptChars?: number;
  messages?: Array<{
    role?: string;
    content?: string;
    reasoning?: string;
    toolCallId?: string;
  }>;
  sectionLabels?: Record<string, string>;
  sectionsDetailed?: Array<{
    id?: string;
    label?: string;
    content?: string;
    tokens?: number;
    truncated?: boolean;
    reason?: string;
  }>;
  report?: {
    generatedAt?: string;
    truncated?: boolean;
    sections?: Array<{
      id?: string;
      tokens?: number;
      truncated?: boolean;
      reason?: string;
    }>;
  };
  tools?: string[];
  skills?: string[];
};

export type ParsedRunEvent = {
  row: ThreadRunEvent;
  runId: string;
  chatEventType: string;
  agentEventName: string;
  summary: string;
  toolName: string;
  toolCallId: string;
  promptReport: PromptReportShape | null;
  rawRecord: Record<string, unknown> | null;
  createdAtMs: number;
};

export type RunSummary = {
  runId: string;
  firstAt: string;
  lastAt: string;
  eventCount: number;
  status: "running" | "completed" | "error" | "aborted" | "unknown";
  lastEvent: string;
};

export type PromptRunSnapshot = {
  runId: string;
  createdAt: string;
  payload: PromptReportShape;
};

export type GatewayDebugEvent = GatewayEvent & {
  __key: string;
  runId: string;
  sessionDisplayId: string;
};

export type OverviewTabProps = {
  t: TranslateFn;
  refreshOverview: () => void;
  gatewayStatus?: StatusReportEntity;
  health?: HealthSnapshotEntity;
  url: string;
  statusLabel: string;
  status: ConnectionStatus;
  metrics: RealtimeMetrics;
  selectedTopic: string;
  setSelectedTopic: (value: string) => void;
  topicOptions: string[];
  visibleMessages: RealtimeEvent[];
  messages: Record<string, RealtimeEvent[]>;
  clearMessages: () => void;
};

export type EventsTabProps = {
  t: TranslateFn;
  gatewayFilteredEvents: GatewayDebugEvent[];
  formatDateTime: (value?: string | number) => string;
  formatRuntimeTime: (value?: string | number) => string;
};

export type RealtimeLogsCardProps = {
  t: TranslateFn;
  logLevel: string;
  setLogLevel: (value: string) => void;
  logsLoading: boolean;
  logsError: boolean;
  logRecords: LogRecordEntity[];
  formatRuntimeTime: (value?: string | number) => string;
};

export type RunTraceTabProps = {
  t: TranslateFn;
  filteredRunEvents: ParsedRunEvent[];
  runEventsLoading: boolean;
  runEventsError: boolean;
  formatDateTime: (value?: string | number) => string;
};

export type RunSummaryCardProps = {
  t: TranslateFn;
  runSummaries: RunSummary[];
  selectedRunId: string;
  setSelectedRunId: (value: string) => void;
  formatDateTime: (value?: string | number) => string;
  statusLabelClass: (status: RunSummary["status"]) => string;
  formatRunStatus: (status: RunSummary["status"]) => string;
  emptyText: string;
};

export type PromptTabProps = {
  t: TranslateFn;
  selectedRunId: string;
  runEventsLoading: boolean;
  runEventsError: boolean;
  selectedPromptRun: PromptRunSnapshot | null;
  formatDateTime: (value?: string | number) => string;
};

export type ChannelsTabProps = {
  t: TranslateFn;
  isLoading: boolean;
  hasError: boolean;
  isFetching: boolean;
  data?: ChannelDebugSnapshot[];
  refetch: () => void;
  formatDateTime: (value?: string | number) => string;
};

export type FrameworkTabProps = {
  t: TranslateFn;
  logLevel: string;
  setLogLevel: (value: string) => void;
  logsLoading: boolean;
  logsError: boolean;
  logRecords: LogRecordEntity[];
  formatRuntimeTime: (value?: string | number) => string;
  showToastPreview: () => void;
  showNotificationPreview: () => void;
  showDialogPreview: () => void;
  sendOsNotification: () => void;
  publishBackendDebug: () => void;
};

export type CallRecordsTabProps = {
  t: TranslateFn;
  threads: Array<{ id: string; title: string; updatedAt?: string }>;
  optionRecords: LLMCallRecord[];
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
  records: LLMCallRecord[];
  selectedRecord: LLMCallRecord | null;
  setSelectedRecordId: (value: string) => void;
  isLoading: boolean;
  hasError: boolean;
  isDetailLoading: boolean;
  refresh: () => void;
  formatDateTime: (value?: string | number) => string;
  inspectRun: (record: LLMCallRecord) => void;
};
