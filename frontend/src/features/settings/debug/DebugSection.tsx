import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Call } from "@wailsio/runtime";
import { Activity, BarChart3, Brain, Database, FileText, Plug2, RefreshCw, ScrollText, Wrench } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/shared/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import {
  useLLMCallRecord,
  useLLMCallRecords,
  useThreadRunEvents,
  useThreads,
  type LLMCallRecord,
  type ThreadRunEvent,
} from "@/shared/query/threads";
import { useGatewayHealth, useGatewayLogsTail, useGatewayStatus } from "@/shared/query/diagnostics";
import { useChannelsDebug } from "@/shared/query/channels";
import {
  DEFAULT_DEBUG_TOPICS,
  REALTIME_TOPICS,
  registerTopic,
  subscribeGatewayEvents,
  type GatewayEvent,
  useRealtimeStore,
} from "@/shared/realtime";
import { messageBus } from "@/shared/message";
import { useUpdateStore } from "@/shared/store/update";
import { ChannelsTab } from "./tabs/ChannelsTab";
import { CallRecordsTab } from "./tabs/CallRecordsTab";
import { EventsTab } from "./tabs/EventsTab";
import { FrameworkTab } from "./tabs/FrameworkTab";
import { OverviewTab } from "./tabs/OverviewTab";
import { PromptTab } from "./tabs/PromptTab";
import { RunTraceTab } from "./tabs/RunTraceTab";
import { ContextFilterMenu, ContextTableFooter } from "./tabs/ContextTabControls";
import { formatRelativeDebugTime } from "./utils/time";
import type {
  GatewayDebugEvent,
  ParsedRunEvent,
  PromptReportShape,
  PromptRunSnapshot,
  RunSummary,
} from "./types";

const MAX_GATEWAY_EVENTS = 600;
const RUN_EVENT_LIMIT = 1200;
const DEBUG_TAB_VALUES = ["overview", "calls", "context", "channels", "framework"] as const;
type DebugTabValue = (typeof DEBUG_TAB_VALUES)[number];
const CONTEXT_TAB_VALUES = ["events", "trace", "prompt"] as const;
type ContextTabValue = (typeof CONTEXT_TAB_VALUES)[number];

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function firstString(record: Record<string, unknown> | null, ...keys: string[]): string {
  if (!record) {
    return "";
  }
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed) {
        return trimmed;
      }
    }
  }
  return "";
}

function firstStringInRecords(records: Array<Record<string, unknown> | null>, ...keys: string[]): string {
  for (const record of records) {
    const value = firstString(record, ...keys);
    if (value) {
      return value;
    }
  }
  return "";
}

function resolveGatewayCandidateRecords(event: GatewayEvent): Array<Record<string, unknown> | null> {
  const eventRecord = asRecord(event as unknown);
  const payloadRecord = asRecord(event.payload);
  const payloadData = asRecord(payloadRecord?.data);
  const payloadMeta = asRecord(payloadRecord?.meta);
  const payloadContext = asRecord(payloadRecord?.context);
  const payloadPayload = asRecord(payloadRecord?.payload);
  const nestedPayloadData = asRecord(payloadPayload?.data);
  const nestedPayloadMeta = asRecord(payloadPayload?.meta);
  const nestedPayloadContext = asRecord(payloadPayload?.context);
  return [
    eventRecord,
    payloadRecord,
    payloadData,
    payloadMeta,
    payloadContext,
    payloadPayload,
    nestedPayloadData,
    nestedPayloadMeta,
    nestedPayloadContext,
  ];
}

function resolveGatewayRunId(event: GatewayEvent): string {
  return firstStringInRecords(
    resolveGatewayCandidateRecords(event),
    "runId",
    "run_id",
    "runID",
    "RunID",
    "parentRunId",
    "parent_run_id"
  );
}

function resolveGatewaySessionId(event: GatewayEvent): string {
  return firstStringInRecords(
    resolveGatewayCandidateRecords(event),
    "sessionId",
    "session_id",
    "sessionID",
    "SessionID",
    "threadId",
    "thread_id",
    "threadID",
    "ThreadID",
    "childSessionId",
    "child_session_id"
  );
}

function resolveGatewaySessionDisplay(event: GatewayEvent, sessionId: string): string {
  if (sessionId) {
    return sessionId;
  }
  const sessionKey = (event.sessionKey ?? "").trim();
  if (sessionKey) {
    return sessionKey;
  }
  return firstStringInRecords(resolveGatewayCandidateRecords(event), "sessionKey", "session_key", "SessionKey");
}

function parseJSON(value: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

function formatDateTime(value?: string | number): string {
  if (typeof value === "number") {
    if (!Number.isFinite(value)) {
      return "-";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "-";
    }
    return date.toLocaleString();
  }
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatRuntimeTime(value?: string | number): string {
  if (typeof value === "number") {
    if (!Number.isFinite(value)) {
      return "-";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "-";
    }
    return date.toLocaleTimeString();
  }
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleTimeString();
}

function parseRunEvent(row: ThreadRunEvent): ParsedRunEvent {
  const payload = parseJSON(row.payloadJson);
  const record = asRecord(payload);
  const chatEventType = firstString(record, "type");
  const eventDataRecord = asRecord(record?.data);
  const toolName =
    firstString(eventDataRecord, "toolName", "tool_name", "name") || firstString(record, "toolName", "tool_name");
  const toolCallId =
    firstString(eventDataRecord, "toolCallId", "tool_call_id") || firstString(record, "toolCallId", "tool_call_id");

  let runId = row.runId.trim();
  let agentEventName = "";
  let promptReport: PromptReportShape | null = null;

  if (chatEventType === "data-agent-event" && eventDataRecord) {
    agentEventName = firstString(eventDataRecord, "event");
    if (!runId) {
      runId = firstString(eventDataRecord, "runId");
    }
  }
  if (chatEventType === "prompt.report" && eventDataRecord) {
    promptReport = eventDataRecord as PromptReportShape;
    if (!runId) {
      runId = String(promptReport.runId ?? "").trim();
    }
  }

  const summary = agentEventName || chatEventType || row.eventType || "unknown";
  const parsedTime = Date.parse(row.createdAt);

  return {
    row,
    runId,
    chatEventType,
    agentEventName,
    summary,
    toolName,
    toolCallId,
    promptReport,
    rawRecord: record,
    createdAtMs: Number.isFinite(parsedTime) ? parsedTime : 0,
  };
}

function resolveRunStatus(events: ParsedRunEvent[]): RunSummary["status"] {
  let status: RunSummary["status"] = "unknown";
  for (const item of events) {
    const event = item.agentEventName;
    if (!event) {
      continue;
    }
    if (event === "run_error") {
      status = "error";
    } else if (event === "run_abort") {
      status = "aborted";
    } else if (event === "run_end") {
      status = "completed";
    } else if (event === "run_start" && status === "unknown") {
      status = "running";
    }
  }
  return status;
}

export function DebugSection() {
  const { language, t } = useI18n();
  const { status, url, topics, messages, metrics, clearMessages } = useRealtimeStore(
    useShallow((state) => ({
      status: state.status,
      url: state.url,
      topics: state.topics,
      messages: state.messages,
      metrics: state.metrics,
      clearMessages: state.clearMessages,
    }))
  );
  const currentAppVersion = useUpdateStore((state) => state.info.currentVersion);
  const latestAppVersion = useUpdateStore((state) => state.info.latestVersion);
  const openWhatsNewPreview = useUpdateStore((state) => state.openWhatsNewPreview);
  const [selectedTopic, setSelectedTopic] = useState<string>(DEFAULT_DEBUG_TOPICS[0]);
  const [selectedThreadId, setSelectedThreadId] = useState<string>("");
  const [selectedCallThreadId, setSelectedCallThreadId] = useState<string>("");
  const [activeTab, setActiveTab] = useState<DebugTabValue>("overview");
  const [activeContextTab, setActiveContextTab] = useState<ContextTabValue>("events");
  const [selectedRunId, setSelectedRunId] = useState<string>("all");
  const [callSource, setCallSource] = useState<string>("");
  const [callStatus, setCallStatus] = useState<string>("");
  const [callProviderFilter, setCallProviderFilter] = useState<string>("");
  const [callModelFilter, setCallModelFilter] = useState<string>("");
  const [callRunFilter, setCallRunFilter] = useState<string>("");
  const [selectedCallRecordId, setSelectedCallRecordId] = useState<string>("");
  const [logLevel, setLogLevel] = useState<string>("info");
  const [selectedGatewayEvent, setSelectedGatewayEvent] = useState<string>("all");
  const [eventRowsPerPage, setEventRowsPerPage] = useState(20);
  const [eventPageIndex, setEventPageIndex] = useState(0);
  const [gatewaySubscriptionVersion, setGatewaySubscriptionVersion] = useState(0);
  const [gatewayEvents, setGatewayEvents] = useState<GatewayDebugEvent[]>([]);
  const subscriptionsRef = useRef<Map<string, () => void>>(new Map());

  const { data: threads = [] } = useThreads(false);
  const runEventsQuery = useThreadRunEvents(selectedThreadId || null, { limit: RUN_EVENT_LIMIT });
  const healthQuery = useGatewayHealth();
  const gatewayStatusQuery = useGatewayStatus();
  const logsQuery = useGatewayLogsTail(
    { level: logLevel, limit: 200 },
    {
      realtime: activeTab === "framework",
      intervalMs: 2000,
    }
  );
  const channelDebugQuery = useChannelsDebug();
  const llmCallRecordListQuery = useMemo(
    () => ({
      threadId: selectedCallThreadId || undefined,
      runId: callRunFilter.trim() || undefined,
      providerId: callProviderFilter.trim() || undefined,
      modelName: callModelFilter.trim() || undefined,
      requestSource: callSource || undefined,
      status: callStatus || undefined,
      limit: 200,
    }),
    [callModelFilter, callProviderFilter, callRunFilter, callSource, callStatus, selectedCallThreadId]
  );
  const llmCallRecordOptionListQuery = useMemo(
    () => ({
      threadId: selectedCallThreadId || undefined,
      requestSource: callSource || undefined,
      status: callStatus || undefined,
      limit: 200,
    }),
    [callSource, callStatus, selectedCallThreadId]
  );
  const llmCallRecordsQuery = useLLMCallRecords(llmCallRecordListQuery);
  const llmCallRecordOptionsQuery = useLLMCallRecords(llmCallRecordOptionListQuery);
  const llmCallRecordQuery = useLLMCallRecord(selectedCallRecordId || null);

  const sortedThreads = useMemo(
    () => [...threads].sort((a, b) => (b.updatedAt || "").localeCompare(a.updatedAt || "")),
    [threads]
  );
  const defaultContextThreadId = sortedThreads[0]?.id ?? "";
  const selectedContextThread = useMemo(
    () => sortedThreads.find((thread) => thread.id === selectedThreadId) ?? null,
    [selectedThreadId, sortedThreads]
  );

  useEffect(() => {
    if (selectedThreadId || sortedThreads.length === 0) {
      return;
    }
    const latest = sortedThreads[0];
    if (latest?.id) {
      setSelectedThreadId(latest.id);
    }
  }, [selectedThreadId, sortedThreads]);

  useEffect(() => {
    const defaults = [
      ...DEFAULT_DEBUG_TOPICS,
      REALTIME_TOPICS.chat.threadUpdated,
      REALTIME_TOPICS.library.operation,
      REALTIME_TOPICS.library.file,
      REALTIME_TOPICS.library.history,
      REALTIME_TOPICS.library.workspace,
      "update.status",
    ];
    const cleanup = defaults.map((topic) => {
      const unsubscribe = registerTopic(topic);
      subscriptionsRef.current.set(topic, unsubscribe);
      return unsubscribe;
    });
    return () => {
      cleanup.forEach((unsubscribe) => unsubscribe());
      subscriptionsRef.current.forEach((unsubscribe) => unsubscribe());
      subscriptionsRef.current.clear();
    };
  }, []);

  useEffect(() => {
    const unsubscribe = subscribeGatewayEvents((event) => {
      const runId = resolveGatewayRunId(event);
      const resolvedSessionId = resolveGatewaySessionId(event);
      const normalizedSessionId = (event.sessionId ?? "").trim() || resolvedSessionId;
      const sessionDisplayId = resolveGatewaySessionDisplay(event, normalizedSessionId);
      const key = `${event.timestamp}-${event.event}-${runId}-${normalizedSessionId}-${Math.random()
        .toString(36)
        .slice(2, 8)}`;
      setGatewayEvents((prev) =>
        [...prev, { ...event, runId, sessionId: normalizedSessionId, sessionDisplayId, __key: key }].slice(
          -MAX_GATEWAY_EVENTS
        )
      );
    });
    return () => unsubscribe();
  }, [gatewaySubscriptionVersion]);

  useEffect(() => {
    if (selectedTopic === "all") {
      return;
    }
    if (topics.length > 0 && !topics.includes(selectedTopic)) {
      setSelectedTopic(topics[0]);
    }
  }, [selectedTopic, topics]);

  const refreshOverview = useCallback(() => {
    void healthQuery.refetch();
    void gatewayStatusQuery.refetch();
    void logsQuery.refetch();
    void runEventsQuery.refetch();
  }, [gatewayStatusQuery, healthQuery, logsQuery, runEventsQuery]);

  const sendOsNotification = async () => {
    try {
      const authorized = await Call.ByName(
        "dreamcreator/internal/presentation/wails.RealtimeHandler.RequestSystemNotificationAuthorization"
      );
      if (!authorized) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.debug.message.realtime.notifyDenied"),
        });
        return;
      }
      await Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.SendSystemNotification", {
        title: t("settings.debug.message.realtime.notifyTitle"),
        body: t("settings.debug.message.realtime.notifyBody"),
        subtitle: "DreamCreator",
        data: { source: "debug", ts: new Date().toISOString() },
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.debug.message.realtime.notifyError"),
        description: String(error),
      });
    }
  };

  const publishBackendDebug = async () => {
    try {
      const targetTopic =
        !selectedTopic || selectedTopic === "all" || selectedTopic === REALTIME_TOPICS.system.hello
          ? REALTIME_TOPICS.debug.echo
          : selectedTopic;
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.RealtimeHandler.PublishDebugEvent",
        targetTopic,
        {
          message: "Hello from frontend",
          ts: new Date().toISOString(),
          source: "settings.debug",
        }
      );
      messageBus.publishToast({
        intent: "success",
        title: t("settings.debug.message.realtime.publishSuccess"),
        description: `${t("settings.debug.message.realtime.topicLabel")}: ${targetTopic}`,
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.debug.message.realtime.publishFailed"),
        description: String(error),
      });
    }
  };

  const showToastPreview = useCallback(() => {
    messageBus.publishToast({
      intent: "warning",
      title: t("settings.debug.message.frontend.toastTitle"),
      description: t("settings.debug.message.frontend.toastDesc"),
    });
  }, [t]);

  const showNotificationPreview = useCallback(() => {
    const actionLabel = t("settings.debug.message.frontend.action");
    let notificationId = "";
    notificationId = messageBus.publishNotification({
      intent: "success",
      title: t("settings.debug.message.frontend.notificationTitle"),
      description: t("settings.debug.message.frontend.notificationDesc"),
      actions: [
        {
          label: actionLabel,
          onClick: () => messageBus.dismiss(notificationId),
        },
      ],
    });
  }, [t]);

  const showDialogPreview = useCallback(() => {
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.debug.message.frontend.dialogTitle"),
      description: t("settings.debug.message.frontend.dialogDesc"),
      confirmLabel: t("settings.debug.message.frontend.dialogConfirm"),
    });
  }, [t]);

  const showWhatsNewPreview = useCallback(() => {
    const version = currentAppVersion.trim() || latestAppVersion.trim() || "2.0.7";
    openWhatsNewPreview(
      {
        version,
        currentVersion: version,
        changelog: t("settings.debug.message.frontend.whatsNewMarkdown"),
      },
      "settings"
    );
  }, [currentAppVersion, latestAppVersion, openWhatsNewPreview, t]);

  const statusLabel = useMemo(() => {
    if (status === "connected") {
      return t("settings.debug.message.realtime.status.connected");
    }
    if (status === "connecting") {
      return t("settings.debug.message.realtime.status.connecting");
    }
    return t("settings.debug.message.realtime.status.disconnected");
  }, [status, t]);

  const topicOptions = useMemo(() => Array.from(new Set([...(topics ?? []), "all"])), [topics]);

  const llmCallRecords = useMemo(() => llmCallRecordsQuery.data ?? [], [llmCallRecordsQuery.data]);
  const selectedCallRecord = useMemo<LLMCallRecord | null>(() => {
    if (!selectedCallRecordId) {
      return null;
    }
    if (llmCallRecordQuery.data?.id === selectedCallRecordId) {
      return llmCallRecordQuery.data;
    }
    return llmCallRecords.find((item) => item.id === selectedCallRecordId) ?? null;
  }, [llmCallRecordQuery.data, llmCallRecords, selectedCallRecordId]);

  useEffect(() => {
    if (!selectedCallRecordId) {
      return;
    }
    const exists = llmCallRecords.some((item) => item.id === selectedCallRecordId);
    if (!exists) {
      setSelectedCallRecordId("");
    }
  }, [llmCallRecords, selectedCallRecordId]);

  const visibleMessages = useMemo(() => {
    if (selectedTopic === "all") {
      return Object.values(messages)
        .flat()
        .sort((a, b) => b.ts - a.ts);
    }
    return [...(messages[selectedTopic] ?? [])].sort((a, b) => b.ts - a.ts);
  }, [messages, selectedTopic]);

  const parsedRunEvents = useMemo(() => {
    const rawEvents = runEventsQuery.data ?? [];
    return rawEvents
      .map(parseRunEvent)
      .sort((a, b) => b.createdAtMs - a.createdAtMs || b.row.id - a.row.id);
  }, [runEventsQuery.data]);

  const runSummaries = useMemo(() => {
    const grouped = new Map<string, ParsedRunEvent[]>();
    for (const item of parsedRunEvents) {
      const runId = item.runId.trim();
      if (!runId) {
        continue;
      }
      const list = grouped.get(runId) ?? [];
      list.push(item);
      grouped.set(runId, list);
    }
    const summaries: RunSummary[] = [];
    for (const [runId, list] of grouped.entries()) {
      const sorted = [...list].sort((a, b) => a.createdAtMs - b.createdAtMs || a.row.id - b.row.id);
      const first = sorted[0];
      const last = sorted[sorted.length - 1];
      summaries.push({
        runId,
        firstAt: first?.row.createdAt ?? "",
        lastAt: last?.row.createdAt ?? "",
        eventCount: sorted.length,
        status: resolveRunStatus(sorted),
        lastEvent: last?.summary ?? "",
      });
    }
    return summaries.sort((a, b) => {
      const left = Date.parse(a.lastAt);
      const right = Date.parse(b.lastAt);
      return right - left;
    });
  }, [parsedRunEvents]);

  useEffect(() => {
    if (selectedRunId === "all") {
      return;
    }
    const exists = runSummaries.some((item) => item.runId === selectedRunId);
    if (!exists) {
      setSelectedRunId("all");
    }
  }, [selectedRunId, runSummaries]);

  const filteredRunEvents = useMemo(() => {
    if (selectedRunId === "all") {
      return parsedRunEvents;
    }
    return parsedRunEvents.filter((item) => item.runId === selectedRunId);
  }, [parsedRunEvents, selectedRunId]);

  const promptRuns = useMemo(() => {
    const latest = new Map<string, PromptRunSnapshot>();
    for (const item of parsedRunEvents) {
      if (!item.promptReport) {
        continue;
      }
      const runId = item.runId.trim() || String(item.promptReport.runId ?? "").trim();
      if (!runId) {
        continue;
      }
      const current = latest.get(runId);
      if (!current || Date.parse(item.row.createdAt) > Date.parse(current.createdAt)) {
        latest.set(runId, {
          runId,
          createdAt: item.row.createdAt,
          payload: item.promptReport,
        });
      }
    }
    return Array.from(latest.values()).sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
  }, [parsedRunEvents]);

  const selectedPromptRun = useMemo(() => {
    if (selectedRunId === "all") {
      return promptRuns[0] ?? null;
    }
    return promptRuns.find((item) => item.runId === selectedRunId) ?? null;
  }, [promptRuns, selectedRunId]);

  const gatewayEventsForThread = useMemo(() => {
    if (!selectedThreadId) {
      return gatewayEvents;
    }
    return gatewayEvents.filter((event) => {
      const sessionId = (event.sessionId ?? "").trim();
      return !sessionId || sessionId === selectedThreadId;
    });
  }, [gatewayEvents, selectedThreadId]);

  const gatewayEventOptions = useMemo(() => {
    const map = new Map<string, number>();
    for (const item of gatewayEventsForThread) {
      const key = item.event.trim() || t("settings.debug.gateway.eventUnknown");
      map.set(key, (map.get(key) ?? 0) + 1);
    }
    return Array.from(map.entries()).sort((a, b) => b[1] - a[1]);
  }, [gatewayEventsForThread, t]);

  useEffect(() => {
    if (selectedGatewayEvent === "all") {
      return;
    }
    const exists = gatewayEventOptions.some(([eventName]) => eventName === selectedGatewayEvent);
    if (!exists) {
      setSelectedGatewayEvent("all");
    }
  }, [selectedGatewayEvent, gatewayEventOptions]);

  const gatewayFilteredEvents = useMemo(() => {
    if (selectedGatewayEvent === "all") {
      return gatewayEventsForThread;
    }
    return gatewayEventsForThread.filter((item) => {
      const name = item.event.trim() || t("settings.debug.gateway.eventUnknown");
      return name === selectedGatewayEvent;
    });
  }, [gatewayEventsForThread, selectedGatewayEvent, t]);

  const sortedGatewayEvents = useMemo(
    () =>
      [...gatewayFilteredEvents].sort((left, right) => {
        const leftTime = Number.isFinite(left.timestamp) ? left.timestamp : 0;
        const rightTime = Number.isFinite(right.timestamp) ? right.timestamp : 0;
        return rightTime - leftTime;
      }),
    [gatewayFilteredEvents]
  );

  const gatewayEventPageCount = useMemo(
    () => Math.max(1, Math.ceil(sortedGatewayEvents.length / eventRowsPerPage)),
    [eventRowsPerPage, sortedGatewayEvents.length]
  );

  useEffect(() => {
    setEventPageIndex((current) => Math.min(current, gatewayEventPageCount - 1));
  }, [gatewayEventPageCount]);

  useEffect(() => {
    setEventPageIndex(0);
  }, [selectedThreadId, selectedGatewayEvent]);

  const paginatedGatewayEvents = useMemo(() => {
    const start = eventPageIndex * eventRowsPerPage;
    return sortedGatewayEvents.slice(start, start + eventRowsPerPage);
  }, [eventPageIndex, eventRowsPerPage, sortedGatewayEvents]);

  const gatewayEventsFirstAt = useMemo(
    () =>
      sortedGatewayEvents.length > 0
        ? formatRelativeDebugTime(sortedGatewayEvents[sortedGatewayEvents.length - 1]?.timestamp, language, t("common.justNow"))
        : "-",
    [language, sortedGatewayEvents, t]
  );

  const gatewayEventsLastAt = useMemo(
    () =>
      sortedGatewayEvents.length > 0
        ? formatRelativeDebugTime(sortedGatewayEvents[0]?.timestamp, language, t("common.justNow"))
        : "-",
    [language, sortedGatewayEvents, t]
  );

  const contextThreadOptions = useMemo(() => {
    if (sortedThreads.length === 0) {
      return [{ value: "", label: t("settings.debug.thread.empty") }];
    }
    return sortedThreads.map((thread) => ({
      value: thread.id,
      label: thread.title || thread.id,
    }));
  }, [sortedThreads, t]);

  const contextGatewayEventOptions = useMemo(
    () => [
      { value: "all", label: t("settings.debug.gateway.filterAll") },
      ...gatewayEventOptions.map(([eventName, count]) => ({
        value: eventName,
        label: `${eventName} (${count})`,
      })),
    ],
    [gatewayEventOptions, t]
  );

  const contextRunOptions = useMemo(
    () => [
      { value: "all", label: t("settings.debug.contextPanel.filters.allRuns") },
      ...runSummaries.map((item) => ({
        value: item.runId,
        label: item.runId,
      })),
    ],
    [runSummaries, t]
  );

  const contextFilterCount = useMemo(() => {
    const threadCount = selectedThreadId && selectedThreadId !== defaultContextThreadId ? 1 : 0;
    if (activeContextTab === "events") {
      return threadCount + (selectedGatewayEvent !== "all" ? 1 : 0);
    }
    return threadCount + (selectedRunId !== "all" ? 1 : 0);
  }, [activeContextTab, defaultContextThreadId, selectedGatewayEvent, selectedRunId, selectedThreadId]);

  const contextFilterTriggerLabel = useMemo(() => {
    const threadLabel = selectedContextThread?.title || selectedContextThread?.id || "";
    if (activeContextTab === "events" && selectedGatewayEvent !== "all") {
      return selectedGatewayEvent;
    }
    if ((activeContextTab === "trace" || activeContextTab === "prompt") && selectedRunId !== "all") {
      return selectedRunId;
    }
    return threadLabel || t("settings.debug.contextPanel.actions.filter");
  }, [activeContextTab, selectedContextThread, selectedGatewayEvent, selectedRunId, t]);

  const clearContextFilters = useCallback(() => {
    setSelectedThreadId(defaultContextThreadId);
    setSelectedGatewayEvent("all");
    setSelectedRunId("all");
  }, [defaultContextThreadId]);

  const inspectCallRecordRun = useCallback(
    (record: LLMCallRecord) => {
      if (record.threadId) {
        setSelectedThreadId(record.threadId);
      }
      setActiveTab("context");
      if (record.runId) {
        setActiveContextTab("trace");
        setSelectedRunId(record.runId);
      } else {
        setActiveContextTab("events");
        setSelectedRunId("all");
      }
    },
    []
  );

  const refreshContextTab = useCallback(() => {
    if (activeContextTab === "events") {
      setGatewaySubscriptionVersion((current) => current + 1);
      return;
    }
    if (activeContextTab === "trace" || activeContextTab === "prompt") {
      void runEventsQuery.refetch();
    }
  }, [activeContextTab, runEventsQuery]);

  return (
    <div className="flex min-h-0 flex-1 flex-col space-y-4">
      <Tabs
        value={activeTab}
        onValueChange={(value) => {
          if ((DEBUG_TAB_VALUES as readonly string[]).includes(value)) {
            setActiveTab(value as DebugTabValue);
          }
        }}
        className="flex min-h-0 flex-1 flex-col space-y-4"
      >
        <div className="flex justify-center">
          <TabsList className="w-fit max-w-full justify-center overflow-x-auto overflow-y-hidden">
            <TabsTrigger value="overview" className="min-w-0">
              <BarChart3 className="h-4 w-4" />
              <span className="truncate">{t("settings.debug.tabs.overview")}</span>
            </TabsTrigger>
            <TabsTrigger value="calls" className="min-w-0">
              <Database className="h-4 w-4" />
              <span className="truncate">{t("settings.debug.tabs.calls")}</span>
            </TabsTrigger>
            <TabsTrigger value="context" className="min-w-0">
              <Brain className="h-4 w-4" />
              <span className="truncate">{t("settings.debug.tabs.context")}</span>
            </TabsTrigger>
            <TabsTrigger value="channels" className="min-w-0">
              <Plug2 className="h-4 w-4" />
              <span className="truncate">{t("settings.debug.tabs.channels")}</span>
            </TabsTrigger>
            <TabsTrigger value="framework" className="min-w-0">
              <Wrench className="h-4 w-4" />
              <span className="truncate">{t("settings.debug.tabs.framework")}</span>
            </TabsTrigger>
          </TabsList>
        </div>
        {activeTab === "overview" ? (
          <OverviewTab
            t={t}
            refreshOverview={refreshOverview}
            gatewayStatus={gatewayStatusQuery.data}
            health={healthQuery.data}
            url={url}
            statusLabel={statusLabel}
            status={status}
            metrics={metrics}
            selectedTopic={selectedTopic}
            setSelectedTopic={setSelectedTopic}
            topicOptions={topicOptions}
            visibleMessages={visibleMessages}
            messages={messages}
            clearMessages={() => clearMessages()}
          />
        ) : null}

        {activeTab === "calls" ? (
          <div className="min-h-0 flex-1 overflow-hidden">
            <CallRecordsTab
              t={t}
              threads={sortedThreads}
              optionRecords={llmCallRecordOptionsQuery.data ?? []}
              selectedThreadId={selectedCallThreadId}
              setSelectedThreadId={setSelectedCallThreadId}
              callSource={callSource}
              setCallSource={setCallSource}
              callStatus={callStatus}
              setCallStatus={setCallStatus}
              providerFilter={callProviderFilter}
              setProviderFilter={setCallProviderFilter}
              modelFilter={callModelFilter}
              setModelFilter={setCallModelFilter}
              runFilter={callRunFilter}
              setRunFilter={setCallRunFilter}
              records={llmCallRecords}
              selectedRecord={selectedCallRecord}
              setSelectedRecordId={setSelectedCallRecordId}
              isLoading={llmCallRecordsQuery.isLoading}
              hasError={Boolean(llmCallRecordsQuery.error)}
              isDetailLoading={llmCallRecordQuery.isLoading}
              refresh={() => {
                void llmCallRecordsQuery.refetch();
                if (selectedCallRecordId) {
                  void llmCallRecordQuery.refetch();
                }
              }}
              formatDateTime={formatDateTime}
              inspectRun={inspectCallRecordRun}
            />
          </div>
        ) : null}

        {activeTab === "context" ? (
          <div className="min-h-0 flex-1 overflow-hidden">
            <Tabs
              value={activeContextTab}
              onValueChange={(value) => {
                if ((CONTEXT_TAB_VALUES as readonly string[]).includes(value)) {
                  setActiveContextTab(value as ContextTabValue);
                }
              }}
              className="flex h-full min-h-0 flex-1 flex-col gap-3"
            >
              <div className="flex min-w-0 flex-nowrap items-center justify-between gap-3 overflow-x-auto pb-1 -mb-1">
                <TabsList className="h-auto shrink-0 rounded-none bg-transparent p-0">
                  <TabsTrigger
                    value="events"
                    className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  >
                    <Activity className="h-4 w-4" />
                    <span className="truncate">{t("settings.debug.tabs.events")}</span>
                  </TabsTrigger>
                  <TabsTrigger
                    value="trace"
                    className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  >
                    <ScrollText className="h-4 w-4" />
                    <span className="truncate">{t("settings.debug.tabs.trace")}</span>
                  </TabsTrigger>
                  <TabsTrigger
                    value="prompt"
                    className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  >
                    <FileText className="h-4 w-4" />
                    <span className="truncate">{t("settings.debug.tabs.prompt")}</span>
                  </TabsTrigger>
                </TabsList>

                <div className="ml-auto flex min-w-0 items-center gap-2">
                  <ContextFilterMenu
                    t={t}
                    triggerLabel={contextFilterTriggerLabel}
                    filterCount={contextFilterCount}
                    fields={
                      activeContextTab === "events"
                        ? [
                            {
                              label: t("settings.debug.contextPanel.filters.thread"),
                              value: selectedThreadId,
                              onChange: setSelectedThreadId,
                              options: contextThreadOptions,
                            },
                            {
                              label: t("settings.debug.contextPanel.filters.event"),
                              value: selectedGatewayEvent,
                              onChange: setSelectedGatewayEvent,
                              options: contextGatewayEventOptions,
                            },
                          ]
                        : [
                            {
                              label: t("settings.debug.contextPanel.filters.thread"),
                              value: selectedThreadId,
                              onChange: setSelectedThreadId,
                              options: contextThreadOptions,
                            },
                            {
                              label: t("settings.debug.contextPanel.filters.runId"),
                              value: selectedRunId,
                              onChange: setSelectedRunId,
                              options: contextRunOptions,
                            },
                          ]
                    }
                    onClearAll={clearContextFilters}
                    disabled={sortedThreads.length === 0}
                  />
                  <Button
                    size="compactIcon"
                    variant="outline"
                    className="shrink-0"
                    aria-label={t("common.refresh")}
                    title={t("common.refresh")}
                    onClick={refreshContextTab}
                  >
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
                {activeContextTab === "events" ? (
                  <EventsTab
                    t={t}
                    gatewayFilteredEvents={paginatedGatewayEvents}
                    formatDateTime={formatDateTime}
                    formatRuntimeTime={formatRuntimeTime}
                  />
                ) : null}

                {activeContextTab === "trace" ? (
                  <RunTraceTab
                    t={t}
                    filteredRunEvents={filteredRunEvents}
                    runEventsLoading={runEventsQuery.isLoading}
                    runEventsError={Boolean(runEventsQuery.error)}
                    formatDateTime={formatDateTime}
                  />
                ) : null}

                {activeContextTab === "prompt" ? (
                  <PromptTab
                    t={t}
                    selectedRunId={selectedRunId}
                    runEventsLoading={runEventsQuery.isLoading}
                    runEventsError={Boolean(runEventsQuery.error)}
                    selectedPromptRun={selectedPromptRun}
                    formatDateTime={formatDateTime}
                  />
                ) : null}
              </div>

              {activeContextTab === "events" ? (
                <ContextTableFooter
                  t={t}
                  className="gap-y-2"
                  stats={[
                    { label: t("settings.debug.contextPanel.footer.eventCount"), value: sortedGatewayEvents.length },
                    { label: t("settings.debug.contextPanel.footer.firstAt"), value: gatewayEventsFirstAt },
                    { label: t("settings.debug.contextPanel.footer.lastAt"), value: gatewayEventsLastAt },
                  ]}
                  rowsPerPage={eventRowsPerPage}
                  onRowsPerPageChange={(value) => {
                    setEventRowsPerPage(value);
                    setEventPageIndex(0);
                  }}
                  pageIndex={eventPageIndex}
                  pageCount={gatewayEventPageCount}
                  onPrevPage={() => setEventPageIndex((current) => Math.max(0, current - 1))}
                  onNextPage={() => setEventPageIndex((current) => Math.min(gatewayEventPageCount - 1, current + 1))}
                />
              ) : null}
            </Tabs>
          </div>
        ) : null}

        {activeTab === "channels" ? (
          <ChannelsTab
            t={t}
            isLoading={channelDebugQuery.isLoading}
            hasError={Boolean(channelDebugQuery.error)}
            isFetching={channelDebugQuery.isFetching}
            data={channelDebugQuery.data}
            refetch={() => {
              void channelDebugQuery.refetch();
            }}
            formatDateTime={formatDateTime}
          />
        ) : null}

        {activeTab === "framework" ? (
          <FrameworkTab
            t={t}
            logLevel={logLevel}
            setLogLevel={setLogLevel}
            logsLoading={logsQuery.isLoading}
            logsError={Boolean(logsQuery.error)}
            logRecords={logsQuery.data?.records ?? []}
            formatRuntimeTime={formatRuntimeTime}
            showToastPreview={showToastPreview}
            showNotificationPreview={showNotificationPreview}
            showDialogPreview={showDialogPreview}
            showWhatsNewPreview={showWhatsNewPreview}
            sendOsNotification={() => {
              void sendOsNotification();
            }}
            publishBackendDebug={() => {
              void publishBackendDebug();
            }}
          />
        ) : null}
      </Tabs>
    </div>
  );
}
