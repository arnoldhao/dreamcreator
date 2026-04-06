import * as React from "react";
import { Browser, Events } from "@wailsio/runtime";
import { ArrowRight, Ban, Check, HelpCircle, Loader2, Minus, Search } from "lucide-react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty";
import { Skeleton } from "@/shared/ui/skeleton";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useAssistants } from "@/shared/query/assistant";
import { useSettings } from "@/shared/query/settings";
import {
  useInspectSkill,
  useInstallSkill,
  useRemoveInstalledSkill,
  useSearchSkills,
  useSkillsCatalog,
  useSkillsStatus,
  useUpdateSkill,
} from "@/shared/query/skills";

import type {
  LocalSkillListItem,
  RemoteSkillListItem,
  SkillDetailContentTab,
  SkillListItem,
} from "../types";
import {
  isRecord,
  formatByteSize,
  formatRuntimeInstallSpec,
  isSkillVersionOutdated,
  resolveSkillsActionErrorInfo,
  stripMarkdownFrontmatter,
} from "../utils/calls-utils";
import {
  defaultBuiltinLocalSourceName,
  defaultRemoteSourceName,
  parseSkillsSourcesState,
  resolveSkillsSettingsState,
} from "../utils/skills-settings-utils";
import { CallsCard } from "./CallsCard";
import { SKILLS_PAGE_CARD_CLASS, SKILLS_PANEL_CARD_CLASS } from "./skills-page-styles";
import { SkillsStackedCardsIllustration } from "./SkillsStackedCardsIllustration";

type SkillGroup<TItem> = {
  key: string;
  label: string;
  items: TItem[];
};

function resolveCatalogLocalSourceLabel(
  item: LocalSkillListItem
): string {
  const sourceType = item.skill.sourceType?.trim().toLowerCase() ?? "";
  const sourceName = item.skill.sourceName?.trim() ?? "";
  if (sourceName) {
    return sourceName;
  }
  if (sourceType === "workspace") {
    return defaultBuiltinLocalSourceName("workspace");
  }
  return item.skill.sourceId?.trim() || item.skill.providerId?.trim() || "Local";
}

function resolveCatalogRemoteSourceLabel(
  item: RemoteSkillListItem
): string {
  const source = item.result.source?.trim().toLowerCase() ?? "";
  if (source === "clawhub" || source === "") {
    return defaultRemoteSourceName("clawhub");
  }
  return item.result.source?.trim() || source;
}

function groupLocalSkills(
  items: LocalSkillListItem[]
): SkillGroup<LocalSkillListItem>[] {
  const groups = new Map<string, SkillGroup<LocalSkillListItem>>();
  items.forEach((item) => {
    const label = resolveCatalogLocalSourceLabel(item);
    const key = `${item.skill.sourceId?.trim().toLowerCase() || label.toLowerCase() || "local"}`;
    const current = groups.get(key);
    if (current) {
      current.items.push(item);
      return;
    }
    groups.set(key, { key, label, items: [item] });
  });
  return Array.from(groups.values());
}

function groupRemoteSkills(
  items: RemoteSkillListItem[]
): SkillGroup<RemoteSkillListItem>[] {
  const groups = new Map<string, SkillGroup<RemoteSkillListItem>>();
  items.forEach((item) => {
    const label = resolveCatalogRemoteSourceLabel(item);
    const key = `${item.result.source?.trim().toLowerCase() || "clawhub"}`;
    const current = groups.get(key);
    if (current) {
      current.items.push(item);
      return;
    }
    groups.set(key, { key, label, items: [item] });
  });
  return Array.from(groups.values());
}

export function SkillsCatalogTab() {
  const { t, language } = useI18n();

  const handleConfirmRemove = (name: string, onConfirm: () => void) => {
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.calls.action.removeTitle"),
      description: t("settings.calls.action.removeDescription").replace("{name}", name),
      confirmLabel: t("settings.calls.action.remove"),
      cancelLabel: t("common.cancel"),
      onConfirm,
    });
  };

  const renderToolStatusBadge = React.useCallback(
    (status: { allowed: boolean } | null) => {
      if (!status) {
        return null;
      }
      const label = status.allowed
        ? t("settings.tools.status.allowed")
        : t("settings.tools.status.blocked");
      const Icon = status.allowed ? Check : Ban;
      const variant = status.allowed ? "secondary" : "outline";
      return (
        <Badge
          variant={variant}
          className="h-5 w-5 justify-center p-0"
          title={label}
          aria-label={label}
        >
          <Icon className="h-3 w-3" />
          <span className="sr-only">{label}</span>
        </Badge>
      );
    },
    [t]
  );

  const renderInstalledStatusBadge = React.useCallback(() => {
    const label = t("settings.calls.skillsInstalled");
    return (
      <Badge
        variant="secondary"
        className="h-5 w-5 justify-center p-0"
        title={label}
        aria-label={label}
      >
        <Check className="h-3 w-3" />
        <span className="sr-only">{label}</span>
      </Badge>
    );
  }, [t]);

  const assistantsQuery = useAssistants(true);
  const settingsQuery = useSettings();
  const assistants = assistantsQuery.data ?? [];
  const activeAssistantId = React.useMemo(() => {
    const defaultAssistant = assistants.find((item) => item.isDefault);
    if (defaultAssistant?.id) {
      return defaultAssistant.id;
    }
    const enabledAssistant = assistants.find((item) => item.enabled);
    return enabledAssistant?.id ?? assistants[0]?.id ?? "";
  }, [assistants]);
  const settingsToolsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.tools;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.tools]);
  const settingsSkillsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.skills;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.skills]);
  const skillsSourcesState = React.useMemo(() => {
    const { skillsConfig } = resolveSkillsSettingsState(settingsToolsRaw, settingsSkillsRaw);
    return parseSkillsSourcesState(skillsConfig);
  }, [settingsSkillsRaw, settingsToolsRaw]);
  const clawhubSource = React.useMemo(
    () => skillsSourcesState.remote.find((item) => item.provider === "clawhub") ?? null,
    [skillsSourcesState.remote]
  );
  const remoteSearchConfigured = clawhubSource ? clawhubSource.enabled && clawhubSource.searchEnabled : true;
  const remoteInstallConfigured = clawhubSource ? clawhubSource.enabled && clawhubSource.installEnabled : true;

  const skillsStatus = useSkillsStatus({ assistantId: activeAssistantId });
  const skillsQuery = useSkillsCatalog();
  const installSkill = useInstallSkill();
  const updateSkill = useUpdateSkill();
  const removeInstalledSkill = useRemoveInstalledSkill();
  const discoveredSkills = skillsQuery.data ?? [];
  const clawhubReady = skillsStatus.data?.clawhubReady ?? false;
  const skillsMutationPending =
    installSkill.isPending ||
    updateSkill.isPending ||
    removeInstalledSkill.isPending;
  const [skillsActionPending, setSkillsActionPending] = React.useState<"install" | "update" | "remove" | null>(null);
  React.useEffect(() => {
    if (!skillsMutationPending) {
      setSkillsActionPending((previous) => (previous === null ? previous : null));
    }
  }, [skillsMutationPending]);
  const installedSkillLookup = React.useMemo(() => {
    const result = new Set<string>();
    discoveredSkills.forEach((item) => {
      const id = item.id?.trim().toLowerCase();
      if (id) {
        result.add(id);
      }
    });
    return result;
  }, [discoveredSkills]);
  const installedSkillVersionLookup = React.useMemo(() => {
    const result = new Map<string, string>();
    discoveredSkills.forEach((item) => {
      const id = item.id?.trim().toLowerCase();
      if (!id) {
        return;
      }
      const version = item.version?.trim() ?? "";
      if (version) {
        result.set(id, version);
      }
    });
    return result;
  }, [discoveredSkills]);

  const [skillsQueryValue, setSkillsQueryValue] = React.useState("");
  const [skillsQuerySubmitted, setSkillsQuerySubmitted] = React.useState("");
  const [selectedSkillKey, setSelectedSkillKey] = React.useState<string | null>(null);
  const [skillDetailContentTab, setSkillDetailContentTab] = React.useState<SkillDetailContentTab>("skill_md");

  const minSkillsQueryLength = 2;
  const submittedSkillsQuery = skillsQuerySubmitted.trim();
  const normalizedSubmittedSkillsQuery = submittedSkillsQuery.toLowerCase();
  const skillsSearchActive = submittedSkillsQuery !== "";
  const remoteQueryReady = submittedSkillsQuery.length >= minSkillsQueryLength;
  const remoteSearchEnabled = skillsSearchActive && remoteQueryReady && clawhubReady && remoteSearchConfigured;
  const remoteSearchBlocked =
    skillsSearchActive && remoteQueryReady && (!clawhubReady || !remoteSearchConfigured);
  const effectiveSkillsQuery = remoteSearchEnabled ? submittedSkillsQuery : "";
  const skillsSearch = useSearchSkills({
    query: effectiveSkillsQuery,
    limit: 20,
    assistantId: activeAssistantId,
  });
  const localSkills = React.useMemo(() => {
    if (!skillsSearchActive || normalizedSubmittedSkillsQuery === "") {
      return discoveredSkills;
    }
    return discoveredSkills.filter((item) => {
      const id = item.id?.toLowerCase() ?? "";
      const name = item.name?.toLowerCase() ?? "";
      const description = item.description?.toLowerCase() ?? "";
      return (
        id.includes(normalizedSubmittedSkillsQuery) ||
        name.includes(normalizedSubmittedSkillsQuery) ||
        description.includes(normalizedSubmittedSkillsQuery)
      );
    });
  }, [discoveredSkills, normalizedSubmittedSkillsQuery, skillsSearchActive]);
  const remoteSearchStatus: "idle" | "loading" | "error" | "blocked" | "too_short" = !skillsSearchActive
    ? "idle"
    : remoteSearchBlocked
    ? "blocked"
    : !remoteQueryReady
    ? "too_short"
    : skillsSearch.isFetching
      ? "loading"
      : skillsSearch.isError
        ? "error"
        : "idle";
  const remoteSearchResults =
    remoteSearchEnabled && remoteSearchStatus === "idle" ? skillsSearch.data ?? [] : [];
  const localSkillsList: LocalSkillListItem[] = localSkills.map((skill) => ({
    kind: "local",
    key: `local:${skill.id}`,
    id: skill.id,
    skill,
  }));
  const remoteSkillsList: RemoteSkillListItem[] =
    skillsSearchActive && remoteSearchStatus === "idle"
      ? remoteSearchResults.map((result) => ({
          kind: "search",
          key: `remote:${result.id}`,
          id: result.id,
          result,
        }))
      : [];
  const localSkillGroups = React.useMemo(() => groupLocalSkills(localSkillsList), [localSkillsList]);
  const remoteSkillGroups = React.useMemo(() => groupRemoteSkills(remoteSkillsList), [remoteSkillsList]);
  const remoteSourceLabel = React.useMemo(() => {
    const configuredName = clawhubSource?.name?.trim() ?? "";
    return configuredName || defaultRemoteSourceName("clawhub");
  }, [clawhubSource]);
  const skillsList: SkillListItem[] = skillsSearchActive
    ? [...localSkillsList, ...remoteSkillsList]
    : localSkillsList;
  const handleSkillsSearchSubmit = React.useCallback(() => {
    const query = skillsQueryValue.trim();
    if (query.length === 0) {
      setSkillsQuerySubmitted("");
      return;
    }
    if (query === submittedSkillsQuery && remoteSearchEnabled) {
      void skillsSearch.refetch();
      return;
    }
    setSkillsQuerySubmitted(query);
  }, [remoteSearchEnabled, skillsQueryValue, skillsSearch, submittedSkillsQuery]);

  React.useEffect(() => {
    if (selectedSkillKey && skillsList.some((item) => item.key === selectedSkillKey)) {
      return;
    }
    setSelectedSkillKey(null);
  }, [skillsList, selectedSkillKey]);

  React.useEffect(() => {
    setSkillDetailContentTab("skill_md");
  }, [selectedSkillKey]);

  const selectedSkillItem = skillsList.find((item) => item.key === selectedSkillKey) ?? null;
  const selectedSkillDisplayName = React.useMemo(() => {
    if (!selectedSkillItem) {
      return "";
    }
    if (selectedSkillItem.kind === "local") {
      return selectedSkillItem.skill.name || selectedSkillItem.skill.id;
    }
    return selectedSkillItem.result.name || selectedSkillItem.result.id;
  }, [selectedSkillItem]);
  const selectedSearchItemInstalled =
    selectedSkillItem?.kind === "local"
      ? true
      : selectedSkillItem?.kind === "search"
        ? installedSkillLookup.has((selectedSkillItem.result.id ?? "").trim().toLowerCase())
        : false;
  const selectedSearchSkillID = React.useMemo(() => {
    if (!selectedSkillItem) {
      return "";
    }
    if (selectedSkillItem.kind === "local") {
      return selectedSkillItem.skill.id?.trim() ?? "";
    }
    return selectedSkillItem.result.id?.trim() ?? "";
  }, [selectedSkillItem]);
  const selectedSearchSkillDetail = useInspectSkill(
    selectedSearchSkillID
      ? {
          skill: selectedSearchSkillID,
          assistantId: activeAssistantId,
        }
      : undefined
  );
  const resolvedSearchSkillDetail = React.useMemo(() => {
    if (!selectedSkillItem) {
      return null;
    }
    const detail = selectedSearchSkillDetail.data;
    const fallbackSummary =
      selectedSkillItem.kind === "local" ? selectedSkillItem.skill.description || "" : selectedSkillItem.result.description || "";
    const fallbackURL = selectedSkillItem.kind === "search" ? selectedSkillItem.result.url || "" : "";
    const fallbackCurrentVersion =
      installedSkillVersionLookup.get(selectedSearchSkillID.toLowerCase()) ??
      (selectedSkillItem.kind === "local" ? selectedSkillItem.skill.version?.trim() ?? "" : "");
    return {
      id: detail?.id || selectedSearchSkillID,
      name: detail?.name || selectedSkillDisplayName || selectedSearchSkillID,
      summary: detail?.summary || fallbackSummary,
      url: detail?.url || fallbackURL,
      owner: detail?.owner || "",
      currentVersion: detail?.currentVersion || fallbackCurrentVersion,
      latestVersion: detail?.latestVersion || "",
      selectedVersion: detail?.selectedVersion || "",
      createdAt: detail?.createdAt,
      updatedAt: detail?.updatedAt,
      files: detail?.files ?? [],
      skillMarkdown: detail?.skillMarkdown || "",
      runtimeRequirements: detail?.runtimeRequirements,
    };
  }, [installedSkillVersionLookup, selectedSearchSkillDetail.data, selectedSearchSkillID, selectedSkillDisplayName, selectedSkillItem]);
  const selectedSearchDetailLoading =
    Boolean(selectedSkillItem) && selectedSearchSkillDetail.isLoading && !selectedSearchSkillDetail.data;
  const selectedSearchRuntime = resolvedSearchSkillDetail?.runtimeRequirements;
  const selectedSearchSkillMarkdownRaw = resolvedSearchSkillDetail?.skillMarkdown ?? "";
  const selectedSearchSkillMarkdown = React.useMemo(
    () => stripMarkdownFrontmatter(selectedSearchSkillMarkdownRaw),
    [selectedSearchSkillMarkdownRaw]
  );
  const selectedSearchSkillFiles = resolvedSearchSkillDetail?.files ?? [];
  const selectedSearchSummary = React.useMemo(() => {
    if (!selectedSkillItem) {
      return "";
    }
    const detailSummary = selectedSearchSkillDetail.data?.summary?.trim();
    if (detailSummary) {
      return detailSummary;
    }
    if (selectedSkillItem.kind === "local") {
      return selectedSkillItem.skill.description?.trim() ?? "";
    }
    return selectedSkillItem.result.description?.trim() ?? "";
  }, [selectedSearchSkillDetail.data?.summary, selectedSkillItem]);
  const skillMarkdownComponents = React.useMemo<Components>(
    () => ({
      h1: ({ children }) => <h2 className="text-sm font-semibold text-foreground">{children}</h2>,
      h2: ({ children }) => <h3 className="text-xs font-semibold text-foreground">{children}</h3>,
      h3: ({ children }) => <h4 className="text-xs font-medium text-foreground">{children}</h4>,
      p: ({ children }) => <p className="text-xs leading-relaxed text-foreground">{children}</p>,
      ul: ({ children }) => <ul className="ml-4 list-disc space-y-1 text-xs text-foreground">{children}</ul>,
      ol: ({ children }) => <ol className="ml-4 list-decimal space-y-1 text-xs text-foreground">{children}</ol>,
      li: ({ children }) => <li className="leading-relaxed">{children}</li>,
      a: ({ href, children, ...props }) => (
        <a
          href={href}
          className="text-primary underline underline-offset-4"
          onClick={(event) => {
            if (!href) {
              return;
            }
            event.preventDefault();
            Browser.OpenURL(href);
          }}
          {...props}
        >
          {children}
        </a>
      ),
      code: ({ className, children, ...props }) => {
        const content = String(children ?? "").replace(/\n$/, "");
        if (!className) {
          return <code className="font-mono text-[0.85em]" {...props}>{content}</code>;
        }
        return (
          <code
            className="block whitespace-pre-wrap break-words rounded-md border border-border/70 p-2 font-mono text-[0.85em]"
            {...props}
          >
            {content}
          </code>
        );
      },
      pre: ({ children }) => <pre className="whitespace-pre-wrap break-words rounded-md border border-border/70 p-2 text-xs">{children}</pre>,
      table: ({ children }) => (
        <div className="overflow-x-auto rounded-md border border-border/70">
          <table className="w-full border-collapse text-xs">{children}</table>
        </div>
      ),
      th: ({ children }) => <th className="border-b border-border/70 px-2 py-1 text-left font-medium text-foreground">{children}</th>,
      td: ({ children }) => <td className="border-b border-border/60 px-2 py-1 text-foreground">{children}</td>,
    }),
    []
  );
  const selectedSearchCurrentVersion = resolvedSearchSkillDetail?.currentVersion?.trim() ?? "";
  const selectedSearchLatestVersion = resolvedSearchSkillDetail?.latestVersion?.trim() ?? "";
  const selectedSearchUpdateAvailable =
    selectedSearchItemInstalled && isSkillVersionOutdated(selectedSearchCurrentVersion, selectedSearchLatestVersion);
  const installActionPending = skillsActionPending === "install";
  const updateActionPending = skillsActionPending === "update";
  const removeActionPending = skillsActionPending === "remove";
  const formatSkillTimestamp = React.useCallback((value?: number) => {
    if (!value || !Number.isFinite(value)) {
      return "";
    }
    const ts = value > 1_000_000_000_000 ? value : value * 1000;
    const date = new Date(ts);
    if (Number.isNaN(date.getTime())) {
      return "";
    }
    return new Intl.DateTimeFormat(language || undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }, [language]);
  const openExternalToolsSettings = React.useCallback(() => {
    Events.Emit("settings:navigate", "external-tools");
  }, []);
  const refreshSkillsViews = React.useCallback(() => {
    void skillsQuery.refetch();
    void skillsStatus.refetch();
    if (remoteSearchEnabled) {
      void skillsSearch.refetch();
    }
    if (selectedSearchSkillID) {
      void selectedSearchSkillDetail.refetch();
    }
  }, [
    remoteSearchEnabled,
    selectedSearchSkillDetail,
    selectedSearchSkillID,
    skillsQuery,
    skillsSearch,
    skillsStatus,
  ]);
  const notifySkillsActionError = React.useCallback(
    (title: string, error: unknown) => {
      const errorInfo = resolveSkillsActionErrorInfo(error);
      const rateLimitHint = t("settings.calls.skillsRateLimitedHint");
      const description = errorInfo.rateLimited
        ? `${rateLimitHint}\n${errorInfo.message}`.trim()
        : errorInfo.message;
      messageBus.publishToast({
        title,
        description,
        intent: "warning",
      });
    },
    [t]
  );
  const requestInstallOrUpdate = React.useCallback(
    (params: { skillID: string; displayName: string; installed: boolean; force: boolean }) => {
      const normalizedID = params.skillID.trim();
      if (normalizedID === "") {
        return;
      }
      const action = params.installed ? "update" : "install";
      const mutation = params.installed ? updateSkill : installSkill;
      const title = params.installed
        ? t("settings.calls.skillsUpdateError")
        : t("settings.calls.skillsInstallError");
      setSkillsActionPending(action);
      mutation.mutate(
        {
          skill: normalizedID,
          assistantId: activeAssistantId,
          force: params.force,
        },
        {
          onError: (error) => {
            const errorInfo = resolveSkillsActionErrorInfo(error);
            if (errorInfo.requiresForce && !params.force) {
              messageBus.publishDialog({
                intent: "danger",
                title: t("settings.calls.skillsForceInstallTitle"),
                description: `${t("settings.calls.skillsForceInstallDescription")}\n${params.displayName}\n\n${errorInfo.message}`,
                confirmLabel: params.installed
                  ? t("settings.calls.skillsForceUpdate")
                  : t("settings.calls.skillsForceInstall"),
                cancelLabel: t("common.cancel"),
                onConfirm: () => {
                  requestInstallOrUpdate({
                    ...params,
                    force: true,
                  });
                },
              });
              return;
            }
            notifySkillsActionError(title, error);
          },
          onSettled: () => {
            setSkillsActionPending(null);
            refreshSkillsViews();
          },
        }
      );
    },
    [activeAssistantId, installSkill, notifySkillsActionError, refreshSkillsViews, t, updateSkill]
  );
  const handleInstallOrUpdate = React.useCallback(() => {
    if (!selectedSkillItem || !selectedSearchSkillID) {
      return;
    }
    if (!remoteInstallConfigured) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.calls.skillsRemoteInstallDisabled"),
        description: t("settings.calls.skillsRemoteInstallDisabledHint"),
      });
      return;
    }
    requestInstallOrUpdate({
      skillID: selectedSearchSkillID,
      displayName: selectedSkillDisplayName || selectedSearchSkillID,
      installed: selectedSearchItemInstalled,
      force: false,
    });
  }, [remoteInstallConfigured, requestInstallOrUpdate, selectedSearchItemInstalled, selectedSearchSkillID, selectedSkillDisplayName, selectedSkillItem, t]);
  const handleRemoveSkill = React.useCallback(
    (skillID: string, displayName?: string) => {
      const normalizedID = skillID.trim();
      if (normalizedID === "") {
        return;
      }
      handleConfirmRemove(displayName?.trim() || normalizedID, () => {
        setSkillsActionPending("remove");
        removeInstalledSkill.mutate(
          { skill: normalizedID, assistantId: activeAssistantId },
          {
            onError: (error) => notifySkillsActionError(t("settings.calls.skillsRemoveError"), error),
            onSettled: () => {
              setSkillsActionPending(null);
              refreshSkillsViews();
            },
          }
        );
      });
    },
    [activeAssistantId, handleConfirmRemove, notifySkillsActionError, refreshSkillsViews, removeInstalledSkill, t]
  );
  const renderSkillsSelectionEmpty = () => (
    <Empty className="py-10">
      <EmptyHeader>
        <EmptyMedia>
          <SkillsStackedCardsIllustration />
        </EmptyMedia>
        <EmptyTitle>{t("settings.calls.skillsSelectTitle")}</EmptyTitle>
        <EmptyDescription>
          {t("settings.calls.skillsSelectDescription")}
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );

  const skillsLeftHeader = (
    <div className="flex h-8 items-center gap-2 rounded-lg border border-border/70 bg-background/80 px-2">
      <Search className="h-4 w-4 text-muted-foreground" />
      <Input
        value={skillsQueryValue}
        onChange={(event) => {
          const nextValue = event.target.value;
          setSkillsQueryValue(nextValue);
          if (nextValue.trim() == "") {
            setSkillsQuerySubmitted("");
          }
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            handleSkillsSearchSubmit();
          }
        }}
        placeholder={t("settings.calls.skillsSearchPlaceholder")}
        size="compact"
        className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
      />
    </div>
  );

  const skillsFooter = (
    <TooltipProvider delayDuration={0}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex h-2 w-2 rounded-full ${clawhubReady ? "bg-emerald-500" : "bg-red-500"}`}
            aria-hidden="true"
          />
          <span className="text-xs text-muted-foreground">{remoteSourceLabel}</span>
          {!clawhubReady ? (
            <Button
              variant="outline"
              size="compact"
              onClick={openExternalToolsSettings}
            >
              {t("settings.calls.openExternalTools")}
            </Button>
          ) : null}
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="compactIcon"
              aria-label={t("settings.calls.action.remove")}
              disabled={!clawhubReady || !selectedSkillItem || selectedSkillItem.kind !== "local" || skillsMutationPending}
              onClick={() => {
                if (selectedSkillItem?.kind !== "local") {
                  return;
                }
                const skill = selectedSkillItem.skill;
                handleRemoveSkill(skill.id, skill.name || skill.id);
              }}
            >
              {removeActionPending ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Minus className="h-3 w-3" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("settings.calls.action.remove")}</TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );

  return (
    <CallsCard
      className={SKILLS_PAGE_CARD_CLASS}
      leftHeader={skillsLeftHeader}
      leftFooter={skillsFooter}
      leftList={
        <div className="space-y-3">
          {localSkillGroups.length === 0 ? (
            skillsSearchActive ? (
              <div className="px-2 py-1 text-xs text-muted-foreground">
                {t("settings.calls.skillsSearchLocalEmpty")}
              </div>
            ) : null
          ) : (
            <div className="space-y-3">
              {localSkillGroups.map((group) => (
                <div key={group.key}>
                  <div className="px-2 pb-1 pt-2 text-[11px] font-medium tracking-wide text-muted-foreground">
                    {group.label}
                  </div>
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const isSelected = item.key === selectedSkillKey;
                      const allowed = item.skill.enabled !== false;
                      return (
                        <SidebarMenuItem key={item.key}>
                          <SidebarMenuButton
                            isActive={isSelected}
                            size="sm"
                            className="justify-between px-2.5"
                            onClick={() => setSelectedSkillKey(item.key)}
                          >
                            <span className="truncate text-xs font-medium">
                              {item.skill.name || item.skill.id}
                            </span>
                            {renderToolStatusBadge({ allowed })}
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      );
                    })}
                  </SidebarMenu>
                </div>
              ))}
            </div>
          )}

          {skillsSearchActive ? (
            <div>
              <div className="px-2 pb-1 pt-2 text-[11px] font-medium tracking-wide text-muted-foreground">
                {remoteSourceLabel}
              </div>
              {remoteSearchStatus === "blocked" ? (
                <div className="space-y-2 px-2 py-1 text-xs text-muted-foreground">
                  <div>
                    {remoteSearchConfigured
                      ? t("settings.calls.skillsUnavailableHint")
                      : t("settings.calls.skillsRemoteDisabledHint")}
                  </div>
                  {!remoteSearchConfigured ? null : (
                    <Button variant="outline" size="compact" onClick={openExternalToolsSettings}>
                      {t("settings.calls.openExternalTools")}
                    </Button>
                  )}
                </div>
              ) : remoteSearchStatus === "too_short" ? (
                <div className="px-2 py-1 text-xs text-muted-foreground">
                  {t("settings.calls.skillsRemoteMinLength")}
                </div>
              ) : remoteSearchStatus === "loading" ? (
                <div className="px-2 py-1 text-xs text-muted-foreground">
                  {t("settings.calls.skillsSearchLoading")}
                </div>
              ) : remoteSearchStatus === "error" ? (
                <div className="px-2 py-1 text-xs text-muted-foreground">
                  {t("settings.calls.skillsSearchError")}
                </div>
              ) : remoteSkillGroups.length === 0 ? (
                <div className="px-2 py-1 text-xs text-muted-foreground">
                  {t("settings.calls.skillsSearchRemoteEmpty")}
                </div>
              ) : (
                <div className="space-y-3">
                  {remoteSkillGroups.map((group) => (
                    <div key={group.key}>
                      {remoteSkillGroups.length > 1 ? (
                        <div className="px-2 pb-1 pt-2 text-[11px] font-medium tracking-wide text-muted-foreground">
                          {group.label}
                        </div>
                      ) : null}
                      <SidebarMenu>
                        {group.items.map((item) => {
                          const isSelected = item.key === selectedSkillKey;
                          const installed = installedSkillLookup.has((item.id ?? "").trim().toLowerCase());
                          return (
                            <SidebarMenuItem key={item.key}>
                              <SidebarMenuButton
                                isActive={isSelected}
                                size="sm"
                                className="justify-between px-2.5"
                                onClick={() => setSelectedSkillKey(item.key)}
                              >
                                <span className="truncate text-xs font-medium">
                                  {item.result.name || item.result.id}
                                </span>
                                {installed ? renderInstalledStatusBadge() : null}
                              </SidebarMenuButton>
                            </SidebarMenuItem>
                          );
                        })}
                      </SidebarMenu>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : null}
        </div>
      }
      rightContent={
        !selectedSkillItem ? (
          <div className="flex min-h-full items-center justify-center">{renderSkillsSelectionEmpty()}</div>
        ) : (
          <div className="flex h-full min-h-0 flex-col gap-3">
            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(280px,340px)]">
              <Card className={SKILLS_PANEL_CARD_CLASS}>
                <CardContent size="compact" className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="truncate text-sm font-semibold text-foreground">
                      {resolvedSearchSkillDetail?.name || selectedSkillDisplayName || selectedSearchSkillID}
                    </div>
                    {resolvedSearchSkillDetail?.owner ? (
                      <Badge variant="outline">
                        {t("settings.calls.skillsDetailOwner")}: {resolvedSearchSkillDetail.owner}
                      </Badge>
                    ) : null}
                    {selectedSearchItemInstalled ? (
                      <Badge variant="secondary">{t("settings.calls.skillsInstalled")}</Badge>
                    ) : null}
                  </div>
                  {selectedSearchDetailLoading ? (
                    <div className="space-y-2 pt-1">
                      <Skeleton className="h-4 w-11/12" />
                      <Skeleton className="h-4 w-4/5" />
                    </div>
                  ) : selectedSearchSummary ? (
                    <div className="text-xs leading-relaxed text-muted-foreground">{selectedSearchSummary}</div>
                  ) : null}
                </CardContent>
              </Card>

              <Card className={SKILLS_PANEL_CARD_CLASS}>
                <CardContent className="p-0">
                  {selectedSearchDetailLoading ? (
                    <>
                      <div className="flex items-center justify-between gap-3 px-4 py-3">
                        <Skeleton className="h-4 w-16" />
                        <div className="flex items-center gap-1.5">
                          <Skeleton className="h-5 w-20 rounded-full" />
                          <Skeleton className="h-3 w-3 rounded-full" />
                          <Skeleton className="h-5 w-20 rounded-full" />
                        </div>
                      </div>
                      <Separator />
                      <div className="flex items-center justify-between gap-3 px-4 py-3">
                        <Skeleton className="h-4 w-16" />
                        <Skeleton className="h-3 w-24" />
                      </div>
                      <Separator />
                      <div className="flex items-start justify-between gap-3 px-4 py-3">
                        <Skeleton className="h-4 w-12" />
                        <div className="flex items-center gap-2">
                          <Skeleton className="h-8 w-20 rounded-md" />
                          <Skeleton className="h-8 w-20 rounded-md" />
                        </div>
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="flex items-center justify-between gap-3 px-4 py-3">
                        <div className="text-xs text-muted-foreground">{t("settings.calls.skillsDetailVersions")}</div>
                        <div className="flex items-center gap-1.5">
                          <Badge variant={selectedSearchCurrentVersion ? "secondary" : "outline"}>
                            {selectedSearchCurrentVersion ||
                              t("settings.calls.skillsNotInstalled")}
                          </Badge>
                          <ArrowRight className="h-3.5 w-3.5 text-muted-foreground" />
                          <Badge variant="outline">{selectedSearchLatestVersion || "-"}</Badge>
                        </div>
                      </div>
                      <Separator />
                      <div className="flex items-center justify-between gap-3 px-4 py-3">
                        <div className="text-xs text-muted-foreground">{t("settings.calls.skillsDetailUpdated")}</div>
                        <div className="text-xs text-muted-foreground">
                          {formatSkillTimestamp(resolvedSearchSkillDetail?.updatedAt) || "-"}
                        </div>
                      </div>
                      <Separator />
                      <div className="flex items-start justify-between gap-3 px-4 py-3">
                        <div className="text-xs text-muted-foreground">{t("settings.calls.skillsDetailAction")}</div>
                        <div className="flex flex-wrap justify-end gap-2">
                          {!selectedSearchItemInstalled ? (
                            <Button
                              size="compact"
                              disabled={!clawhubReady || !remoteInstallConfigured || skillsMutationPending || !selectedSearchSkillID}
                              onClick={handleInstallOrUpdate}
                              className="gap-1.5"
                            >
                              {installActionPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                              {t("settings.calls.skillsInstall")}
                            </Button>
                          ) : selectedSearchUpdateAvailable ? (
                            <>
                              <Button
                                size="compact"
                                disabled={!clawhubReady || !remoteInstallConfigured || skillsMutationPending || !selectedSearchSkillID}
                                onClick={handleInstallOrUpdate}
                                className="gap-1.5"
                              >
                                {updateActionPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                                {t("settings.calls.skillsUpdate")}
                              </Button>
                              <Button
                                variant="outline"
                                size="compact"
                                disabled={!clawhubReady || skillsMutationPending || !selectedSearchSkillID}
                                className="gap-1.5"
                                onClick={() => handleRemoveSkill(selectedSearchSkillID, selectedSkillDisplayName || selectedSearchSkillID)}
                              >
                                {removeActionPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                                {t("settings.calls.skillsUninstall")}
                              </Button>
                            </>
                          ) : (
                            <Button
                              variant="outline"
                              size="compact"
                              disabled={!clawhubReady || skillsMutationPending || !selectedSearchSkillID}
                              className="gap-1.5"
                              onClick={() => handleRemoveSkill(selectedSearchSkillID, selectedSkillDisplayName || selectedSearchSkillID)}
                            >
                              {removeActionPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                              {t("settings.calls.skillsUninstall")}
                            </Button>
                          )}
                          {!clawhubReady ? (
                            <Button variant="outline" size="compact" onClick={openExternalToolsSettings}>
                              {t("settings.calls.openExternalTools")}
                            </Button>
                          ) : !remoteInstallConfigured && !selectedSearchItemInstalled ? (
                            <div className="text-xs text-muted-foreground">
                              {t("settings.calls.skillsRemoteInstallDisabledHint")}
                            </div>
                          ) : null}
                        </div>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>

            {selectedSearchSkillDetail.isError ? (
              <div className="rounded-md border border-dashed border-amber-500/40 bg-amber-500/5 p-2 text-[11px] text-muted-foreground">
                {t("settings.calls.skillsDetailPartial")}
              </div>
            ) : null}

            <Card className={SKILLS_PANEL_CARD_CLASS}>
              <CardHeader size="compact" className="pb-2">
                <div className="flex items-center gap-1.5">
                  <CardTitle className="text-sm">
                    {t("settings.calls.skillsDetailRuntime")}
                  </CardTitle>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                        aria-label={t("settings.calls.skillsDetailRuntimeHintLabel")}
                      >
                        <HelpCircle className="h-3.5 w-3.5" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="top" className="max-w-80 text-xs">
                      {t("settings.calls.skillsDetailRuntimeHint")}
                    </TooltipContent>
                  </Tooltip>
                </div>
              </CardHeader>
              <CardContent size="compact" className="space-y-2">
                {selectedSearchDetailLoading ? (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-3 w-20" />
                      <Skeleton className="h-5 w-24 rounded-full" />
                      <Skeleton className="h-5 w-16 rounded-full" />
                    </div>
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-3 w-24" />
                      <Skeleton className="h-5 w-20 rounded-full" />
                    </div>
                    <Skeleton className="h-10 w-full rounded-md" />
                  </div>
                ) : null}
                {!selectedSearchDetailLoading && !selectedSearchRuntime ? (
                  <div className="px-2 py-1 text-xs text-muted-foreground">
                    {t("settings.calls.skillsDetailRuntimeEmpty")}
                  </div>
                ) : null
                }
                {selectedSearchRuntime ? (
                  <div className="space-y-2">
                      {selectedSearchRuntime.primaryEnv ? (
                        <div className="flex flex-wrap items-center gap-2 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimePrimaryEnv")}</span>
                          <Badge variant="secondary">{selectedSearchRuntime.primaryEnv}</Badge>
                        </div>
                      ) : null}
                      {selectedSearchRuntime.os && selectedSearchRuntime.os.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimeOS")}</span>
                          {selectedSearchRuntime.os.map((item) => (
                            <Badge key={`os:${item}`} variant="outline">
                              {item}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.bins && selectedSearchRuntime.bins.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimeBins")}</span>
                          {selectedSearchRuntime.bins.map((item) => (
                            <Badge key={`bin:${item}`} variant="outline">
                              {item}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.anyBins && selectedSearchRuntime.anyBins.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimeAnyBins")}</span>
                          {selectedSearchRuntime.anyBins.map((item) => (
                            <Badge key={`any-bin:${item}`} variant="outline">
                              {item}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.env && selectedSearchRuntime.env.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimeEnv")}</span>
                          {selectedSearchRuntime.env.map((item) => (
                            <Badge key={`env:${item}`} variant="outline">
                              {item}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.config && selectedSearchRuntime.config.length > 0 ? (
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          <span className="text-muted-foreground">{t("settings.calls.skillsRuntimeConfig")}</span>
                          {selectedSearchRuntime.config.map((item) => (
                            <Badge key={`config:${item}`} variant="outline">
                              {item}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.install && selectedSearchRuntime.install.length > 0 ? (
                        <div className="space-y-1 text-xs">
                          <div className="text-muted-foreground">{t("settings.calls.skillsRuntimeInstall")}</div>
                          {selectedSearchRuntime.install.map((spec, index) => (
                            <div
                              key={`install:${spec.kind ?? "unknown"}:${index}`}
                              className="rounded-md border bg-muted/20 px-2 py-1.5 text-muted-foreground"
                            >
                              {formatRuntimeInstallSpec(spec)}
                            </div>
                          ))}
                        </div>
                      ) : null}
                      {selectedSearchRuntime.nix ? (
                        <div className="space-y-1 text-xs">
                          <div className="text-muted-foreground">{t("settings.calls.skillsRuntimeNix")}</div>
                          <pre className="max-h-36 overflow-auto whitespace-pre-wrap rounded-md border bg-muted/20 p-2 text-[11px] text-muted-foreground">
                            {selectedSearchRuntime.nix}
                          </pre>
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </CardContent>
              </Card>

              <Card className={SKILLS_PANEL_CARD_CLASS}>
                <CardContent size="compact" className="space-y-3">
                  <Tabs
                    value={skillDetailContentTab}
                    onValueChange={(value) => setSkillDetailContentTab(value as SkillDetailContentTab)}
                    className="space-y-3"
                  >
                    <TabsList>
                      <TabsTrigger value="skill_md">
                        {t("settings.calls.skillsDetailMarkdown")}
                      </TabsTrigger>
                      <TabsTrigger value="files">
                        {t("settings.calls.skillsDetailFileList")}
                      </TabsTrigger>
                    </TabsList>
                    <Separator />
                    <TabsContent value="skill_md" className="mt-0 space-y-2">
                      {selectedSearchDetailLoading ? (
                        <div className="space-y-2">
                          <Skeleton className="h-4 w-1/3" />
                          <Skeleton className="h-4 w-11/12" />
                          <Skeleton className="h-4 w-9/12" />
                          <Skeleton className="h-20 w-full rounded-md" />
                        </div>
                      ) : selectedSearchSkillMarkdown ? (
                        <ReactMarkdown remarkPlugins={[remarkGfm]} components={skillMarkdownComponents}>
                          {selectedSearchSkillMarkdown}
                        </ReactMarkdown>
                      ) : (
                        <div className="text-xs text-muted-foreground">
                          {t("settings.calls.skillsDetailMarkdownEmpty")}
                        </div>
                      )}
                    </TabsContent>
                    <TabsContent value="files" className="mt-0 space-y-2">
                      {selectedSearchDetailLoading ? (
                        <div className="space-y-1.5">
                          <Skeleton className="h-14 w-full rounded-md" />
                          <Skeleton className="h-14 w-full rounded-md" />
                          <Skeleton className="h-14 w-full rounded-md" />
                        </div>
                      ) : selectedSearchSkillFiles.length > 0 ? (
                        <div className="space-y-1.5">
                          {selectedSearchSkillFiles.map((file) => (
                            <div key={file.path} className="rounded-md border bg-muted/20 px-3 py-2">
                              <div className="truncate text-xs font-medium text-foreground">{file.path}</div>
                              <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
                                {file.size ? <Badge variant="outline">{formatByteSize(file.size)}</Badge> : null}
                                {file.contentType ? <Badge variant="outline">{file.contentType}</Badge> : null}
                                {file.sha256 ? (
                                  <span className="truncate font-mono text-[10px]">{file.sha256}</span>
                                ) : null}
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="text-xs text-muted-foreground">
                          {t("settings.calls.skillsDetailFileListEmpty")}
                        </div>
                      )}
                    </TabsContent>
                  </Tabs>
                </CardContent>
              </Card>
            </div>
          )
        }
      />
  );
}
