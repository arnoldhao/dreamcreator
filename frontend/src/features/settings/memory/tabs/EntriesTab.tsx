import { Events } from "@wailsio/runtime";
import { Fragment, type FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { Globe, RefreshCw, Search, User, Users, X } from "lucide-react";

import { useHttpBaseUrl } from "@/features/settings/gateway/ui/useHttpBaseUrl";
import { messageBus } from "@/shared/message";
import { useAssistants } from "@/shared/query/assistant";
import { useChannelsDebug } from "@/shared/query/channels";
import {
  type MemoryEntry,
  type MemoryPrincipalItem,
  useMemoryBrowseOptions,
  useMemoryEntries,
  useMemoryPrincipalList,
  useMemoryRefreshPrincipal,
  useMemoryRecall,
  useMemoryStats,
} from "@/shared/query/memory";
import { useThreads } from "@/shared/query/threads";
import { useChatRuntimeStore } from "@/shared/store/chat-runtime";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";

const browserViewValues = ["global", "user", "group"] as const;
type BrowserViewValue = (typeof browserViewValues)[number];

const allScopeValue = "all";
const allCategoryValue = "all";
const defaultLimitValue = "20";
const memoryCategoryValues = ["preference", "fact", "decision", "entity", "reflection", "other"] as const;

interface EntriesTabProps {
  t: (key: string) => string;
  language: string;
}

function formatTime(value?: string): string {
  const normalized = value?.trim() ?? "";
  if (!normalized) {
    return "-";
  }
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) {
    return normalized;
  }
  return parsed.toLocaleString();
}

function formatRelativeTime(value: string | undefined, language: string): string {
  const normalized = value?.trim() ?? "";
  if (!normalized) {
    return "-";
  }
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) {
    return "-";
  }
  const diffSeconds = Math.round((parsed.getTime() - Date.now()) / 1000);
  const absSeconds = Math.abs(diffSeconds);
  const rtf = new Intl.RelativeTimeFormat(language, { numeric: "auto" });
  if (absSeconds < 60) {
    return rtf.format(diffSeconds, "second");
  }
  const diffMinutes = Math.round(diffSeconds / 60);
  if (Math.abs(diffMinutes) < 60) {
    return rtf.format(diffMinutes, "minute");
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (Math.abs(diffHours) < 24) {
    return rtf.format(diffHours, "hour");
  }
  const diffDays = Math.round(diffHours / 24);
  if (Math.abs(diffDays) < 30) {
    return rtf.format(diffDays, "day");
  }
  const diffMonths = Math.round(diffDays / 30);
  if (Math.abs(diffMonths) < 12) {
    return rtf.format(diffMonths, "month");
  }
  const diffYears = Math.round(diffMonths / 12);
  return rtf.format(diffYears, "year");
}

function buildContentPreview(content: string, maxLength = 120): string {
  const normalized = content.replace(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) {
    return normalized;
  }
  return `${normalized.slice(0, maxLength)}...`;
}

function formatCategoryLabel(t: (key: string) => string, category?: string): string {
  const normalized = category?.trim() ?? "";
  if (!normalized) {
    return "-";
  }
  return t(`settings.memory.entries.categoryOption.${normalized}`);
}

function formatThreadLabel(threadID: string | undefined, threadTitleByID: Map<string, string>): string {
  const normalizedThreadID = threadID?.trim() ?? "";
  if (!normalizedThreadID) {
    return "-";
  }
  const title = threadTitleByID.get(normalizedThreadID)?.trim() ?? "";
  return title || normalizedThreadID;
}

function formatSourceJSON(value?: string): string {
  const normalized = value?.trim() ?? "";
  if (!normalized) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(normalized), null, 2);
  } catch {
    return normalized;
  }
}

function normalizeLimit(value: string): number {
  const parsed = Number(value);
  if (Number.isNaN(parsed)) {
    return 20;
  }
  if (parsed < 1) {
    return 1;
  }
  if (parsed > 100) {
    return 100;
  }
  return Math.round(parsed);
}

function resolvePrincipalDisplayName(item?: MemoryPrincipalItem | null): string {
  if (!item) {
    return "";
  }
  const fromName = item.name?.trim() ?? "";
  if (fromName) {
    return fromName;
  }
  const fromUsername = item.username?.trim() ?? "";
  if (fromUsername) {
    return fromUsername.startsWith("@") ? fromUsername : `@${fromUsername}`;
  }
  return item.principalId?.trim() ?? "";
}

function resolvePrincipalIdentityLine(item?: MemoryPrincipalItem | null): string {
  if (!item) {
    return "";
  }
  const channel = item.channel?.trim() ?? "";
  const principalID = item.principalId?.trim() ?? "";
  if (channel && principalID) {
    return `${channel} · ${principalID}`;
  }
  return principalID || channel;
}

function buildAvatarFallback(name: string): string {
  const normalized = name.trim();
  if (!normalized) {
    return "?";
  }
  const segments = normalized.split(/\s+/).filter(Boolean);
  if (segments.length >= 2) {
    return `${segments[0]?.[0] ?? ""}${segments[1]?.[0] ?? ""}`.toUpperCase();
  }
  return normalized.slice(0, 2).toUpperCase();
}

function normalizeAccountDisplayName(value?: string): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed) {
    return "";
  }
  if (trimmed.startsWith("@")) {
    return trimmed.replace(/\s*\(default\)\s*$/i, "").trim();
  }
  return `@${trimmed}`.replace(/\s*\(default\)\s*$/i, "").trim();
}

function sanitizeAccountLabel(label: string, accountID: string): string {
  const trimmedLabel = label.trim();
  if (!trimmedLabel) {
    return "";
  }
  const trimmedAccountID = accountID.trim();
  if (!trimmedAccountID) {
    return trimmedLabel.replace(/\s*\(default\)\s*$/i, "").trim();
  }
  const escapedAccountID = trimmedAccountID.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return trimmedLabel
    .replace(new RegExp(`\\s*\\(${escapedAccountID}\\)\\s*$`, "i"), "")
    .replace(/\s*\(default\)\s*$/i, "")
    .trim();
}

function buildMemoryAvatarApiURL(httpBaseUrl: string, avatarKey: string): string {
  const trimmedBase = httpBaseUrl.trim().replace(/\/+$/, "");
  const trimmedKey = avatarKey.trim();
  if (!trimmedBase || !trimmedKey) {
    return "";
  }
  return `${trimmedBase}/api/memory/avatar?key=${encodeURIComponent(trimmedKey)}`;
}

function buildLibraryAssetApiURL(httpBaseUrl: string, path: string): string {
  const trimmedBase = httpBaseUrl.trim().replace(/\/+$/, "");
  const trimmedPath = path.trim();
  if (!trimmedBase || !trimmedPath) {
    return "";
  }
  return `${trimmedBase}/api/library/asset?path=${encodeURIComponent(trimmedPath)}`;
}

function resolvePrincipalAvatarSource(item: MemoryPrincipalItem | null | undefined, httpBaseUrl: string): { primary: string; fallback: string } {
  const avatarKey = item?.avatarKey?.trim() ?? "";
  const avatarUrl = item?.avatarUrl?.trim() ?? "";
  const keyURL = avatarKey ? buildMemoryAvatarApiURL(httpBaseUrl, avatarKey) : "";
  const directURL = /^https?:\/\//i.test(avatarUrl) ? avatarUrl : buildLibraryAssetApiURL(httpBaseUrl, avatarUrl);
  if (keyURL) {
    return {
      primary: keyURL,
      fallback: directURL && directURL !== keyURL ? directURL : "",
    };
  }
  return {
    primary: directURL,
    fallback: "",
  };
}

function renderPrincipalAvatar(item: MemoryPrincipalItem | null | undefined, httpBaseUrl: string, sizeClass = "h-8 w-8") {
  const avatarSource = resolvePrincipalAvatarSource(item, httpBaseUrl);
  const avatarUrl = avatarSource.primary;
  const displayName = resolvePrincipalDisplayName(item);
  const fallback = buildAvatarFallback(displayName || item?.principalId || "");
  return (
    <span
      className={`relative inline-flex ${sizeClass} shrink-0 items-center justify-center overflow-hidden rounded-full border border-border/70 bg-muted text-[11px] font-medium text-muted-foreground`}
      aria-hidden
    >
      {avatarUrl ? (
        <img
          src={avatarUrl}
          data-fallback-src={avatarSource.fallback}
          alt=""
          className="h-full w-full object-cover"
          loading="lazy"
          referrerPolicy="no-referrer"
          onError={(event) => {
            const fallbackSource = event.currentTarget.dataset.fallbackSrc?.trim() ?? "";
            if (fallbackSource && event.currentTarget.src !== fallbackSource) {
              event.currentTarget.dataset.fallbackSrc = "";
              event.currentTarget.style.removeProperty("display");
              event.currentTarget.src = fallbackSource;
              return;
            }
            event.currentTarget.style.display = "none";
            const fallbackElement = event.currentTarget.nextElementSibling as HTMLElement | null;
            if (fallbackElement) {
              fallbackElement.style.display = "flex";
            }
          }}
        />
      ) : null}
      <span className={avatarUrl ? "hidden h-full w-full items-center justify-center" : "flex h-full w-full items-center justify-center"}>
        {fallback}
      </span>
    </span>
  );
}

function renderMemoryRows(params: {
  t: (key: string) => string;
  entries: MemoryEntry[];
  isLoading: boolean;
  errorText: string;
  hasAssistant: boolean;
  expandedEntryId: string | null;
  threadTitleByID: Map<string, string>;
  onToggleExpand: (id: string) => void;
}) {
  const { t, entries, isLoading, errorText, hasAssistant, expandedEntryId, threadTitleByID, onToggleExpand } = params;
  if (!hasAssistant) {
    return (
      <TableRow>
        <TableCell colSpan={4} className="px-2 py-2 text-muted-foreground">
          {t("settings.memory.entries.assistantRequired")}
        </TableCell>
      </TableRow>
    );
  }
  if (isLoading) {
    return (
      <TableRow>
        <TableCell colSpan={4} className="px-2 py-2 text-muted-foreground">
          {t("settings.memory.entries.loading")}
        </TableCell>
      </TableRow>
    );
  }
  if (errorText) {
    return (
      <TableRow>
        <TableCell colSpan={4} className="px-2 py-2 text-destructive">
          {t("settings.memory.entries.error")}: {errorText}
        </TableCell>
      </TableRow>
    );
  }
  if (entries.length === 0) {
    return (
      <TableRow>
        <TableCell colSpan={4} className="px-2 py-2 text-muted-foreground">
          {t("settings.memory.entries.empty")}
        </TableCell>
      </TableRow>
    );
  }
  return entries.map((entry, index) => {
    const rowKey = entry.id?.trim() || `${entry.updatedAt}-${index}`;
    const isExpanded = expandedEntryId === rowKey;
    const categoryLabel = formatCategoryLabel(t, entry.category);
    const threadLabel = formatThreadLabel(entry.threadId, threadTitleByID);
    const scoreValue =
      typeof entry.score === "number"
        ? entry.score.toFixed(3)
        : typeof entry.confidence === "number"
          ? entry.confidence.toFixed(2)
          : "-";
    return (
      <Fragment key={rowKey}>
        <TableRow className="cursor-pointer select-none" onClick={() => onToggleExpand(rowKey)}>
          <TableCell
            className="truncate whitespace-nowrap font-mono text-[11px] text-muted-foreground"
            title={formatTime(entry.updatedAt || entry.createdAt)}
          >
            {formatTime(entry.updatedAt || entry.createdAt)}
          </TableCell>
          <TableCell className="truncate whitespace-nowrap text-center font-mono text-[11px]" title={categoryLabel}>
            {categoryLabel}
          </TableCell>
          <TableCell className="truncate whitespace-nowrap text-center font-mono text-[11px]">{scoreValue}</TableCell>
          <TableCell className="truncate font-mono text-[11px]" title={entry.content || "-"}>
            {buildContentPreview(entry.content || "") || "-"}
          </TableCell>
        </TableRow>
        {isExpanded ? (
          <TableRow>
            <TableCell colSpan={4} className="border-0 px-2 pb-2">
              <div className="space-y-2 rounded-md border bg-muted/20 p-2 select-text">
                <div className="grid grid-cols-1 gap-1 text-[11px] text-muted-foreground sm:grid-cols-2">
                  <div className="truncate">
                    {t("settings.memory.entries.detail.id")}:
                    <span className="font-mono text-foreground"> {entry.id}</span>
                  </div>
                  <div className="truncate">
                    {t("settings.memory.entries.detail.thread")}:
                    <span className="font-mono text-foreground" title={entry.threadId || "-"}>
                      {" "}
                      {threadLabel}
                    </span>
                  </div>
                  <div className="truncate">
                    {t("settings.memory.entries.confidenceLabel")}
                    : <span className="font-mono text-foreground">{scoreValue}</span>
                  </div>
                  <div className="truncate">
                    {t("settings.memory.entries.category")}:
                    <span className="font-mono text-foreground"> {categoryLabel}</span>
                  </div>
                </div>
                <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-background/80 p-2 text-[11px] text-foreground">
                  {entry.content || "-"}
                </pre>
                {entry.sourceJson?.trim() ? (
                  <div className="space-y-1">
                    <div className="text-[11px] text-muted-foreground">{t("settings.memory.entries.detail.source")}</div>
                    <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-background/80 p-2 text-[11px] text-muted-foreground">
                      {formatSourceJSON(entry.sourceJson)}
                    </pre>
                  </div>
                ) : null}
              </div>
            </TableCell>
          </TableRow>
        ) : null}
      </Fragment>
    );
  });
}

function renderPrincipalSidebar(params: {
  t: (key: string) => string;
  browserView: BrowserViewValue;
  principalItems: MemoryPrincipalItem[];
  selectedPrincipalID: string;
  setSelectedPrincipalID: (id: string) => void;
  httpBaseUrl: string;
  isLoading: boolean;
  errorText: string;
}) {
  const { t, browserView, principalItems, selectedPrincipalID, setSelectedPrincipalID, httpBaseUrl, isLoading, errorText } = params;
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="px-1 pb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {browserView === "group"
          ? t("settings.memory.entries.sidebar.groups")
          : t("settings.memory.entries.sidebar.users")}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1">
        {isLoading ? (
          <div className="px-2 py-2 text-xs text-muted-foreground">{t("settings.memory.entries.loading")}</div>
        ) : errorText ? (
          <div className="px-2 py-2 text-xs text-destructive">
            {t("settings.memory.entries.error")}: {errorText}
          </div>
        ) : principalItems.length === 0 ? (
          <div className="rounded-md border border-dashed border-border/70 bg-muted/20 px-3 py-3 text-xs text-muted-foreground">
            {t("settings.memory.entries.sidebar.empty")}
          </div>
        ) : (
          <div className="space-y-1">
            {principalItems.map((item) => {
              const selected = item.principalId === selectedPrincipalID;
              const displayName = resolvePrincipalDisplayName(item) || item.principalId;
              const identityLine = resolvePrincipalIdentityLine(item);
              return (
                <button
                  key={item.principalId}
                  type="button"
                  className={`flex w-full items-center justify-between rounded-md border px-2 py-1.5 text-left ${
                    selected ? "border-primary/40 bg-primary/10" : "border-border/70 bg-background hover:bg-muted/30"
                  }`}
                  onClick={() => setSelectedPrincipalID(item.principalId)}
                >
                  <span className="min-w-0 flex items-center gap-2">
                    {renderPrincipalAvatar(item, httpBaseUrl)}
                    <span className="min-w-0">
                      <span className="block truncate text-xs font-medium text-foreground">{displayName}</span>
                      <span className="block truncate font-mono text-[10px] text-muted-foreground">{identityLine}</span>
                    </span>
                  </span>
                  <span className="ml-2 shrink-0 text-[11px] text-muted-foreground">{item.count}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

export function EntriesTab({ t, language }: EntriesTabProps) {
  const assistantsQuery = useAssistants(true);
  const assistants = assistantsQuery.data ?? [];
  const threadsQuery = useThreads(false);
  const threads = threadsQuery.data ?? [];
  const channelsDebugQuery = useChannelsDebug();
  const currentAssistantId = useChatRuntimeStore((state) => state.assistantId);
  const httpBaseUrl = useHttpBaseUrl();

  const [selectedAssistantId, setSelectedAssistantId] = useState<string>("");
  const [browserView, setBrowserView] = useState<BrowserViewValue>("global");
  const [scope, setScope] = useState<string>(allScopeValue);
  const [channel, setChannel] = useState<string>("");
  const [accountId, setAccountId] = useState<string>("");
  const [category, setCategory] = useState<string>(allCategoryValue);
  const [limitInput, setLimitInput] = useState<string>(defaultLimitValue);
  const [queryInput, setQueryInput] = useState<string>("");
  const [query, setQuery] = useState<string>("");
  const [expandedEntryId, setExpandedEntryId] = useState<string | null>(null);
  const [selectedUserID, setSelectedUserID] = useState<string>("");
  const [selectedGroupID, setSelectedGroupID] = useState<string>("");

  const normalizedCurrentAssistantId = currentAssistantId.trim();
  const hasAssistant = selectedAssistantId.trim().length > 0;
  const limit = useMemo(() => normalizeLimit(limitInput), [limitInput]);
  const normalizedQuery = query.trim();
  const normalizedScope = scope.trim() || allScopeValue;
  const normalizedChannel = channel.trim();
  const normalizedAccountId = accountId.trim();
  const normalizedCategory = category === allCategoryValue ? "" : category.trim();
  const isGlobalView = browserView === "global";
  const principalType = browserView === "group" ? "group" : "user";
  const threadTitleByID = useMemo(() => {
    const map = new Map<string, string>();
    for (const item of threads) {
      const id = item.id?.trim() ?? "";
      if (!id) {
        continue;
      }
      if (selectedAssistantId && item.assistantId && item.assistantId !== selectedAssistantId) {
        continue;
      }
      const title = item.title?.trim() ?? "";
      if (!title) {
        continue;
      }
      map.set(id, title);
    }
    return map;
  }, [threads, selectedAssistantId]);

  useEffect(() => {
    if (selectedAssistantId && assistants.some((item) => item.id === selectedAssistantId)) {
      return;
    }
    const next =
      assistants.find((item) => item.id === normalizedCurrentAssistantId) ??
      assistants.find((item) => item.isDefault) ??
      assistants[0];
    setSelectedAssistantId(next?.id ?? "");
  }, [assistants, normalizedCurrentAssistantId, selectedAssistantId]);

  const optionsQuery = useMemoryBrowseOptions(
    {
      assistantId: selectedAssistantId,
      scope: normalizedScope,
      channel: normalizedChannel,
    },
    hasAssistant
  );
  const options = optionsQuery.data;
  const optionScopes = useMemo(
    () => (Array.isArray(options?.scopes) ? options.scopes.filter((item): item is string => typeof item === "string" && item.trim().length > 0) : []),
    [options?.scopes]
  );
  const optionChannels = useMemo(
    () => (Array.isArray(options?.channels) ? options.channels.filter((item): item is string => typeof item === "string" && item.trim().length > 0) : []),
    [options?.channels]
  );
  const optionAccountIds = useMemo(
    () => (Array.isArray(options?.accountIds) ? options.accountIds.filter((item): item is string => typeof item === "string" && item.trim().length > 0) : []),
    [options?.accountIds]
  );
  const optionCategories = useMemo(
    () => (Array.isArray(options?.categories) ? options.categories.filter((item): item is string => typeof item === "string" && item.trim().length > 0) : []),
    [options?.categories]
  );
  const accountNameByChannelAndID = useMemo(() => {
    const map = new Map<string, string>();
    for (const snapshot of channelsDebugQuery.data ?? []) {
      const channelID = snapshot.channelId?.trim() ?? "";
      if (!channelID) {
        continue;
      }
      for (const account of snapshot.accounts ?? []) {
        const accountID = account.accountId?.trim() ?? "";
        if (!accountID) {
          continue;
        }
        const displayName = normalizeAccountDisplayName(account.botUsername) || accountID;
        map.set(`${channelID}\u0000${accountID}`, displayName);
      }
    }
    return map;
  }, [channelsDebugQuery.data]);
  const accountOptionItems = useMemo(() => {
    return optionAccountIds.map((item) => {
      const accountID = item.trim();
      if (!accountID) {
        return { value: item, label: item };
      }
      const scopedChannel = normalizedChannel;
      const resolveName = (channelID: string) => accountNameByChannelAndID.get(`${channelID}\u0000${accountID}`) ?? "";
      if (scopedChannel) {
        const accountName = sanitizeAccountLabel(resolveName(scopedChannel), accountID);
        if (accountName && accountName !== accountID) {
          return { value: accountID, label: accountName };
        }
        return { value: accountID, label: accountID };
      }
      for (const [key, candidateName] of accountNameByChannelAndID) {
        const [channelID, candidateAccountID] = key.split("\u0000");
        if (candidateAccountID !== accountID) {
          continue;
        }
        const normalizedCandidateName = sanitizeAccountLabel(candidateName, accountID);
        if (normalizedCandidateName && normalizedCandidateName !== accountID) {
          return { value: accountID, label: normalizedCandidateName };
        }
        if (channelID) {
          return { value: accountID, label: `${channelID}/${accountID}` };
        }
      }
      return { value: accountID, label: accountID };
    });
  }, [accountNameByChannelAndID, normalizedChannel, optionAccountIds]);
  const categoryOptions = useMemo(() => {
    const base: string[] = [...memoryCategoryValues];
    const extras = optionCategories.filter((item) => !base.includes(item)).sort();
    return [...base, ...extras];
  }, [optionCategories]);

  useEffect(() => {
    if (scope === allScopeValue) {
      return;
    }
    if (!optionScopes.includes(scope)) {
      setScope(allScopeValue);
    }
  }, [optionScopes, scope]);

  useEffect(() => {
    if (!channel) {
      return;
    }
    if (!optionChannels.includes(channel)) {
      setChannel("");
    }
  }, [optionChannels, channel]);

  useEffect(() => {
    if (!accountId) {
      return;
    }
    if (!optionAccountIds.includes(accountId)) {
      setAccountId("");
    }
  }, [optionAccountIds, accountId]);

  useEffect(() => {
    if (category === allCategoryValue) {
      return;
    }
    if (!categoryOptions.includes(category)) {
      setCategory(allCategoryValue);
    }
  }, [categoryOptions, category]);

  const principalListQuery = useMemoryPrincipalList(
    {
      assistantId: selectedAssistantId,
      scope: normalizedScope,
      channel: normalizedChannel,
      accountId: normalizedAccountId,
      category: normalizedCategory,
      principalType,
      limit: 200,
    },
    hasAssistant && !isGlobalView
  );
  const principalItems = principalListQuery.data ?? [];
  const selectedPrincipalID = browserView === "group" ? selectedGroupID : selectedUserID;
  const selectedPrincipalItem = useMemo(
    () => principalItems.find((item) => item.principalId === selectedPrincipalID) ?? null,
    [principalItems, selectedPrincipalID]
  );
  const selectedPrincipalName = resolvePrincipalDisplayName(selectedPrincipalItem) || selectedPrincipalID;
  const selectedPrincipalIdentityLine =
    resolvePrincipalIdentityLine(selectedPrincipalItem) ||
    (normalizedChannel ? `${normalizedChannel} · ${selectedPrincipalID}` : selectedPrincipalID);

  useEffect(() => {
    if (browserView !== "user") {
      return;
    }
    if (selectedUserID && principalItems.some((item) => item.principalId === selectedUserID)) {
      return;
    }
    setSelectedUserID(principalItems[0]?.principalId ?? "");
  }, [browserView, principalItems, selectedUserID]);

  useEffect(() => {
    if (browserView !== "group") {
      return;
    }
    if (selectedGroupID && principalItems.some((item) => item.principalId === selectedGroupID)) {
      return;
    }
    setSelectedGroupID(principalItems[0]?.principalId ?? "");
  }, [browserView, principalItems, selectedGroupID]);

  const filteredUserID = browserView === "user" ? selectedPrincipalID : "";
  const filteredGroupID = browserView === "group" ? selectedPrincipalID : "";

  const globalListQuery = useMemoryEntries(
    {
      assistantId: selectedAssistantId,
      category: normalizedCategory,
      scope: normalizedScope,
      channel: normalizedChannel,
      accountId: normalizedAccountId,
      limit,
      offset: 0,
    },
    hasAssistant && isGlobalView && normalizedQuery.length === 0
  );

  const globalRecallQuery = useMemoryRecall(
    {
      assistantId: selectedAssistantId,
      query: normalizedQuery,
      topK: limit,
      category: normalizedCategory,
      scope: normalizedScope,
      channel: normalizedChannel,
      accountId: normalizedAccountId,
    },
    hasAssistant && isGlobalView && normalizedQuery.length > 0
  );

  const isRecallMode = normalizedQuery.length > 0;
  const activeGlobalQuery = isRecallMode ? globalRecallQuery : globalListQuery;
  const globalEntries: MemoryEntry[] = isRecallMode ? (globalRecallQuery.data?.entries ?? []) : (globalListQuery.data ?? []);
  const globalLoading = activeGlobalQuery.isLoading || activeGlobalQuery.isFetching;
  const globalErrorText = activeGlobalQuery.error ? String(activeGlobalQuery.error) : "";

  const principalStatsQuery = useMemoryStats(
    {
      assistantId: selectedAssistantId,
      scope: normalizedScope,
      channel: normalizedChannel,
      accountId: normalizedAccountId,
      userId: filteredUserID,
      groupId: filteredGroupID,
    },
    hasAssistant && !isGlobalView && selectedPrincipalID.length > 0
  );

  const principalEntriesQuery = useMemoryEntries(
    {
      assistantId: selectedAssistantId,
      scope: normalizedScope,
      channel: normalizedChannel,
      accountId: normalizedAccountId,
      category: normalizedCategory,
      userId: filteredUserID,
      groupId: filteredGroupID,
      limit,
      offset: 0,
    },
    hasAssistant && !isGlobalView && selectedPrincipalID.length > 0
  );

  const principalStats = principalStatsQuery.data;
  const principalEntries = principalEntriesQuery.data ?? [];
  const principalSummaryLoading = principalStatsQuery.isLoading || principalStatsQuery.isFetching || principalEntriesQuery.isLoading;
  const principalSummaryError = principalStatsQuery.error ? String(principalStatsQuery.error) : principalEntriesQuery.error ? String(principalEntriesQuery.error) : "";
  const principalCategories = Object.entries(principalStats?.categoryCounts ?? {}).sort((a, b) => b[1] - a[1]);
  const refreshPrincipalMutation = useMemoryRefreshPrincipal();

  const headerCount = isGlobalView ? globalEntries.length : principalStats?.totalCount ?? 0;

  const refresh = useCallback(() => {
    if (isGlobalView) {
      void activeGlobalQuery.refetch();
      void optionsQuery.refetch();
      return;
    }
    void principalListQuery.refetch();
    void principalStatsQuery.refetch();
    void principalEntriesQuery.refetch();
    void optionsQuery.refetch();
  }, [activeGlobalQuery, isGlobalView, optionsQuery, principalEntriesQuery, principalListQuery, principalStatsQuery]);

  useEffect(() => {
    if (!hasAssistant || isGlobalView) {
      return;
    }
    const offPrincipalAvatarUpdated = Events.On("memory:principal-avatar-updated", (event: any) => {
      const payload = event?.data ?? event;
      const payloadAssistantID = typeof payload?.assistantId === "string" ? payload.assistantId.trim() : "";
      if (!payloadAssistantID || payloadAssistantID !== selectedAssistantId) {
        return;
      }
      const payloadPrincipalType = typeof payload?.principalType === "string" ? payload.principalType.trim().toLowerCase() : "";
      if (payloadPrincipalType && payloadPrincipalType !== principalType) {
        return;
      }
      refresh();
    });
    return () => {
      offPrincipalAvatarUpdated();
    };
  }, [hasAssistant, isGlobalView, principalType, refresh, selectedAssistantId]);

  const handleSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setQuery(queryInput.trim());
    setExpandedEntryId(null);
  };

  const clearSearch = () => {
    setQuery("");
    setQueryInput("");
    setExpandedEntryId(null);
  };

  const refreshSelectedPrincipalProfile = async () => {
    if (isGlobalView || !hasAssistant || !selectedPrincipalID) {
      return;
    }
    try {
      const result = await refreshPrincipalMutation.mutateAsync({
        assistantId: selectedAssistantId,
        scope: normalizedScope,
        channel: normalizedChannel,
        accountId: normalizedAccountId,
        principalType,
        principalId: selectedPrincipalID,
      });
      if ((result.updatedRows ?? 0) > 0) {
        messageBus.publishToast({
          intent: "success",
          title: t("settings.memory.entries.summary.refreshSuccess"),
          description: `${t("settings.memory.entries.summary.refreshUpdatedRows")}: ${result.updatedRows}`,
        });
      } else {
        const hasAvatar = (result.avatarUrl?.trim() ?? "") !== "";
        messageBus.publishToast({
          intent: hasAvatar ? "success" : "warning",
          title: hasAvatar
            ? t("settings.memory.entries.summary.refreshNoChange")
            : t("settings.memory.entries.summary.refreshNoAvatar"),
        });
      }
      refresh();
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.memory.entries.summary.refreshError"),
        description: String(error),
      });
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
      <div className="pb-2">
        <div className="flex items-center gap-3">
          <Tabs
            value={browserView}
            onValueChange={(value) => {
              if ((browserViewValues as readonly string[]).includes(value)) {
                setBrowserView(value as BrowserViewValue);
                setExpandedEntryId(null);
              }
            }}
            className="min-w-0"
          >
            <TabsList className="h-auto shrink-0 rounded-none bg-transparent p-0">
              <TabsTrigger
                value="global"
                className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
              >
                <Globe className="h-4 w-4" />
                <span className="truncate">{t("settings.memory.entries.view.global")}</span>
              </TabsTrigger>
              <TabsTrigger
                value="user"
                className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
              >
                <User className="h-4 w-4" />
                <span className="truncate">{t("settings.memory.entries.view.user")}</span>
              </TabsTrigger>
              <TabsTrigger
                value="group"
                className="-mb-px rounded-none border-b-2 border-transparent px-2 py-2 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none"
              >
                <Users className="h-4 w-4" />
                <span className="truncate">{t("settings.memory.entries.view.group")}</span>
              </TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="ml-auto flex min-w-0 items-center gap-2">
            <span className="whitespace-nowrap text-xs text-muted-foreground">
              {t("settings.memory.entries.count")}: {headerCount}
            </span>
            <Select
              id="memory-browser-assistant"
              value={selectedAssistantId}
              onChange={(event) => {
                setSelectedAssistantId(event.target.value);
                setExpandedEntryId(null);
              }}
              className="w-[240px] min-w-0 shrink focus-visible:ring-inset focus-visible:ring-offset-0"
              aria-label={t("settings.memory.entries.assistant")}
            >
              {assistants.length === 0 ? (
                <option value="">{t("settings.memory.entries.assistantEmpty")}</option>
              ) : (
                assistants.map((assistant) => (
                  <option key={assistant.id} value={assistant.id}>
                    {assistant.identity?.name?.trim() || assistant.id}
                  </option>
                ))
              )}
            </Select>
            <Select
              id="memory-browser-scope"
              value={scope}
              onChange={(event) => {
                setScope(event.target.value);
                setExpandedEntryId(null);
              }}
              className="w-[160px] min-w-0 shrink focus-visible:ring-inset focus-visible:ring-offset-0"
              aria-label={t("settings.memory.entries.scope")}
            >
              <option value={allScopeValue}>{t("settings.memory.entries.scopeAll")}</option>
              {optionScopes.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </Select>
            <Button
              type="button"
              variant="outline"
              size="compactIcon"
              className="shrink-0"
              onClick={refresh}
              aria-label={t("common.refresh")}
              title={t("common.refresh")}
            >
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-4 [&>*]:min-w-0">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground" htmlFor="memory-browser-channel">
              {t("settings.memory.entries.channel")}
            </label>
            <Select
              id="memory-browser-channel"
              value={channel}
              onChange={(event) => {
                setChannel(event.target.value);
                setAccountId("");
                setExpandedEntryId(null);
              }}
              className="w-full focus-visible:ring-inset focus-visible:ring-offset-0"
            >
              <option value="">{t("settings.memory.entries.filter.channelAll")}</option>
              {optionChannels.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground" htmlFor="memory-browser-account-id">
              {t("settings.memory.entries.accountId")}
            </label>
            <Select
              id="memory-browser-account-id"
              value={accountId}
              onChange={(event) => {
                setAccountId(event.target.value);
                setExpandedEntryId(null);
              }}
              className="w-full focus-visible:ring-inset focus-visible:ring-offset-0"
            >
              <option value="">{t("settings.memory.entries.filter.accountAll")}</option>
              {accountOptionItems.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground" htmlFor="memory-browser-category">
              {t("settings.memory.entries.category")}
            </label>
            <Select
              id="memory-browser-category"
              value={category}
              onChange={(event) => {
                setCategory(event.target.value);
                setExpandedEntryId(null);
              }}
              className="w-full focus-visible:ring-inset focus-visible:ring-offset-0"
            >
              <option value={allCategoryValue}>{t("settings.memory.entries.categoryAll")}</option>
              {categoryOptions.map((item) => (
                <option key={item} value={item}>
                  {t(`settings.memory.entries.categoryOption.${item}`)}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground" htmlFor="memory-browser-limit">
              {t("settings.memory.entries.limit")}
            </label>
            <Input
              id="memory-browser-limit"
              type="number"
              min={1}
              max={100}
              value={limitInput}
              size="compact"
              className="w-full focus-visible:ring-inset focus-visible:ring-offset-0"
              onChange={(event) => {
                setLimitInput(event.target.value);
                setExpandedEntryId(null);
              }}
            />
          </div>
        </div>

        <div className="border-b border-border/70" />

        {isGlobalView ? (
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            <form className="flex flex-wrap items-center gap-2" onSubmit={handleSearchSubmit}>
              <div className="min-w-0 flex-1">
                <Input
                  value={queryInput}
                  onChange={(event) => setQueryInput(event.target.value)}
                  placeholder={t("settings.memory.entries.queryPlaceholder")}
                  size="compact"
                  className="focus-visible:ring-inset focus-visible:ring-offset-0"
                />
              </div>
              <Button type="submit" variant="outline" size="compact" className="gap-1">
                <Search className="h-4 w-4" />
                <span>{t("settings.memory.entries.search")}</span>
              </Button>
              {normalizedQuery ? (
                <Button type="button" variant="ghost" size="compact" onClick={clearSearch} className="gap-1">
                  <X className="h-4 w-4" />
                  <span>{t("settings.memory.entries.clear")}</span>
                </Button>
              ) : null}
            </form>

            <div className="flex min-h-0 flex-1 flex-col rounded-lg border border-border bg-card">
              <div className="flex min-h-0 flex-1 flex-col p-2">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "24%" }} />
                    <col style={{ width: "14%" }} />
                    <col style={{ width: "12%" }} />
                    <col style={{ width: "50%" }} />
                  </colgroup>
                  <TableHeader className="app-table-dense-head [&_tr]:border-b">
                    <TableRow>
                      <TableHead className="whitespace-nowrap">{t("settings.memory.entries.table.updatedAt")}</TableHead>
                      <TableHead className="whitespace-nowrap text-center">{t("settings.memory.entries.table.category")}</TableHead>
                      <TableHead className="whitespace-nowrap text-center">{t("settings.memory.entries.confidenceLabel")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.memory.entries.table.content")}</TableHead>
                    </TableRow>
                  </TableHeader>
                </Table>

                <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
                  <Table className="text-xs table-fixed w-full">
                    <colgroup>
                      <col style={{ width: "24%" }} />
                      <col style={{ width: "14%" }} />
                      <col style={{ width: "12%" }} />
                      <col style={{ width: "50%" }} />
                    </colgroup>
                    <TableBody>
                      {renderMemoryRows({
                        t,
                        entries: globalEntries,
                        isLoading: globalLoading,
                        errorText: globalErrorText,
                        hasAssistant,
                        expandedEntryId,
                        threadTitleByID,
                        onToggleExpand: (id) => setExpandedEntryId((current) => (current === id ? null : id)),
                      })}
                    </TableBody>
                  </Table>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-hidden rounded-lg border bg-card">
            <div className="flex h-full min-h-0 overflow-hidden p-3">
              <div className="flex min-h-0 w-[280px] shrink-0 flex-col overflow-hidden">
                {renderPrincipalSidebar({
                  t,
                  browserView,
                  principalItems,
                  selectedPrincipalID,
                  setSelectedPrincipalID: browserView === "group" ? setSelectedGroupID : setSelectedUserID,
                  httpBaseUrl,
                  isLoading: principalListQuery.isLoading || principalListQuery.isFetching,
                  errorText: principalListQuery.error ? String(principalListQuery.error) : "",
                })}
              </div>

              <div className="mx-3 self-stretch">
                <div className="h-full w-px bg-border/70" />
              </div>

              <div className="min-h-0 min-w-0 flex-1 overflow-y-auto overflow-x-hidden">
                {!selectedPrincipalID ? (
                  <div className="flex h-full min-h-[240px] items-center justify-center">
                    <div className="max-w-md rounded-md border border-dashed border-border/70 bg-muted/20 px-5 py-6 text-center">
                      <Users className="mx-auto mb-3 h-5 w-5 text-muted-foreground" />
                      <div className="text-sm text-foreground">{t("settings.memory.entries.summary.selectHint")}</div>
                    </div>
                  </div>
                ) : principalSummaryLoading ? (
                  <div className="text-sm text-muted-foreground">{t("settings.memory.entries.summary.loading")}</div>
                ) : principalSummaryError ? (
                  <div className="text-sm text-destructive">
                    {t("settings.memory.entries.error")}: {principalSummaryError}
                  </div>
                ) : (
                  <div className="space-y-3">
                    <div className="rounded-md border border-border/60 bg-muted/15 p-3">
                      <div className="flex min-w-0 items-center gap-3">
                        {renderPrincipalAvatar(selectedPrincipalItem, httpBaseUrl, "h-12 w-12")}
                        <div className="min-w-0 flex-1 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1">
                          <div className="truncate text-base font-medium text-foreground">{selectedPrincipalName}</div>
                          <div className="text-xs text-muted-foreground">
                            {t("settings.memory.entries.summary.total")}:{" "}
                            <span className="font-medium text-foreground">{principalStats?.totalCount ?? 0}</span>
                          </div>
                          <div className="truncate font-mono text-xs text-muted-foreground">{selectedPrincipalIdentityLine}</div>
                          <div className="text-xs text-muted-foreground">
                            {t("settings.memory.entries.summary.updatedAt")}:{" "}
                            <span className="font-medium text-foreground">{formatRelativeTime(principalStats?.lastUpdatedAt, language)}</span>
                          </div>
                        </div>
                        <Button
                          type="button"
                          variant="outline"
                          size="compactIcon"
                          className="shrink-0"
                          onClick={() => {
                            void refreshSelectedPrincipalProfile();
                          }}
                          disabled={refreshPrincipalMutation.isPending}
                          aria-label={t("settings.memory.entries.summary.refreshProfile")}
                          title={t("settings.memory.entries.summary.refreshProfile")}
                        >
                          <RefreshCw className={`h-4 w-4${refreshPrincipalMutation.isPending ? " animate-spin" : ""}`} />
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <div className="text-sm font-medium text-muted-foreground">{t("settings.memory.entries.summary.categories")}</div>
                      <div className="flex flex-wrap gap-2 text-xs">
                        {principalCategories.length === 0 ? (
                          <span className="text-muted-foreground">-</span>
                        ) : (
                          principalCategories.map(([itemCategory, count]) => (
                            <span key={itemCategory} className="rounded-full bg-muted px-2 py-0.5 text-muted-foreground">
                              {t(`settings.memory.entries.categoryOption.${itemCategory}`)} ({count})
                            </span>
                          ))
                        )}
                      </div>
                    </div>

                    <div className="space-y-2">
                      <div className="text-sm font-medium text-muted-foreground">{t("settings.memory.entries.summary.recent")}</div>
                      {principalEntries.length === 0 ? (
                        <div className="rounded-md border border-dashed border-border/70 bg-muted/20 p-3 text-sm text-muted-foreground">
                          {t("settings.memory.entries.summary.empty")}
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {principalEntries.slice(0, 8).map((entry, index) => (
                            <div key={entry.id || `${entry.updatedAt}-${index}`} className="rounded-md border border-border/70 bg-muted/10 p-2">
                              <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                                <span>{formatTime(entry.updatedAt || entry.createdAt)}</span>
                                <span className="font-mono">{formatCategoryLabel(t, entry.category)}</span>
                              </div>
                              <div className="mt-1 text-xs text-foreground">{buildContentPreview(entry.content || "", 200) || "-"}</div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
