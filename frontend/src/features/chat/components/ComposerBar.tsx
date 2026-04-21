import * as React from "react";
import * as WailsRuntime from "@wailsio/runtime";
import {
  ComposerPrimitive,
  useAssistantApi,
  useAssistantState,
  useThreadComposer,
} from "@assistant-ui/react";
import {
  ArrowUp,
  CircleStop,
  PieChart,
  ShieldCheck,
  Unlock,
} from "lucide-react";

import {
  ComposerAddAttachmentButton,
  ComposerAttachments,
} from "@/components/assistant-ui/attachment";
import {
  ModelSelectorContent,
  ModelSelectorRoot,
  ModelSelectorTrigger,
  type ModelOption,
} from "@/components/assistant-ui/model-selector";
import {
  SelectContent as AuiSelectContent,
  SelectItem as AuiSelectItem,
  SelectRoot as AuiSelectRoot,
  SelectTrigger as AuiSelectTrigger,
} from "@/components/assistant-ui/select";
import { cn } from "@/lib/utils";
import { resolvePersistedThreadId } from "@/shared/assistant/thread-identities";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useAssistants, useUpdateAssistant } from "@/shared/query/assistant";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import { useUpdateSettings } from "@/shared/query/settings";
import type { Assistant } from "@/shared/store/assistant";
import {
  useChatRuntimeStore,
  type ContextTokenSnapshot,
} from "@/shared/store/chat-runtime";
import type { ProviderModel, ProviderWithModels } from "@/shared/store/providers";
import { useSettingsStore } from "@/shared/store/settings";
import { Button } from "@/shared/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";

import { useChatReadiness } from "../lib/readiness";
import type { ModelGroup } from "../types";

type SelectedAttachmentFile = {
  path?: string;
  fileName?: string;
  mime?: string;
  sizeBytes?: number;
  contentBase64?: string;
};

const ATTACHMENT_DEFAULT_MIME = "application/octet-stream";
const MODEL_MISSING_VALUE_PREFIX = "__model_ref__:";
const EXEC_PERMISSION_MODE_KEY = "execPermissionMode";
const EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS = "default permissions";
const EXEC_PERMISSION_MODE_FULL_ACCESS = "full access";
const ASSISTANT_TRIGGER_MAX_WIDTH_STYLE: React.CSSProperties = {
  maxWidth: "var(--chat-composer-assistant-max-width, 240px)",
  flexBasis: "auto",
  flexGrow: 0,
  flexShrink: 1,
};
const MODEL_TRIGGER_MAX_WIDTH_STYLE: React.CSSProperties = {
  maxWidth: "var(--chat-composer-model-max-width, 320px)",
  flexBasis: "auto",
  flexGrow: 0,
  flexShrink: 2,
};
const PERMISSION_TRIGGER_MAX_WIDTH_STYLE: React.CSSProperties = {
  maxWidth: "var(--chat-composer-permission-max-width, 220px)",
};

type ExecPermissionMode =
  | typeof EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS
  | typeof EXEC_PERMISSION_MODE_FULL_ACCESS;

const parseModelRef = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) {
    return { providerId: "", modelName: "" };
  }
  const slashIndex = trimmed.indexOf("/");
  if (slashIndex > 0) {
    return {
      providerId: trimmed.slice(0, slashIndex).trim(),
      modelName: trimmed.slice(slashIndex + 1).trim(),
    };
  }
  const colonIndex = trimmed.indexOf(":");
  if (colonIndex > 0) {
    return {
      providerId: trimmed.slice(0, colonIndex).trim(),
      modelName: trimmed.slice(colonIndex + 1).trim(),
    };
  }
  return { providerId: "", modelName: trimmed };
};

const buildModelRef = (providerId: string, modelName: string) => {
  const provider = providerId.trim();
  const model = modelName.trim();
  if (!provider || !model) {
    return "";
  }
  return `${provider}/${model}`;
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const normalizeExecPermissionMode = (value: unknown): ExecPermissionMode => {
  if (typeof value !== "string") {
    return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
  }
  const normalized = value.trim().toLowerCase();
  if (
    normalized === EXEC_PERMISSION_MODE_FULL_ACCESS ||
    normalized === "full_access" ||
    normalized === "all-access" ||
    normalized === "all_access" ||
    normalized === "all access" ||
    normalized === "full"
  ) {
    return EXEC_PERMISSION_MODE_FULL_ACCESS;
  }
  if (
    normalized === EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS ||
    normalized === "default_permissions" ||
    normalized === "default permission" ||
    normalized === "default" ||
    normalized === "standard" ||
    normalized === "safe" ||
    normalized === "ask"
  ) {
    return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
  }
  return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
};

const resolveToolsConfig = (raw: unknown) => (isRecord(raw) ? { ...raw } : {});

const resolveExecPermissionMode = (callsToolsRaw: unknown): ExecPermissionMode => {
  const toolsConfig = resolveToolsConfig(callsToolsRaw);
  const direct = toolsConfig[EXEC_PERMISSION_MODE_KEY];
  if (direct !== undefined) {
    return normalizeExecPermissionMode(direct);
  }
  const fallback = toolsConfig.permissionMode;
  if (fallback !== undefined) {
    return normalizeExecPermissionMode(fallback);
  }
  return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
};

const patchCallsToolsExecPermissionMode = (callsToolsRaw: unknown, mode: ExecPermissionMode) => {
  const toolsConfig = resolveToolsConfig(callsToolsRaw);
  return {
    ...toolsConfig,
    [EXEC_PERMISSION_MODE_KEY]: mode,
  };
};

type ThreadContextTokensResponse = {
  promptTokens?: number;
  totalTokens?: number;
  contextWindowTokens?: number;
  updatedAt?: string;
  fresh?: boolean;
};

const toFiniteTokenValue = (value: unknown): number => {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.floor(value));
};

const isCancelledErrorMessage = (error: unknown) => {
  const message = error instanceof Error ? error.message : String(error ?? "");
  return message.toLowerCase().includes("cancel");
};

const parseThreadContextSnapshot = (payload: unknown): ContextTokenSnapshot | null => {
  if (!payload || typeof payload !== "object") {
    return null;
  }
  const data = payload as ThreadContextTokensResponse;
  const promptTokens = toFiniteTokenValue(data.promptTokens);
  const totalTokens = toFiniteTokenValue(data.totalTokens);
  const contextWindowTokens = toFiniteTokenValue(data.contextWindowTokens);
  const updatedAtRaw = typeof data.updatedAt === "string" ? data.updatedAt.trim() : "";
  const updatedAt = updatedAtRaw ? Date.parse(updatedAtRaw) : Number.NaN;
  if (promptTokens <= 0 && totalTokens <= 0 && contextWindowTokens <= 0 && !Number.isFinite(updatedAt)) {
    return null;
  }
  return {
    promptTokens,
    totalTokens,
    contextWindowTokens: contextWindowTokens > 0 ? contextWindowTokens : undefined,
    contextFresh: data.fresh !== false,
    updatedAt: Number.isFinite(updatedAt) ? updatedAt : undefined,
  };
};

const formatTokenCount = (value: number | undefined): string => {
  const normalized = Number(value ?? 0);
  if (!Number.isFinite(normalized) || normalized <= 0) {
    return "0";
  }
  return Math.round(normalized).toLocaleString();
};

const toModelGroups = (items: ProviderWithModels[]): ModelGroup[] => {
  const map = new Map<string, ModelGroup>();
  for (const entry of items) {
    const visibleModels = entry.models.filter((model) => model.showInUi !== false);
    if (visibleModels.length === 0) {
      continue;
    }
    const existing = map.get(entry.provider.id);
    if (existing) {
      existing.models.push(...visibleModels);
      continue;
    }
    map.set(entry.provider.id, {
      provider: entry.provider,
      models: [...visibleModels],
    });
  }
  return Array.from(map.values()).sort((left, right) =>
    left.provider.name.localeCompare(right.provider.name)
  );
};

const modelDisplayName = (model: ProviderModel | null | undefined) => {
  if (!model) {
    return "";
  }
  return model.displayName?.trim() || model.name;
};

const assistantSelectLabel = (assistant: Assistant, fallback: string) => {
  const name = assistant.identity?.name?.trim() || fallback;
  const emoji = assistant.identity?.emoji?.trim();
  return emoji ? `${emoji} ${name}` : name;
};

const assistantSelectEmoji = (assistant: Assistant | null | undefined) =>
  assistant?.identity?.emoji?.trim() || "";

function ThemedProviderIcon({ icon, className }: { icon: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn("size-4 shrink-0 bg-current text-muted-foreground", className)}
      style={{
        WebkitMaskImage: `url(${icon})`,
        maskImage: `url(${icon})`,
        WebkitMaskRepeat: "no-repeat",
        maskRepeat: "no-repeat",
        WebkitMaskPosition: "center",
        maskPosition: "center",
        WebkitMaskSize: "contain",
        maskSize: "contain",
      }}
    />
  );
}

const decodeBase64ToBytes = (value: string) => {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
};

const resolveAttachmentName = (item: SelectedAttachmentFile) => {
  const direct = item.fileName?.trim();
  if (direct) {
    return direct;
  }
  const path = item.path?.trim() ?? "";
  if (path) {
    const normalized = path.replace(/\\/g, "/");
    const segments = normalized.split("/").filter(Boolean);
    const last = segments[segments.length - 1];
    if (last) {
      return last;
    }
  }
  return "attachment";
};

export function useRuntimeSelections() {
  const { data: providersWithModels = [], isLoading: providersLoading } = useEnabledProvidersWithModels();
  const { data: assistants = [], isLoading: assistantsLoading } = useAssistants(false);
  const assistantId = useChatRuntimeStore((state) => state.assistantId);
  const setAssistantId = useChatRuntimeStore((state) => state.setAssistantId);

  const modelGroups = React.useMemo(() => toModelGroups(providersWithModels), [providersWithModels]);
  const selectedAssistant = React.useMemo(
    () => assistants.find((assistant) => assistant.id === assistantId) ?? null,
    [assistantId, assistants]
  );
  const agentModelRef = selectedAssistant?.model?.agent?.primary?.trim() ?? "";
  const parsedAgent = React.useMemo(() => parseModelRef(agentModelRef), [agentModelRef]);
  const selectedProviderGroup = React.useMemo(
    () => modelGroups.find((group) => group.provider.id === parsedAgent.providerId) ?? null,
    [modelGroups, parsedAgent.providerId]
  );
  const selectedModel = React.useMemo(
    () => selectedProviderGroup?.models.find((model) => model.name === parsedAgent.modelName) ?? null,
    [parsedAgent.modelName, selectedProviderGroup]
  );

  React.useEffect(() => {
    if (assistants.length === 0) {
      return;
    }
    if (assistants.some((assistant) => assistant.id === assistantId)) {
      return;
    }
    const fallbackAssistant = assistants.find((assistant) => assistant.isDefault) ?? assistants[0];
    if (fallbackAssistant) {
      setAssistantId(fallbackAssistant.id);
    }
  }, [assistantId, assistants, setAssistantId]);

  return {
    assistants,
    assistantsLoading,
    assistantId,
    selectedAssistant,
    setAssistantId,
    modelGroups,
    providersLoading,
    agentModelRef,
    agentProviderId: parsedAgent.providerId,
    agentModelName: parsedAgent.modelName,
    selectedProvider: selectedProviderGroup?.provider ?? null,
    selectedModel,
  };
}

export function useComposerRunConfig() {
  const api = useAssistantApi();
  const threadId = useAssistantState(({ threadListItem }) =>
    resolvePersistedThreadId(threadListItem.remoteId, threadListItem.id)
  );

  React.useEffect(() => {
    if (!threadId) {
      return;
    }
    const custom: Record<string, unknown> = {
      threadId,
    };
    api.composer().setRunConfig({ custom });
  }, [api, threadId]);
}

type ComposerBarProps = {
  assistants: Assistant[];
  assistantId: string;
  selectedAssistant: Assistant | null;
  setAssistantId: (assistantId: string) => void;
  modelGroups: ModelGroup[];
  agentProviderId: string;
  agentModelName: string;
  loading: boolean;
};

type AssistantModelDropdownProps = {
  assistants: Assistant[];
  assistantId: string;
  selectedAssistant: Assistant | null;
  setAssistantId: (assistantId: string) => void;
  modelGroups: ModelGroup[];
  agentProviderId: string;
  agentModelName: string;
  disabled?: boolean;
  mode: "assistant" | "model";
};

function AssistantModelDropdown({
  assistants,
  assistantId,
  selectedAssistant,
  setAssistantId,
  modelGroups,
  agentProviderId,
  agentModelName,
  disabled,
  mode,
}: AssistantModelDropdownProps) {
  const { t } = useI18n();
  const updateAssistant = useUpdateAssistant();
  const modelOptions = React.useMemo(
    () =>
      modelGroups.flatMap((group) =>
        group.models.map((model) => ({
          value: model.id,
          label: `${group.provider.name} / ${modelDisplayName(model)}`,
          providerId: model.providerId,
          modelName: model.name,
          modelRef: buildModelRef(model.providerId, model.name),
          providerName: group.provider.name,
          providerIcon: group.provider.icon?.trim() ?? "",
          modelLabel: modelDisplayName(model),
        }))
      ),
    [modelGroups]
  );
  const selectedModelValue = React.useMemo(() => {
    const providerID = agentProviderId.trim();
    const modelName = agentModelName.trim();
    if (!modelName) {
      return "";
    }
    if (!providerID) {
      const matched = modelOptions.find(
        (option) =>
          option.value === modelName ||
          option.modelName.trim().toLowerCase() === modelName.toLowerCase()
      );
      return matched?.value ?? `${MODEL_MISSING_VALUE_PREFIX}${modelName}`;
    }
    const matched = modelOptions.find(
      (option) =>
        option.providerId.trim().toLowerCase() === providerID.toLowerCase() &&
        option.modelName.trim().toLowerCase() === modelName.toLowerCase()
    );
    return matched?.value ?? `${MODEL_MISSING_VALUE_PREFIX}${buildModelRef(providerID, modelName)}`;
  }, [agentModelName, agentProviderId, modelOptions]);
  const selectedModelMissingInOptions = selectedModelValue.startsWith(MODEL_MISSING_VALUE_PREFIX);
  const selectedModelProviderLabel = React.useMemo(() => {
    if (!agentProviderId) {
      const fallback = modelOptions.find(
        (option) =>
          option.value === agentModelName ||
          option.modelName.trim().toLowerCase() === agentModelName.trim().toLowerCase()
      );
      return fallback?.providerName ?? "";
    }
    const providerGroup = modelGroups.find((group) => group.provider.id === agentProviderId);
    return providerGroup?.provider.name ?? agentProviderId;
  }, [agentModelName, agentProviderId, modelGroups, modelOptions]);
  const selectedModelLabel = React.useMemo(() => {
    if (!agentModelName) {
      return t("chat.composer.selectModel");
    }
    if (!agentProviderId) {
      const fallback = modelOptions.find(
        (option) =>
          option.value === agentModelName ||
          option.modelName.trim().toLowerCase() === agentModelName.trim().toLowerCase()
      );
      return fallback?.modelLabel ?? agentModelName;
    }
    const providerGroup = modelGroups.find((group) => group.provider.id === agentProviderId);
    const model = providerGroup?.models.find((entry) => entry.name === agentModelName);
    return modelDisplayName(model) || agentModelName;
  }, [agentModelName, agentProviderId, modelGroups, modelOptions, t]);
  const selectedModelOption = React.useMemo(
    () => modelOptions.find((option) => option.value === selectedModelValue) ?? null,
    [modelOptions, selectedModelValue]
  );
  const modelSelectorOptions = React.useMemo<ModelOption[]>(() => {
    const result: ModelOption[] = modelOptions.map((option) => ({
      id: option.value,
      name: option.modelLabel,
      description: option.providerName,
      icon: option.providerIcon ? (
        <ThemedProviderIcon icon={option.providerIcon} />
      ) : undefined,
      disabled: false,
    }));
    if (
      selectedModelMissingInOptions &&
      selectedModelValue &&
      !result.some((item) => item.id === selectedModelValue)
    ) {
      result.unshift({
        id: selectedModelValue,
        name: selectedModelLabel,
        description: selectedModelProviderLabel || undefined,
        disabled: false,
      });
    }
    return result;
  }, [
    modelOptions,
    selectedModelLabel,
    selectedModelMissingInOptions,
    selectedModelProviderLabel,
    selectedModelValue,
  ]);

  const commitModel = async (nextValue: string) => {
    if (!selectedAssistant) {
      return;
    }
    const normalizedNextValue = nextValue.trim();
    if (!normalizedNextValue || normalizedNextValue === selectedModelValue) {
      return;
    }
    let primary = "";
    if (normalizedNextValue) {
      const selectedOption = modelOptions.find((option) => option.value === normalizedNextValue);
      if (selectedOption) {
        primary = selectedOption.modelRef;
      } else if (normalizedNextValue.startsWith(MODEL_MISSING_VALUE_PREFIX)) {
        const rawModelRef = normalizedNextValue.slice(MODEL_MISSING_VALUE_PREFIX.length).trim();
        const parsed = parseModelRef(rawModelRef);
        primary = parsed.providerId ? buildModelRef(parsed.providerId, parsed.modelName) : rawModelRef;
      }
    }
    try {
      await updateAssistant.mutateAsync({
        id: selectedAssistant.id,
        model: {
          ...selectedAssistant.model,
          agent: {
            ...(selectedAssistant.model?.agent ?? {}),
            primary,
          },
        },
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };

  if (mode === "assistant") {
    const selectedAssistantLabel = selectedAssistant
      ? assistantSelectLabel(selectedAssistant, t("chat.composer.untitledAssistant"))
      : t("chat.composer.selectAssistant");
    const selectedAssistantEmoji = assistantSelectEmoji(selectedAssistant);
    const selectedAssistantTriggerText = selectedAssistantEmoji || selectedAssistantLabel;
    return (
      <AuiSelectRoot
        value={assistantId}
        onValueChange={setAssistantId}
        disabled={disabled || assistants.length === 0}
      >
        <AuiSelectTrigger
          variant="ghost"
          size="sm"
          title={selectedAssistantLabel}
          aria-label={selectedAssistantLabel}
          className="w-fit min-w-0 max-w-full focus-visible:ring-0 focus-visible:ring-offset-0 [&>svg]:hidden"
          style={ASSISTANT_TRIGGER_MAX_WIDTH_STYLE}
        >
          <span
            className={cn(
              "min-w-0 truncate",
              selectedAssistantEmoji ? "text-base leading-none" : undefined
            )}
          >
            {selectedAssistantTriggerText}
          </span>
        </AuiSelectTrigger>
        <AuiSelectContent align="center" className="max-w-[320px]">
          {assistants.length === 0 ? (
            <AuiSelectItem value="__assistant_empty__" disabled>
              {t("chat.composer.assistantEmpty")}
            </AuiSelectItem>
          ) : (
            assistants.map((assistant) => (
              <AuiSelectItem key={assistant.id} value={assistant.id}>
                {assistantSelectLabel(assistant, t("chat.composer.untitledAssistant"))}
              </AuiSelectItem>
            ))
          )}
        </AuiSelectContent>
      </AuiSelectRoot>
    );
  }

  return (
    <ModelSelectorRoot
      models={modelSelectorOptions}
      value={selectedModelValue}
      disabled={disabled || !selectedAssistant || updateAssistant.isPending}
      onValueChange={(value) => {
        void commitModel(value);
      }}
    >
      <ModelSelectorTrigger
        variant="ghost"
        size="sm"
        title={selectedModelLabel}
        className="w-fit min-w-0 max-w-full focus-visible:ring-0 focus-visible:ring-offset-0 [&>svg]:hidden"
        style={MODEL_TRIGGER_MAX_WIDTH_STYLE}
        disabled={disabled || !selectedAssistant || updateAssistant.isPending}
      >
        <div className="flex min-w-0 items-center gap-1.5 whitespace-nowrap">
          {selectedModelOption?.providerIcon ? (
            <ThemedProviderIcon icon={selectedModelOption.providerIcon} />
          ) : null}
          <span className="min-w-0 truncate">{selectedModelLabel}</span>
        </div>
      </ModelSelectorTrigger>
      <ModelSelectorContent align="center" className="max-w-[420px]" />
    </ModelSelectorRoot>
  );
}

type PermissionModeSelectorProps = {
  value: ExecPermissionMode;
  options: Array<{
    value: ExecPermissionMode;
    label: string;
    Icon: typeof ShieldCheck;
  }>;
  disabled?: boolean;
  onChange: (nextValue: ExecPermissionMode) => void;
  title: string;
};

function PermissionModeSelector({
  value,
  options,
  disabled,
  onChange,
  title,
}: PermissionModeSelectorProps) {
  const selected = options.find((item) => item.value === value) ?? options[0];
  return (
    <AuiSelectRoot
      value={selected.value}
      onValueChange={(nextValue) => onChange(nextValue as ExecPermissionMode)}
      disabled={disabled}
    >
      <AuiSelectTrigger
        variant="ghost"
        size="sm"
        title={title}
        aria-label={title}
        className="w-fit min-w-0 [&>svg]:hidden"
        style={PERMISSION_TRIGGER_MAX_WIDTH_STYLE}
      >
        <span className="inline-flex items-center justify-center">
          <selected.Icon className="size-4 text-muted-foreground" />
        </span>
      </AuiSelectTrigger>
      <AuiSelectContent align="center" className="max-w-[260px]">
        {options.map((option) => (
          <AuiSelectItem key={option.value} value={option.value}>
            <span className="flex items-center gap-2 whitespace-nowrap">
              <option.Icon className="size-4 text-muted-foreground" />
              <span className="truncate">{option.label}</span>
            </span>
          </AuiSelectItem>
        ))}
      </AuiSelectContent>
    </AuiSelectRoot>
  );
}

export function ComposerBar({
  assistants,
  assistantId,
  selectedAssistant,
  setAssistantId,
  modelGroups,
  agentProviderId,
  agentModelName,
  loading,
}: ComposerBarProps) {
  const api = useAssistantApi();
  const { t } = useI18n();
  const isRunning = useAssistantState(({ thread }) => thread.isRunning);
  const isDisabled = useAssistantState(
    ({ thread, composer }) => thread.isDisabled || (composer.dictation?.inputDisabled ?? false)
  );
  const activeThreadId = useAssistantState(({ threadListItem }) =>
    resolvePersistedThreadId(threadListItem.remoteId, threadListItem.id)
  );
  const hasActiveThread = (activeThreadId ?? "").trim().length > 0;
  const setContextTokens = useChatRuntimeStore((state) => state.setContextTokens);
  const contextSnapshot = useChatRuntimeStore((state) => state.contextTokens[activeThreadId]);
  const lastRunUsage = useChatRuntimeStore((state) => state.runUsage[activeThreadId]);
  const selectedModelContextLimit = React.useMemo(() => {
    if (!agentProviderId || !agentModelName) {
      return 0;
    }
    const providerGroup = modelGroups.find((group) => group.provider.id === agentProviderId);
    const model = providerGroup?.models.find((entry) => entry.name === agentModelName);
    return model?.contextWindowTokens && model.contextWindowTokens > 0 ? model.contextWindowTokens : 0;
  }, [agentModelName, agentProviderId, modelGroups]);
  const updateSettings = useUpdateSettings();
  const callsToolsRaw = useSettingsStore((state) => state.settings?.tools);
  const execPermissionMode = React.useMemo(
    () => resolveExecPermissionMode(callsToolsRaw),
    [callsToolsRaw]
  );
  const contextTotal = contextSnapshot?.totalTokens ?? 0;
  const contextLimit =
    selectedModelContextLimit > 0
      ? selectedModelContextLimit
      : contextSnapshot?.contextWindowTokens && contextSnapshot.contextWindowTokens > 0
        ? contextSnapshot.contextWindowTokens
        : 0;
  const contextUsageRatio = contextLimit > 0 ? contextTotal / contextLimit : 0;
  const contextUsagePercent = contextLimit > 0 ? Math.max(0, Math.round(contextUsageRatio * 100)) : 0;
  const contextWarnActive = contextLimit > 0 && contextUsageRatio >= 0.8;
  const contextHardActive = contextLimit > 0 && contextUsageRatio >= 1;
  const contextUsageColor = contextHardActive
    ? "text-destructive"
    : contextWarnActive
      ? "text-amber-500"
      : "text-muted-foreground";
  const showContextUsage = hasActiveThread;
  const contextUsageText = `${contextUsagePercent}%`;
  const contextTotalText = contextTotal.toLocaleString();
  const contextLimitText =
    contextLimit > 0 ? contextLimit.toLocaleString() : t("chat.composer.contextTotalUnknown");
  const latestPromptTokensText = formatTokenCount(lastRunUsage?.promptTokens);
  const latestCompletionTokensText = formatTokenCount(lastRunUsage?.completionTokens);
  const latestTotalTokensText = formatTokenCount(lastRunUsage?.totalTokens);

  React.useEffect(() => {
    const threadID = (activeThreadId ?? "").trim();
    if (!threadID) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const result = await WailsRuntime.Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.GetThreadContextTokens",
          threadID
        );
        if (cancelled) {
          return;
        }
        const snapshot = parseThreadContextSnapshot(result);
        if (!snapshot) {
          return;
        }
        const current = useChatRuntimeStore.getState().contextTokens[threadID];
        if (current?.updatedAt && snapshot.updatedAt && current.updatedAt > snapshot.updatedAt) {
          return;
        }
        setContextTokens(threadID, {
          ...snapshot,
          warnTokens: current?.warnTokens,
          hardTokens: current?.hardTokens,
        });
      } catch {
        // Best effort hydration for thread switching; runtime stream remains source of truth.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [activeThreadId, agentProviderId, agentModelName, setContextTokens]);

  const readiness = useChatReadiness({
    assistant: selectedAssistant,
    modelGroups,
    loading,
  });
  const composerText = useThreadComposer((state) => state.text);
  const sendDisabled =
    isDisabled || readiness.isChecking || !readiness.isReady || composerText.trim().length === 0;

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
              intent: "danger",
              title: t("chat.composer.permission.updateError"),
              description: error instanceof Error ? error.message : String(error ?? ""),
            });
          },
        }
      );
    },
    [callsToolsRaw, execPermissionMode, t, updateSettings]
  );

  const permissionModeOptions = React.useMemo(
    () => [
      {
        value: EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS as ExecPermissionMode,
        label: t("chat.composer.permission.standard"),
        Icon: ShieldCheck,
      },
      {
        value: EXEC_PERMISSION_MODE_FULL_ACCESS as ExecPermissionMode,
        label: t("chat.composer.permission.allAccess"),
        Icon: Unlock,
      },
    ],
    [t]
  );
  const selectedPermissionMode =
    permissionModeOptions.find((item) => item.value === execPermissionMode) ?? permissionModeOptions[0];
  const permissionLabel = selectedPermissionMode?.label ?? t("chat.composer.permission.label");

  const addSelectedAttachments = React.useCallback(
    async (picked: SelectedAttachmentFile[]) => {
      if (picked.length === 0) {
        return;
      }
      let addedCount = 0;
      for (const item of picked) {
        const contentBase64 = item.contentBase64?.trim() ?? "";
        if (!contentBase64) {
          continue;
        }
        const bytes = decodeBase64ToBytes(contentBase64);
        const mime = item.mime?.trim() || ATTACHMENT_DEFAULT_MIME;
        const file = new File([bytes], resolveAttachmentName(item), { type: mime });
        await api.composer().addAttachment(file);
        addedCount += 1;
      }
      if (addedCount === 0) {
        throw new Error(t("chat.composer.attachNoValidFile"));
      }
    },
    [api, t]
  );

  const handleAddAttachment = React.useCallback(() => {
    void (async () => {
      try {
        const selection = await WailsRuntime.Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.SelectAttachmentFiles",
          t("chat.composer.attachDialogTitle")
        );
        const picked = (Array.isArray(selection) ? selection : []) as SelectedAttachmentFile[];
        if (picked.length === 0) {
          return;
        }
        await addSelectedAttachments(picked);
      } catch (error) {
        if (isCancelledErrorMessage(error)) {
          return;
        }
        const message = error instanceof Error ? error.message : String(error);
        messageBus.publishToast({
          title: t("chat.composer.attachError"),
          description: message,
          intent: "warning",
        });
      }
    })();
  }, [addSelectedAttachments, t]);

  React.useEffect(() => {
    const offDropped = WailsRuntime.Events.On("chat:attachments-dropped", (event: any) => {
      const payload = event?.data ?? event;
      const files = (payload?.files ?? []) as SelectedAttachmentFile[];
      if (files.length === 0) {
        return;
      }
      void (async () => {
        try {
          await addSelectedAttachments(files);
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          messageBus.publishToast({
            title: t("chat.composer.attachError"),
            description: message,
            intent: "warning",
          });
        }
      })();
    });
    const offError = WailsRuntime.Events.On("chat:attachments-drop-error", (event: any) => {
      const message = (event?.data ?? event) as string;
      if (!message) {
        return;
      }
      messageBus.publishToast({
        title: t("chat.composer.attachError"),
        description: message,
        intent: "warning",
      });
    });

    return () => {
      offDropped();
      offError();
    };
  }, [addSelectedAttachments, t]);

  const handleAttachmentDrop = React.useCallback(
    (event: React.DragEvent<HTMLTextAreaElement>) => {
      if (typeof window !== "undefined" && (window as any)?._wails?.flags?.enableFileDrop) {
        return;
      }
      if (event.defaultPrevented) {
        return;
      }
      const files = event.dataTransfer?.files;
      if (!files || files.length === 0) {
        return;
      }
      event.preventDefault();
      void (async () => {
        try {
          let addedCount = 0;
          for (const file of Array.from(files)) {
            await api.composer().addAttachment(file);
            addedCount += 1;
          }
          if (addedCount === 0) {
            throw new Error(t("chat.composer.attachNoValidFile"));
          }
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          messageBus.publishToast({
            title: t("chat.composer.attachError"),
            description: message,
            intent: "warning",
          });
        }
      })();
    },
    [api, t]
  );

  const handleAttachmentDragOver = React.useCallback((event: React.DragEvent<HTMLTextAreaElement>) => {
    if (typeof window !== "undefined" && (window as any)?._wails?.flags?.enableFileDrop) {
      return;
    }
    const types = event.dataTransfer?.types;
    if (types && Array.from(types).includes("Files")) {
      event.preventDefault();
    }
  }, []);

  const handleComposerInputKeyDown = React.useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key !== "Enter" || event.shiftKey) {
        return;
      }
      const nativeEvent = event.nativeEvent as KeyboardEvent & { keyCode?: number; which?: number };
      // Some IME flows report Enter with keyCode=229 while committing composition.
      const imeComposing = nativeEvent.isComposing || nativeEvent.keyCode === 229 || nativeEvent.which === 229;
      if (imeComposing || isRunning) {
        return;
      }
      event.preventDefault();
      event.currentTarget.closest("form")?.requestSubmit();
    },
    [isRunning]
  );

  return (
    <ComposerPrimitive.AttachmentDropzone className="wails-no-drag">
      <ComposerPrimitive.Root
        data-file-drop-target="chat-composer"
        className="flex w-full flex-col gap-2 rounded-2xl border border-border/60 bg-card/90 p-[var(--app-sidebar-padding)] shadow-sm"
      >
        <ComposerAttachments />
        <ComposerPrimitive.Input
          placeholder={t("chat.composer.placeholder")}
          disabled={isDisabled}
          submitOnEnter={false}
          onKeyDown={handleComposerInputKeyDown}
          onDrop={handleAttachmentDrop}
          onDragOver={handleAttachmentDragOver}
          className={cn(
            "min-h-[44px] w-full resize-none rounded-md bg-transparent px-1 py-2 text-sm text-foreground",
            "focus-visible:outline-none"
          )}
        />
        <div className="flex w-full min-w-0 items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2 overflow-hidden">
            <div className="shrink-0">
              <ComposerAddAttachmentButton
                aria-label={t("chat.composer.attach")}
                disabled={isRunning}
                onClick={handleAddAttachment}
              />
            </div>
            <AssistantModelDropdown
              assistants={assistants}
              assistantId={assistantId}
              selectedAssistant={selectedAssistant}
              setAssistantId={setAssistantId}
              modelGroups={modelGroups}
              agentProviderId={agentProviderId}
              agentModelName={agentModelName}
              disabled={isRunning}
              mode="assistant"
            />
            <div className="shrink-0">
              <PermissionModeSelector
                value={execPermissionMode}
                options={permissionModeOptions}
                disabled={isRunning || updateSettings.isPending}
                onChange={handleExecPermissionModeChange}
                title={permissionLabel}
              />
            </div>
            {showContextUsage ? (
              <div className="shrink-0">
                <TooltipProvider delayDuration={0}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        className="inline-flex h-7 items-center gap-1 rounded-md px-1.5 hover:bg-muted/40"
                        aria-label={t("chat.composer.contextUsage")}
                      >
                        <PieChart className={cn("h-4 w-4", contextUsageColor)} aria-hidden />
                        <span className={cn("text-xs font-medium tabular-nums", contextUsageColor)}>
                          {contextUsageText}
                        </span>
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="top" align="start">
                      <div className="space-y-1 text-xs">
                        <div>
                          {t("chat.composer.contextCurrent")}: {contextTotalText}
                        </div>
                        <div>
                          {t("chat.composer.contextTotal")}: {contextLimitText}
                        </div>
                        <div>
                          {t("chat.composer.contextUsageShort")}: {contextUsageText}
                        </div>
                        {lastRunUsage ? (
                          <>
                            <div className="pt-1 text-muted-foreground">
                              {t("chat.composer.lastRunUsage")}
                            </div>
                            <div>
                              {t("chat.composer.lastInputTokens")}: {latestPromptTokensText}
                            </div>
                            <div>
                              {t("chat.composer.lastOutputTokens")}: {latestCompletionTokensText}
                            </div>
                            <div>
                              {t("chat.composer.lastTotalTokens")}: {latestTotalTokensText}
                            </div>
                          </>
                        ) : null}
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
            ) : null}
            <AssistantModelDropdown
              assistants={assistants}
              assistantId={assistantId}
              selectedAssistant={selectedAssistant}
              setAssistantId={setAssistantId}
              modelGroups={modelGroups}
              agentProviderId={agentProviderId}
              agentModelName={agentModelName}
              disabled={isRunning}
              mode="model"
            />
          </div>

          <div className="flex shrink-0 items-center gap-2">
            {isRunning ? (
              <TooltipProvider delayDuration={0}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <ComposerPrimitive.Cancel asChild>
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className="h-9 w-9 rounded-full border border-border/60 bg-background/78 text-muted-foreground shadow-sm hover:bg-accent/60 hover:text-foreground"
                        aria-label={t("chat.composer.stop")}
                        title={t("chat.composer.stop")}
                      >
                        <CircleStop className="h-4 w-4" />
                      </Button>
                    </ComposerPrimitive.Cancel>
                  </TooltipTrigger>
                  <TooltipContent side="top">{t("chat.composer.stop")}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : null}
            <TooltipProvider delayDuration={0}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <ComposerPrimitive.Send asChild disabled={sendDisabled}>
                    <Button
                      type="submit"
                      size="icon"
                      variant="secondary"
                      className="h-9 w-9 rounded-full border border-border/50 shadow-sm"
                      disabled={sendDisabled}
                      aria-label={t("chat.composer.send")}
                      title={t("chat.composer.send")}
                    >
                      <ArrowUp className="h-4 w-4" />
                    </Button>
                  </ComposerPrimitive.Send>
                </TooltipTrigger>
                <TooltipContent side="top">{t("chat.composer.send")}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      </ComposerPrimitive.Root>
    </ComposerPrimitive.AttachmentDropzone>
  );
}
