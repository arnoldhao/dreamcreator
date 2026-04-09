import * as React from "react";
import {
  ArrowDown,
  ArrowUp,
  FolderOpen,
  FolderSearch,
  FolderTree,
  Globe,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Trash2,
} from "lucide-react";

import { canAdoptIncomingDraftSnapshot, shouldSkipDraftPersist } from "@/app/settings/draftSync";
import { cn } from "@/lib/utils";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty";
import { Skeleton } from "@/shared/ui/skeleton";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useSelectDirectory, useSettings, useUpdateSettings } from "@/shared/query/settings";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";

import { isRecord } from "../utils/calls-utils";
import {
  buildLocalExtraSource,
  buildNextSettingsTools,
  defaultBuiltinLocalSourceName,
  defaultRemoteSourceName,
  normalizeLocalSources,
  normalizeRemoteSources,
  parseSkillsSecurity,
  parseSkillsSourcesState,
  resolveSkillsSettingsState,
  toSourcesPayload,
  type LocalSkillSourceItem,
  type RemoteSkillSourceItem,
  type SkillSourceTrust,
} from "../utils/skills-settings-utils";
import {
  SKILLS_LINE_TABS_LIST_CLASS,
  SKILLS_LINE_TABS_TRIGGER_CLASS,
  SKILLS_PAGE_CARD_CLASS,
  SKILLS_SELECT_TEXT_CLASS,
} from "./skills-page-styles";

type SourcesTabValue = "local" | "remote";

function cloneLocalSources(items: LocalSkillSourceItem[]): LocalSkillSourceItem[] {
  return items.map((item) => ({ ...item }));
}

function cloneRemoteSources(items: RemoteSkillSourceItem[]): RemoteSkillSourceItem[] {
  return items.map((item) => ({ ...item }));
}

function trustBadgeVariant(trust: SkillSourceTrust): "default" | "secondary" | "outline" {
  if (trust === "trusted") {
    return "default";
  }
  if (trust === "community") {
    return "secondary";
  }
  return "outline";
}

function resolveTrustLabel(
  t: (key: string) => string,
  trust: SkillSourceTrust
): string {
  return t(`settings.calls.skills.sources.trust.${trust}`);
}

function resolveBuiltinPathHint(
  t: (key: string) => string,
  type: LocalSkillSourceItem["type"]
): string {
  return t(`settings.calls.skills.sources.pathHint.${type}`);
}

function resolveLocalSourcePathValue(
  t: (key: string) => string,
  item: LocalSkillSourceItem
): string {
  const path = item.path.trim();
  if (path) {
    return path;
  }
  if (item.type === "extra") {
    return t("settings.calls.skills.sources.pathPlaceholder");
  }
  return resolveBuiltinPathHint(t, item.type);
}

function resolveLocalSourceDisplayName(item: LocalSkillSourceItem): string {
  return item.name.trim() || defaultBuiltinLocalSourceName(item.type);
}

function resolveRemoteSourceDisplayName(
  _t: (key: string) => string,
  item: RemoteSkillSourceItem
): string {
  const defaultName = defaultRemoteSourceName(item.provider);
  if (!item.name.trim() || item.name.trim() === defaultName) {
    return defaultName;
  }
  return item.name.trim() || defaultName;
}

export function SkillsSourcesTab() {
  const { t } = useI18n();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();
  const selectDirectory = useSelectDirectory();

  const settingsToolsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.tools;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.tools]);
  const settingsSkillsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.skills;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.skills]);
  const { toolsConfig, skillsConfig } = React.useMemo(
    () => resolveSkillsSettingsState(settingsToolsRaw, settingsSkillsRaw),
    [settingsSkillsRaw, settingsToolsRaw]
  );
  const sourcesState = React.useMemo(() => parseSkillsSourcesState(skillsConfig), [skillsConfig]);
  const security = React.useMemo(() => parseSkillsSecurity(skillsConfig), [skillsConfig]);

  const [tab, setTab] = React.useState<SourcesTabValue>("local");
  const [draftLocalSources, setDraftLocalSources] = React.useState<LocalSkillSourceItem[]>([]);
  const [draftRemoteSources, setDraftRemoteSources] = React.useState<RemoteSkillSourceItem[]>([]);
  const [expandedLocalSourceIds, setExpandedLocalSourceIds] = React.useState<string[]>([]);
  const [expandedRemoteSourceIds, setExpandedRemoteSourceIds] = React.useState<string[]>([]);
  const [editingLocalNameId, setEditingLocalNameId] = React.useState<string | null>(null);
  const lastPersistedDraftSignatureRef = React.useRef("");
  const pendingDraftSignatureRef = React.useRef("");

  const isSaving = updateSettings.isPending;
  const writeMode = security.actionModes.source_write ?? "ask";
  const blocked = writeMode === "deny";
  const normalizedCurrentLocal = React.useMemo(() => normalizeLocalSources(sourcesState.local), [sourcesState.local]);
  const normalizedDraftLocal = React.useMemo(() => normalizeLocalSources(draftLocalSources), [draftLocalSources]);
  const normalizedCurrentRemote = React.useMemo(() => normalizeRemoteSources(sourcesState.remote), [sourcesState.remote]);
  const normalizedDraftRemote = React.useMemo(() => normalizeRemoteSources(draftRemoteSources), [draftRemoteSources]);
  const currentSourcesSignature = React.useMemo(
    () => JSON.stringify({ local: normalizedCurrentLocal, remote: normalizedCurrentRemote }),
    [normalizedCurrentLocal, normalizedCurrentRemote]
  );
  const draftSourcesSignature = React.useMemo(
    () => JSON.stringify({ local: normalizedDraftLocal, remote: normalizedDraftRemote }),
    [normalizedDraftLocal, normalizedDraftRemote]
  );
  const previousSourcesSignatureRef = React.useRef(currentSourcesSignature);

  React.useEffect(() => {
    const previousSourcesSignature = previousSourcesSignatureRef.current;
    previousSourcesSignatureRef.current = currentSourcesSignature;
    const canAdoptRemote = canAdoptIncomingDraftSnapshot({
      draftSignature: draftSourcesSignature,
      currentRemoteSignature: currentSourcesSignature,
      previousRemoteSignature: previousSourcesSignature,
      lastPersistedSignature: lastPersistedDraftSignatureRef.current,
    });
    if (!canAdoptRemote) {
      return;
    }
    setDraftLocalSources(cloneLocalSources(sourcesState.local));
    setDraftRemoteSources(cloneRemoteSources(sourcesState.remote));
  }, [currentSourcesSignature, draftSourcesSignature, sourcesState.local, sourcesState.remote]);

  const dirty = React.useMemo(() => {
    return currentSourcesSignature !== draftSourcesSignature;
  }, [currentSourcesSignature, draftSourcesSignature]);

  const persistSources = React.useCallback(() => {
    if (blocked) {
      return;
    }
    const submittedSignature = draftSourcesSignature;
    pendingDraftSignatureRef.current = submittedSignature;
    const nextSkillsConfig: Record<string, unknown> = {
      ...skillsConfig,
      sources: toSourcesPayload({
        local: draftLocalSources,
        remote: draftRemoteSources,
      }),
    };
    const nextToolsConfig: Record<string, unknown> = {
      ...toolsConfig,
    };
    const nextTools = buildNextSettingsTools(nextToolsConfig);
    updateSettings.mutate(
      {
        tools: nextTools,
        skills: nextSkillsConfig,
      },
      {
        onSuccess: () => {
          lastPersistedDraftSignatureRef.current = submittedSignature;
          if (pendingDraftSignatureRef.current === submittedSignature) {
            pendingDraftSignatureRef.current = "";
          }
        },
        onError: (error) => {
          if (pendingDraftSignatureRef.current === submittedSignature) {
            pendingDraftSignatureRef.current = "";
          }
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.calls.skills.sources.saveFailed"),
            description: error instanceof Error ? error.message : String(error ?? ""),
          });
        },
      }
    );
  }, [blocked, draftLocalSources, draftRemoteSources, skillsConfig, t, toolsConfig, updateSettings]);

  React.useEffect(() => {
    if (blocked || !dirty) {
      return;
    }
    if (
      shouldSkipDraftPersist({
        draftSignature: draftSourcesSignature,
        lastPersistedSignature: lastPersistedDraftSignatureRef.current,
        pendingSubmittedSignature: pendingDraftSignatureRef.current,
      })
    ) {
      return;
    }
    const timer = window.setTimeout(() => {
      if (updateSettings.isPending) {
        return;
      }
      persistSources();
    }, 500);
    return () => window.clearTimeout(timer);
  }, [blocked, dirty, draftSourcesSignature, persistSources, updateSettings.isPending]);

  React.useEffect(() => {
    if (!dirty && currentSourcesSignature === draftSourcesSignature) {
      lastPersistedDraftSignatureRef.current = draftSourcesSignature;
      if (pendingDraftSignatureRef.current === draftSourcesSignature) {
        pendingDraftSignatureRef.current = "";
      }
    }
  }, [currentSourcesSignature, dirty, draftSourcesSignature]);

  React.useEffect(() => {
    setExpandedLocalSourceIds((previous) => previous.filter((id) => draftLocalSources.some((item) => item.id === id)));
  }, [draftLocalSources]);

  React.useEffect(() => {
    setExpandedRemoteSourceIds((previous) => previous.filter((id) => draftRemoteSources.some((item) => item.id === id)));
  }, [draftRemoteSources]);

  React.useEffect(() => {
    if (!editingLocalNameId) {
      return;
    }
    if (draftLocalSources.some((item) => item.id === editingLocalNameId)) {
      return;
    }
    setEditingLocalNameId(null);
  }, [draftLocalSources, editingLocalNameId]);

  const handleRefresh = React.useCallback(() => {
    void settingsQuery.refetch();
  }, [settingsQuery]);

  const handleAddLocalSource = React.useCallback(async () => {
    if (blocked || isSaving) {
      return;
    }
    try {
      const selected = (
        await selectDirectory.mutateAsync({
          title: t("settings.calls.skills.sources.selectDirectory"),
        })
      ).trim();
      if (!selected) {
        return;
      }
      const nextItem = buildLocalExtraSource({ path: selected, priority: draftLocalSources.length });
      setDraftLocalSources((previous) => normalizeLocalSources([
        ...cloneLocalSources(previous),
        nextItem,
      ]));
      setExpandedLocalSourceIds((previous) => (previous.includes(nextItem.id) ? previous : [...previous, nextItem.id]));
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.calls.skills.sources.selectFailed"),
        description: error instanceof Error ? error.message : String(error ?? ""),
      });
    }
  }, [blocked, draftLocalSources.length, isSaving, selectDirectory, t]);

  const handlePickLocalSourceDirectory = React.useCallback(async (index: number) => {
    try {
      const current = draftLocalSources[index];
      if (!current || current.builtin) {
        return;
      }
      const selected = (
        await selectDirectory.mutateAsync({
          title: t("settings.calls.skills.sources.selectDirectory"),
          initialDir: current?.path?.trim() ?? "",
        })
      ).trim();
      if (!selected) {
        return;
      }
      setDraftLocalSources((previous) => {
        if (!previous[index]) {
          return previous;
        }
        const next = cloneLocalSources(previous);
        next[index] = {
          ...next[index],
          path: selected,
          name: next[index].builtin ? next[index].name : next[index].name || "",
        };
        return next;
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.calls.skills.sources.selectFailed"),
        description: error instanceof Error ? error.message : String(error ?? ""),
      });
    }
  }, [draftLocalSources, selectDirectory, t]);

  const patchLocalSource = React.useCallback((index: number, patch: Partial<LocalSkillSourceItem>) => {
    setDraftLocalSources((previous) => {
      if (!previous[index] || previous[index].builtin) {
        return previous;
      }
      const next = cloneLocalSources(previous);
      next[index] = {
        ...next[index],
        ...patch,
      };
      return next;
    });
  }, []);

  const moveLocalSource = React.useCallback((index: number, delta: number) => {
    setDraftLocalSources((previous) => {
      const target = index + delta;
      if (target < 0 || target >= previous.length) {
        return previous;
      }
      if (previous[index]?.builtin || previous[target]?.builtin) {
        return previous;
      }
      const next = cloneLocalSources(previous);
      const [item] = next.splice(index, 1);
      next.splice(target, 0, item);
      return normalizeLocalSources(
        next.map((entry, entryIndex) => ({
          ...entry,
          priority: entryIndex,
        }))
      );
    });
  }, []);

  const removeLocalSource = React.useCallback((index: number) => {
    setDraftLocalSources((previous) => {
      if (!previous[index] || previous[index].builtin) {
        return previous;
      }
      const next = cloneLocalSources(previous);
      next.splice(index, 1);
      return normalizeLocalSources(next);
    });
  }, []);

  const patchRemoteSource = React.useCallback((index: number, patch: Partial<RemoteSkillSourceItem>) => {
    setDraftRemoteSources((previous) => {
      if (!previous[index] || previous[index].builtin) {
        return previous;
      }
      const next = cloneRemoteSources(previous);
      next[index] = {
        ...next[index],
        ...patch,
      };
      return next;
    });
  }, []);

  const toggleLocalSourceExpanded = React.useCallback((sourceId: string) => {
    setExpandedLocalSourceIds((previous) => (
      previous.includes(sourceId)
        ? previous.filter((id) => id !== sourceId)
        : [...previous, sourceId]
    ));
    if (editingLocalNameId === sourceId) {
      setEditingLocalNameId(null);
    }
  }, [editingLocalNameId]);

  const toggleRemoteSourceExpanded = React.useCallback((sourceId: string) => {
    setExpandedRemoteSourceIds((previous) => (
      previous.includes(sourceId)
        ? previous.filter((id) => id !== sourceId)
        : [...previous, sourceId]
    ));
  }, []);

  const stopInteraction = React.useCallback((event: React.SyntheticEvent) => {
    event.stopPropagation();
  }, []);

  const handleLocalNameEditStart = React.useCallback((sourceId: string) => {
    setEditingLocalNameId(sourceId);
  }, []);

  const handleLocalNameEditEnd = React.useCallback(() => {
    setEditingLocalNameId(null);
    if (blocked || updateSettings.isPending) {
      return;
    }
    persistSources();
  }, [blocked, persistSources, updateSettings.isPending]);

  const handleConfirmRemoveLocalSource = React.useCallback((item: LocalSkillSourceItem, index: number) => {
    if (item.builtin || blocked || isSaving) {
      return;
    }
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.calls.action.removeTitle"),
      description: t("settings.calls.action.removeDescription")
        .replace("{name}", resolveLocalSourceDisplayName(item)),
      confirmLabel: t("common.delete"),
      cancelLabel: t("common.cancel"),
      onConfirm: () => removeLocalSource(index),
    });
  }, [blocked, isSaving, removeLocalSource, t]);

  if (settingsQuery.isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-52 w-full" />
      </div>
    );
  }

  if (settingsQuery.isError) {
    return (
      <Card className={SKILLS_PAGE_CARD_CLASS}>
        <CardContent size="compact" className="flex items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            {t("settings.calls.skills.sources.loadFailed")}
          </p>
          <Button variant="outline" size="compact" onClick={() => void settingsQuery.refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            {t("common.retry")}
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col space-y-3">
      {blocked ? (
        <Card className="rounded-xl border-amber-500/40 bg-amber-50/40 dark:bg-amber-950/20">
          <CardContent size="compact" className="text-xs text-muted-foreground">
            {t("settings.calls.skills.sources.blocked")}
          </CardContent>
        </Card>
      ) : null}

      <Tabs
        value={tab}
        onValueChange={(value) => setTab(value as SourcesTabValue)}
        className="flex min-h-0 flex-1 flex-col space-y-3"
      >
        <div className="flex items-center gap-3">
          <TabsList className={`shrink-0 ${SKILLS_LINE_TABS_LIST_CLASS}`}>
            <TabsTrigger value="local" className={SKILLS_LINE_TABS_TRIGGER_CLASS}>
              <FolderTree className="h-4 w-4" />
              <span className="truncate">{t("settings.calls.skills.sources.tabs.local")}</span>
            </TabsTrigger>
            <TabsTrigger value="remote" className={SKILLS_LINE_TABS_TRIGGER_CLASS}>
              <Globe className="h-4 w-4" />
              <span className="truncate">{t("settings.calls.skills.sources.tabs.remote")}</span>
            </TabsTrigger>
          </TabsList>

          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="outline"
              size="compactIcon"
              className="shrink-0"
              aria-label={t("settings.calls.skills.sources.add")}
              title={t("settings.calls.skills.sources.add")}
              disabled={blocked || isSaving || tab !== "local"}
              onClick={() => void handleAddLocalSource()}
            >
              <Plus className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="compactIcon"
              className="shrink-0"
              aria-label={t("common.refresh")}
              title={t("common.refresh")}
              disabled={isSaving}
              onClick={handleRefresh}
            >
              {isSaving ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            </Button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          {tab === "local" ? (
            draftLocalSources.length === 0 ? (
              <Card className={SKILLS_PAGE_CARD_CLASS}>
                <CardContent size="compact" className="py-10">
                  <Empty>
                    <EmptyHeader>
                      <EmptyMedia>
                        <FolderSearch className="h-5 w-5 text-muted-foreground" />
                      </EmptyMedia>
                      <EmptyTitle>{t("settings.calls.skills.sources.emptyTitle")}</EmptyTitle>
                      <EmptyDescription>
                        {t("settings.calls.skills.sources.emptyDescription")}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-3">
                {draftLocalSources.map((item, index) => {
                  const displayName = resolveLocalSourceDisplayName(item);
                  const indexedDisplayName = `#${index + 1} ${displayName}`;
                  const pathValue = resolveLocalSourcePathValue(t, item);
                  const expanded = expandedLocalSourceIds.includes(item.id);
                  return (
                    <Card
                      key={`${item.id}-${index}`}
                      className={cn(
                        SKILLS_PAGE_CARD_CLASS,
                        "transition-colors",
                        expanded ? "border-primary/40 bg-background/80" : "hover:border-border"
                      )}
                    >
                      <CardContent size="compact" className="p-0">
                        <div
                          role="button"
                          tabIndex={0}
                          aria-expanded={expanded}
                          className="flex flex-col p-4 text-left outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
                          onClick={() => toggleLocalSourceExpanded(item.id)}
                          onKeyDown={(event) => {
                            if (event.key !== "Enter" && event.key !== " ") {
                              return;
                            }
                            event.preventDefault();
                            toggleLocalSourceExpanded(item.id);
                          }}
                        >
                          <div className="flex items-center gap-3">
                            <div className="min-w-0 flex-1 space-y-2">
                              <div className="flex min-w-0 flex-wrap items-center gap-2">
                                <div className="truncate text-sm font-semibold text-foreground">{indexedDisplayName}</div>
                                <Badge variant={trustBadgeVariant(item.trust)}>{resolveTrustLabel(t, item.trust)}</Badge>
                              </div>
                              <div
                                className="truncate font-mono text-[11px] text-muted-foreground/60"
                                title={pathValue}
                              >
                                {pathValue}
                              </div>
                            </div>

                            <div
                              className="flex shrink-0 items-center"
                              onClick={stopInteraction}
                              onKeyDown={stopInteraction}
                            >
                              <div className="flex items-center gap-1">
                                <Switch
                                  checked={item.enabled}
                                  onCheckedChange={(checked) => patchLocalSource(index, { enabled: checked === true })}
                                  disabled={blocked || isSaving || item.builtin}
                                />
                                <Button
                                  variant="outline"
                                  size="compactIcon"
                                  className="h-8 w-8"
                                  onClick={() => moveLocalSource(index, -1)}
                                  disabled={blocked || isSaving || item.builtin || index === 0 || draftLocalSources[index - 1]?.builtin}
                                >
                                  <ArrowUp className="h-3.5 w-3.5" />
                                </Button>
                                <Button
                                  variant="outline"
                                  size="compactIcon"
                                  className="h-8 w-8"
                                  onClick={() => moveLocalSource(index, 1)}
                                  disabled={
                                    blocked
                                    || isSaving
                                    || item.builtin
                                    || index >= draftLocalSources.length - 1
                                    || draftLocalSources[index + 1]?.builtin
                                  }
                                >
                                  <ArrowDown className="h-3.5 w-3.5" />
                                </Button>
                                <Button
                                  variant="outline"
                                  size="compactIcon"
                                  className="h-8 w-8"
                                  onClick={() => handleConfirmRemoveLocalSource(item, index)}
                                  disabled={blocked || isSaving || item.builtin}
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                              </div>
                            </div>
                          </div>

                          {expanded ? (
                            <div
                              className="mt-3 grid gap-3 border-t border-border/70 pt-3 md:grid-cols-[minmax(0,2fr)_minmax(0,2fr)_minmax(0,1fr)]"
                              onClick={stopInteraction}
                              onKeyDown={stopInteraction}
                            >
                              <div className="space-y-1.5 text-xs">
                                <span className="text-muted-foreground">
                                  {t("settings.calls.skills.sources.fields.name")}
                                </span>
                                {editingLocalNameId === item.id ? (
                                  <Input
                                    value={item.name}
                                    onChange={(event) => patchLocalSource(index, { name: event.target.value })}
                                    onBlur={handleLocalNameEditEnd}
                                    onKeyDown={(event) => {
                                      if (event.key === "Enter") {
                                        event.preventDefault();
                                        event.currentTarget.blur();
                                      }
                                    }}
                                    disabled={blocked || isSaving}
                                    size="compact"
                                    autoFocus
                                  />
                                ) : (
                                  <div className="app-control-compact flex min-w-0 items-center px-0">
                                    <div className="inline-flex min-w-0 max-w-full items-center gap-1">
                                      <span
                                        className="min-w-0 truncate leading-tight text-foreground"
                                        title={indexedDisplayName}
                                      >
                                        {indexedDisplayName}
                                      </span>
                                      {!item.builtin ? (
                                        <TooltipProvider delayDuration={0}>
                                          <Tooltip>
                                            <TooltipTrigger asChild>
                                              <Button
                                                type="button"
                                                variant="ghost"
                                                size="compactIcon"
                                                className="h-7 w-7 shrink-0"
                                                disabled={blocked || isSaving}
                                                onClick={() => handleLocalNameEditStart(item.id)}
                                                aria-label={t("common.edit")}
                                              >
                                                <Pencil className="h-4 w-4" />
                                              </Button>
                                            </TooltipTrigger>
                                            <TooltipContent side="top">
                                              {t("common.edit")}
                                            </TooltipContent>
                                          </Tooltip>
                                        </TooltipProvider>
                                      ) : null}
                                    </div>
                                  </div>
                                )}
                              </div>

                              <div className="space-y-1.5 text-xs">
                                <span className="text-muted-foreground">
                                  {t("settings.calls.skills.sources.fields.path")}
                                </span>
                                <div className="app-control-compact flex min-w-0 items-center px-0">
                                  <div className="inline-flex min-w-0 max-w-full items-center gap-1">
                                    <span
                                      className="min-w-0 truncate font-mono leading-tight text-muted-foreground"
                                      title={pathValue}
                                    >
                                      {pathValue}
                                    </span>
                                    {!item.builtin ? (
                                      <TooltipProvider delayDuration={0}>
                                        <Tooltip>
                                          <TooltipTrigger asChild>
                                            <Button
                                              type="button"
                                              variant="outline"
                                              size="compactIcon"
                                              className="h-7 w-7 shrink-0"
                                              disabled={blocked || isSaving}
                                              onClick={() => void handlePickLocalSourceDirectory(index)}
                                              aria-label={t("settings.calls.skills.sources.pickDirectory")}
                                            >
                                              <FolderOpen className="h-4 w-4" />
                                            </Button>
                                          </TooltipTrigger>
                                          <TooltipContent side="top">
                                            {t("settings.calls.skills.sources.pickDirectory")}
                                          </TooltipContent>
                                        </Tooltip>
                                      </TooltipProvider>
                                    ) : null}
                                  </div>
                                </div>
                              </div>

                              <label className="space-y-1.5 text-xs">
                                <span className="text-muted-foreground">
                                  {t("settings.calls.skills.sources.fields.trust")}
                                </span>
                                <Select
                                  value={item.trust}
                                  onChange={(event) => patchLocalSource(index, { trust: event.target.value as SkillSourceTrust })}
                                  disabled={blocked || isSaving || item.builtin}
                                  className={SKILLS_SELECT_TEXT_CLASS}
                                >
                                  <option value="local">{t("settings.calls.skills.sources.trust.local")}</option>
                                  <option value="trusted">{t("settings.calls.skills.sources.trust.trusted")}</option>
                                  <option value="community">{t("settings.calls.skills.sources.trust.community")}</option>
                                </Select>
                              </label>
                            </div>
                          ) : null}
                        </div>
                      </CardContent>
                    </Card>
                  );
                })}
              </div>
            )
          ) : (
            <div className="space-y-3">
              {draftRemoteSources.map((item, index) => {
                const displayName = resolveRemoteSourceDisplayName(t, item);
                const indexedDisplayName = `#${index + 1} ${displayName}`;
                const expanded = expandedRemoteSourceIds.includes(item.id);
                return (
                  <Card
                    key={`${item.id}-${index}`}
                    className={cn(
                      SKILLS_PAGE_CARD_CLASS,
                      "transition-colors",
                      expanded ? "border-primary/40 bg-background/80" : "hover:border-border"
                    )}
                  >
                    <CardContent size="compact" className="p-0">
                      <div
                        role="button"
                        tabIndex={0}
                        aria-expanded={expanded}
                        className="flex flex-col p-4 text-left outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
                        onClick={() => toggleRemoteSourceExpanded(item.id)}
                        onKeyDown={(event) => {
                          if (event.key !== "Enter" && event.key !== " ") {
                            return;
                          }
                          event.preventDefault();
                          toggleRemoteSourceExpanded(item.id);
                        }}
                      >
                        <div className="flex items-center gap-3">
                          <div className="min-w-0 flex-1 space-y-2">
                            <div className="flex min-w-0 flex-wrap items-center gap-2">
                              <div className="truncate text-sm font-semibold text-foreground">{indexedDisplayName}</div>
                              {item.builtin ? (
                                <Badge variant="secondary">
                                  {t("settings.calls.skills.sources.builtin")}
                                </Badge>
                              ) : null}
                            </div>
                            <div
                              className="truncate font-mono text-[11px] text-muted-foreground/60"
                              title={item.provider}
                            >
                              {item.provider}
                            </div>
                          </div>

                          <div
                            className="flex shrink-0 items-center"
                            onClick={stopInteraction}
                            onKeyDown={stopInteraction}
                          >
                            <Switch
                              checked={item.enabled}
                              onCheckedChange={(checked) => patchRemoteSource(index, { enabled: checked === true })}
                              disabled={blocked || isSaving || item.builtin}
                            />
                          </div>
                        </div>

                        {expanded ? (
                          <div
                            className="mt-3 grid gap-3 border-t border-border/70 pt-3 md:grid-cols-4"
                            onClick={stopInteraction}
                            onKeyDown={stopInteraction}
                          >
                            <div className="space-y-1.5 text-xs">
                              <span className="text-muted-foreground">
                                {t("settings.calls.skills.sources.fields.name")}
                              </span>
                              <div className="app-control-compact flex min-w-0 items-center px-0">
                                <span
                                  className="min-w-0 truncate leading-tight text-foreground"
                                  title={indexedDisplayName}
                                >
                                  {indexedDisplayName}
                                </span>
                              </div>
                            </div>

                            <div className="space-y-1.5 text-xs">
                              <span className="text-muted-foreground">
                                {t("settings.calls.skills.sources.remote.provider")}
                              </span>
                              <div className="app-control-compact flex min-w-0 items-center px-0">
                                <span
                                  className="min-w-0 truncate font-mono leading-tight text-muted-foreground"
                                  title={item.provider}
                                >
                                  {item.provider}
                                </span>
                              </div>
                            </div>

                            <div className="space-y-1.5 text-xs">
                              <span className="text-muted-foreground">
                                {t("settings.calls.skills.sources.remote.searchEnabled")}
                              </span>
                              <div className="app-control-compact flex items-center px-0">
                                <Switch
                                  checked={item.searchEnabled}
                                  onCheckedChange={(checked) => patchRemoteSource(index, { searchEnabled: checked === true })}
                                  disabled={blocked || isSaving || item.builtin || !item.enabled}
                                />
                              </div>
                            </div>

                            <div className="space-y-1.5 text-xs">
                              <span className="text-muted-foreground">
                                {t("settings.calls.skills.sources.remote.installEnabled")}
                              </span>
                              <div className="app-control-compact flex items-center px-0">
                                <Switch
                                  checked={item.installEnabled}
                                  onCheckedChange={(checked) => patchRemoteSource(index, { installEnabled: checked === true })}
                                  disabled={blocked || isSaving || item.builtin || !item.enabled}
                                />
                              </div>
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      </Tabs>
    </div>
  );
}
