import * as React from "react";
import {
  AudioLines,
  Brain,
  Clipboard,
  Eye,
  EyeOff,
  HelpCircle,
  Image,
  Loader2,
  Minus,
  Plus,
  RefreshCw,
  RotateCcw,
  Ruler,
  Search,
  Settings2,
  Trash2,
  Video,
  Wrench,
} from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Item, ItemActions, ItemContent, ItemGroup, ItemTitle } from "@/shared/ui/item";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import { SETTINGS_ROW_CLASS, SETTINGS_ROW_LABEL_CLASS, SETTINGS_WIDE_CONTROL_WIDTH_CLASS, SettingsSeparator } from "@/shared/ui/settings-layout";
import { Switch } from "@/shared/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message/store";
import { cn } from "@/lib/utils";
import { formatContextWindow, parseModelMeta } from "@/shared/utils/modelMeta";
import {
  useDeleteProvider,
  useProviderModels,
  useProviders,
  useProviderSecret,
  useReplaceProviderModels,
  useSyncProviderModels,
  useUpdateProviderModel,
  useUpsertProvider,
  useUpsertProviderSecret,
} from "@/shared/query/providers";
import type { Provider, ProviderModel } from "@/shared/store/providers";

interface ProviderState {
  apiKey: string;
  orgRef: string;
  modelQuery: string;
  isDirty: boolean;
  showApiKey: boolean;
}

interface CustomProviderState {
  providerId: string;
  name: string;
  endpoint: string;
  type: string;
  compatibility: string;
  isDirty: boolean;
}

interface NewProviderDraft {
  name: string;
  type: string;
  compatibility: string;
  endpoint: string;
}

interface EditableProviderModel {
  draftKey: string;
  originalName: string;
  id: string;
  name: string;
  displayName: string;
  capabilitiesJson: string;
  contextWindowTokens?: number;
  maxOutputTokens?: number;
  supportsTools?: boolean;
  supportsReasoning?: boolean;
  supportsVision?: boolean;
  supportsAudio?: boolean;
  supportsVideo?: boolean;
  enabled: boolean;
  showInUi: boolean;
}

const PROVIDER_DEFAULTS = [
  { id: "deepseek", label: "DeepSeek", endpoint: "https://api.deepseek.com", enabled: false, type: "openai", compatibility: "deepseek" },
  { id: "openrouter", label: "OpenRouter", endpoint: "https://openrouter.ai/api/v1", enabled: false, type: "openai", compatibility: "openrouter" },
  { id: "openai", label: "OpenAI", endpoint: "https://api.openai.com/v1", enabled: false, type: "openai", compatibility: "openai" },
  { id: "anthropic", label: "Anthropic", endpoint: "https://api.anthropic.com/v1", enabled: false, type: "anthropic", compatibility: "anthropic" },
  { id: "google", label: "Google Gemini", endpoint: "https://generativelanguage.googleapis.com/v1beta/openai", enabled: false, type: "openai", compatibility: "google" },
  { id: "aihubmix", label: "AIHubMix", endpoint: "https://aihubmix.com/v1", enabled: false, type: "openai", compatibility: "openai" },
  { id: "moonshotai", label: "Moonshot AI", endpoint: "https://api.moonshot.ai/v1", enabled: false, type: "openai", compatibility: "openai" },
  { id: "zai", label: "Z.AI", endpoint: "https://api.z.ai/api/paas/v4", enabled: false, type: "openai", compatibility: "openai" },
  { id: "github-copilot", label: "GitHub Copilot", endpoint: "https://api.githubcopilot.com", enabled: false, type: "openai", compatibility: "openai" },
];
const API_TYPE_OPTIONS = [
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
];
const API_COMPATIBILITY_OPTIONS: Record<string, Array<{ value: string; label: string }>> = {
  openai: [
    { value: "openai", label: "OpenAI Compatible" },
    { value: "deepseek", label: "DeepSeek Compatible" },
    { value: "openrouter", label: "OpenRouter Compatible" },
    { value: "google", label: "Google Gemini Compatible" },
  ],
  anthropic: [{ value: "anthropic", label: "Anthropic Compatible" }],
};

const defaultCompatibilityForType = (type: string) =>
  API_COMPATIBILITY_OPTIONS[type]?.[0]?.value ?? "openai";

const extractErrorMessage = (error: unknown): string => {
  if (!error) {
    return "";
  }
  if (typeof error === "string") {
    return error;
  }
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof (error as { message?: unknown }).message === "string") {
    return (error as { message: string }).message;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return "";
  }
};

const formatModelsSyncError = (error: unknown): string => {
  const raw = extractErrorMessage(error);
  if (!raw) {
    return "";
  }
  const marker = "models request failed:";
  const index = raw.indexOf(marker);
  const detail = index >= 0 ? raw.slice(index + marker.length).trim() : raw.trim();
  if (!detail) {
    return raw.trim();
  }
  try {
    const parsed = JSON.parse(detail) as { error?: { message?: unknown } };
    if (typeof parsed?.error?.message === "string") {
      return parsed.error.message;
    }
  } catch {
    // ignore parse failures
  }
  return detail;
};

export function ProviderSection() {
  const { t } = useI18n();
  const { data: providers = [], isLoading: providersLoading } = useProviders();
  const [activeProviderId, setActiveProviderId] = React.useState<string | null>(null);
  const [stateByProvider, setStateByProvider] = React.useState<Record<string, ProviderState>>({});
  const [customDraft, setCustomDraft] = React.useState<CustomProviderState | null>(null);
  const [providerQuery, setProviderQuery] = React.useState("");
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
  const [resetDialogOpen, setResetDialogOpen] = React.useState(false);
  const [resetPending, setResetPending] = React.useState(false);
  const [manageModelsDialogOpen, setManageModelsDialogOpen] = React.useState(false);
  const [editableModels, setEditableModels] = React.useState<EditableProviderModel[]>([]);
  const manageModelsAddButtonRef = React.useRef<HTMLButtonElement | null>(null);
  const modelDraftSequence = React.useRef(0);
  const savedEditableModelsSnapshot = React.useRef("[]");
  const [newProviderDraft, setNewProviderDraft] = React.useState<NewProviderDraft>({
    name: "",
    type: "openai",
    compatibility: defaultCompatibilityForType("openai"),
    endpoint: "",
  });
  const syncModels = useSyncProviderModels();
  const updateModel = useUpdateProviderModel();
  const createProvider = useUpsertProvider();
  const resetProvider = useUpsertProvider();
  const updateProvider = useUpsertProvider();
  const replaceProviderModels = useReplaceProviderModels();
  const saveSecret = useUpsertProviderSecret();
  const deleteProvider = useDeleteProvider();
  const { data: providerSecret } = useProviderSecret(activeProviderId);
  const { data: providerModels = [], isLoading: modelsLoading } = useProviderModels(activeProviderId);

  React.useEffect(() => {
    if (providers.length === 0) {
      if (activeProviderId) {
        setActiveProviderId(null);
      }
      return;
    }
    if (!activeProviderId || !providers.some((provider) => provider.id === activeProviderId)) {
      setActiveProviderId(providers[0].id);
    }
  }, [activeProviderId, providers]);

  React.useEffect(() => {
    if (!activeProviderId) {
      return;
    }
    setStateByProvider((prev) => {
      const current = prev[activeProviderId];
      if (current?.isDirty) {
        return prev;
      }
      return {
        ...prev,
        [activeProviderId]: {
          apiKey: providerSecret?.apiKey ?? "",
          orgRef: providerSecret?.orgRef ?? "",
          modelQuery: current?.modelQuery ?? "",
          isDirty: false,
          showApiKey: current?.showApiKey ?? false,
        },
      };
    });
  }, [activeProviderId, providerSecret?.apiKey, providerSecret?.orgRef]);

  const activeProvider = providers.find((provider) => provider.id === activeProviderId) ?? providers[0];
  const activeState = activeProviderId ? stateByProvider[activeProviderId] : undefined;
  const modelQuery = (activeState?.modelQuery ?? "").trim().toLowerCase();
  const filteredModels = React.useMemo(() => {
    if (!modelQuery) {
      return providerModels;
    }
    return providerModels.filter((model) => model.name.toLowerCase().includes(modelQuery));
  }, [providerModels, modelQuery]);
  const providerQueryNormalized = providerQuery.trim().toLowerCase();
  const filteredProviders = React.useMemo(() => {
    if (!providerQueryNormalized) {
      return providers;
    }
    return providers.filter((provider) => {
      const haystack = `${provider.name} ${provider.type} ${provider.compatibility} ${provider.endpoint}`.toLowerCase();
      return haystack.includes(providerQueryNormalized);
    });
  }, [providers, providerQueryNormalized]);
  const isCustomProvider = (provider: Provider) => !provider.builtin;
  const activeProviderIsCustom = activeProvider ? isCustomProvider(activeProvider) : false;
  React.useEffect(() => {
    if (!activeProvider || !isCustomProvider(activeProvider)) {
      setCustomDraft(null);
      return;
    }
    setCustomDraft((prev) => {
      if (prev?.providerId === activeProvider.id && prev.isDirty) {
        return prev;
      }
      return {
        providerId: activeProvider.id,
        name: activeProvider.name,
        endpoint: activeProvider.endpoint,
        type: activeProvider.type,
        compatibility: activeProvider.compatibility || defaultCompatibilityForType(activeProvider.type),
        isDirty: false,
      };
    });
  }, [activeProvider, activeProviderIsCustom]);
  const typeOptions = React.useMemo(() => API_TYPE_OPTIONS, []);
  const compatibilityOptions = React.useMemo(
    () => API_COMPATIBILITY_OPTIONS[customDraft?.type ?? "openai"] ?? API_COMPATIBILITY_OPTIONS.openai,
    [customDraft?.type]
  );
  const newProviderCompatibilityOptions = React.useMemo(
    () => API_COMPATIBILITY_OPTIONS[newProviderDraft.type] ?? API_COMPATIBILITY_OPTIONS.openai,
    [newProviderDraft.type]
  );
  const buildModelDraft = React.useCallback((model?: ProviderModel): EditableProviderModel => {
    modelDraftSequence.current += 1;
    if (!model) {
      return {
        draftKey: `draft-${modelDraftSequence.current}`,
        originalName: "",
        id: "",
        name: "",
        displayName: "",
        capabilitiesJson: "",
        contextWindowTokens: undefined,
        maxOutputTokens: undefined,
        supportsTools: undefined,
        supportsReasoning: undefined,
        supportsVision: undefined,
        supportsAudio: undefined,
        supportsVideo: undefined,
        enabled: true,
        showInUi: true,
      };
    }
    return {
      draftKey: `draft-${modelDraftSequence.current}`,
      originalName: model.name,
      id: model.id,
      name: model.name,
      displayName: model.displayName,
      capabilitiesJson: model.capabilitiesJson,
      contextWindowTokens: model.contextWindowTokens,
      maxOutputTokens: model.maxOutputTokens,
      supportsTools: model.supportsTools,
      supportsReasoning: model.supportsReasoning,
      supportsVision: model.supportsVision,
      supportsAudio: model.supportsAudio,
      supportsVideo: model.supportsVideo,
      enabled: model.enabled,
      showInUi: model.showInUi,
    };
  }, []);

  const toEditableModelsSnapshot = React.useCallback((models: EditableProviderModel[]) => {
    const payload = models
      .map((item) => ({
        id: item.id,
        originalName: item.originalName,
        name: item.name.trim(),
        displayName: item.displayName.trim(),
        capabilitiesJson: item.capabilitiesJson.trim(),
        contextWindowTokens: item.contextWindowTokens,
        maxOutputTokens: item.maxOutputTokens,
        supportsTools: item.supportsTools,
        supportsReasoning: item.supportsReasoning,
        supportsVision: item.supportsVision,
        supportsAudio: item.supportsAudio,
        supportsVideo: item.supportsVideo,
        enabled: item.enabled,
        showInUi: item.showInUi,
      }))
      .filter((item) => item.name);
    return JSON.stringify(payload);
  }, []);

  const updateProviderState = (providerId: string, next: Partial<ProviderState>) => {
    setStateByProvider((prev) => ({
      ...prev,
      [providerId]: {
        ...(prev[providerId] ?? {
          apiKey: "",
          orgRef: "",
          modelQuery: "",
          isDirty: false,
          showApiKey: false,
        }),
        ...next,
      },
    }));
  };

  const updateCustomDraft = (next: Partial<CustomProviderState>) => {
    setCustomDraft((prev) => {
      if (!prev) {
        return prev;
      }
      return {
        ...prev,
        ...next,
        isDirty: true,
      };
    });
  };

  const updateEditableModel = (
    draftKey: string,
    updater: Partial<EditableProviderModel> | ((current: EditableProviderModel) => Partial<EditableProviderModel>)
  ) => {
    setEditableModels((prev) =>
      prev.map((item) => {
        if (item.draftKey !== draftKey) {
          return item;
        }
        const nextPatch = typeof updater === "function" ? updater(item) : updater;
        return {
          ...item,
          ...nextPatch,
        };
      })
    );
  };

  const addEditableModel = () => {
    setEditableModels((prev) => [...prev, buildModelDraft()]);
  };

  const removeEditableModel = (draftKey: string) => {
    setEditableModels((prev) => prev.filter((item) => item.draftKey !== draftKey));
  };

  const openManageModelsDialog = () => {
    const drafts = providerModels.map((model) => buildModelDraft(model));
    const nextDrafts = drafts.length > 0 ? drafts : [buildModelDraft()];
    savedEditableModelsSnapshot.current = toEditableModelsSnapshot(nextDrafts);
    setEditableModels(nextDrafts);
    setManageModelsDialogOpen(true);
  };

  const commitCustomDraft = (override?: Partial<CustomProviderState>) => {
    if (!activeProvider || !customDraft) {
      return;
    }
    const nextDraft = {
      ...customDraft,
      ...override,
    };
    const trimmedName = nextDraft.name.trim();
    const trimmedEndpoint = nextDraft.endpoint.trim();
    if (!trimmedName || !trimmedEndpoint) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.provider.updateError"),
        description: t("settings.provider.custom.updateInvalid"),
      });
      return;
    }
    if (updateProvider.isPending) {
      return;
    }
    updateProvider.mutate(
      {
        id: activeProvider.id,
        name: trimmedName,
        type: nextDraft.type,
        compatibility: nextDraft.compatibility,
        endpoint: trimmedEndpoint,
        enabled: activeProvider.enabled,
      },
      {
        onSuccess: () => {
          setCustomDraft((prev) => {
            if (!prev) {
              return prev;
            }
            return {
              ...prev,
              name: trimmedName,
              endpoint: trimmedEndpoint,
              type: nextDraft.type,
              compatibility: nextDraft.compatibility,
              isDirty: false,
            };
          });
        },
        onError: (error) => {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.provider.updateError"),
            description:
              extractErrorMessage(error) ||
              t("settings.provider.updateErrorFallback"),
          });
        },
      }
    );
  };

  const commitSecret = (onSuccess?: () => void) => {
    if (!activeProviderId) {
      return;
    }
    if (saveSecret.isPending) {
      return;
    }
    const apiKey = activeState?.apiKey ?? "";
    const orgRef = activeState?.orgRef ?? "";
    saveSecret.mutate(
      { providerId: activeProviderId, apiKey, orgRef },
      {
        onSuccess: () => {
          updateProviderState(activeProviderId, { isDirty: false });
          onSuccess?.();
        },
        onError: (error) => {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.provider.updateError"),
            description:
              extractErrorMessage(error) ||
              t("settings.provider.updateErrorFallback"),
          });
        },
      }
    );
  };

  const handleFetchModels = () => {
    if (!activeProviderId) {
      return;
    }
    const apiKey = activeState?.apiKey ?? "";
    const handleSyncError = (error: unknown) => {
      const detail =
        formatModelsSyncError(error) ||
        t("settings.provider.models.syncErrorFallback");
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.provider.models.syncErrorTitle"),
        description: detail,
      });
    };
    if (activeState?.isDirty) {
      commitSecret(() => {
        syncModels.mutate({ providerId: activeProviderId, apiKey }, { onError: handleSyncError });
      });
    } else {
      syncModels.mutate({ providerId: activeProviderId, apiKey }, { onError: handleSyncError });
    }
  };

  const handleSaveEditableModels = async (options?: {
    silentValidation?: boolean;
    drafts?: EditableProviderModel[];
    forceReplace?: boolean;
  }) => {
    if (!activeProvider) {
      return;
    }
    const sourceModels = options?.drafts ?? editableModels;
    const nextModels = sourceModels
      .map((item) => ({
        ...item,
        name: item.name.trim(),
        displayName: item.displayName.trim(),
        capabilitiesJson: item.capabilitiesJson.trim(),
      }))
      .filter((item) => item.name);
    const nameSet = new Set<string>();
    for (const item of nextModels) {
      const key = item.name.toLowerCase();
      if (nameSet.has(key)) {
        if (!options?.silentValidation) {
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.provider.models.manage.invalidTitle"),
            description: t("settings.provider.models.manage.invalidDuplicate"),
          });
        }
        return;
      }
      nameSet.add(key);
    }
    try {
      const nextSnapshot = toEditableModelsSnapshot(nextModels);
      const hasEnabledModel = nextModels.some((item) => item.enabled);
      const shouldEnableProvider = hasEnabledModel && !activeProvider.enabled;
      const shouldDisableProvider = !hasEnabledModel && activeProvider.enabled;
      if (!options?.forceReplace && nextSnapshot === savedEditableModelsSnapshot.current) {
        if (shouldEnableProvider || shouldDisableProvider) {
          await updateProvider.mutateAsync({
            id: activeProvider.id,
            name: activeProvider.name,
            type: activeProvider.type,
            endpoint: activeProvider.endpoint,
            enabled: shouldEnableProvider,
          });
        }
        return;
      }
      await replaceProviderModels.mutateAsync({
        providerId: activeProvider.id,
        models: nextModels.map((item) => ({
          id: item.id,
          providerId: activeProvider.id,
          name: item.name,
          displayName: item.originalName === item.name ? item.displayName : "",
          capabilitiesJson: item.capabilitiesJson,
          contextWindowTokens: item.contextWindowTokens,
          maxOutputTokens: item.maxOutputTokens,
          supportsTools: item.supportsTools,
          supportsReasoning: item.supportsReasoning,
          supportsVision: item.supportsVision,
          supportsAudio: item.supportsAudio,
          supportsVideo: item.supportsVideo,
          enabled: item.enabled,
          showInUi: item.showInUi,
        })),
      });
      if (shouldEnableProvider || shouldDisableProvider) {
        await updateProvider.mutateAsync({
          id: activeProvider.id,
          name: activeProvider.name,
          type: activeProvider.type,
          endpoint: activeProvider.endpoint,
          enabled: shouldEnableProvider,
        });
      }
      savedEditableModelsSnapshot.current = nextSnapshot;
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.provider.updateError"),
        description:
          extractErrorMessage(error) ||
          t("settings.provider.updateErrorFallback"),
      });
    }
  };

  const handleEditableModelsTableBlur = (event: React.FocusEvent<HTMLDivElement>) => {
    const nextFocusTarget = event.relatedTarget as Node | null;
    if (nextFocusTarget && event.currentTarget.contains(nextFocusTarget)) {
      return;
    }
    void handleSaveEditableModels({ silentValidation: true });
  };

  const handleRemoveEditableModel = (draftKey: string) => {
    const nextDrafts = editableModels.filter((item) => item.draftKey !== draftKey);
    removeEditableModel(draftKey);
    void handleSaveEditableModels({ silentValidation: true, drafts: nextDrafts });
  };

  const handleManageModelsDialogOpenChange = (open: boolean) => {
    if (!open) {
      void handleSaveEditableModels({ silentValidation: true, forceReplace: true });
    }
    setManageModelsDialogOpen(open);
  };

  const handleApiKeyBlur = () => {
    if (!activeProviderId || !activeState?.isDirty) {
      return;
    }
    commitSecret();
  };

  const applyApiKey = (value: string) => {
    if (!activeProviderId) {
      return;
    }
    updateProviderState(activeProviderId, { apiKey: value, isDirty: true });
  };

  const handlePasteApiKey = async () => {
    if (!activeProviderId || typeof navigator === "undefined" || !navigator.clipboard) {
      return;
    }
    try {
      const text = await navigator.clipboard.readText();
      applyApiKey(text);
    } catch {
      // ignore clipboard failures
    }
  };

  const handleModelEnabledToggle = (model: ProviderModel, nextEnabled: boolean) => {
    if (!activeProvider) {
      return;
    }
    const shouldEnableProvider = nextEnabled && !activeProvider.enabled;
    const shouldDisableProvider =
      !nextEnabled &&
      activeProvider.enabled &&
      model.enabled &&
      !providerModels.some((item) => item.id !== model.id && item.enabled);
    void (async () => {
      try {
        if (shouldEnableProvider) {
          await updateProvider.mutateAsync({
            id: activeProvider.id,
            name: activeProvider.name,
            type: activeProvider.type,
            endpoint: activeProvider.endpoint,
            enabled: true,
          });
        }
        await updateModel.mutateAsync({
          id: model.id,
          providerId: model.providerId,
          enabled: nextEnabled,
          showInUi: nextEnabled ? true : model.showInUi,
        });
        if (shouldDisableProvider) {
          await updateProvider.mutateAsync({
            id: activeProvider.id,
            name: activeProvider.name,
            type: activeProvider.type,
            endpoint: activeProvider.endpoint,
            enabled: false,
          });
        }
      } catch (error) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.provider.updateError"),
          description:
            extractErrorMessage(error) ||
            t("settings.provider.updateErrorFallback"),
        });
      }
    })();
  };

  const handleProviderEnabledChange = async (checked: boolean) => {
    if (!activeProvider) {
      return;
    }
    try {
      if (checked) {
        await updateProvider.mutateAsync({
          id: activeProvider.id,
          name: activeProvider.name,
          type: activeProvider.type,
          endpoint: activeProvider.endpoint,
          enabled: true,
        });
        const hasEnabledModel = providerModels.some((model) => model.enabled);
        if (!hasEnabledModel && providerModels.length > 0) {
          const firstModel = providerModels[0];
          await updateModel.mutateAsync({
            id: firstModel.id,
            providerId: firstModel.providerId,
            enabled: true,
            showInUi: true,
          });
        }
        return;
      }

      await updateProvider.mutateAsync({
        id: activeProvider.id,
        name: activeProvider.name,
        type: activeProvider.type,
        endpoint: activeProvider.endpoint,
        enabled: false,
      });
      const enabledModels = providerModels.filter((model) => model.enabled);
      if (enabledModels.length > 0) {
        for (const model of enabledModels) {
          await updateModel.mutateAsync({
            id: model.id,
            providerId: model.providerId,
            enabled: false,
            showInUi: model.showInUi,
          });
        }
      }
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.provider.updateError"),
        description: extractErrorMessage(error) || t("settings.provider.updateErrorFallback"),
      });
    }
  };

  const resetNewProviderDraft = () => {
    setNewProviderDraft({
      name: "",
      type: "openai",
      compatibility: defaultCompatibilityForType("openai"),
      endpoint: "",
    });
  };

  const handleCreateDialogOpenChange = (open: boolean) => {
    setCreateDialogOpen(open);
    if (!open) {
      resetNewProviderDraft();
    }
  };

  const handleCreateProvider = async () => {
    const trimmedName = newProviderDraft.name.trim();
    const trimmedEndpoint = newProviderDraft.endpoint.trim();
    if (!trimmedName || !trimmedEndpoint) {
      return;
    }
    const result = await createProvider.mutateAsync({
      name: trimmedName,
      type: newProviderDraft.type,
      compatibility: newProviderDraft.compatibility,
      endpoint: trimmedEndpoint,
      enabled: true,
    });
    setActiveProviderId(result.id);
    handleCreateDialogOpenChange(false);
  };

  const handleDeleteProvider = async () => {
    if (!activeProvider || !activeProviderIsCustom) {
      return;
    }
    await deleteProvider.mutateAsync(activeProvider.id);
    setDeleteDialogOpen(false);
  };

  const handleResetProviders = async () => {
    setResetPending(true);
    try {
      for (const provider of providers) {
        if (isCustomProvider(provider)) {
          await deleteProvider.mutateAsync(provider.id);
        }
      }
      for (const option of PROVIDER_DEFAULTS) {
        await resetProvider.mutateAsync({
          id: option.id,
          name: option.label,
          type: option.type,
          endpoint: option.endpoint,
          enabled: option.enabled,
        });
      }
      setResetDialogOpen(false);
    } finally {
      setResetPending(false);
    }
  };

  const apiKey = activeState?.apiKey ?? "";
  const showApiKey = activeState?.showApiKey ?? false;
  const canFetchModels = Boolean(activeProviderId);
  const visibleModels = filteredModels;
  const configRowClassName = SETTINGS_ROW_CLASS;
  const customFieldWidthClass = SETTINGS_WIDE_CONTROL_WIDTH_CLASS;

  return (
    <div className="providers-card flex min-h-0 min-w-0 flex-1">
      <Card className="flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden">
        <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
              <div className="flex h-7 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder={t("settings.provider.searchPlaceholder")}
                  value={providerQuery}
                  onChange={(event) => setProviderQuery(event.target.value)}
                  size="compact"
                  className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              {providersLoading ? (
                <div className="py-6 text-center text-sm text-muted-foreground">Loading providers...</div>
              ) : providers.length === 0 ? (
                <div className="py-6 text-center text-sm text-muted-foreground">No providers configured.</div>
              ) : filteredProviders.length === 0 ? (
                <div className="py-6 text-center text-sm text-muted-foreground">
                  No providers match your search.
                </div>
              ) : (
                <SidebarMenu className="gap-[6px]">
                {filteredProviders.map((provider) => {
                  const isSelected = provider.id === activeProviderId;
                  const displayName =
                    provider.id === customDraft?.providerId && customDraft?.name
                      ? customDraft.name
                      : provider.name;
                  return (
                    <SidebarMenuItem key={provider.id}>
                        <SidebarMenuButton
                          type="button"
                          variant="default"
                          isActive={isSelected}
                          size="default"
                          onClick={() => setActiveProviderId(provider.id)}
                          className="justify-start text-sidebar-foreground"
                        >
                          {provider.icon ? (
                            <span
                              aria-hidden
                              className="h-4 w-4 shrink-0 bg-current text-sidebar-foreground"
                              style={{
                                WebkitMaskImage: `url(${provider.icon})`,
                                maskImage: `url(${provider.icon})`,
                                WebkitMaskRepeat: "no-repeat",
                                maskRepeat: "no-repeat",
                                WebkitMaskPosition: "center",
                                maskPosition: "center",
                                WebkitMaskSize: "contain",
                                maskSize: "contain",
                              }}
                            />
                          ) : (
                            <span className="h-4 w-4 shrink-0" aria-hidden />
                          )}
                          <span className="flex-1 truncate">{displayName}</span>
                          <div
                            className={cn(
                              "h-2 w-2 shrink-0 rounded-full",
                              provider.enabled ? "bg-emerald-500" : "bg-muted-foreground/40"
                            )}
                          />
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              )}
            </div>
            <div className="flex h-7 shrink-0 items-center justify-between border-t px-[var(--app-sidebar-padding)]">
              <TooltipProvider delayDuration={0}>
                <div className="flex w-full items-center justify-between">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => setResetDialogOpen(true)}
                        disabled={resetPending || providersLoading}
                        aria-label={t("settings.provider.footer.reset")}
                      >
                        <RotateCcw className="h-3 w-3 text-destructive" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {t("settings.provider.footer.reset")}
                    </TooltipContent>
                  </Tooltip>
                  <div className="flex items-center gap-2">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          variant="ghost"
                          size="compactIcon"
                          onClick={() => handleCreateDialogOpenChange(true)}
                          aria-label={t("settings.provider.footer.add")}
                          disabled={createProvider.isPending || resetPending}
                        >
                          <Plus className="h-3 w-3" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("settings.provider.footer.add")}</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          variant="ghost"
                          size="compactIcon"
                          onClick={() => setDeleteDialogOpen(true)}
                          aria-label={t("settings.provider.footer.remove")}
                          disabled={!activeProviderIsCustom || deleteProvider.isPending || resetPending}
                        >
                          <Minus className="h-3 w-3" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("settings.provider.footer.remove")}</TooltipContent>
                    </Tooltip>
                  </div>
                </div>
              </TooltipProvider>
            </div>
          </div>

          <Separator orientation="vertical" className="self-stretch" />

          <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-[var(--app-sidebar-padding)] overflow-hidden px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
            {activeProvider ? (
              <TooltipProvider delayDuration={0}>
                <div className="space-y-2 text-sm">
                  {activeProviderIsCustom && customDraft ? (
                    <>
                      <div className={configRowClassName}>
                        <div className={SETTINGS_ROW_LABEL_CLASS}>
                          {t("settings.provider.custom.fields.name")}
                        </div>
                        <Input
                          value={customDraft.name}
                          onChange={(event) => updateCustomDraft({ name: event.target.value })}
                          onBlur={() => commitCustomDraft()}
                          size="compact"
                          className={cn(customFieldWidthClass, "text-right")}
                          disabled={updateProvider.isPending}
                        />
                      </div>
                      <SettingsSeparator />
                      <div className={configRowClassName}>
                        <div className={SETTINGS_ROW_LABEL_CLASS}>
                          {t("settings.provider.custom.fields.type")}
                        </div>
                        <Select
                          value={customDraft.type}
                          onChange={(event) => {
                            const nextType = event.target.value;
                            const nextCompatibility = defaultCompatibilityForType(nextType);
                            updateCustomDraft({ type: nextType, compatibility: nextCompatibility });
                            commitCustomDraft({ type: nextType, compatibility: nextCompatibility });
                          }}
                          className={cn(customFieldWidthClass, "text-right")}
                          disabled={updateProvider.isPending}
                        >
                          {typeOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <SettingsSeparator />
                      <div className={configRowClassName}>
                        <div className={SETTINGS_ROW_LABEL_CLASS}>
                          {t("settings.provider.custom.fields.compatibility")}
                        </div>
                        <Select
                          value={customDraft.compatibility}
                          onChange={(event) => {
                            const nextCompatibility = event.target.value;
                            updateCustomDraft({ compatibility: nextCompatibility });
                            commitCustomDraft({ compatibility: nextCompatibility });
                          }}
                          className={cn(customFieldWidthClass, "text-right")}
                          disabled={updateProvider.isPending}
                        >
                          {compatibilityOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <SettingsSeparator />
                      <div className={configRowClassName}>
                        <div className={SETTINGS_ROW_LABEL_CLASS}>
                          {t("settings.provider.custom.fields.endpoint")}
                        </div>
                        <Input
                          value={customDraft.endpoint}
                          onChange={(event) => updateCustomDraft({ endpoint: event.target.value })}
                          onBlur={() => commitCustomDraft()}
                          size="compact"
                          className={cn(customFieldWidthClass, "text-right")}
                          disabled={updateProvider.isPending}
                        />
                      </div>
                      <SettingsSeparator />
                    </>
                  ) : null}

                  <div className={configRowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.provider.credentials.enable")}
                    </div>
                    <Switch
                      checked={activeProvider.enabled}
                      onCheckedChange={(checked) => void handleProviderEnabledChange(checked)}
                      disabled={updateProvider.isPending || updateModel.isPending}
                      aria-label={t("settings.provider.credentials.enable")}
                    />
                  </div>
                  <SettingsSeparator />
                  <div className="flex items-center justify-between gap-4">
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.provider.credentials.apiKey")}
                    </div>
                    <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
                      <div className="relative w-48">
                        <Input
                          type={showApiKey ? "text" : "password"}
                          placeholder={t("settings.provider.credentials.apiKeyPlaceholder")}
                          value={apiKey}
                          onChange={(event) =>
                            updateProviderState(activeProvider.id, {
                              apiKey: event.target.value,
                              isDirty: true,
                            })
                          }
                          onBlur={handleApiKeyBlur}
                          size="compact"
                          className="!pr-14"
                        />
                        <div className="absolute right-1 top-1/2 flex -translate-y-1/2 items-center gap-1">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="ghost"
                                size="compactIcon"
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={handlePasteApiKey}
                                aria-label={t("settings.provider.credentials.paste")}
                              >
                                <Clipboard className="h-3 w-3" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                              {t("settings.provider.credentials.paste")}
                            </TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="ghost"
                                size="compactIcon"
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={() =>
                                  updateProviderState(activeProvider.id, { showApiKey: !showApiKey })
                                }
                                aria-label={
                                  showApiKey
                                    ? t("settings.provider.credentials.hide")
                                    : t("settings.provider.credentials.show")
                                }
                              >
                                {showApiKey ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                              {showApiKey
                                ? t("settings.provider.credentials.hide")
                                : t("settings.provider.credentials.show")}
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      </div>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            variant="ghost"
                            size="compactIcon"
                            onClick={handleFetchModels}
                            disabled={!canFetchModels || syncModels.isPending || saveSecret.isPending}
                            aria-label={t("settings.provider.credentials.fetch")}
                          >
                            {syncModels.isPending ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : (
                              <RefreshCw className="h-3 w-3" />
                            )}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("settings.provider.credentials.fetch")}</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                </div>

                <Card className="flex min-h-0 flex-1 flex-col border bg-card">
                  <div className="px-[var(--app-sidebar-padding)] pb-[6px] pt-[var(--app-sidebar-padding)]">
                    <div className="flex items-center gap-2">
                      <div className="flex h-7 flex-1 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
                        <Search className="h-4 w-4 text-muted-foreground" />
                        <Input
                          placeholder={t("settings.provider.models.searchPlaceholder")}
                          value={activeState?.modelQuery ?? ""}
                          onChange={(event) =>
                            updateProviderState(activeProvider.id, { modelQuery: event.target.value })
                          }
                          size="compact"
                          className="border-0 bg-transparent text-xs shadow-none placeholder:text-xs focus-visible:ring-0 focus-visible:ring-offset-0"
                        />
                      </div>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            size="compactIcon"
                            variant="outline"
                            onClick={openManageModelsDialog}
                            aria-label={t("settings.provider.models.manage.open")}
                            disabled={replaceProviderModels.isPending}
                          >
                            <Settings2 className="h-3.5 w-3.5" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("settings.provider.models.manage.open")}</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                  <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden px-[var(--app-sidebar-padding)] pb-[var(--app-sidebar-padding)] pt-0 text-xs">
                    <div className="min-h-0 flex-1 overflow-y-auto pr-1">
                      {modelsLoading ? (
                        <div className="py-6 text-center text-xs text-muted-foreground">
                          {t("settings.provider.models.loading")}
                        </div>
                      ) : filteredModels.length === 0 ? (
                        <div className="py-6 text-center text-xs text-muted-foreground">
                          {t("settings.provider.models.empty")}
                        </div>
                      ) : (
                        <ItemGroup className="gap-[3px]">
                          {visibleModels.map((model: ProviderModel) => {
                            const meta = parseModelMeta(model);
                            const capabilityIcons = (
                              <>
                                {meta.capabilities.map((capability) => {
                                  const label =
                                    capability === "tools"
                                      ? t("settings.provider.models.capability.tools")
                                      : capability === "reasoning"
                                        ? t("settings.provider.models.capability.reasoning")
                                        : capability === "image"
                                          ? t("settings.provider.models.capability.image")
                                          : capability === "audio"
                                            ? t("settings.provider.models.capability.audio")
                                            : t("settings.provider.models.capability.video");
                                  const Icon =
                                    capability === "tools"
                                      ? Wrench
                                      : capability === "reasoning"
                                        ? Brain
                                        : capability === "image"
                                          ? Image
                                          : capability === "audio"
                                            ? AudioLines
                                            : Video;
                                  return (
                                    <Tooltip key={`${model.id}-${capability}`}>
                                      <TooltipTrigger asChild>
                                        <span className="flex h-4 w-4 items-center justify-center">
                                          <Icon className="h-3.5 w-3.5" />
                                        </span>
                                      </TooltipTrigger>
                                      <TooltipContent>{label}</TooltipContent>
                                    </Tooltip>
                                  );
                                })}
                                {meta.contextWindow ? (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="flex h-4 w-4 items-center justify-center">
                                        <Ruler className="h-3.5 w-3.5" />
                                      </span>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      {t("settings.provider.models.context")}{" "}
                                      {formatContextWindow(meta.contextWindow)}
                                    </TooltipContent>
                                  </Tooltip>
                                ) : null}
                              </>
                            );
                            const itemNode = (
                              <Item
                                size="compact"
                                variant="default"
                                className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3"
                              >
                                <ItemContent className="min-w-0">
                                  <div className="flex min-w-0 items-center gap-2">
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <div className="min-w-0 flex-1">
                                          <ItemTitle className="min-w-0 w-full truncate text-xs font-medium text-sidebar-foreground">
                                            {model.name}
                                          </ItemTitle>
                                        </div>
                                      </TooltipTrigger>
                                      <TooltipContent>{model.name}</TooltipContent>
                                    </Tooltip>
                                    <div className="flex shrink-0 items-center gap-1 text-sidebar-foreground/70">
                                      {capabilityIcons}
                                    </div>
                                  </div>
                                </ItemContent>
                                <ItemActions className="shrink-0 gap-2">
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="inline-flex items-center">
                                        <Switch
                                          checked={model.enabled}
                                          onCheckedChange={(checked) =>
                                            handleModelEnabledToggle(model, checked)
                                          }
                                          disabled={updateModel.isPending || updateProvider.isPending}
                                          aria-label={t("settings.provider.models.toggleEnabled")}
                                        />
                                      </span>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      {t("settings.provider.models.toggleEnabled")}
                                    </TooltipContent>
                                  </Tooltip>
                                </ItemActions>
                              </Item>
                            );
                            return <React.Fragment key={model.id}>{itemNode}</React.Fragment>;
                          })}
                        </ItemGroup>
                      )}
                    </div>
                  </div>
                </Card>
              </TooltipProvider>
            ) : (
              <div className="flex min-h-0 flex-1 items-center justify-center text-center text-sm text-muted-foreground">
                Select a provider to configure.
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Dialog open={createDialogOpen} onOpenChange={handleCreateDialogOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("settings.provider.custom.title")}</DialogTitle>
            <DialogDescription>
              {t("settings.provider.custom.description")}
            </DialogDescription>
          </DialogHeader>
          <Card>
            <CardContent className="p-0">
              <div className="flex min-h-9 items-center justify-between gap-4 px-3 py-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("settings.provider.custom.fields.name")}
                </span>
                <Input
                  value={newProviderDraft.name}
                  onChange={(event) =>
                    setNewProviderDraft((prev) => ({ ...prev, name: event.target.value }))
                  }
                  placeholder={t("settings.provider.custom.fields.namePlaceholder")}
                  size="compact"
                  className={cn(customFieldWidthClass, "text-right")}
                />
              </div>
              <Separator />
              <div className="flex min-h-9 items-center justify-between gap-4 px-3 py-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("settings.provider.custom.fields.type")}
                </span>
                <Select
                  value={newProviderDraft.type}
                  onChange={(event) => {
                    const nextType = event.target.value;
                    setNewProviderDraft((prev) => ({
                      ...prev,
                      type: nextType,
                      compatibility: defaultCompatibilityForType(nextType),
                    }));
                  }}
                  className={cn(customFieldWidthClass, "text-right")}
                >
                  {typeOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </div>
              <Separator />
              <div className="flex min-h-9 items-center justify-between gap-4 px-3 py-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("settings.provider.custom.fields.compatibility")}
                </span>
                <Select
                  value={newProviderDraft.compatibility}
                  onChange={(event) =>
                    setNewProviderDraft((prev) => ({ ...prev, compatibility: event.target.value }))
                  }
                  className={cn(customFieldWidthClass, "text-right")}
                >
                  {newProviderCompatibilityOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </div>
              <Separator />
              <div className="flex min-h-9 items-center justify-between gap-4 px-3 py-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("settings.provider.custom.fields.endpoint")}
                </span>
                <Input
                  value={newProviderDraft.endpoint}
                  onChange={(event) =>
                    setNewProviderDraft((prev) => ({ ...prev, endpoint: event.target.value }))
                  }
                  placeholder={t("settings.provider.custom.fields.endpointPlaceholder")}
                  size="compact"
                  required
                  className={cn(customFieldWidthClass, "text-right")}
                />
              </div>
            </CardContent>
          </Card>
          <div className="flex flex-col-reverse gap-2 pt-4 sm:flex-row sm:items-center sm:justify-end">
            <DialogClose asChild>
              <Button size="compact" variant="outline">
                {t("common.cancel")}
              </Button>
            </DialogClose>
            <Button
              size="compact"
              onClick={handleCreateProvider}
              disabled={
                !newProviderDraft.name.trim() || !newProviderDraft.endpoint.trim() || createProvider.isPending
              }
            >
              {createProvider.isPending
                ? t("settings.provider.custom.actions.creating")
                : t("settings.provider.custom.actions.create")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("settings.provider.delete.title")}</DialogTitle>
            <DialogDescription>
              {activeProvider
                ? t("settings.provider.delete.descriptionWithName").replace("{name}", activeProvider.name)
                : t("settings.provider.delete.description")}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col-reverse gap-2 pt-4 sm:flex-row sm:items-center sm:justify-end">
            <DialogClose asChild>
              <Button size="compact" variant="outline">
                {t("common.cancel")}
              </Button>
            </DialogClose>
            <Button
              size="compact"
              variant="destructive"
              onClick={handleDeleteProvider}
              disabled={!activeProviderIsCustom || deleteProvider.isPending}
            >
              {deleteProvider.isPending
                ? t("settings.provider.delete.actions.deleting")
                : t("settings.provider.delete.actions.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={resetDialogOpen} onOpenChange={setResetDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset providers</DialogTitle>
            <DialogDescription>
              This resets the provider list to defaults and removes custom providers.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col-reverse gap-2 pt-4 sm:flex-row sm:items-center sm:justify-end">
            <DialogClose asChild>
              <Button size="compact" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button
              size="compact"
              variant="destructive"
              onClick={handleResetProviders}
              disabled={resetPending}
            >
              {resetPending ? "Resetting..." : "Reset"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={manageModelsDialogOpen} onOpenChange={handleManageModelsDialogOpenChange}>
        <DialogContent
          className="max-w-3xl [&>button]:hidden"
          onOpenAutoFocus={(event) => {
            event.preventDefault();
            manageModelsAddButtonRef.current?.focus();
          }}
        >
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5">
                <span className="text-sm font-medium">
                  {t("settings.provider.models.manage.title")}
                </span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="compactIcon"
                      aria-label={t("settings.provider.models.manage.description")}
                    >
                      <HelpCircle className="h-3.5 w-3.5 text-muted-foreground" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {t("settings.provider.models.manage.description")}
                  </TooltipContent>
                </Tooltip>
              </div>
              <Button
                ref={manageModelsAddButtonRef}
                type="button"
                size="compact"
                variant="outline"
                onClick={addEditableModel}
              >
                <Plus className="mr-1 h-3.5 w-3.5" />
                {t("settings.provider.models.manage.add")}
              </Button>
            </div>
            <div
              className="max-h-[52vh] overflow-auto rounded-md border"
              onBlurCapture={handleEditableModelsTableBlur}
            >
              <Table className="table-fixed">
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[55%]">
                      {t("settings.provider.models.manage.columnModelId")}
                    </TableHead>
                    <TableHead className="w-32 text-center">
                      {t("settings.provider.models.manage.columnEnabled")}
                    </TableHead>
                    <TableHead className="w-36 text-center">
                      {t("settings.provider.models.manage.columnVisible")}
                    </TableHead>
                    <TableHead className="w-20 text-right">
                      {t("settings.provider.models.manage.columnActions")}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {editableModels.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-xs text-muted-foreground">
                        {t("settings.provider.models.manage.empty")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    editableModels.map((item) => (
                      <TableRow key={item.draftKey}>
                        <TableCell>
                          <Input
                            value={item.name}
                            onChange={(event) =>
                              updateEditableModel(item.draftKey, { name: event.target.value })
                            }
                            placeholder={t("settings.provider.models.manage.idPlaceholder")}
                            size="compact"
                          />
                        </TableCell>
                        <TableCell className="text-center">
                          <div className="inline-flex items-center">
                            <Switch
                              checked={item.enabled}
                              onCheckedChange={(checked) =>
                                updateEditableModel(item.draftKey, {
                                  enabled: checked,
                                  showInUi: checked ? true : item.showInUi,
                                })
                              }
                              aria-label={t("settings.provider.models.toggleEnabled")}
                            />
                          </div>
                        </TableCell>
                        <TableCell className="text-center">
                          <div className="inline-flex items-center">
                            <Switch
                              checked={item.showInUi}
                              onCheckedChange={(checked) =>
                                updateEditableModel(item.draftKey, { showInUi: checked })
                              }
                              aria-label={t("settings.provider.models.manage.visible")}
                            />
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            type="button"
                            variant="ghost"
                            size="compactIcon"
                            onClick={() => handleRemoveEditableModel(item.draftKey)}
                            aria-label={t("settings.provider.models.manage.remove")}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
          <div className="flex justify-end pt-2">
            <DialogClose asChild>
              <Button size="compact" variant="outline">
                {t("settings.provider.models.manage.close")}
              </Button>
            </DialogClose>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
