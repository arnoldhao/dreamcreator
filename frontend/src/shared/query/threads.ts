import { useQuery } from "@tanstack/react-query";
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
