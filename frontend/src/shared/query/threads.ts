import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import type { ThreadSummary } from "@/shared/store/threads";

export interface ThreadMessage {
  id: string;
  role: string;
  content: string;
  createdAt: string;
}

export interface ThreadRunEvent {
  id: number;
  runId: string;
  threadId: string;
  eventType: string;
  payloadJson: string;
  createdAt: string;
}

export interface LLMCallRecord {
  id: string;
  sessionId: string;
  threadId: string;
  runId: string;
  providerId: string;
  modelName: string;
  requestSource: string;
  operation: string;
  status: string;
  finishReason: string;
  errorText: string;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  contextPromptTokens: number;
  contextTotalTokens: number;
  contextWindowTokens: number;
  requestPayloadJson: string;
  responsePayloadJson: string;
  payloadTruncated: boolean;
  startedAt: string;
  finishedAt: string;
  durationMs: number;
}

export type LLMCallRecordListQuery = {
  threadId?: string;
  runId?: string;
  providerId?: string;
  modelName?: string;
  requestSource?: string;
  status?: string;
  startAt?: string;
  endAt?: string;
  limit?: number;
};

export const threadsKey = (includeDeleted: boolean) => ["threads", { includeDeleted }];

export const threadMessagesKey = (threadId: string, limit: number) => ["threads", threadId, "messages", limit];
export const threadRunEventsKey = (threadId: string, afterId: number, limit: number, eventTypePrefix: string) => [
  "threads",
  threadId,
  "run-events",
  afterId,
  limit,
  eventTypePrefix,
];
export const llmCallRecordsKey = (query: LLMCallRecordListQuery) => ["threads", "llm-call-records", query];
export const llmCallRecordKey = (id: string) => ["threads", "llm-call-record", id];

export function useThreads(includeDeleted = false) {
  return useQuery({
    queryKey: threadsKey(includeDeleted),
    queryFn: async (): Promise<ThreadSummary[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.ListThreads",
        includeDeleted
      );
      return (result as ThreadSummary[]) ?? [];
    },
    staleTime: 5_000,
  });
}

export function useThreadMessages(threadId: string | null, limit = 200) {
  return useQuery({
    queryKey: threadId ? threadMessagesKey(threadId, limit) : ["threads", "messages"],
    queryFn: async (): Promise<ThreadMessage[]> => {
      if (!threadId) {
        return [];
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.ListMessages",
        threadId,
        limit
      );
      return (result as ThreadMessage[]) ?? [];
    },
    enabled: Boolean(threadId),
  });
}

export function useThreadRunEvents(
  threadId: string | null,
  options?: { afterId?: number; limit?: number; eventTypePrefix?: string }
) {
  const afterId = options?.afterId ?? 0;
  const limit = options?.limit ?? 500;
  const eventTypePrefix = options?.eventTypePrefix ?? "";
  return useQuery({
    queryKey: threadId
      ? threadRunEventsKey(threadId, afterId, limit, eventTypePrefix)
      : ["threads", "run-events"],
    queryFn: async (): Promise<ThreadRunEvent[]> => {
      if (!threadId) {
        return [];
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.ListThreadRunEvents",
        {
          threadId,
          afterId,
          limit,
          eventTypePrefix,
        }
      );
      return (result as ThreadRunEvent[]) ?? [];
    },
    enabled: Boolean(threadId),
    staleTime: 3_000,
  });
}

export function useLLMCallRecords(query: LLMCallRecordListQuery) {
  return useQuery({
    queryKey: llmCallRecordsKey(query),
    queryFn: async (): Promise<LLMCallRecord[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.ListLLMCallRecords",
        query
      );
      return (result as LLMCallRecord[]) ?? [];
    },
    staleTime: 3_000,
  });
}

export function useLLMCallRecord(id: string | null) {
  return useQuery({
    queryKey: id ? llmCallRecordKey(id) : ["threads", "llm-call-record"],
    queryFn: async (): Promise<LLMCallRecord | null> => {
      if (!id) {
        return null;
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.GetLLMCallRecord",
        id
      );
      return (result as LLMCallRecord) ?? null;
    },
    enabled: Boolean(id),
    staleTime: 3_000,
  });
}

export function usePruneExpiredLLMCallRecords() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (): Promise<number> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.PruneExpiredLLMCallRecords"
      );
      return Number(result ?? 0);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["threads", "llm-call-records"] });
      void queryClient.invalidateQueries({ queryKey: ["threads", "llm-call-record"] });
    },
  });
}

export function useClearLLMCallRecords() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (): Promise<number> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ThreadHandler.ClearLLMCallRecords"
      );
      return Number(result ?? 0);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["threads", "llm-call-records"] });
      void queryClient.invalidateQueries({ queryKey: ["threads", "llm-call-record"] });
    },
  });
}
