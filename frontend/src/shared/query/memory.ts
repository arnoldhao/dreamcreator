import { useMutation, useQuery } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

export interface MemorySummary {
  assistantId?: string;
  summary?: string;
  totalMemories: number;
  assistantCount: number;
  threadCount?: number;
  userCount?: number;
  groupCount?: number;
  channelCount?: number;
  accountCount?: number;
  categoryCounts?: Record<string, number>;
  scopeCounts?: Record<string, number>;
  channelCounts?: Record<string, number>;
  accountCounts?: Record<string, number>;
  principalCounts?: Record<string, number>;
  storage?: MemorySummaryStorage;
  lastUpdatedAt?: string;
}

export interface MemorySummaryStorage {
  totalBytes?: number;
  collectionsBytes?: number;
  chunksBytes?: number;
  assistantSummaryBytes?: number;
  avatarCacheBytes?: number;
}

export interface MemoryStats {
  totalCount: number;
  assistantCount?: number;
  categoryCounts?: Record<string, number>;
  lastUpdatedAt?: string;
  lastMemoryAt?: string;
  hasEmbeddings?: boolean;
  hasFts?: boolean;
  configuredModel?: string;
}

export interface MemoryEntry {
  id: string;
  assistantId?: string;
  threadId?: string;
  content: string;
  category: string;
  confidence: number;
  score?: number;
  sourceJson?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MemoryBrowseOptionsRequest {
  assistantId: string;
  threadId?: string;
  scope?: string;
  channel?: string;
}

export interface MemoryBrowseOptions {
  scopes: string[];
  channels: string[];
  accountIds: string[];
  categories: string[];
}

export interface MemoryListRequest {
  assistantId: string;
  threadId?: string;
  channel?: string;
  accountId?: string;
  userId?: string;
  groupId?: string;
  category?: string;
  scope?: string;
  limit?: number;
  offset?: number;
}

export interface MemoryRecallRequest {
  assistantId: string;
  threadId?: string;
  channel?: string;
  accountId?: string;
  userId?: string;
  groupId?: string;
  query: string;
  topK?: number;
  category?: string;
  scope?: string;
}

export interface MemoryRetrieval {
  entries: MemoryEntry[];
}

export interface MemoryPrincipalListRequest {
  assistantId: string;
  threadId?: string;
  scope?: string;
  channel?: string;
  accountId?: string;
  category?: string;
  principalType: "user" | "group";
  query?: string;
  limit?: number;
}

export interface MemoryPrincipalItem {
  principalId: string;
  channel?: string;
  name?: string;
  username?: string;
  avatarUrl?: string;
  avatarKey?: string;
  count: number;
  lastUpdatedAt?: string;
}

export interface MemoryPrincipalRefreshRequest {
  assistantId: string;
  threadId?: string;
  scope?: string;
  channel?: string;
  accountId?: string;
  principalType: "user" | "group";
  principalId: string;
}

export interface MemoryPrincipalRefreshResult {
  principalId: string;
  name?: string;
  username?: string;
  avatarUrl?: string;
  avatarKey?: string;
  updatedRows: number;
  lastUpdatedAt?: string;
}

export const memorySummaryKey = (assistantId?: string) => ["memory", "summary", assistantId ?? "global"];
export const memoryStatsKey = (request: {
  assistantId: string;
  threadId?: string;
  scope?: string;
  channel?: string;
  accountId?: string;
  userId?: string;
  groupId?: string;
}) => ["memory", "stats", request];
export const memoryBrowseOptionsKey = (request: MemoryBrowseOptionsRequest) => ["memory", "browse-options", request];
export const memoryEntriesKey = (request: MemoryListRequest) => ["memory", "entries", request];
export const memoryRecallKey = (request: MemoryRecallRequest) => ["memory", "recall", request];
export const memoryPrincipalListKey = (request: MemoryPrincipalListRequest) => ["memory", "principals", request];

export function useMemorySummary(assistantId?: string) {
  return useQuery({
    queryKey: memorySummaryKey(assistantId),
    queryFn: async (): Promise<MemorySummary> => {
      const normalizedAssistantId = (assistantId ?? "").trim();
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.MemoryHandler.GetSummary",
        { assistantId: normalizedAssistantId }
      );
      const summary = (result as MemorySummary | null) ?? null;
      return (
        summary ?? {
          assistantId: normalizedAssistantId,
          summary: "",
          totalMemories: 0,
          assistantCount: 0,
          threadCount: 0,
          userCount: 0,
          groupCount: 0,
          channelCount: 0,
          accountCount: 0,
          categoryCounts: {},
          scopeCounts: {},
          channelCounts: {},
          accountCounts: {},
          principalCounts: {},
          storage: {
            totalBytes: 0,
            collectionsBytes: 0,
            chunksBytes: 0,
            assistantSummaryBytes: 0,
            avatarCacheBytes: 0,
          },
          lastUpdatedAt: "",
        }
      );
    },
    staleTime: 5_000,
  });
}

export function useMemoryEntries(request: MemoryListRequest, enabled = true) {
  const assistantId = request.assistantId?.trim() ?? "";
  return useQuery({
    queryKey: memoryEntriesKey(request),
    queryFn: async (): Promise<MemoryEntry[]> => {
      if (!assistantId) {
        return [];
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.List", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
        accountId: request.accountId?.trim() ?? "",
        userId: request.userId?.trim() ?? "",
        groupId: request.groupId?.trim() ?? "",
        category: request.category?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
        limit: request.limit ?? 50,
        offset: request.offset ?? 0,
      });
      return (result as MemoryEntry[]) ?? [];
    },
    enabled: enabled && assistantId.length > 0,
    staleTime: 3_000,
  });
}

export function useMemoryStats(
  request: {
    assistantId: string;
    threadId?: string;
    scope?: string;
    channel?: string;
    accountId?: string;
    userId?: string;
    groupId?: string;
  },
  enabled = true
) {
  const assistantId = request.assistantId?.trim() ?? "";
  return useQuery({
    queryKey: memoryStatsKey(request),
    queryFn: async (): Promise<MemoryStats> => {
      if (!assistantId) {
        return { totalCount: 0, assistantCount: 0, categoryCounts: {}, lastUpdatedAt: "" };
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.Stats", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
        accountId: request.accountId?.trim() ?? "",
        userId: request.userId?.trim() ?? "",
        groupId: request.groupId?.trim() ?? "",
      });
      return (
        (result as MemoryStats | null) ?? {
          totalCount: 0,
          assistantCount: 0,
          categoryCounts: {},
          lastUpdatedAt: "",
        }
      );
    },
    enabled: enabled && assistantId.length > 0,
    staleTime: 3_000,
  });
}

export function useMemoryBrowseOptions(request: MemoryBrowseOptionsRequest, enabled = true) {
  const assistantId = request.assistantId?.trim() ?? "";
  return useQuery({
    queryKey: memoryBrowseOptionsKey(request),
    queryFn: async (): Promise<MemoryBrowseOptions> => {
      if (!assistantId) {
        return { scopes: [], channels: [], accountIds: [], categories: [] };
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.BrowseOptions", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
      });
      const options = (result as Partial<MemoryBrowseOptions> | null) ?? {};
      const asStringArray = (value: unknown): string[] =>
        Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
      return {
        scopes: asStringArray(options.scopes),
        channels: asStringArray(options.channels),
        accountIds: asStringArray(options.accountIds),
        categories: asStringArray(options.categories),
      };
    },
    enabled: enabled && assistantId.length > 0,
    staleTime: 3_000,
  });
}

export function useMemoryRecall(request: MemoryRecallRequest, enabled = true) {
  const assistantId = request.assistantId?.trim() ?? "";
  const query = request.query?.trim() ?? "";
  return useQuery({
    queryKey: memoryRecallKey(request),
    queryFn: async (): Promise<MemoryRetrieval> => {
      if (!assistantId || !query) {
        return { entries: [] };
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.Recall", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
        accountId: request.accountId?.trim() ?? "",
        userId: request.userId?.trim() ?? "",
        groupId: request.groupId?.trim() ?? "",
        query,
        topK: request.topK ?? 20,
        category: request.category?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
      });
      return (result as MemoryRetrieval) ?? { entries: [] };
    },
    enabled: enabled && assistantId.length > 0 && query.length > 0,
    staleTime: 3_000,
  });
}

export function useMemoryPrincipalList(request: MemoryPrincipalListRequest, enabled = true) {
  const assistantId = request.assistantId?.trim() ?? "";
  const principalType = request.principalType;
  return useQuery({
    queryKey: memoryPrincipalListKey(request),
    queryFn: async (): Promise<MemoryPrincipalItem[]> => {
      if (!assistantId) {
        return [];
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.ListPrincipals", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
        accountId: request.accountId?.trim() ?? "",
        category: request.category?.trim() ?? "",
        principalType,
        query: request.query?.trim() ?? "",
        limit: request.limit ?? 120,
      });
      return (result as MemoryPrincipalItem[]) ?? [];
    },
    enabled: enabled && assistantId.length > 0 && (principalType === "user" || principalType === "group"),
    staleTime: 3_000,
  });
}

export function useMemoryRefreshPrincipal() {
  return useMutation({
    mutationFn: async (request: MemoryPrincipalRefreshRequest): Promise<MemoryPrincipalRefreshResult> => {
      const assistantId = request.assistantId?.trim() ?? "";
      const principalId = request.principalId?.trim() ?? "";
      if (!assistantId) {
        throw new Error("assistantId is required");
      }
      if (!principalId) {
        throw new Error("principalId is required");
      }
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.MemoryHandler.RefreshPrincipal", {
        assistantId,
        threadId: request.threadId?.trim() ?? "",
        scope: request.scope?.trim() ?? "",
        channel: request.channel?.trim() ?? "",
        accountId: request.accountId?.trim() ?? "",
        principalType: request.principalType,
        principalId,
      });
      return (
        (result as MemoryPrincipalRefreshResult | null) ?? {
          principalId,
          updatedRows: 0,
        }
      );
    },
  });
}
