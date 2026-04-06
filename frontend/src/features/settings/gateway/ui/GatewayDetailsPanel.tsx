import * as React from "react";
import { ChevronsUpDown, Loader2 } from "lucide-react";
import { Events } from "@wailsio/runtime";

import { gatewayRequest } from "@/shared/api/gateway";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useHeartbeatLastStatus } from "@/shared/query/heartbeat";
import { useNotices } from "@/shared/query/notices";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import { useSettings, useShowMainWindow, useUpdateSettings } from "@/shared/query/settings";
import { useThreads } from "@/shared/query/threads";
import {
  useSetTTSConfig,
  useSetTalkConfig,
  useSetVoiceWake,
  useTTSStatus,
  useTalkConfig,
  useVoiceWake,
} from "@/shared/query/voice";
import type { TalkConfigEntity, TTSConfigEntity } from "@/entities/voice";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import type { UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import { GatewayAgentLoopPanel } from "./panels/GatewayAgentLoopPanel";
import { Assistant3DAvatar } from "./Assistant3DAvatar";
import { AssistantEmojiPicker } from "./AssistantEmojiPicker";
import { GatewayContextPanel } from "./panels/GatewayContextPanel";
import { GatewayCorePanel } from "./panels/GatewayCorePanel";
import { GatewayCronPanel } from "./panels/GatewayCronPanel";
import { GatewayHeartbeatPanel } from "./panels/GatewayHeartbeatPanel";
import { GatewayHttpPanel } from "./panels/GatewayHttpPanel";
import { GatewayQueuePanel } from "./panels/GatewayQueuePanel";
import { GatewaySubagentsPanel } from "./panels/GatewaySubagentsPanel";
import { GatewayTalkPanel, GatewayVoiceWakePanel } from "./panels/GatewayVoicePanels";
import {
  DETAILS_SECTIONS,
  EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS,
  EXEC_PERMISSION_MODE_FULL_ACCESS,
  buildEmptyHeartbeatSpecDraft,
  buildHeartbeatSpecDraftFromChecklist,
  formatVoiceAliases,
  parseCommaList,
  patchCallsToolsExecPermissionMode,
  resolveExecPermissionMode,
  toCommaList,
  type DetailsSectionId,
  type ExecPermissionMode,
  type GatewayDetailsPanelProps,
  type HeartbeatSpecDraft,
  type HeartbeatSpecItemDraft,
  type HeartbeatTriggerResponse,
} from "./gateway-details-panel.utils";

type HeartbeatTriggerFeedback = {
  intent: "success" | "warning" | "info";
  message: string;
};

export function GatewayDetailsPanel({
  assistant,
  assistants,
  currentAssistantId,
  onSelectAssistant,
}: GatewayDetailsPanelProps) {
  const { t } = useI18n();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();
  const showMainWindow = useShowMainWindow();
  const providersQuery = useEnabledProvidersWithModels();
  const threadsQuery = useThreads(false);

  const talkConfig = useTalkConfig();
  const setTalkConfig = useSetTalkConfig();
  const [talkDraft, setTalkDraft] = React.useState<TalkConfigEntity>({});
  const [talkAliasesInput, setTalkAliasesInput] = React.useState<string>("");
  const ttsStatus = useTTSStatus();
  const setTTSConfig = useSetTTSConfig();
  const [ttsDraft, setTtsDraft] = React.useState<TTSConfigEntity>({});

  const voiceWake = useVoiceWake();
  const setVoiceWake = useSetVoiceWake();
  const [wakeInput, setWakeInput] = React.useState<string>("");
  const [heartbeatSpecDraft, setHeartbeatSpecDraft] = React.useState<HeartbeatSpecDraft>(
    buildEmptyHeartbeatSpecDraft()
  );
  const [heartbeatSpecVersion, setHeartbeatSpecVersion] = React.useState<number>(0);
  const [heartbeatSpecUpdatedAt, setHeartbeatSpecUpdatedAt] = React.useState<string>("");
  const [heartbeatSpecSaving, setHeartbeatSpecSaving] = React.useState<boolean>(false);
  const [heartbeatTriggerPending, setHeartbeatTriggerPending] = React.useState<boolean>(false);
  const [heartbeatTriggerFeedback, setHeartbeatTriggerFeedback] =
    React.useState<HeartbeatTriggerFeedback | null>(null);

  const [activeSectionId, setActiveSectionId] = React.useState<DetailsSectionId>("gateway");
  const [contextTab, setContextTab] = React.useState<"guard" | "compaction" | "memory">("guard");
  const [heartbeatTab, setHeartbeatTab] = React.useState<"general" | "runtime" | "checklist">(
    "general"
  );
  const [httpTab, setHttpTab] = React.useState<"endpoints" | "files" | "images">("endpoints");
  const [talkTab, setTalkTab] = React.useState<"talk" | "tts">("talk");

  React.useEffect(() => {
    if (talkConfig.data) {
      setTalkDraft(talkConfig.data);
      setTalkAliasesInput(formatVoiceAliases(talkConfig.data.voiceAliases));
    }
  }, [talkConfig.data]);

  React.useEffect(() => {
    if (voiceWake.data) {
      setWakeInput(toCommaList(voiceWake.data.triggers));
    }
  }, [voiceWake.data]);

  React.useEffect(() => {
    if (ttsStatus.data?.config) {
      setTtsDraft(ttsStatus.data.config);
    }
  }, [ttsStatus.data?.config]);

  React.useEffect(() => {
    if (DETAILS_SECTIONS.some((item) => item.id === activeSectionId)) {
      return;
    }
    setActiveSectionId("gateway");
  }, [activeSectionId]);

  const gateway = settingsQuery.data?.gateway;
  const heartbeatSessionKey = gateway?.heartbeat.runSession?.trim() ?? "";
  const heartbeatLastStatus = useHeartbeatLastStatus(heartbeatSessionKey, Boolean(heartbeatSessionKey));
  const noticesQuery = useNotices({ surface: "center", limit: 20 });
  const heartbeatSessionOptions = React.useMemo(() => {
    const items = [...(threadsQuery.data ?? [])].sort((a, b) => {
      const left = (a.lastInteractiveAt || a.updatedAt || "").trim();
      const right = (b.lastInteractiveAt || b.updatedAt || "").trim();
      return right.localeCompare(left);
    });
    return items
      .filter((item) => (item.id ?? "").trim() !== "")
      .map((item) => {
        const id = item.id.trim();
        const title = (item.title ?? "").trim() || t("sidebar.thread.untitled");
        return {
          value: id,
          label: `${title} · ${id.slice(0, 8)}`,
        };
      });
  }, [t, threadsQuery.data]);
  const callsToolsRaw = settingsQuery.data?.tools;
  const execPermissionMode = React.useMemo(
    () => resolveExecPermissionMode(callsToolsRaw),
    [callsToolsRaw]
  );
  const permissionModeOptions = React.useMemo(
    () => [
      {
        value: EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS as ExecPermissionMode,
        label: t("chat.composer.permission.standard"),
      },
      {
        value: EXEC_PERMISSION_MODE_FULL_ACCESS as ExecPermissionMode,
        label: t("chat.composer.permission.allAccess"),
      },
    ],
    [t]
  );
  const activeSection =
    DETAILS_SECTIONS.find((item) => item.id === activeSectionId) ?? DETAILS_SECTIONS[0];
  const activeSectionDescription = activeSection.descriptionKey
    ? t(activeSection.descriptionKey)
    : "";
  const isBusy = updateSettings.isPending || settingsQuery.isLoading;
  const isDisabled = isBusy || !gateway;
  const latestHeartbeatNotice = React.useMemo(
    () => (noticesQuery.data ?? []).find((notice) => notice.source?.producer === "heartbeat") ?? null,
    [noticesQuery.data]
  );

  const updateGateway = (payload: UpdateGatewaySettingsRequest) => {
    if (!gateway) {
      return;
    }
    updateSettings.mutate({ gateway: payload });
  };

  const handleExecPermissionModeChange = React.useCallback(
    (nextMode: ExecPermissionMode) => {
      if (nextMode === execPermissionMode) {
        return;
      }
      const nextCallsTools = patchCallsToolsExecPermissionMode(callsToolsRaw, nextMode);
      updateSettings.mutate(
        {
          tools: nextCallsTools,
        },
        {
          onError: (error) => {
            messageBus.publishToast({
              intent: "warning",
              title: t("chat.composer.permission.updateError"),
              description: error instanceof Error ? error.message : String(error ?? ""),
            });
          },
        }
      );
    },
    [callsToolsRaw, execPermissionMode, t, updateSettings]
  );

  React.useEffect(() => {
    const checklist = gateway?.heartbeat.checklist;
    setHeartbeatSpecDraft(buildHeartbeatSpecDraftFromChecklist(checklist));
    setHeartbeatSpecVersion(
      checklist && typeof checklist.version === "number" ? checklist.version : 0
    );
    setHeartbeatSpecUpdatedAt(
      checklist && typeof checklist.updatedAt === "string" ? checklist.updatedAt : ""
    );
  }, [gateway?.heartbeat.checklist]);

  const updateHeartbeatSpecItem = (index: number, patch: Partial<HeartbeatSpecItemDraft>) => {
    setHeartbeatSpecDraft((previous) => {
      if (!previous.items[index]) {
        return previous;
      }
      const items = [...previous.items];
      items[index] = {
        ...items[index],
        ...patch,
      };
      return {
        ...previous,
        items,
      };
    });
  };

  const normalizeHeartbeatSpecItems = (items: HeartbeatSpecItemDraft[]): HeartbeatSpecItemDraft[] =>
    items
      .map((item) => ({
        id: item.id.trim(),
        text: item.text.trim(),
        done: item.done,
        priority: item.priority.trim(),
      }))
      .filter((item) => item.id !== "" || item.text !== "");

  const handleSaveHeartbeatSpec = () => {
    const normalizedItems = normalizeHeartbeatSpecItems(heartbeatSpecDraft.items);
    if (!gateway) {
      return;
    }
    setHeartbeatSpecSaving(true);
    updateSettings.mutate(
      {
        gateway: {
          heartbeat: {
            checklist: {
              title: heartbeatSpecDraft.title.trim(),
              notes: heartbeatSpecDraft.notes.trim(),
              items: normalizedItems.map((item) => ({
                id: item.id,
                text: item.text,
                done: item.done,
                priority: item.priority,
              })),
              version: heartbeatSpecVersion,
            },
          },
        },
      },
      {
        onSuccess: () => {
          setHeartbeatSpecDraft((previous) => ({
            ...previous,
            items: normalizedItems,
          }));
          messageBus.publishToast({
            intent: "success",
            title: t("settings.gateway.detailsPanel.heartbeat.spec.saved"),
          });
        },
        onError: (error) => {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.gateway.detailsPanel.heartbeat.spec.saveFailed"),
            description: String(error),
          });
        },
        onSettled: () => {
          setHeartbeatSpecSaving(false);
        },
      }
    );
  };

  const handleClearHeartbeatSpec = () => {
    if (!gateway) {
      return;
    }
    setHeartbeatSpecSaving(true);
    updateSettings.mutate(
      {
        gateway: {
          heartbeat: {
            checklist: {
              title: "",
              notes: "",
              items: [],
              version: 0,
              updatedAt: "",
            },
          },
        },
      },
      {
        onSuccess: () => {
          setHeartbeatSpecDraft(buildEmptyHeartbeatSpecDraft());
          setHeartbeatSpecVersion(0);
          setHeartbeatSpecUpdatedAt("");
          messageBus.publishToast({
            intent: "success",
            title: t("settings.gateway.detailsPanel.heartbeat.spec.cleared"),
          });
        },
        onError: (error) => {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.gateway.detailsPanel.heartbeat.spec.clearFailed"),
            description: String(error),
          });
        },
        onSettled: () => {
          setHeartbeatSpecSaving(false);
        },
      }
    );
  };

  const handleTriggerHeartbeat = async () => {
    const resolveReasonText = (reasonRaw?: string) => {
      const reason = reasonRaw?.trim().toLowerCase() ?? "";
      if (reason === "run_session_unset") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.runSessionUnset");
      }
      if (reason === "target_unavailable") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.targetUnavailable");
      }
      if (reason === "outside_active_hours") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.outsideActiveHours");
      }
      if (reason === "busy") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.busy");
      }
      if (reason === "not_due") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.notDue");
      }
      if (reason === "disabled") {
        return t("settings.gateway.detailsPanel.heartbeat.triggerReason.disabled");
      }
      return "";
    };

    setHeartbeatTriggerPending(true);
    try {
      const result = await gatewayRequest<HeartbeatTriggerResponse>("heartbeat.trigger", {
        reason: "manual-settings",
      });
      if (!result.ok || !result.accepted) {
        const message = t("settings.gateway.detailsPanel.heartbeat.triggerRejected");
        setHeartbeatTriggerFeedback({ intent: "warning", message });
        messageBus.publishToast({
          intent: "warning",
          title: message,
        });
        return;
      }
      if (result.executedStatus === "failed") {
        const reasonText =
          resolveReasonText(result.reason) ||
          t("settings.gateway.detailsPanel.heartbeat.triggerReason.failed");
        const message = `${t("settings.gateway.detailsPanel.heartbeat.triggerFailed")} · ${reasonText}`;
        setHeartbeatTriggerFeedback({ intent: "warning", message });
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.gateway.detailsPanel.heartbeat.triggerFailed"),
          description: reasonText,
        });
        return;
      }
      if (result.executedStatus === "skipped") {
        const reasonText =
          resolveReasonText(result.reason) ||
          t("settings.gateway.detailsPanel.heartbeat.triggerReason.skipped");
        const message = `${t("settings.gateway.detailsPanel.heartbeat.triggerSkipped")} · ${reasonText}`;
        setHeartbeatTriggerFeedback({ intent: "info", message });
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.gateway.detailsPanel.heartbeat.triggerSkipped"),
          description: reasonText,
        });
        return;
      }
      if (result.executedStatus === "queued") {
        const reasonText = resolveReasonText(result.reason);
        const message = reasonText
          ? `${t("settings.gateway.detailsPanel.heartbeat.triggerQueued")} · ${reasonText}`
          : t("settings.gateway.detailsPanel.heartbeat.triggerQueued");
        setHeartbeatTriggerFeedback({ intent: "success", message });
        messageBus.publishToast({
          intent: "success",
          title: t("settings.gateway.detailsPanel.heartbeat.triggerQueued"),
          description: reasonText || undefined,
        });
        return;
      }
      setHeartbeatTriggerFeedback({
        intent: "success",
        message: t("settings.gateway.detailsPanel.heartbeat.triggered"),
      });
      messageBus.publishToast({
        intent: "success",
        title: t("settings.gateway.detailsPanel.heartbeat.triggered"),
        description: resolveReasonText(result.reason) || undefined,
      });
    } catch {
      const message = t("settings.gateway.detailsPanel.heartbeat.triggerFailed");
      setHeartbeatTriggerFeedback({
        intent: "warning",
        message: `${message} · ${t("settings.gateway.detailsPanel.heartbeat.triggerReason.runtimeError")}`,
      });
      messageBus.publishToast({
        intent: "warning",
        title: message,
        description: t("settings.gateway.detailsPanel.heartbeat.triggerReason.runtimeError"),
      });
    } finally {
      setHeartbeatTriggerPending(false);
      if (heartbeatSessionKey) {
        void heartbeatLastStatus.refetch();
      }
      void noticesQuery.refetch();
    }
  };

  const subagentModelOptions = React.useMemo(() => {
    const options: Array<{ value: string; label: string }> = [
      {
        value: "",
        label: t("settings.gateway.detailsPanel.subagents.modelAuto"),
      },
    ];
    const seen = new Set<string>();
    for (const providerWithModels of providersQuery.data ?? []) {
      const providerID = providerWithModels.provider.id.trim();
      if (!providerID) {
        continue;
      }
      const providerLabel = providerWithModels.provider.name?.trim() || providerID;
      for (const model of providerWithModels.models ?? []) {
        if (!model.enabled) {
          continue;
        }
        const modelName = model.name.trim();
        if (!modelName) {
          continue;
        }
        const value = `${providerID}/${modelName}`;
        if (seen.has(value)) {
          continue;
        }
        seen.add(value);
        options.push({
          value,
          label: `${providerLabel} / ${model.displayName?.trim() || modelName}`,
        });
      }
    }
    const currentValue = gateway?.subagents.model?.trim() ?? "";
    if (currentValue && !seen.has(currentValue)) {
      options.push({
        value: currentValue,
        label: `${t("settings.gateway.detailsPanel.subagents.modelCustom")} / ${currentValue}`,
      });
    }
    return options;
  }, [gateway?.subagents.model, providersQuery.data, t]);

  const subagentThinkingOptions = React.useMemo(() => {
    const options: Array<{ value: string; label: string }> = [
      {
        value: "",
        label: t("settings.gateway.detailsPanel.subagents.thinkingAuto"),
      },
      { value: "off", label: "off" },
      { value: "minimal", label: "minimal" },
      { value: "low", label: "low" },
      { value: "medium", label: "medium" },
      { value: "high", label: "high" },
      { value: "xhigh", label: "xhigh" },
    ];
    const currentValue = gateway?.subagents.thinking?.trim() ?? "";
    if (currentValue && !options.some((item) => item.value === currentValue)) {
      options.push({
        value: currentValue,
        label: `${t("settings.gateway.detailsPanel.subagents.thinkingCustom")} / ${currentValue}`,
      });
    }
    return options;
  }, [gateway?.subagents.thinking, t]);

  const normalizeTalkDraft = (draft: TalkConfigEntity): TalkConfigEntity => {
    const trimOrUndefined = (value?: string) => {
      const trimmed = value?.trim() ?? "";
      return trimmed ? trimmed : undefined;
    };
    const aliases = draft.voiceAliases;
    const normalizedAliases = aliases
      ? Object.entries(aliases).reduce<Record<string, string>>((acc, [key, value]) => {
          const trimmedKey = key.trim();
          const trimmedValue = value.trim();
          if (!trimmedKey || !trimmedValue) {
            return acc;
          }
          acc[trimmedKey] = trimmedValue;
          return acc;
        }, {})
      : undefined;
    return {
      voiceId: trimOrUndefined(draft.voiceId),
      voiceAliases:
        normalizedAliases && Object.keys(normalizedAliases).length > 0 ? normalizedAliases : undefined,
      modelId: trimOrUndefined(draft.modelId),
      outputFormat: trimOrUndefined(draft.outputFormat),
      apiKey: trimOrUndefined(draft.apiKey),
      interruptOnSpeech: draft.interruptOnSpeech,
    };
  };

  const normalizeTtsDraft = (draft: TTSConfigEntity): TTSConfigEntity => {
    const trimOrUndefined = (value?: string) => {
      const trimmed = value?.trim() ?? "";
      return trimmed ? trimmed : undefined;
    };
    return {
      providerId: trimOrUndefined(draft.providerId),
      voiceId: trimOrUndefined(draft.voiceId),
      modelId: trimOrUndefined(draft.modelId),
      format: trimOrUndefined(draft.format),
    };
  };

  const commitTalkConfig = async (next: TalkConfigEntity) => {
    const normalized = normalizeTalkDraft(next);
    setTalkDraft(normalized);
    try {
      await setTalkConfig.mutateAsync(normalized);
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.gateway.detailsPanel.talk.updateFailed"),
        description: String(error),
      });
    }
  };

  const commitTTSConfig = async (next: TTSConfigEntity) => {
    const normalized = normalizeTtsDraft(next);
    setTtsDraft(normalized);
    try {
      await setTTSConfig.mutateAsync(normalized);
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.gateway.detailsPanel.tts.updateFailed"),
        description: String(error),
      });
    }
  };

  const commitVoiceWake = async (value: string) => {
    const triggers = parseCommaList(value);
    try {
      await setVoiceWake.mutateAsync({ triggers });
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.gateway.detailsPanel.voiceWakeTriggers.updateFailed"),
        description: String(error),
      });
    }
  };

  const handleCronEnabledChange = (value: boolean) => {
    updateGateway({ cron: { enabled: value } });
  };

  const handleOpenNotificationCenter = React.useCallback(async () => {
    await showMainWindow.mutateAsync();
    await Events.Emit("main:navigate", "notifications");
  }, [showMainWindow]);

  const renderPanel = () => {
    if (activeSection.id === "gateway") {
      return (
        <GatewayCorePanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          execPermissionMode={execPermissionMode}
          permissionModeOptions={permissionModeOptions}
          onExecPermissionModeChange={handleExecPermissionModeChange}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "context") {
      return (
        <GatewayContextPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          contextTab={contextTab}
          onContextTabChange={setContextTab}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "agentLoop") {
      return (
        <GatewayAgentLoopPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "queue") {
      return (
        <GatewayQueuePanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "cron") {
      return (
        <GatewayCronPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          updateGateway={updateGateway}
          onCronEnabledChange={handleCronEnabledChange}
        />
      );
    }
    if (activeSection.id === "heartbeat") {
      return (
        <GatewayHeartbeatPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          heartbeatSessionOptions={heartbeatSessionOptions}
          heartbeatTab={heartbeatTab}
          onHeartbeatTabChange={setHeartbeatTab}
          updateGateway={updateGateway}
          heartbeatSpecDraft={heartbeatSpecDraft}
          setHeartbeatSpecDraft={setHeartbeatSpecDraft}
          heartbeatSpecVersion={heartbeatSpecVersion}
          heartbeatSpecUpdatedAt={heartbeatSpecUpdatedAt}
          heartbeatSpecSaving={heartbeatSpecSaving}
          heartbeatTriggerPending={heartbeatTriggerPending}
          heartbeatTriggerFeedback={heartbeatTriggerFeedback}
          latestHeartbeatStatus={heartbeatLastStatus.data ?? null}
          latestHeartbeatNotice={latestHeartbeatNotice}
          onOpenNotificationCenter={handleOpenNotificationCenter}
          onUpdateHeartbeatSpecItem={updateHeartbeatSpecItem}
          onTriggerHeartbeat={handleTriggerHeartbeat}
          onClearHeartbeatSpec={handleClearHeartbeatSpec}
          onSaveHeartbeatSpec={handleSaveHeartbeatSpec}
        />
      );
    }
    if (activeSection.id === "subagents") {
      return (
        <GatewaySubagentsPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          subagentModelOptions={subagentModelOptions}
          subagentThinkingOptions={subagentThinkingOptions}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "http") {
      return (
        <GatewayHttpPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          httpTab={httpTab}
          onHttpTabChange={setHttpTab}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "talkMode") {
      return (
        <GatewayTalkPanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          talkTab={talkTab}
          onTalkTabChange={setTalkTab}
          talkDraft={talkDraft}
          setTalkDraft={setTalkDraft}
          talkAliasesInput={talkAliasesInput}
          setTalkAliasesInput={setTalkAliasesInput}
          commitTalkConfig={commitTalkConfig}
          ttsStatus={ttsStatus.data}
          ttsDraft={ttsDraft}
          setTtsDraft={setTtsDraft}
          commitTTSConfig={commitTTSConfig}
          updateGateway={updateGateway}
        />
      );
    }
    if (activeSection.id === "voiceWakeTriggers") {
      return (
        <GatewayVoiceWakePanel
          t={t}
          gateway={gateway}
          isDisabled={isDisabled}
          wakeInput={wakeInput}
          setWakeInput={setWakeInput}
          commitVoiceWake={commitVoiceWake}
          updateGateway={updateGateway}
        />
      );
    }
    return null;
  };

  return (
    <div className="flex min-h-0 min-w-0 flex-1 overflow-x-hidden">
      <Card className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
        <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              <SidebarMenu className="gap-1">
                {DETAILS_SECTIONS.map((item) => (
                  <SidebarMenuItem key={item.id}>
                    <SidebarMenuButton
                      isActive={activeSectionId === item.id}
                      onClick={() => setActiveSectionId(item.id)}
                    >
                      <span className="truncate">{t(item.labelKey)}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </div>
            <div className="px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              <div className="flex flex-col items-center gap-3">
                <div className="w-full max-w-[180px]">
                  <Assistant3DAvatar
                    assistant={assistant}
                    className="aspect-square w-full"
                    iconClassName="h-6 w-6"
                  />
                </div>
                <div className="flex w-full items-center gap-2">
                  <div className="flex min-w-0 flex-1 items-center gap-2">
                    <AssistantEmojiPicker assistant={assistant} emojiClassName="text-base" />
                    <span className="min-w-0 truncate text-left text-sm font-semibold uppercase tracking-wide text-foreground">
                      {assistant.identity?.name}
                    </span>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="outline"
                        size="compactIcon"
                        className="h-7 w-7 rounded-full"
                        aria-label={t("settings.gateway.action.switch")}
                      >
                        <ChevronsUpDown className="h-3.5 w-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-56">
                      {assistants.map((item) => {
                        const emoji = item.identity?.emoji?.trim() || "🙂";
                        return (
                          <DropdownMenuCheckboxItem
                            key={item.id}
                            checked={item.id === currentAssistantId}
                            onSelect={(event) => {
                              event.preventDefault();
                              if (item.id !== currentAssistantId) {
                                onSelectAssistant(item.id);
                              }
                            }}
                            className="gap-2"
                          >
                            <span className="text-base">{emoji}</span>
                            <span className="min-w-0 flex-1 truncate">{item.identity?.name}</span>
                          </DropdownMenuCheckboxItem>
                        );
                      })}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </div>
          </div>

          <Separator orientation="vertical" className="self-stretch" />

          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto px-3 py-4">
              {activeSectionDescription ? (
                <p className="mb-3 text-xs text-muted-foreground">{activeSectionDescription}</p>
              ) : null}
              {renderPanel()}
              {!gateway ? (
                <Badge variant="subtle" className="mt-2 w-fit">
                  {t("settings.gateway.detailsPanel.unavailable")}
                </Badge>
              ) : null}
              {isBusy ? (
                <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  {t("settings.gateway.detailsPanel.loading")}
                </div>
              ) : null}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
