import { useEffect, useRef } from "react";
import type { PropsWithChildren } from "react";
import { QueryClientProvider } from "@tanstack/react-query";

import {
  registerTopic,
  REALTIME_TOPICS,
  subscribeGatewayEvents,
  startRealtime,
  type GatewayEvent,
} from "@/shared/realtime";
import { AssistantUIRuntimeProvider } from "@/shared/assistant/runtime-provider";
import { createQueryClient } from "./query-client";
import { normalizeUpdateInfo, useUpdateStore, type UpdateInfo } from "@/shared/store/update";
import { toOperationListItem, useLibraryRealtimeStore } from "@/shared/store/libraryRealtime";
import type {
  LibraryFileDTO,
  LibraryHistoryRecordDTO,
  LibraryOperationDTO,
  OperationListItemDTO,
  WorkspaceStateRecordDTO,
} from "@/shared/contracts/library";
import {
  LIBRARY_DETAIL_QUERY_KEY,
  LIBRARY_FILE_EVENTS_QUERY_KEY,
  LIBRARY_HISTORY_QUERY_KEY,
  LIBRARY_LIST_QUERY_KEY,
  LIBRARY_OPERATIONS_QUERY_KEY,
  LIBRARY_WORKSPACE_QUERY_KEY,
  LIBRARY_WORKSPACE_PROJECT_QUERY_KEY,
} from "@/shared/query/library";
import { Call, Events } from "@wailsio/runtime";
import { assistantsKey } from "@/shared/query/assistant";
import { ENABLED_PROVIDERS_WITH_MODELS_KEY, PROVIDERS_QUERY_KEY } from "@/shared/query/providers";
import { queryKeys } from "@/shared/query/keys";
import { EXTERNAL_TOOLS_QUERY_KEY } from "@/shared/query/externalTools";
import {
  noticeSeverityToIntent,
  resolveNoticeBody,
  resolveNoticeSummary,
  resolveNoticeTitle,
  type Notice,
  type NoticeAction,
} from "@/shared/contracts/notice";
import {
  SKILLS_CATALOG_QUERY_KEY,
  SKILLS_DETAIL_QUERY_KEY,
  SKILLS_SEARCH_QUERY_KEY,
  SKILLS_STATUS_QUERY_KEY,
} from "@/shared/query/skills";
import { threadsKey } from "@/shared/query/threads";
import { messageBus } from "@/shared/message";
import { t } from "@/shared/i18n";
import { useThreadStore, type ThreadSummary, type ThreadStatus } from "@/shared/store/threads";
import { TelemetryManager } from "@/shared/telemetry/manager";

const queryClient = createQueryClient();

type ThreadUpdatedEventPayload = {
  threadId?: string;
  change?: string;
  reason?: string;
  thread?: Partial<ThreadSummary> | null;
};

const parseThreadTime = (value: string) => {
  if (!value) {
    return 0;
  }
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

const normalizeThreadStatus = (value: unknown): ThreadStatus =>
  typeof value === "string" && value.trim().toLowerCase() === "archived" ? "archived" : "regular";

const normalizeThreadSummaryPayload = (
  raw: Partial<ThreadSummary> | null | undefined,
  threadId: string,
  current?: ThreadSummary
): ThreadSummary | null => {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const resolvedId = typeof raw.id === "string" && raw.id.trim() ? raw.id.trim() : threadId;
  if (!resolvedId) {
    return null;
  }
  const resolvedUpdatedAt =
    typeof raw.updatedAt === "string" && raw.updatedAt.trim()
      ? raw.updatedAt
      : current?.updatedAt ?? "";
  const resolvedLastInteractiveAt =
    typeof raw.lastInteractiveAt === "string" && raw.lastInteractiveAt.trim()
      ? raw.lastInteractiveAt
      : current?.lastInteractiveAt ?? resolvedUpdatedAt;
  return {
    id: resolvedId,
    assistantId:
      typeof raw.assistantId === "string" ? raw.assistantId : current?.assistantId ?? "",
    title: typeof raw.title === "string" && raw.title.trim() ? raw.title : current?.title ?? "",
    titleIsDefault:
      typeof raw.titleIsDefault === "boolean"
        ? raw.titleIsDefault
        : current?.titleIsDefault ?? true,
    titleChangedBy:
      raw.titleChangedBy === "user" || raw.titleChangedBy === "summary"
        ? raw.titleChangedBy
        : current?.titleChangedBy,
    status:
      raw.status !== undefined
        ? normalizeThreadStatus(raw.status)
        : current?.status ?? "regular",
    createdAt:
      typeof raw.createdAt === "string" ? raw.createdAt : current?.createdAt ?? "",
    updatedAt: resolvedUpdatedAt,
    lastInteractiveAt: resolvedLastInteractiveAt,
    deletedAt:
      typeof raw.deletedAt === "string" ? raw.deletedAt : current?.deletedAt ?? "",
    purgeAfter:
      typeof raw.purgeAfter === "string" ? raw.purgeAfter : current?.purgeAfter ?? "",
    workspaceName:
      typeof raw.workspaceName === "string"
        ? raw.workspaceName
        : current?.workspaceName ?? "",
  };
};

const upsertThreadInList = (
  previous: ThreadSummary[] | undefined,
  thread: ThreadSummary,
  includeDeleted: boolean
) => {
  const next = (previous ?? []).filter((item) => item.id !== thread.id);
  if (includeDeleted || !thread.deletedAt) {
    next.unshift(thread);
  }
  next.sort(
    (left, right) =>
      parseThreadTime(right.lastInteractiveAt || right.updatedAt) -
      parseThreadTime(left.lastInteractiveAt || left.updatedAt)
  );
  return next;
};

const removeThreadFromList = (previous: ThreadSummary[] | undefined, threadId: string) =>
  (previous ?? []).filter((item) => item.id !== threadId);

const upsertOperationListItem = (
  previous: OperationListItemDTO[] | undefined,
  operation: LibraryOperationDTO
) => {
  const next = toOperationListItem(operation);
  const items = previous ?? [];
  const index = items.findIndex((item) => item.operationId === next.operationId);
  if (index === -1) {
    return [next, ...items];
  }
  const updated = [...items];
  updated[index] = { ...updated[index], ...next };
  return updated;
};

const shouldRefreshOperationListItem = (
  current: OperationListItemDTO | undefined,
  operation: LibraryOperationDTO
) => {
  if (!current) {
    return true;
  }
  const next = toOperationListItem(operation);
  return (
    current.libraryId !== next.libraryId ||
    current.name !== next.name ||
    current.kind !== next.kind ||
    current.status !== next.status ||
    current.domain !== next.domain ||
    current.sourceIcon !== next.sourceIcon ||
    current.platform !== next.platform ||
    current.uploader !== next.uploader ||
    current.publishTime !== next.publishTime ||
    current.startedAt !== next.startedAt ||
    current.finishedAt !== next.finishedAt ||
    current.createdAt !== next.createdAt
  );
};

const removeOperationListItem = (
  previous: OperationListItemDTO[] | undefined,
  operationId: string
) => (previous ?? []).filter((item) => item.operationId !== operationId);

const isTerminalOperationStatus = (status: string) => {
  const normalized = status.trim().toLowerCase();
  return normalized === "succeeded" || normalized === "failed" || normalized === "canceled";
};

const asNotice = (value: unknown): Notice | null => {
  if (!value || typeof value !== "object") {
    return null;
  }
  const candidate = value as Notice;
  return typeof candidate.id === "string" && candidate.id.trim() ? candidate : null;
};

const performNoticeAction = (action?: NoticeAction) => {
  if (!action || !action.type) {
    return;
  }
  switch (action.type) {
    case "open_thread":
      if (action.target) {
        void Events.Emit("chat:navigate", action.target);
      }
      return;
    case "open_route":
      void Events.Emit("main:navigate", action.target || "notifications");
      return;
    default:
      if (action.target) {
        void Events.Emit("main:navigate", action.target);
      }
    }
};

const publishNoticeSurfaces = (notice: Notice) => {
  const title = resolveNoticeTitle(notice, t);
  const summary = resolveNoticeSummary(notice, t);
  const body = resolveNoticeBody(notice, t);
  const description = body || summary;
  const intent = noticeSeverityToIntent(notice.severity);
  const descriptionKey = notice.i18n.bodyKey || notice.i18n.summaryKey;
  const actionLabelKey = notice.action?.labelKey?.trim() ?? "";
  const hasAction = Boolean(actionLabelKey && notice.action?.type);
  if (notice.surfaces.includes("toast")) {
    messageBus.publishToast({
      id: `notice-toast-${notice.id}-${notice.occurrenceCount}`,
      intent,
      title,
      description,
      i18n: {
        titleKey: notice.i18n.titleKey,
        descriptionKey,
        params: notice.i18n.params,
      },
      action: hasAction
        ? {
            label: actionLabelKey ? t(actionLabelKey) : "",
            labelKey: actionLabelKey,
            onClick: () => performNoticeAction(notice.action),
          }
        : undefined,
    });
  }
  if (notice.surfaces.includes("popup")) {
    messageBus.publishNotification({
      id: `notice-popup-${notice.id}-${notice.occurrenceCount}`,
      intent,
      title,
      description,
      i18n: {
        titleKey: notice.i18n.titleKey,
        descriptionKey,
        params: notice.i18n.params,
      },
      actions: hasAction
        ? [
            {
              label: actionLabelKey ? t(actionLabelKey) : "",
              labelKey: actionLabelKey,
              onClick: () => performNoticeAction(notice.action),
            },
          ]
        : undefined,
    });
  }
  if (notice.surfaces.includes("os")) {
    void Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.SendSystemNotification", {
      id: `notice-os-${notice.id}-${notice.occurrenceCount}`,
      title,
      body: description,
      data: {
        noticeId: notice.id,
        code: notice.code,
      },
    }).catch((error) => {
      console.warn("[notice] os notification failed", error);
    });
  }
};

export function AppProviders({ children }: PropsWithChildren) {
  const setUpdateInfo = useUpdateStore((state) => state.setInfo);
  const threadResyncTimerRef = useRef<number | null>(null);
  const libraryResyncTimerRef = useRef<number | null>(null);
  const skillsRefreshTimerRef = useRef<number | null>(null);

  useEffect(() => {
    const telemetry = new TelemetryManager();
    void telemetry.start();
    return () => {
      telemetry.stop();
    };
  }, []);

  useEffect(() => {
    startRealtime().catch((error) => {
      console.warn("[realtime] failed to start", error);
    });
    // Seed update state on load to keep update button visible after reloads.
    Call.ByName("dreamcreator/internal/presentation/wails.UpdateHandler.GetState")
      .then((result) => {
        setUpdateInfo(normalizeUpdateInfo(result as Partial<UpdateInfo>));
      })
      .catch((error) => {
        console.warn("[update] get state failed", error);
      });

    const offThreadUpdated = registerTopic(REALTIME_TOPICS.chat.threadUpdated, (event) => {
      const scheduleThreadListResync = () => {
        if (threadResyncTimerRef.current !== null) {
          window.clearTimeout(threadResyncTimerRef.current);
        }
        threadResyncTimerRef.current = window.setTimeout(() => {
          threadResyncTimerRef.current = null;
          queryClient.invalidateQueries({ queryKey: ["threads"] });
        }, 120);
      };

      if ((event?.type ?? "").trim().toLowerCase() === "resync-required") {
        scheduleThreadListResync();
        return;
      }
      // WS replay may emit many historical updates in a burst; collapse to one refresh
      // to avoid repeated layout churn in sidebar components.
      if (event?.replay) {
        scheduleThreadListResync();
        return;
      }
      const payload = event?.payload as ThreadUpdatedEventPayload | null;
      const threadId = (payload?.threadId ?? "").trim();
      const change = (payload?.change ?? "").trim().toLowerCase();
      const reason = (payload?.reason ?? "").trim().toLowerCase();

      if (!threadId) {
        queryClient.invalidateQueries({ queryKey: ["threads"] });
        return;
      }

      const store = useThreadStore.getState();
      const current = store.threads[threadId];
      const normalized = normalizeThreadSummaryPayload(payload?.thread, threadId, current);
      const shouldRemove = change === "purge";
      const shouldUpsert =
        change === "upsert" &&
        normalized !== null &&
        parseThreadTime(normalized.updatedAt) >= parseThreadTime(current?.updatedAt ?? "");

      if (shouldRemove) {
        store.removeThread(threadId);
        queryClient.setQueryData(
          threadsKey(false),
          (previous: ThreadSummary[] | undefined) => removeThreadFromList(previous, threadId)
        );
        queryClient.setQueryData(
          threadsKey(true),
          (previous: ThreadSummary[] | undefined) => removeThreadFromList(previous, threadId)
        );
      } else if (shouldUpsert && normalized) {
        store.upsertThread(normalized);
        queryClient.setQueryData(
          threadsKey(false),
          (previous: ThreadSummary[] | undefined) => upsertThreadInList(previous, normalized, false)
        );
        queryClient.setQueryData(
          threadsKey(true),
          (previous: ThreadSummary[] | undefined) => upsertThreadInList(previous, normalized, true)
        );
      } else {
        queryClient.invalidateQueries({ queryKey: ["threads"] });
      }

      if (reason === "append-message" || reason === "new-thread" || reason === "purge") {
        queryClient.invalidateQueries({ queryKey: ["threads", threadId, "messages"] });
      }
    });
    const offAssistantsUpdated = Events.On("assistants:updated", () => {
      queryClient.invalidateQueries({ queryKey: assistantsKey(false), refetchType: "all" });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true), refetchType: "all" });
    });
    const offProvidersUpdated = Events.On("providers:updated", () => {
      queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" });
      queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" });
    });
    const offExternalToolsUpdated = Events.On("external-tools:updated", () => {
      queryClient.invalidateQueries({ queryKey: EXTERNAL_TOOLS_QUERY_KEY, refetchType: "all" });
      queryClient.invalidateQueries({ queryKey: ["external-tools-updates"], refetchType: "all" });
      queryClient.invalidateQueries({ queryKey: ["external-tools-install-state"], refetchType: "active" });
    });
    const offNoticeCreated = registerTopic(REALTIME_TOPICS.notices.created, (event) => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
      if (event?.replay) {
        return;
      }
      const payload = (event?.payload ?? {}) as { notice?: Notice };
      const notice = asNotice(payload.notice);
      if (notice) {
        publishNoticeSurfaces(notice);
      }
    });
    const offNoticeUpdated = registerTopic(REALTIME_TOPICS.notices.updated, (event) => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
      if (event?.replay) {
        return;
      }
      const payload = (event?.payload ?? {}) as { notice?: Notice };
      const notice = asNotice(payload.notice);
      if (notice) {
        publishNoticeSurfaces(notice);
      }
    });
    const offNoticeUnread = registerTopic(REALTIME_TOPICS.notices.unread, () => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
    });
    const pendingThreadMessageRefresh = new Set<string>();
    let threadMessageRefreshTimer: number | null = null;
    const scheduleThreadMessageRefresh = (threadID: string) => {
      const trimmed = threadID.trim();
      if (!trimmed) {
        return;
      }
      pendingThreadMessageRefresh.add(trimmed);
      if (threadMessageRefreshTimer !== null) {
        return;
      }
      threadMessageRefreshTimer = window.setTimeout(() => {
        threadMessageRefreshTimer = null;
        pendingThreadMessageRefresh.forEach((id) => {
          queryClient.invalidateQueries({
            queryKey: ["threads", id, "messages"],
            refetchType: "active",
          });
        });
        pendingThreadMessageRefresh.clear();
      }, 120);
    };
    const invalidateLibraryQueries = (libraryId?: string) => {
      queryClient.invalidateQueries({ queryKey: LIBRARY_LIST_QUERY_KEY, refetchType: "active" });
      queryClient.invalidateQueries({ queryKey: LIBRARY_OPERATIONS_QUERY_KEY, refetchType: "active" });
      queryClient.invalidateQueries({ queryKey: LIBRARY_HISTORY_QUERY_KEY, refetchType: "active" });
      queryClient.invalidateQueries({ queryKey: LIBRARY_FILE_EVENTS_QUERY_KEY, refetchType: "active" });
      if (libraryId) {
        queryClient.invalidateQueries({ queryKey: [...LIBRARY_DETAIL_QUERY_KEY, libraryId], refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_QUERY_KEY, libraryId], refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, libraryId], refetchType: "active" });
      } else {
        queryClient.invalidateQueries({ queryKey: LIBRARY_DETAIL_QUERY_KEY, refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: LIBRARY_WORKSPACE_QUERY_KEY, refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, refetchType: "active" });
      }
    };
    const scheduleLibraryResync = (libraryId?: string) => {
      if (libraryResyncTimerRef.current !== null) {
        window.clearTimeout(libraryResyncTimerRef.current);
      }
      libraryResyncTimerRef.current = window.setTimeout(() => {
        libraryResyncTimerRef.current = null;
        invalidateLibraryQueries(libraryId);
      }, 120);
    };
    const getStringField = (value: unknown, key: string) => {
      if (!value || typeof value !== "object") {
        return "";
      }
      const record = value as Record<string, unknown>;
      const raw = record[key];
      return typeof raw === "string" ? raw.trim() : "";
    };
    const resolveLibraryID = (payload: unknown) => {
      return getStringField(payload, "libraryId");
    };
    const resolveDeletedID = (payload: unknown) => {
      return getStringField(payload, "id") || getStringField(payload, "operationId") || getStringField(payload, "fileId") || getStringField(payload, "recordId");
    };
    const handleLibraryEvent = (topic: string, type: string, payload: unknown) => {
      const normalizedTopic = topic.trim().toLowerCase();
      const normalizedType = type.trim().toLowerCase();
      if (!normalizedTopic) {
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      if (normalizedType === "resync-required") {
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      const store = useLibraryRealtimeStore.getState();
      if (normalizedTopic === REALTIME_TOPICS.library.operation) {
        if (normalizedType === "delete") {
          const id = resolveDeletedID(payload);
          if (id) {
            store.deleteOperation(id);
            queryClient.setQueriesData({ queryKey: LIBRARY_OPERATIONS_QUERY_KEY }, (current) =>
              Array.isArray(current) ? removeOperationListItem(current as OperationListItemDTO[], id) : current
            );
          }
          scheduleLibraryResync(resolveLibraryID(payload));
          return;
        }
        if (payload && typeof payload === "object") {
          const operation = payload as LibraryOperationDTO;
          store.upsertOperation(operation);
          queryClient.setQueryData([...LIBRARY_OPERATIONS_QUERY_KEY, "detail", operation.id], operation);
          queryClient.setQueriesData({ queryKey: LIBRARY_OPERATIONS_QUERY_KEY }, (current) => {
            if (!Array.isArray(current)) {
              return current;
            }
            const items = current as OperationListItemDTO[];
            const existing = items.find((item) => item.operationId === operation.id);
            return shouldRefreshOperationListItem(existing, operation)
              ? upsertOperationListItem(items, operation)
              : current;
          });
          if (isTerminalOperationStatus(operation.status)) {
            scheduleLibraryResync(operation.libraryId);
          }
          return;
        }
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      if (normalizedTopic === REALTIME_TOPICS.library.file) {
        if (normalizedType === "delete") {
          const id = resolveDeletedID(payload);
          if (id) {
            store.deleteFile(id);
          }
          scheduleLibraryResync(resolveLibraryID(payload));
          return;
        }
        if (payload && typeof payload === "object") {
          const file = payload as LibraryFileDTO;
          store.upsertFile(file);
          scheduleLibraryResync(file.libraryId);
          return;
        }
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      if (normalizedTopic === REALTIME_TOPICS.library.history) {
        if (payload && typeof payload === "object") {
          const history = payload as LibraryHistoryRecordDTO;
          store.upsertHistory(history);
          scheduleLibraryResync(history.libraryId);
          return;
        }
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      if (normalizedTopic === REALTIME_TOPICS.library.workspace) {
        if (payload && typeof payload === "object") {
          const workspace = payload as WorkspaceStateRecordDTO;
          store.replaceWorkspaceHead(workspace);
          queryClient.setQueryData([...LIBRARY_WORKSPACE_QUERY_KEY, workspace.libraryId], workspace);
          scheduleLibraryResync(workspace.libraryId);
          return;
        }
        scheduleLibraryResync(resolveLibraryID(payload));
        return;
      }
      if (normalizedTopic === REALTIME_TOPICS.library.workspaceProject) {
        scheduleLibraryResync(resolveLibraryID(payload));
      }
    };
    const handleGatewayLibraryEvent = (event: GatewayEvent) => {
      const eventName = (event.event ?? "").trim().toLowerCase();
      if (!eventName.startsWith("library.")) {
        return;
      }
      const bridgePayload = (event.payload ?? {}) as
        | { topic?: string; type?: string; payload?: unknown }
        | undefined;
      const topic = (bridgePayload?.topic ?? "").trim().toLowerCase() || eventName.replace(/\.(updated|deleted)$/, "");
      const type = (bridgePayload?.type ?? "").trim().toLowerCase() || (eventName.endsWith(".deleted") ? "delete" : "upsert");
      handleLibraryEvent(topic, type, bridgePayload?.payload);
    };
    const offGatewayRefresh = subscribeGatewayEvents((event) => {
      const eventName = (event.event ?? "").trim().toLowerCase();
      const runID = (event.runId ?? "").trim();
      const sessionID = (event.sessionId ?? "").trim();

      handleGatewayLibraryEvent(event);

      if (eventName === "cron.status") {
        queryClient.invalidateQueries({ queryKey: queryKeys.cronStatus(), refetchType: "active" });
      } else if (eventName === "cron.list") {
        queryClient.invalidateQueries({ queryKey: queryKeys.cronStatus(), refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: queryKeys.cronJobs(), refetchType: "active" });
      } else if (eventName === "cron.runs") {
        queryClient.invalidateQueries({ queryKey: ["cron", "runs"], refetchType: "active" });
        queryClient.invalidateQueries({ queryKey: queryKeys.cronStatus(), refetchType: "active" });
      } else if (eventName === "cron.rundetail") {
        queryClient.invalidateQueries({ queryKey: ["cron", "runs"], refetchType: "active" });
        if (runID) {
          queryClient.invalidateQueries({
            queryKey: queryKeys.cronRunDetail(runID),
            refetchType: "active",
          });
        } else {
          queryClient.invalidateQueries({ queryKey: ["cron", "runDetail"], refetchType: "active" });
        }
      } else if (eventName === "cron.runevents") {
        queryClient.invalidateQueries({ queryKey: ["cron", "runEvents"], refetchType: "active" });
      }

      if (eventName.startsWith("skills.") && eventName.endsWith(".completed")) {
        if (skillsRefreshTimerRef.current !== null) {
          window.clearTimeout(skillsRefreshTimerRef.current);
        }
        skillsRefreshTimerRef.current = window.setTimeout(() => {
          skillsRefreshTimerRef.current = null;
          queryClient.invalidateQueries({ queryKey: SKILLS_CATALOG_QUERY_KEY, refetchType: "active" });
          queryClient.invalidateQueries({ queryKey: SKILLS_SEARCH_QUERY_KEY, refetchType: "active" });
          queryClient.invalidateQueries({ queryKey: SKILLS_STATUS_QUERY_KEY, refetchType: "active" });
          queryClient.invalidateQueries({ queryKey: SKILLS_DETAIL_QUERY_KEY, refetchType: "active" });
        }, 180);
      }

      if (sessionID) {
        queryClient.invalidateQueries({ queryKey: ["threads"], refetchType: "active" });
        scheduleThreadMessageRefresh(sessionID);
      }
    });
    const offLibraryOperation = registerTopic(REALTIME_TOPICS.library.operation, (event) => {
      handleLibraryEvent(REALTIME_TOPICS.library.operation, (event?.type ?? "upsert").trim().toLowerCase(), event?.payload);
    });
    const offLibraryFile = registerTopic(REALTIME_TOPICS.library.file, (event) => {
      handleLibraryEvent(REALTIME_TOPICS.library.file, (event?.type ?? "upsert").trim().toLowerCase(), event?.payload);
    });
    const offLibraryHistory = registerTopic(REALTIME_TOPICS.library.history, (event) => {
      handleLibraryEvent(REALTIME_TOPICS.library.history, (event?.type ?? "upsert").trim().toLowerCase(), event?.payload);
    });
    const offLibraryWorkspace = registerTopic(REALTIME_TOPICS.library.workspace, (event) => {
      handleLibraryEvent(REALTIME_TOPICS.library.workspace, (event?.type ?? "upsert").trim().toLowerCase(), event?.payload);
    });
    const offLibraryWorkspaceProject = registerTopic(REALTIME_TOPICS.library.workspaceProject, (event) => {
      handleLibraryEvent(REALTIME_TOPICS.library.workspaceProject, (event?.type ?? "upsert").trim().toLowerCase(), event?.payload);
    });
    return () => {
      if (threadResyncTimerRef.current !== null) {
        window.clearTimeout(threadResyncTimerRef.current);
      }
      if (libraryResyncTimerRef.current !== null) {
        window.clearTimeout(libraryResyncTimerRef.current);
      }
      if (skillsRefreshTimerRef.current !== null) {
        window.clearTimeout(skillsRefreshTimerRef.current);
      }
      if (threadMessageRefreshTimer !== null) {
        window.clearTimeout(threadMessageRefreshTimer);
      }
      offThreadUpdated();
      offAssistantsUpdated();
      offProvidersUpdated();
      offExternalToolsUpdated();
      offNoticeCreated();
      offNoticeUpdated();
      offNoticeUnread();
      offGatewayRefresh();
      offLibraryFile();
      offLibraryHistory();
      offLibraryWorkspace();
      offLibraryWorkspaceProject();
      offLibraryOperation();
    };
  }, []);
  return (
    <QueryClientProvider client={queryClient}>
      <AssistantUIRuntimeProvider>
        {children}
      </AssistantUIRuntimeProvider>
    </QueryClientProvider>
  );
}
