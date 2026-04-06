import * as React from "react";
import {
  CircleOff,
  Loader2,
  RefreshCw,
  Search,
  Trash2,
} from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { SectionCard } from "@/shared/ui/SectionCard";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import {
  CHANNELS_QUERY_KEY,
  useChannelPairingApprove,
  useChannelPairingList,
  useChannelPairingReject,
  useChannels,
  useLogoutChannel,
  useProbeChannel,
  useSyncChannelMenu,
} from "@/shared/query/channels";
import { useConfigGet, useConfigPatch } from "@/shared/query/config";
import { gatewayToolsKey } from "@/shared/query/tools";
import type { ChannelOverview } from "@/shared/store/channels";
import { messageBus } from "@/shared/message";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import {
  FIELD_BOT_TOKEN,
  FIELD_DM_POLICY,
  FIELD_GROUP_POLICY,
  FIELD_STREAM_MODE_TELEGRAM,
  REQUIRED_FIELD_LABELS,
  STATUS_META,
  areCustomCommandsEqual,
  areGroupEntriesEqual,
  areListsEqual,
  formatGatewayError,
  normalizeTelegramCommandName,
  resolveAllowFromList,
  resolveChannelFields,
  resolveChannelStatus,
  resolveCommandSettingValue,
  resolveConfigEnabled,
  resolveCustomCommandList,
  resolveGroupAllowList,
  resolveMissingRequiredFields,
  resolveRecord,
  resolveTelegramBotToken,
  sanitizeAllowFromDraft,
  sanitizeCustomCommandDraft,
  sanitizeGroupAllowDraft,
  stableStringify,
  telegramCommandNamePattern,
  type ChannelField,
  type GroupAllowEntry,
  type TelegramCustomCommand,
} from "./channels-section.utils";
import {
  AllowFromTable,
  GroupAllowFromTable,
  GroupAllowListTable,
  PairingRequestsPanel,
} from "./components/TelegramConfigTables";
import { TelegramMenuConfigCard } from "./components/TelegramMenuConfigCard";

export function ChannelsSection() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const channels = useChannels();
  const probeChannel = useProbeChannel();
  const logoutChannel = useLogoutChannel();
  const syncMenu = useSyncChannelMenu();
  const pairingApprove = useChannelPairingApprove();
  const pairingReject = useChannelPairingReject();
  const configQuery = useConfigGet("/channels");
  const patchConfig = useConfigPatch();
  const [configDraft, setConfigDraft] = React.useState<Record<string, any>>({});
  const lastConfigSnapshotRef = React.useRef<string | null>(null);
  const [autoSaveStatus, setAutoSaveStatus] = React.useState<"idle" | "saving" | "error">("idle");
  const [autoSaveError, setAutoSaveError] = React.useState<string | null>(null);
  const [allowFromDraft, setAllowFromDraft] = React.useState<string[]>([]);
  const [groupAllowFromDraft, setGroupAllowFromDraft] = React.useState<string[]>([]);
  const [groupAllowListDraft, setGroupAllowListDraft] = React.useState<GroupAllowEntry[]>([]);
  const [customCommandsDraft, setCustomCommandsDraft] = React.useState<TelegramCustomCommand[]>([]);
  const [allowFromEditingIndex, setAllowFromEditingIndex] = React.useState<number | null>(null);
  const [groupAllowFromEditingIndex, setGroupAllowFromEditingIndex] = React.useState<number | null>(null);
  const [groupAllowListEditingIndex, setGroupAllowListEditingIndex] = React.useState<number | null>(
    null
  );
  const [customCommandsEditingIndex, setCustomCommandsEditingIndex] = React.useState<number | null>(null);
  const [customCommandsError, setCustomCommandsError] = React.useState<string | null>(null);
  const [pairingBusyCode, setPairingBusyCode] = React.useState<string | null>(null);

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [query, setQuery] = React.useState("");
  const saveTimeoutRef = React.useRef<number | null>(null);
  const saveInFlightRef = React.useRef(false);
  const saveQueuedRef = React.useRef<Record<string, any> | null>(null);
  const skipAutoSaveRef = React.useRef(false);
  const configVersionRef = React.useRef<number>(0);
  const customCommandsTouchedRef = React.useRef(false);
  const customCommandsSyncRef = React.useRef(false);

  const items = channels.data ?? [];
  const trimmedQuery = query.trim().toLowerCase();
  const filteredItems = trimmedQuery
    ? items.filter((item) => {
        const label = item.displayName || item.channelId;
        return label.toLowerCase().includes(trimmedQuery) || item.channelId.toLowerCase().includes(trimmedQuery);
      })
    : items;

  React.useEffect(() => {
    if (selectedId && !items.some((item) => item.channelId === selectedId)) {
      setSelectedId(null);
    }
  }, [items, selectedId]);

  React.useEffect(() => {
    if (!selectedId && filteredItems.length > 0) {
      setSelectedId(filteredItems[0].channelId);
    }
  }, [filteredItems, selectedId]);

  const selected = items.find((item) => item.channelId === selectedId) ?? null;
  const configSnapshot = resolveRecord(configQuery.data?.config);
  const selectedConfig = resolveRecord(
    selectedId ? configDraft[selectedId] ?? configSnapshot[selectedId] : null
  );
  const baselineSelectedConfig = resolveRecord(selectedId ? configSnapshot[selectedId] : null);
  const configFields = resolveChannelFields(selectedId);
  const normalizeChannelConfig = React.useCallback(
    (channelId: string | null, value: Record<string, any>) => {
      if (!channelId) {
        return {};
      }
      if (channelId !== "telegram") {
        return value;
      }
      const sanitized = { ...value };
      const commandsConfig = resolveRecord(sanitized.commands);
      const cleanedCommands: Record<string, any> = {};
      if (typeof commandsConfig.native === "boolean") {
        cleanedCommands.native = commandsConfig.native;
      }
      if (typeof commandsConfig.nativeSkills === "boolean") {
        cleanedCommands.nativeSkills = commandsConfig.nativeSkills;
      }
      if (Object.keys(cleanedCommands).length > 0) {
        sanitized.commands = cleanedCommands;
      } else {
        delete sanitized.commands;
      }
      const customCommands = sanitizeCustomCommandDraft(
        resolveCustomCommandList(sanitized.customCommands)
      );
      if (customCommands.length > 0) {
        sanitized.customCommands = customCommands;
      } else {
        delete sanitized.customCommands;
      }
      const allowFromList = resolveAllowFromList(sanitized.allowFrom);
      if (allowFromList.length > 0) {
        sanitized.allowFrom = allowFromList;
      } else {
        delete sanitized.allowFrom;
      }
      return sanitized;
    },
    []
  );
  const normalizedDraft = React.useMemo(
    () => normalizeChannelConfig(selectedId, selectedConfig),
    [normalizeChannelConfig, selectedConfig, selectedId]
  );
  const normalizedBaseline = React.useMemo(
    () => normalizeChannelConfig(selectedId, baselineSelectedConfig),
    [normalizeChannelConfig, baselineSelectedConfig, selectedId]
  );
  const normalizedDraftSnapshot = React.useMemo(
    () => stableStringify(normalizedDraft),
    [normalizedDraft]
  );
  const normalizedBaselineSnapshot = React.useMemo(
    () => stableStringify(normalizedBaseline),
    [normalizedBaseline]
  );
  const hasConfigChanges = selectedId ? normalizedDraftSnapshot !== normalizedBaselineSnapshot : false;
  const configReady = Boolean(configQuery.data);
  const configBusy = configQuery.isLoading || configQuery.isFetching;
  const configEnabled = resolveConfigEnabled(selected, selectedConfig);
  const missingRequiredFields = configReady
    ? resolveMissingRequiredFields(selectedId, selectedConfig)
    : [];
  const requiresConfig = configReady && configEnabled && missingRequiredFields.length > 0;
  const state = resolveChannelStatus(selectedId, selected, selectedConfig, configReady);
  const status = STATUS_META[state] ?? STATUS_META.unknown;
  const missingFieldLabels = missingRequiredFields.map((field) => {
    const entry = REQUIRED_FIELD_LABELS[field];
    return entry ? t(entry.labelKey) : field;
  });
  const isBusy = probeChannel.isPending || logoutChannel.isPending;
  const actionsDisabled = isBusy || !configEnabled || requiresConfig;
  const menuReady =
    selectedId === "telegram" &&
    configReady &&
    configEnabled &&
    resolveTelegramBotToken(selectedConfig) !== "";
  const dmPolicyValue =
    typeof selectedConfig.dmPolicy === "string" && selectedConfig.dmPolicy
      ? selectedConfig.dmPolicy
      : "pairing";
  const groupPolicyValue =
    typeof selectedConfig.groupPolicy === "string" && selectedConfig.groupPolicy
      ? selectedConfig.groupPolicy
      : "allowlist";
  const showAllowFrom = selectedId === "telegram";
  const showPairing = selectedId === "telegram" && dmPolicyValue === "pairing";
  const showGroupAllowList = selectedId === "telegram";
  const showGroupAllowFrom = selectedId === "telegram";
  const pairingListQuery = useChannelPairingList(
    { channelId: selectedId ?? "", accountId: selected?.accountId },
    showPairing && Boolean(selectedId)
  );
  const customCommandsList = sanitizeCustomCommandDraft(
    resolveCustomCommandList(selectedConfig.customCommands)
  );
  const allowFromList = resolveAllowFromList(selectedConfig.allowFrom);
  const groupAllowFromList = resolveAllowFromList(selectedConfig.groupAllowFrom);
  const groupAllowList = resolveGroupAllowList(selectedConfig.groups);
  const lastError = selected?.lastError?.trim();
  const updatedAt = selected?.updatedAt ? new Date(selected.updatedAt).toLocaleString() : "-";
  const autoSaveState =
    autoSaveStatus === "error"
      ? "error"
      : autoSaveStatus === "saving"
        ? "saving"
        : hasConfigChanges
          ? "pending"
          : "saved";
  const autoSaveLabel =
    autoSaveState === "error"
      ? t("settings.integration.channels.config.autoSave.error")
      : autoSaveState === "saving"
        ? t("settings.integration.channels.config.autoSave.saving")
        : t("settings.integration.channels.config.autoSave.pending");
  const showAutoSaveStatus = configReady && autoSaveState !== "saved";
  const showAutoSaveError = autoSaveState === "error" && autoSaveError;
  const getChannelConfig = React.useCallback(
    (channelId: string) => resolveRecord(configDraft[channelId] ?? configSnapshot[channelId]),
    [configDraft, configSnapshot]
  );

  const updateChannelOverview = React.useCallback(
    (channelId: string, patch: Partial<ChannelOverview>) => {
      queryClient.setQueryData(CHANNELS_QUERY_KEY, (current) => {
        if (!Array.isArray(current)) {
          return current;
        }
        return (current as ChannelOverview[]).map((item) =>
          item.channelId === channelId ? { ...item, ...patch } : item
        );
      });
    },
    [queryClient]
  );


  React.useEffect(() => {
    if (!configQuery.data) {
      return;
    }
    configVersionRef.current = configQuery.data.version ?? 0;
    const snapshot = resolveRecord(configQuery.data.config);
    const snapshotSerialized = stableStringify(snapshot);
    const draftSerialized = stableStringify(configDraft);
    const draftMatchesLastSnapshot =
      lastConfigSnapshotRef.current != null && draftSerialized === lastConfigSnapshotRef.current;
    if (!hasConfigChanges || draftMatchesLastSnapshot) {
      setConfigDraft(snapshot);
    }
    lastConfigSnapshotRef.current = snapshotSerialized;
  }, [configQuery.data?.version, configDraft, hasConfigChanges]);

  const handleProbe = async () => {
    if (!selected) {
      return;
    }
    try {
      const result = await probeChannel.mutateAsync({ channelId: selected.channelId });
      if (!result.success) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.integration.channels.actions.probe"),
          description: result.error || t("settings.integration.channels.probeError"),
        });
        return;
      }
      messageBus.publishToast({
        intent: "success",
        title: t("settings.integration.channels.actions.probe"),
        description: t("settings.integration.channels.probeSuccess"),
      });
      updateChannelOverview(selected.channelId, {
        lastError: undefined,
        updatedAt: result.checkedAt ?? new Date().toISOString(),
        ...(result.state ? { state: result.state } : {}),
      });
      void channels.refetch();
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.integration.channels.actions.probe"),
        description: formatGatewayError(error),
      });
    }
  };

  const handleLogout = async () => {
    if (!selected) {
      return;
    }
    try {
      const result = await logoutChannel.mutateAsync({ channelId: selected.channelId });
      if (!result.success) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.integration.channels.actions.logout"),
          description: result.error || t("settings.integration.channels.logoutError"),
        });
        return;
      }
      messageBus.publishToast({
        intent: "success",
        title: t("settings.integration.channels.actions.logout"),
        description: t("settings.integration.channels.logoutSuccess"),
      });
      updateChannelOverview(selected.channelId, {
        lastError: undefined,
        updatedAt: new Date().toISOString(),
      });
      void channels.refetch();
      void queryClient.invalidateQueries({ queryKey: ["config", "/channels"] });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.integration.channels.actions.logout"),
        description: formatGatewayError(error),
      });
    }
  };

  const handleMenuSync = async () => {
    if (!selected || selected.channelId !== "telegram") {
      return;
    }
    try {
      const result = await syncMenu.mutateAsync({ channelId: selected.channelId });
      if (!result.synced) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.integration.channels.menu.sync"),
          description: result.error || t("settings.integration.channels.menu.syncError"),
        });
        return;
      }
      const issueText = result.issues && result.issues.length > 0 ? result.issues.join("\n") : "";
      const syncTemplate = t("settings.integration.channels.menu.syncSuccess");
      messageBus.publishToast({
        intent: issueText || (result.overflowCount ?? 0) > 0 ? "warning" : "success",
        title: t("settings.integration.channels.menu.sync"),
        description:
          issueText ||
          syncTemplate.replace("{{count}}", String(result.commands ?? 0)),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.integration.channels.menu.sync"),
        description: formatGatewayError(error),
      });
    }
  };

  const formatPairingMeta = (meta?: Record<string, string>) => {
    if (!meta) {
      return "";
    }
    const name = [meta.firstName, meta.lastName].filter(Boolean).join(" ");
    const username = meta.username ? `@${meta.username}` : "";
    const account = meta.accountId ? `${t("settings.integration.channels.config.pairing.meta.account")}: ${meta.accountId}` : "";
    return [username, name, account].filter(Boolean).join(" / ");
  };

  const formatPairingTime = (value?: string) => {
    if (!value) {
      return "-";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
  };

  const handlePairingApprove = async (code: string) => {
    if (!selectedId) {
      return;
    }
    setPairingBusyCode(code);
    try {
      const result = await pairingApprove.mutateAsync({
        channelId: selectedId,
        accountId: selected?.accountId,
        code,
        notify: false,
      });
      if (!result.approved) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.integration.channels.config.pairing.actions.approve"),
          description:
            result.error ||
            t("settings.integration.channels.config.pairing.approveError"),
        });
        return;
      }
      messageBus.publishToast({
        intent: "success",
        title: t("settings.integration.channels.config.pairing.actions.approve"),
        description: t("settings.integration.channels.config.pairing.approveSuccess"),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.integration.channels.config.pairing.actions.approve"),
        description: formatGatewayError(error),
      });
    } finally {
      setPairingBusyCode(null);
    }
  };

  const handlePairingReject = async (code: string) => {
    if (!selectedId) {
      return;
    }
    setPairingBusyCode(code);
    try {
      const result = await pairingReject.mutateAsync({
        channelId: selectedId,
        accountId: selected?.accountId,
        code,
      });
      if (!result.rejected) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.integration.channels.config.pairing.actions.reject"),
          description:
            result.error ||
            t("settings.integration.channels.config.pairing.rejectError"),
        });
        return;
      }
      messageBus.publishToast({
        intent: "success",
        title: t("settings.integration.channels.config.pairing.actions.reject"),
        description: t("settings.integration.channels.config.pairing.rejectSuccess"),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.integration.channels.config.pairing.actions.reject"),
        description: formatGatewayError(error),
      });
    } finally {
      setPairingBusyCode(null);
    }
  };

  const handleReload = () => {
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
      saveTimeoutRef.current = null;
    }
    setAutoSaveStatus("idle");
    setAutoSaveError(null);
    const tasks: Array<Promise<unknown>> = [configQuery.refetch(), channels.refetch()];
    if (showPairing) {
      tasks.push(pairingListQuery.refetch());
    }
    void Promise.all(tasks);
  };

  const handleResetConfig = () => {
    if (!selectedId) {
      return;
    }
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
      saveTimeoutRef.current = null;
    }
    saveQueuedRef.current = null;
    messageBus.publishDialog({
      intent: "danger",
      title: t("settings.integration.channels.reset.title"),
      description: t("settings.integration.channels.reset.description"),
      confirmLabel: t("settings.integration.channels.reset.confirm"),
      cancelLabel: t("settings.integration.channels.reset.cancel"),
      onConfirm: async () => {
        try {
          const result = await patchConfig.mutateAsync({
            ops: [{ op: "remove", path: `/channels/${selectedId}` }],
            expectedVersion: configVersionRef.current,
            dryRun: false,
          });
          configVersionRef.current = result.version;
          const nextConfig = { ...configSnapshot };
          delete nextConfig[selectedId];
          queryClient.setQueryData(["config", "/channels"], {
            config: nextConfig,
            version: result.version,
          });
          setConfigDraft(nextConfig);
          setAutoSaveStatus("idle");
          setAutoSaveError(null);
          void queryClient.invalidateQueries({ queryKey: gatewayToolsKey });
        } catch (error) {
          messageBus.publishToast({
            intent: "danger",
            title: t("settings.integration.channels.reset.title"),
            description: formatGatewayError(error),
          });
        }
      },
    });
  };

  const setSelectedConfigValue = (key: string, value: unknown, dropEmpty = false) => {
    if (!selectedId) {
      return;
    }
    setConfigDraft((prev) => {
      const next = { ...prev };
      const current = resolveRecord(next[selectedId]);
      const updated = { ...current };
      if (dropEmpty && (value === "" || value == null)) {
        delete updated[key];
      } else {
        updated[key] = value;
      }
      next[selectedId] = updated;
      return next;
    });
  };


  const runAutoSave = React.useCallback(
    async (payload: Record<string, any>) => {
      if (!selectedId) {
        return false;
      }
      if (saveInFlightRef.current) {
        saveQueuedRef.current = payload;
        return false;
      }
      saveInFlightRef.current = true;
      setAutoSaveStatus("saving");
      setAutoSaveError(null);
      try {
        const result = await patchConfig.mutateAsync({
          ops: [
            {
              op: "replace",
              path: `/channels/${selectedId}`,
              value: payload,
            },
          ],
          expectedVersion: configVersionRef.current,
          dryRun: false,
        });
        configVersionRef.current = result.version;
        queryClient.setQueryData(["config", "/channels"], (current) => {
          const currentConfig = resolveRecord((current as { config?: unknown })?.config);
          return {
            config: { ...currentConfig, [selectedId]: payload },
            version: result.version,
          };
        });
        setConfigDraft((prev) => ({ ...prev, [selectedId]: payload }));
        setAutoSaveStatus("idle");
        setAutoSaveError(null);
        void queryClient.invalidateQueries({ queryKey: CHANNELS_QUERY_KEY });
        void queryClient.invalidateQueries({ queryKey: gatewayToolsKey });
        return true;
      } catch (error) {
        const message = formatGatewayError(error);
        setAutoSaveStatus("error");
        setAutoSaveError(message);
        messageBus.publishToast({
          intent: "danger",
          title: t("settings.integration.channels.config.autoSaveError"),
          description: message,
        });
        return false;
      } finally {
        saveInFlightRef.current = false;
        if (saveQueuedRef.current) {
          const queued = saveQueuedRef.current;
          saveQueuedRef.current = null;
          void runAutoSave(queued);
        }
      }
    },
    [patchConfig, queryClient, selectedId, t]
  );

  const commitConfigDraft = React.useCallback(() => {
    if (!selectedId || !configQuery.data) {
      return;
    }
    if (skipAutoSaveRef.current) {
      return;
    }
    if (saveTimeoutRef.current) {
      window.clearTimeout(saveTimeoutRef.current);
      saveTimeoutRef.current = null;
    }
    if (!hasConfigChanges) {
      return;
    }
    if (saveInFlightRef.current) {
      saveQueuedRef.current = normalizedDraft;
      return;
    }
    void runAutoSave(normalizedDraft);
  }, [configQuery.data, hasConfigChanges, normalizedDraft, runAutoSave, selectedId]);

  React.useEffect(() => () => {
    if (!saveTimeoutRef.current) {
      return;
    }
    window.clearTimeout(saveTimeoutRef.current);
    saveTimeoutRef.current = null;
  }, []);

  const buildNextConfig = React.useCallback(
    (key: string, value: unknown, dropEmpty = false) => {
      if (!selectedId) {
        return null;
      }
      const next = { ...selectedConfig };
      if (dropEmpty && (value === "" || value == null)) {
        delete next[key];
      } else {
        next[key] = value;
      }
      return normalizeChannelConfig(selectedId, next);
    },
    [normalizeChannelConfig, selectedConfig, selectedId]
  );

  const applyImmediateConfig = React.useCallback(
    async (payload: Record<string, any>) => {
      if (!selectedId || !configQuery.data) {
        return false;
      }
      if (saveTimeoutRef.current) {
        window.clearTimeout(saveTimeoutRef.current);
        saveTimeoutRef.current = null;
      }
      skipAutoSaveRef.current = true;
      setConfigDraft((prev) => ({ ...prev, [selectedId]: payload }));
      try {
        return await runAutoSave(payload);
      } finally {
        skipAutoSaveRef.current = false;
      }
    },
    [configQuery.data, runAutoSave, selectedId]
  );

  React.useEffect(() => {
    if (!showAllowFrom) {
      if (allowFromDraft.length > 0) {
        setAllowFromDraft([]);
      }
      if (allowFromEditingIndex != null) {
        setAllowFromEditingIndex(null);
      }
      return;
    }
    const hasEmptyRow = allowFromDraft.some((entry) => entry.trim() === "");
    const normalizedDraft = sanitizeAllowFromDraft(allowFromDraft);
    if (!hasEmptyRow && !areListsEqual(normalizedDraft, allowFromList)) {
      setAllowFromDraft(allowFromList);
    }
    if (allowFromDraft.length === 0 && allowFromList.length > 0) {
      setAllowFromDraft(allowFromList);
    }
  }, [allowFromDraft, allowFromList, showAllowFrom]);

  React.useEffect(() => {
    if (!showGroupAllowFrom) {
      if (groupAllowFromDraft.length > 0) {
        setGroupAllowFromDraft([]);
      }
      if (groupAllowFromEditingIndex != null) {
        setGroupAllowFromEditingIndex(null);
      }
      return;
    }
    const hasEmptyRow = groupAllowFromDraft.some((entry) => entry.trim() === "");
    const normalizedDraft = sanitizeAllowFromDraft(groupAllowFromDraft);
    if (!hasEmptyRow && !areListsEqual(normalizedDraft, groupAllowFromList)) {
      setGroupAllowFromDraft(groupAllowFromList);
    }
    if (groupAllowFromDraft.length === 0 && groupAllowFromList.length > 0) {
      setGroupAllowFromDraft(groupAllowFromList);
    }
  }, [groupAllowFromDraft, groupAllowFromList, showGroupAllowFrom]);

  React.useEffect(() => {
    if (!showGroupAllowList) {
      if (groupAllowListDraft.length > 0) {
        setGroupAllowListDraft([]);
      }
      if (groupAllowListEditingIndex != null) {
        setGroupAllowListEditingIndex(null);
      }
      return;
    }
    const hasEmptyRow = groupAllowListDraft.some((entry) => entry.id.trim() === "");
    const normalizedDraft = sanitizeGroupAllowDraft(groupAllowListDraft);
    if (!hasEmptyRow && !areGroupEntriesEqual(normalizedDraft, groupAllowList)) {
      setGroupAllowListDraft(groupAllowList);
    }
    if (groupAllowListDraft.length === 0 && groupAllowList.length > 0) {
      setGroupAllowListDraft(groupAllowList);
    }
  }, [groupAllowListDraft, groupAllowList, showGroupAllowList]);

  React.useEffect(() => {
    if (selectedId !== "telegram") {
      if (customCommandsDraft.length > 0) {
        setCustomCommandsDraft([]);
      }
      if (customCommandsEditingIndex != null) {
        setCustomCommandsEditingIndex(null);
      }
      if (customCommandsError) {
        setCustomCommandsError(null);
      }
      if (customCommandsTouchedRef.current) {
        customCommandsTouchedRef.current = false;
      }
      if (customCommandsSyncRef.current) {
        customCommandsSyncRef.current = false;
      }
      return;
    }
    const normalizedDraft = sanitizeCustomCommandDraft(customCommandsDraft);
    const isDirty = !areCustomCommandsEqual(normalizedDraft, customCommandsList);
    if (!isDirty && customCommandsTouchedRef.current) {
      customCommandsTouchedRef.current = false;
    }
    if (customCommandsTouchedRef.current) {
      return;
    }
    if (customCommandsDraft.length === 0 && customCommandsList.length > 0) {
      setCustomCommandsDraft(customCommandsList);
      return;
    }
    if (!areCustomCommandsEqual(normalizedDraft, customCommandsList)) {
      setCustomCommandsDraft(customCommandsList);
    }
  }, [
    customCommandsDraft,
    customCommandsEditingIndex,
    customCommandsError,
    customCommandsList,
    selectedId,
  ]);

  const buildCustomCommandsConfig = (draft: TelegramCustomCommand[]) => {
    const next = { ...selectedConfig };
    const sanitized = sanitizeCustomCommandDraft(draft);
    if (sanitized.length > 0) {
      next.customCommands = sanitized;
    } else {
      delete next.customCommands;
    }
    return next;
  };

  const updateCustomCommandsDraft = (draft: TelegramCustomCommand[]) => {
    customCommandsTouchedRef.current = true;
    setCustomCommandsDraft(draft);
    if (customCommandsError) {
      setCustomCommandsError(null);
    }
  };

  const validateCustomCommands = React.useCallback(
    (draft: TelegramCustomCommand[]) => {
      for (const entry of draft) {
        const command = entry.command.trim();
        const description = entry.description.trim();
        if (command === "" && description === "") {
          continue;
        }
        if (!command.startsWith("/")) {
          return t("settings.integration.channels.menu.validation.commandPrefix");
        }
        const normalized = normalizeTelegramCommandName(command);
        if (!telegramCommandNamePattern.test(normalized)) {
          return t("settings.integration.channels.menu.validation.commandInvalid");
        }
        if (description === "") {
          return t("settings.integration.channels.menu.validation.descriptionRequired");
        }
      }
      return null;
    },
    [t]
  );

  const saveCustomCommands = React.useCallback(
    async (draft: TelegramCustomCommand[], syncAfter: boolean) => {
      const validationError = validateCustomCommands(draft);
      if (validationError) {
        setCustomCommandsError(validationError);
        return false;
      }
      setCustomCommandsError(null);
      const sanitized = sanitizeCustomCommandDraft(draft);
      if (areCustomCommandsEqual(sanitized, customCommandsList)) {
        if (syncAfter) {
          customCommandsSyncRef.current = false;
        }
        return true;
      }
      if (syncAfter) {
        customCommandsSyncRef.current = true;
      }
      const saved = await applyImmediateConfig(buildCustomCommandsConfig(draft));
      if (saved && syncAfter) {
        customCommandsSyncRef.current = false;
        await handleMenuSync();
      }
      return saved;
    },
    [
      applyImmediateConfig,
      buildCustomCommandsConfig,
      customCommandsList,
      handleMenuSync,
      validateCustomCommands,
    ]
  );

  React.useEffect(() => {
    if (!customCommandsSyncRef.current) {
      return;
    }
    if (selectedId !== "telegram") {
      customCommandsSyncRef.current = false;
      return;
    }
    if (autoSaveStatus === "saving" || saveInFlightRef.current) {
      return;
    }
    const normalizedDraft = sanitizeCustomCommandDraft(customCommandsDraft);
    if (!areCustomCommandsEqual(normalizedDraft, customCommandsList)) {
      return;
    }
    customCommandsSyncRef.current = false;
    void handleMenuSync();
  }, [autoSaveStatus, customCommandsDraft, customCommandsList, handleMenuSync, selectedId]);

  const buildAllowFromConfig = React.useCallback(
    (draft: string[]) => {
      const next = { ...selectedConfig };
      const sanitized = sanitizeAllowFromDraft(draft);
      if (sanitized.length > 0) {
        next.allowFrom = sanitized;
      } else {
        delete next.allowFrom;
      }
      return next;
    },
    [selectedConfig]
  );

  const updateAllowFromDraftAndSave = React.useCallback(
    (draft: string[]) => {
      setAllowFromDraft(draft);
      const sanitized = sanitizeAllowFromDraft(draft);
      if (!areListsEqual(sanitized, allowFromList)) {
        void applyImmediateConfig(buildAllowFromConfig(draft));
      }
    },
    [allowFromList, applyImmediateConfig, buildAllowFromConfig]
  );

  const buildGroupAllowFromConfig = React.useCallback(
    (draft: string[]) => {
      const next = { ...selectedConfig };
      const sanitized = sanitizeAllowFromDraft(draft);
      if (sanitized.length > 0) {
        next.groupAllowFrom = sanitized;
      } else {
        delete next.groupAllowFrom;
      }
      return next;
    },
    [selectedConfig]
  );

  const updateGroupAllowFromDraftAndSave = React.useCallback(
    (draft: string[]) => {
      setGroupAllowFromDraft(draft);
      const sanitized = sanitizeAllowFromDraft(draft);
      if (!areListsEqual(sanitized, groupAllowFromList)) {
        void applyImmediateConfig(buildGroupAllowFromConfig(draft));
      }
    },
    [applyImmediateConfig, buildGroupAllowFromConfig, groupAllowFromList]
  );

  const buildGroupAllowListConfig = React.useCallback(
    (draft: GroupAllowEntry[]) => {
      const next = { ...selectedConfig };
      const sanitized = sanitizeGroupAllowDraft(draft);
      if (sanitized.length > 0) {
        const groups: Record<string, any> = {};
        sanitized.forEach((entry) => {
          const groupEntry: Record<string, any> = {};
          if (!entry.requireMention) {
            groupEntry.requireMention = false;
          }
          groups[entry.id] = groupEntry;
        });
        next.groups = groups;
      } else {
        delete next.groups;
      }
      return next;
    },
    [selectedConfig]
  );

  const updateGroupAllowListDraftAndSave = React.useCallback(
    (draft: GroupAllowEntry[]) => {
      setGroupAllowListDraft(draft);
      const sanitized = sanitizeGroupAllowDraft(draft);
      if (!areGroupEntriesEqual(sanitized, groupAllowList)) {
        void applyImmediateConfig(buildGroupAllowListConfig(draft));
      }
    },
    [applyImmediateConfig, buildGroupAllowListConfig, groupAllowList]
  );

  const controlClassName = "w-full sm:w-60 min-w-0";

  const renderConfigField = (field: ChannelField) => {
    const label = t(field.labelKey);
    const rawValue =
      field.key === "botToken" && selectedId === "telegram"
        ? resolveTelegramBotToken(selectedConfig)
        : selectedConfig[field.key];
    const rowClass = "flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4";

    if (field.type === "toggle") {
      const fallbackEnabled = selected?.enabled ?? true;
      const checked = typeof rawValue === "boolean" ? rawValue : fallbackEnabled;
      return (
        <React.Fragment key={field.key}>
          <div className={rowClass}>
            <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">{label}</span>
            <Switch
              checked={checked}
              onCheckedChange={(value) => {
                const nextConfig = buildNextConfig(field.key, value);
                if (nextConfig) {
                  void applyImmediateConfig(nextConfig);
                }
              }}
              disabled={configBusy}
            />
          </div>
        </React.Fragment>
      );
    }

    if (field.type === "select") {
      const rawString = typeof rawValue === "string" ? rawValue : "";
      let value = rawString;
      if (field.key === "streamMode" && rawString === "") {
        value = "partial";
      }
      return (
        <React.Fragment key={field.key}>
          <div className={rowClass}>
            <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">{label}</span>
            <Select
              value={value}
              onChange={(event) => {
                const nextConfig = buildNextConfig(field.key, event.target.value, true);
                if (nextConfig) {
                  void applyImmediateConfig(nextConfig);
                }
              }}
              className={controlClassName}
              disabled={configBusy}
            >
              {field.allowDefault !== false && (
                <option value="">
                  {t("settings.integration.channels.config.options.default")}
                </option>
              )}
              {(field.options ?? []).map((option) => (
                <option key={option.value} value={option.value}>
                  {t(option.labelKey)}
                </option>
              ))}
            </Select>
          </div>
        </React.Fragment>
      );
    }

    const value = typeof rawValue === "string" ? rawValue : "";
    const inputType = field.type === "secret" ? "password" : "text";
    const placeholder =
      field.placeholder ??
      t("settings.integration.channels.config.placeholders.token");
    return (
      <React.Fragment key={field.key}>
        <div className={rowClass}>
          <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">{label}</span>
          <Input
            value={value}
            type={inputType}
            placeholder={placeholder}
            onChange={(event) => setSelectedConfigValue(field.key, event.target.value, true)}
            onBlur={commitConfigDraft}
            className={controlClassName}
            size="compact"
            disabled={configBusy}
          />
        </div>
      </React.Fragment>
    );
  };

  const telegramCommandsConfig = resolveRecord(selectedConfig.commands);
  const telegramNativeSetting = resolveCommandSettingValue(telegramCommandsConfig.native);
  const telegramNativeSkillsSetting = resolveCommandSettingValue(telegramCommandsConfig.nativeSkills);

  const updateTelegramCommandSetting = (key: "native" | "nativeSkills", value: "on" | "off") => {
    if (!selectedId) {
      return;
    }
    const current = resolveRecord(selectedConfig);
    const currentCommands = resolveRecord(current.commands);
    const updated = { ...currentCommands };
    updated[key] = value === "on";
    const next = { ...current };
    if (Object.keys(updated).length > 0) {
      next.commands = updated;
    } else {
      delete next.commands;
    }
    const normalized = normalizeChannelConfig(selectedId, next);
    void applyImmediateConfig(normalized);
  };

  const handleAddTelegramCustomCommand = () => {
    const nextIndex = customCommandsDraft.length;
    const next = [...customCommandsDraft, { command: "", description: "" }];
    updateCustomCommandsDraft(next);
    void saveCustomCommands(next, true);
    setCustomCommandsEditingIndex(nextIndex);
  };

  const handleRemoveTelegramCustomCommand = (index: number) => {
    const next = customCommandsDraft.filter((_, idx) => idx !== index);
    updateCustomCommandsDraft(next);
    void saveCustomCommands(next, true);
  };

  const handleTelegramCustomCommandChange = (
    index: number,
    key: keyof TelegramCustomCommand,
    value: string
  ) => {
    const next = customCommandsDraft.map((entry, idx) =>
      idx === index ? { ...entry, [key]: value } : entry
    );
    updateCustomCommandsDraft(next);
  };

  const commitTelegramCustomCommands = () => {
    void saveCustomCommands(customCommandsDraft, true);
  };

  const renderPolicySelectRow = (field: ChannelField, value: string) => (
    <div className={rowClassName}>
      <span className={labelClassName}>{t(field.labelKey)}</span>
      <Select
        value={value}
        onChange={(event) => {
          const nextConfig = buildNextConfig(field.key, event.target.value, true);
          if (nextConfig) {
            void applyImmediateConfig(nextConfig);
          }
        }}
        className={controlClassName}
        disabled={configBusy}
      >
        {(field.options ?? []).map((option) => (
          <option key={option.value} value={option.value}>
            {t(option.labelKey)}
          </option>
        ))}
      </Select>
    </div>
  );

  const renderTelegramConfigCards = () => {
    if (configQuery.isLoading) {
      return (
        <SectionCard contentClassName="space-y-3 px-3 py-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {t("settings.integration.channels.config.loading")}
          </div>
        </SectionCard>
      );
    }
    if (configQuery.isError) {
      return (
        <SectionCard contentClassName="space-y-3 px-3 py-2">
          <div className="text-sm text-destructive">
            {t("settings.integration.channels.config.loadError")}
          </div>
        </SectionCard>
      );
    }

    return (
      <>
        <SectionCard contentClassName="space-y-3 px-3 py-2">
          <div className="space-y-3">
            {[FIELD_BOT_TOKEN, FIELD_STREAM_MODE_TELEGRAM].map((field) => renderConfigField(field))}
          </div>
        </SectionCard>

        <Tabs defaultValue="dm" className="space-y-2">
          <TabsList>
            <TabsTrigger value="dm">
              {t("settings.integration.channels.config.fields.dmPolicy")}
            </TabsTrigger>
            <TabsTrigger value="group">
              {t("settings.integration.channels.config.fields.groupPolicy")}
            </TabsTrigger>
          </TabsList>
          <SectionCard contentClassName="space-y-3 px-3 py-2">
            <TabsContent value="dm" className="mt-0 space-y-3">
              {renderPolicySelectRow(FIELD_DM_POLICY, dmPolicyValue)}
              <AllowFromTable
                t={t}
                show={showAllowFrom}
                rows={allowFromDraft}
                editingIndex={allowFromEditingIndex}
                onSetEditingIndex={setAllowFromEditingIndex}
                onRowsChange={updateAllowFromDraftAndSave}
                disabled={configBusy}
              />
              <PairingRequestsPanel
                t={t}
                show={showPairing}
                requests={pairingListQuery.data?.requests ?? []}
                isLoading={pairingListQuery.isLoading}
                isError={pairingListQuery.isError}
                errorMessage={
                  pairingListQuery.isError ? formatGatewayError(pairingListQuery.error) : ""
                }
                pairingBusy={pairingApprove.isPending || pairingReject.isPending}
                approvePending={pairingApprove.isPending}
                rejectPending={pairingReject.isPending}
                pairingBusyCode={pairingBusyCode}
                onApprove={handlePairingApprove}
                onReject={handlePairingReject}
                formatPairingMeta={formatPairingMeta}
                formatPairingTime={formatPairingTime}
              />
            </TabsContent>
            <TabsContent value="group" className="mt-0 space-y-3">
              {renderPolicySelectRow(FIELD_GROUP_POLICY, groupPolicyValue)}
              <GroupAllowListTable
                t={t}
                show={showGroupAllowList}
                rows={groupAllowListDraft}
                editingIndex={groupAllowListEditingIndex}
                onSetEditingIndex={setGroupAllowListEditingIndex}
                onRowsChange={updateGroupAllowListDraftAndSave}
                disabled={configBusy}
              />
              <GroupAllowFromTable
                t={t}
                show={showGroupAllowFrom}
                rows={groupAllowFromDraft}
                editingIndex={groupAllowFromEditingIndex}
                onSetEditingIndex={setGroupAllowFromEditingIndex}
                onRowsChange={updateGroupAllowFromDraftAndSave}
                disabled={configBusy}
              />
            </TabsContent>
          </SectionCard>
        </Tabs>

        {showAutoSaveStatus ? (
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-2">
              {autoSaveState === "saving" || autoSaveState === "pending" ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : null}
              <span>{autoSaveLabel}</span>
            </div>
          </div>
        ) : null}
        {showAutoSaveError ? <div className="text-xs text-destructive">{autoSaveError}</div> : null}
      </>
    );
  };

  const rowClassName =
    "flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4";
  const labelClassName = "min-w-0 truncate text-sm font-medium text-muted-foreground";

  return (
    <div className="connectors-card flex min-h-0 min-w-0 flex-1">
      <Card className="flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden">
        <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
              <div className="flex h-8 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder={t("settings.integration.channels.searchPlaceholder")}
                  size="compact"
                  className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              {channels.isLoading ? (
                <div className="flex items-center gap-2 p-3 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {t("settings.integration.channels.loading")}
                </div>
              ) : filteredItems.length === 0 ? (
                <div className="p-3 text-sm text-muted-foreground">
                  {t("settings.integration.channels.searchEmpty")}
                </div>
              ) : (
                <SidebarMenu>
                  {filteredItems.map((channel) => {
                    const channelConfig = getChannelConfig(channel.channelId);
                    const channelState = resolveChannelStatus(channel.channelId, channel, channelConfig, configReady);
                    const statusMeta = STATUS_META[channelState] ?? STATUS_META.unknown;
                    const isSelected = channel.channelId === selectedId;
                    const label = channel.displayName || channel.channelId;
                    return (
                      <SidebarMenuItem key={channel.channelId}>
                        <SidebarMenuButton
                          type="button"
                          isActive={isSelected}
                          className="justify-between"
                          onClick={() => setSelectedId(channel.channelId)}
                        >
                          <div className="flex min-w-0 items-center gap-2">
                            <span className="truncate text-xs font-semibold uppercase tracking-[0.24em]">
                              {label.toUpperCase()}
                            </span>
                          </div>
                          <div className="shrink-0">
                            <span
                              className={cn(
                                "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                                statusMeta.className
                              )}
                            >
                              {React.createElement(statusMeta.icon, { className: "h-3.5 w-3.5" })}
                            </span>
                          </div>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              )}
            </div>
          </div>

          <Separator orientation="vertical" className="self-stretch" />

          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto px-3 py-1.5">
              {selected ? (
                <div className="flex min-h-0 flex-col gap-3">
                  <SectionCard contentClassName="space-y-3 px-3 py-2">
                    <div className={rowClassName}>
                      <div className={labelClassName}>
                        {t("settings.integration.channels.detail.status")}
                      </div>
                      <span
                        className={cn(
                          "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                          status.className
                        )}
                      >
                        {React.createElement(status.icon, { className: "h-3.5 w-3.5" })}
                        {t(status.labelKey)}
                      </span>
                    </div>
                    {state === "not_configured" ? (
                      <div className="text-xs text-amber-700 dark:text-amber-200">
                        {t("settings.integration.channels.status.notConfiguredHint")}
                        {missingFieldLabels.length > 0
                          ? ` ${t("settings.integration.channels.status.missing")}: ${missingFieldLabels.join(", ")}`
                          : null}
                      </div>
                    ) : null}
                    <Separator />
                    <div className={rowClassName}>
                      <div className={labelClassName}>
                        {t("settings.integration.channels.config.fields.enabled")}
                      </div>
                      <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-60">
                        <Switch
                          checked={configEnabled}
                          onCheckedChange={(value) => {
                            const nextConfig = buildNextConfig("enabled", value);
                            if (nextConfig) {
                              void applyImmediateConfig(nextConfig);
                            }
                          }}
                          disabled={configBusy}
                        />
                      </div>
                    </div>
                    <Separator />
                    <div className={rowClassName}>
                      <div className={labelClassName}>
                        {t("settings.integration.channels.detail.updatedAt")}
                      </div>
                      <div className="flex w-full items-center justify-end gap-2 sm:w-60">
                        <span className="min-w-0 truncate text-sm text-foreground">{updatedAt}</span>
                        <Button
                          variant="ghost"
                          size="compactIcon"
                          onClick={handleReload}
                          disabled={configBusy}
                          aria-label={t("settings.integration.channels.actions.reload")}
                        >
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                    {lastError ? (
                      <>
                        <Separator />
                        <div className={rowClassName}>
                          <div className={labelClassName}>
                            {t("settings.integration.channels.detail.lastError")}
                          </div>
                          <div className="min-w-0 flex-1 text-right">
                            <span className="block min-w-0 break-words whitespace-pre-wrap text-xs text-destructive">
                              {lastError}
                            </span>
                          </div>
                        </div>
                      </>
                    ) : null}
                    <Separator />
                    <div className="flex flex-wrap items-center justify-center gap-2">
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={handleProbe}
                        disabled={actionsDisabled}
                      >
                        {probeChannel.isPending ? (
                          <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                        ) : (
                          <Search className="mr-1 h-4 w-4" />
                        )}
                        {t("settings.integration.channels.actions.probe")}
                      </Button>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={handleLogout}
                        disabled={actionsDisabled}
                      >
                        {logoutChannel.isPending ? (
                          <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                        ) : (
                          <CircleOff className="mr-1 h-4 w-4" />
                        )}
                        {t("settings.integration.channels.actions.logout")}
                      </Button>
                      {selected?.channelId === "telegram" ? (
                        <Button
                          variant="outline"
                          size="compact"
                          onClick={handleMenuSync}
                          disabled={!menuReady || syncMenu.isPending}
                        >
                          {syncMenu.isPending ? (
                            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                          ) : (
                            <RefreshCw className="mr-1 h-4 w-4" />
                          )}
                          {t("settings.integration.channels.menu.sync")}
                        </Button>
                      ) : null}
                    </div>
                    {requiresConfig ? (
                      <div className="text-xs text-muted-foreground">
                        {t("settings.integration.channels.actions.disabledHint")}
                      </div>
                    ) : null}
                  </SectionCard>

                  {selectedId === "telegram" ? (
                    renderTelegramConfigCards()
                  ) : (
                    <SectionCard contentClassName="space-y-3 px-3 py-2">
                      {configQuery.isLoading ? (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          {t("settings.integration.channels.config.loading")}
                        </div>
                      ) : configQuery.isError ? (
                        <div className="text-sm text-destructive">
                          {t("settings.integration.channels.config.loadError")}
                        </div>
                      ) : configFields.length === 0 ? (
                        <div className="text-sm text-muted-foreground">
                          {t("settings.integration.channels.config.empty")}
                        </div>
                      ) : (
                        <div className="space-y-3">
                          {configFields.map((field) => renderConfigField(field))}
                        </div>
                      )}
                      {showAutoSaveStatus ? (
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                          <div className="flex items-center gap-2">
                            {autoSaveState === "saving" || autoSaveState === "pending" ? (
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            ) : null}
                            <span>{autoSaveLabel}</span>
                          </div>
                        </div>
                      ) : null}
                      {showAutoSaveError ? (
                        <div className="text-xs text-destructive">{autoSaveError}</div>
                      ) : null}
                    </SectionCard>
                  )}

                  {selectedId === "telegram" ? (
                    <TelegramMenuConfigCard
                      t={t}
                      busy={configBusy}
                      controlClassName={controlClassName}
                      nativeSetting={telegramNativeSetting}
                      nativeSkillsSetting={telegramNativeSkillsSetting}
                      customCommands={customCommandsDraft}
                      customCommandsEditingIndex={customCommandsEditingIndex}
                      customCommandsError={customCommandsError}
                      onChangeCommandSetting={updateTelegramCommandSetting}
                      onAddCustomCommand={handleAddTelegramCustomCommand}
                      onRemoveCustomCommand={handleRemoveTelegramCustomCommand}
                      onCustomCommandChange={handleTelegramCustomCommandChange}
                      onSetCustomCommandsEditingIndex={setCustomCommandsEditingIndex}
                      onCommitCustomCommands={commitTelegramCustomCommands}
                    />
                  ) : null}

                  <div className="flex justify-center pt-2">
                    <Button
                      variant="destructive"
                      size="compact"
                      onClick={handleResetConfig}
                      disabled={configBusy || patchConfig.isPending || !selectedId}
                    >
                      <Trash2 className="h-4 w-4" />
                      {t("settings.integration.channels.reset.button")}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="p-4 text-sm text-muted-foreground">
                  {t("settings.integration.channels.empty")}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
