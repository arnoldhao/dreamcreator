export type CronScheduleType = "cron" | "every" | "at";
export type CronSessionTarget = "main" | "isolated";
export type CronWakeMode = "now" | "next-heartbeat";
export type CronPayloadKind = "systemEvent" | "agentTurn";
export type CronDeliveryMode = "none" | "announce" | "webhook";

export interface CronSchedule {
  kind: CronScheduleType;
  at?: string;
  everyMs?: number;
  anchorMs?: number;
  expr?: string;
  tz?: string;
  staggerMs?: number;
}

export interface CronPayloadSpec {
  kind: CronPayloadKind;
  text?: string;
  message?: string;
  model?: string;
  thinking?: string;
  timeoutSeconds?: number;
  lightContext?: boolean;
}

export interface CronFailureDestination {
  mode?: "announce" | "webhook";
  channel?: "default" | "app" | "telegram";
  to?: string;
  accountId?: string;
}

export interface CronDelivery {
  mode: CronDeliveryMode;
  channel?: string;
  to?: string;
  accountId?: string;
  bestEffort?: boolean;
  failureDestination?: CronFailureDestination;
}

export interface CronJobState {
  nextRunAtMs?: number;
  runningAtMs?: number;
  lastRunAtMs?: number;
  lastRunStatus?: string;
  lastError?: string;
  lastDurationMs?: number;
  consecutiveErrors?: number;
  scheduleErrorCount?: number;
  lastDeliveryStatus?: string;
  lastDeliveryError?: string;
  lastDelivered?: boolean;
}

export interface CronJob {
  id: string;
  assistantId: string;
  name: string;
  description?: string;
  schedule: CronSchedule;
  sessionTarget: CronSessionTarget;
  wakeMode: CronWakeMode;
  payload: CronPayloadSpec;
  delivery?: CronDelivery;
  sourceChannel?: string;
  state?: CronJobState;
  deleteAfterRun: boolean;
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface CronRunRecord {
  runId: string;
  jobId: string;
  jobName?: string;
  status: string;
  startedAt: string;
  endedAt?: string;
  deliveryStatus?: string;
  deliveryError?: string;
  model?: string;
  provider?: string;
  sessionKey?: string;
  summary?: string;
  usageJson?: string;
  latestStage?: string;
  error?: string;
}

export interface CronRunEvent {
  eventId: string;
  runId: string;
  jobId: string;
  jobName?: string;
  stage: string;
  status?: string;
  message?: string;
  error?: string;
  channel?: string;
  sessionKey?: string;
  source?: string;
  meta?: Record<string, unknown>;
  createdAt: string;
}

export interface CronRunDetail {
  run: CronRunRecord;
  events: CronRunEvent[];
  eventsTotal: number;
}

export interface CronStatus {
  enabled: boolean;
  jobs: number;
  nextWakeAt?: string;
  nextWakeAtMs?: number;
}

export interface CronJobsResponse {
  items: CronJob[];
  total: number;
  offset: number;
  limit: number;
  hasMore: boolean;
  nextOffset: number;
}

export interface CronRunsResponse {
  items: CronRunRecord[];
  total: number;
  offset: number;
  limit: number;
  hasMore: boolean;
  nextOffset: number;
}

export interface CronRunEventsResponse {
  items: CronRunEvent[];
  total: number;
  offset: number;
  limit: number;
  hasMore: boolean;
  nextOffset: number;
}

export interface CronListQuery {
  includeDisabled?: boolean;
  enabled?: "all" | "enabled" | "disabled";
  query?: string;
  sortBy?: "nextRunAtMs" | "updatedAtMs" | "name";
  sortDir?: "asc" | "desc";
  limit?: number;
  offset?: number;
}

export interface CronCreateRequest {
  id?: string;
  assistantId: string;
  name: string;
  description?: string;
  enabled: boolean;
  deleteAfterRun?: boolean;
  schedule: CronSchedule;
  sessionTarget: CronSessionTarget;
  wakeMode: CronWakeMode;
  payload: CronPayloadSpec;
  delivery?: CronDelivery;
  sessionKey?: string;
}

export interface CronPatchRequest {
  assistantId?: string;
  name?: string;
  description?: string;
  enabled?: boolean;
  deleteAfterRun?: boolean;
  schedule?: CronSchedule;
  sessionTarget?: CronSessionTarget;
  wakeMode?: CronWakeMode;
  payload?: CronPayloadSpec;
  delivery?: CronDelivery;
  sessionKey?: string;
}

export interface CronUpdateRequest {
  id: string;
  patch: CronPatchRequest;
}

export interface CronRemoveRequest {
  id: string;
}

export interface CronRunRequest {
  id: string;
  mode: "due" | "force";
}

export interface CronRunsQuery {
  scope?: "job" | "all";
  id?: string;
  statuses?: string[];
  deliveryStatuses?: string[];
  query?: string;
  sortDir?: "asc" | "desc";
  limit?: number;
  offset?: number;
}

export interface CronRunDetailRequest {
  runId: string;
  eventsLimit?: number;
}

export interface CronRunEventsRequest {
  runId: string;
  sortDir?: "asc" | "desc";
  limit?: number;
  offset?: number;
}
