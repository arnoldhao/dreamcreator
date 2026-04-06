import * as React from "react";
import {
  CheckCircle2,
  Download,
  Loader2,
  RefreshCw,
  Wrench,
} from "lucide-react";

import { messageBus } from "@/shared/message/store";
import { useI18n } from "@/shared/i18n";
import { useAssistants, useUpdateAssistant } from "@/shared/query/assistant";
import {
  useExternalToolInstallState,
  useExternalTools,
  useInstallExternalTool,
  useVerifyExternalTool,
} from "@/shared/query/externalTools";
import {
  useEnabledProvidersWithModels,
  useProviderModels,
  useProviderSecret,
  useProviders,
  useSyncProviderModels,
  useUpdateProviderModel,
  useUpsertProvider,
  useUpsertProviderSecret,
} from "@/shared/query/providers";
import { useUpdateSettings } from "@/shared/query/settings";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import type { ExternalTool } from "@/shared/store/externalTools";
import type { ProviderModel } from "@/shared/store/providers";
import { useSettingsStore } from "@/shared/store/settings";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Progress } from "@/shared/ui/progress";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import { parseModelMeta } from "@/shared/utils/modelMeta";
import {
  type AssistantModelOption,
  buildModelRef,
  modelRefEquals,
  resolveModelSelectValue,
  resolveUsableModelRef,
} from "@/features/settings/gateway/ui/assistant-parameter-panels/model-utils";

import { DOWNLOAD_MODE_REQUIRED_TOOLS, FULL_MODE_REQUIRED_TOOLS, PROVIDER_PRESETS } from "./constants";
import {
  InlineNotice,
  SetupCardRow,
  SetupCardRows,
  SetupCardSection,
  SetupCardSeparator,
  SetupCardStatusHeader,
  SetupPageCard,
  SetupStatusIcon,
} from "./cards";
import { getToolItemId, type SetupNavStatus } from "./nav";
import { isInstalledTool, parseModelRef, resolveToolDependencyIssues } from "./readiness";
import { useSetupCenter } from "./store";

type ProviderOption = {
  id: string;
  label: string;
  endpoint: string;
  type: string;
};

type TranslateFn = (key: string) => string;

const EXTERNAL_INSTALL_STAGES = new Set(["downloading", "extracting", "verifying"]);

const clampProgress = (value?: number) => {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return 0;
  }
  if (value < 0) {
    return 0;
  }
  if (value > 100) {
    return 100;
  }
  return Math.round(value);
};

const resolveProviderOption = (
  allProviders: Array<{ id: string; name: string; endpoint: string; type: string }>
): ProviderOption[] => {
  const options: ProviderOption[] = [];
  const seen = new Set<string>();

  for (const preset of PROVIDER_PRESETS) {
    const matched = allProviders.find((item) => item.id === preset.id);
    options.push({
      id: preset.id,
      label: matched?.name?.trim() || preset.label,
      endpoint: matched?.endpoint?.trim() || preset.endpoint,
      type: matched?.type?.trim() || preset.type,
    });
    seen.add(preset.id);
  }

  for (const provider of allProviders) {
    if (seen.has(provider.id)) {
      continue;
    }
    options.push({
      id: provider.id,
      label: provider.name,
      endpoint: provider.endpoint,
      type: provider.type,
    });
  }

  return options;
};

const resolveModelRefFromOptionValue = (value: string, options: AssistantModelOption[]) => {
  if (!value) {
    return "";
  }
  return options.find((option) => option.value === value)?.modelRef ?? "";
};

const resolveInstallActionLabel = (tool: ExternalTool | null, t: TranslateFn) =>
  String(tool?.status ?? "").trim().toLowerCase() === "invalid"
    ? t("settings.externalTools.actions.repair")
    : t("settings.externalTools.actions.install");

export function SetupInlineProviderCard({
  status,
  showSkip = false,
  onSkip,
}: {
  status: SetupNavStatus;
  showSkip?: boolean;
  onSkip?: () => void;
}) {
  const { t } = useI18n();
  const { clearAiDeferred, clearSkippedItem } = useSetupCenter();
  const assistantsQuery = useAssistants(true);
  const providersListQuery = useProviders();
  const upsertProvider = useUpsertProvider();
  const upsertProviderSecret = useUpsertProviderSecret();
  const syncProviderModels = useSyncProviderModels();
  const updateProviderModel = useUpdateProviderModel();

  const assistants = assistantsQuery.data ?? [];
  const defaultAssistant = React.useMemo(
    () => assistants.find((item) => item.isDefault) ?? assistants[0] ?? null,
    [assistants]
  );
  const parsedDefaultModel = React.useMemo(
    () => parseModelRef(defaultAssistant?.model?.agent?.primary?.trim() ?? ""),
    [defaultAssistant?.model?.agent?.primary]
  );

  const allProviders = providersListQuery.data ?? [];
  const providerOptions = React.useMemo(() => resolveProviderOption(allProviders), [allProviders]);
  const [selectedProviderId, setSelectedProviderId] = React.useState(PROVIDER_PRESETS[0]?.id ?? "openai");
  const [providerApiKey, setProviderApiKey] = React.useState("");
  const [providerError, setProviderError] = React.useState("");

  const selectedProvider = allProviders.find((item) => item.id === selectedProviderId) ?? null;
  const selectedProviderOption = providerOptions.find((item) => item.id === selectedProviderId) ?? null;
  const selectedProviderModelsQuery = useProviderModels(selectedProvider?.id ?? null);
  const selectedProviderSecretQuery = useProviderSecret(selectedProvider?.id ?? null);
  const selectedProviderModels = selectedProviderModelsQuery.data ?? [];
  const providerBusy =
    upsertProvider.isPending || upsertProviderSecret.isPending || syncProviderModels.isPending;
  const selectedProviderApiConfigured = Boolean(
    (selectedProvider ? selectedProviderSecretQuery.data?.apiKey : providerApiKey)?.trim()
  );

  React.useEffect(() => {
    const preferredProviderId = parsedDefaultModel.providerId.trim();
    if (selectedProviderId && providerOptions.some((item) => item.id === selectedProviderId)) {
      return;
    }
    if (preferredProviderId && providerOptions.some((item) => item.id === preferredProviderId)) {
      setSelectedProviderId(preferredProviderId);
      return;
    }
    if (providerOptions[0]?.id) {
      setSelectedProviderId(providerOptions[0].id);
    }
  }, [parsedDefaultModel.providerId, providerOptions, selectedProviderId]);

  React.useEffect(() => {
    if (!selectedProvider) {
      return;
    }
    setProviderApiKey(selectedProviderSecretQuery.data?.apiKey ?? "");
  }, [selectedProvider, selectedProviderSecretQuery.data?.apiKey]);

  const saveProviderConfig = React.useCallback(async () => {
    const providerId = selectedProvider?.id ?? selectedProviderOption?.id ?? "";
    const providerName = selectedProvider?.name ?? selectedProviderOption?.label ?? "";
    const providerType = selectedProvider?.type ?? selectedProviderOption?.type ?? "openai";
    const endpoint = selectedProvider?.endpoint ?? selectedProviderOption?.endpoint ?? "";

    if (!providerId || !providerName || !endpoint) {
      return "";
    }

    await upsertProvider.mutateAsync({
      id: providerId,
      name: providerName,
      type: providerType,
      endpoint,
      enabled: selectedProvider?.enabled ?? false,
    });

    if (providerApiKey.trim()) {
      await upsertProviderSecret.mutateAsync({
        providerId,
        apiKey: providerApiKey.trim(),
        orgRef: "",
      });
    }

    setSelectedProviderId(providerId);
    return providerId;
  }, [
    providerApiKey,
    selectedProvider,
    selectedProviderOption,
    upsertProvider,
    upsertProviderSecret,
  ]);

  const handleRefreshProviderModels = async () => {
    setProviderError("");
    try {
      const providerId = await saveProviderConfig();
      const apiKey =
        providerApiKey.trim() || selectedProviderSecretQuery.data?.apiKey?.trim() || "";
      if (!providerId || !apiKey) {
        return;
      }
      await syncProviderModels.mutateAsync({ providerId, apiKey });
      clearSkippedItem("ai.provider");
      clearAiDeferred();
    } catch (error) {
      setProviderError(error instanceof Error ? error.message : String(error ?? ""));
    }
  };

  const handleProviderModelToggle = async (model: ProviderModel, nextEnabled: boolean) => {
    if (!selectedProvider) {
      return;
    }
    const shouldEnableProvider = nextEnabled && !selectedProvider.enabled;
    const shouldDisableProvider =
      !nextEnabled &&
      selectedProvider.enabled &&
      model.enabled &&
      !selectedProviderModels.some((item) => item.id !== model.id && item.enabled);

    try {
      if (shouldEnableProvider) {
        await upsertProvider.mutateAsync({
          id: selectedProvider.id,
          name: selectedProvider.name,
          type: selectedProvider.type,
          endpoint: selectedProvider.endpoint,
          enabled: true,
        });
      }

      await updateProviderModel.mutateAsync({
        id: model.id,
        providerId: model.providerId,
        enabled: nextEnabled,
        showInUi: nextEnabled ? true : model.showInUi,
      });

      if (shouldDisableProvider) {
        await upsertProvider.mutateAsync({
          id: selectedProvider.id,
          name: selectedProvider.name,
          type: selectedProvider.type,
          endpoint: selectedProvider.endpoint,
          enabled: false,
        });
      }

      if (nextEnabled) {
        clearSkippedItem("ai.provider");
        clearAiDeferred();
      }
    } catch (error) {
      setProviderError(error instanceof Error ? error.message : String(error ?? ""));
    }
  };

  return (
    <SetupPageCard
      title={t("setupCenter.ai.provider")}
      headerRight={
        <SetupCardStatusHeader
          status={status}
          onSkip={onSkip}
          showSkip={showSkip && status === "pending"}
          skipLabel={t("setupCenter.actions.skip")}
        />
      }
    >
      <SetupCardRows>
        <SetupCardRow label={t("setupCenter.ai.providerLabel")}>
          <Select
            value={selectedProviderId}
            className="h-9 min-w-[16rem] text-xs"
            onChange={(event) => {
              setProviderError("");
              setSelectedProviderId(event.target.value);
            }}
          >
            {providerOptions.map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </Select>
        </SetupCardRow>
        <SetupCardSeparator />
        <SetupCardRow label={t("setupCenter.ai.apiKey")}>
          <Input
            type="password"
            value={providerApiKey}
            className="h-9 w-[16rem] text-right"
            onChange={(event) => setProviderApiKey(event.target.value)}
          />
        </SetupCardRow>
      </SetupCardRows>

      <SetupCardSeparator />
      <SetupCardSection>
        <div className="rounded-lg border border-border/70 bg-background/70">
          <div className="flex items-center justify-between gap-3 border-b border-border/70 px-3 py-2">
            <div className="min-w-0 truncate text-xs font-medium text-muted-foreground">
              {selectedProvider?.name ?? selectedProviderOption?.label ?? t("setupCenter.ai.noProvider")}
            </div>
            <Button
              type="button"
              size="compact"
              variant="outline"
              disabled={providerBusy || !providerApiKey.trim()}
              onClick={() => void handleRefreshProviderModels()}
            >
              {providerBusy ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <RefreshCw className="h-3.5 w-3.5" />
              )}
              {t("setupCenter.ai.syncModels")}
            </Button>
          </div>

          <div className="max-h-[14rem] overflow-y-auto">
            {selectedProviderModelsQuery.isLoading ? (
              <div className="flex items-center gap-2 px-3 py-3 text-xs text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("settings.provider.models.loading")}
              </div>
            ) : selectedProviderModels.length > 0 ? (
              <div className="divide-y divide-border/70">
                {selectedProviderModels.map((model) => (
                  <div key={model.id} className="flex items-center justify-between gap-3 px-3 py-3">
                    <div className="min-w-0 text-xs text-muted-foreground">
                      {(model.displayName || model.name || "").trim() || model.name}
                    </div>
                    <Switch
                      checked={model.enabled}
                      disabled={updateProviderModel.isPending || upsertProvider.isPending}
                      onCheckedChange={(checked) => void handleProviderModelToggle(model, checked)}
                    />
                  </div>
                ))}
              </div>
            ) : (
              <div className="px-3 py-4 text-xs text-muted-foreground">
                {selectedProviderApiConfigured
                  ? t("setupCenter.ai.modelsEmpty")
                  : t("setupCenter.ai.refreshHint")}
              </div>
            )}
          </div>
        </div>
      </SetupCardSection>

      {providerError ? (
        <>
          <SetupCardSeparator />
          <SetupCardSection>
            <InlineNotice tone="warning">{providerError}</InlineNotice>
          </SetupCardSection>
        </>
      ) : null}
    </SetupPageCard>
  );
}

export function SetupInlineAgentModelCard({
  status,
  showSkip = false,
  onSkip,
}: {
  status: SetupNavStatus;
  showSkip?: boolean;
  onSkip?: () => void;
}) {
  const { t } = useI18n();
  const { clearAiDeferred, clearSkippedItem } = useSetupCenter();
  const assistantsQuery = useAssistants(true);
  const enabledProvidersQuery = useEnabledProvidersWithModels();
  const updateAssistant = useUpdateAssistant();
  const updateSettings = useUpdateSettings();
  const gatewayEnabled = useSettingsStore(
    (state) => state.settings?.gateway.controlPlaneEnabled ?? false
  );

  const assistants = assistantsQuery.data ?? [];
  const defaultAssistant = React.useMemo(
    () => assistants.find((item) => item.isDefault) ?? assistants[0] ?? null,
    [assistants]
  );
  const enabledProvidersWithModels = enabledProvidersQuery.data ?? [];
  const [modelDraft, setModelDraft] = React.useState("");

  const assistantModelRef = defaultAssistant?.model?.agent?.primary?.trim() ?? "";
  const agentModelOptions = React.useMemo<AssistantModelOption[]>(() => {
    return enabledProvidersWithModels.flatMap((entry) => {
      const providerName = entry.provider.name || entry.provider.id;
      return entry.models
        .filter((model) => model.showInUi !== false)
        .map((model) => ({
          value: model.id,
          label: `${providerName} / ${(model.displayName || model.name || "").trim() || model.name}`,
          providerId: entry.provider.id,
          modelName: model.name,
          modelRef: buildModelRef(entry.provider.id, model.name),
          model,
          meta: parseModelMeta(model),
        }));
    });
  }, [enabledProvidersWithModels]);

  React.useEffect(() => {
    setModelDraft(assistantModelRef);
  }, [assistantModelRef]);

  React.useEffect(() => {
    if (agentModelOptions.length === 0) {
      if (modelDraft) {
        setModelDraft("");
      }
      return;
    }
    const nextModelRef = resolveUsableModelRef(modelDraft, agentModelOptions);
    if (!nextModelRef || modelRefEquals(modelDraft, nextModelRef)) {
      return;
    }
    setModelDraft(nextModelRef);
  }, [agentModelOptions, modelDraft]);

  const applyAgentModel = React.useCallback(
    async (nextModelRef: string) => {
      if (!defaultAssistant) {
        return;
      }
      const normalizedNext = nextModelRef.trim();
      if (!normalizedNext) {
        return;
      }
      const currentPrimary = defaultAssistant.model?.agent?.primary?.trim() ?? "";
      const needsModelUpdate = !modelRefEquals(currentPrimary, normalizedNext);
      const needsGatewayEnable = !gatewayEnabled;
      if (!needsModelUpdate && !needsGatewayEnable) {
        clearSkippedItem("ai.agentModel");
        clearAiDeferred();
        return;
      }
      if (needsModelUpdate) {
        await updateAssistant.mutateAsync({
          id: defaultAssistant.id,
          model: {
            ...defaultAssistant.model,
            agent: {
              ...defaultAssistant.model.agent,
              primary: normalizedNext,
            },
          },
        });
      }
      if (needsGatewayEnable) {
        await updateSettings.mutateAsync({
          gateway: {
            controlPlaneEnabled: true,
          },
        });
      }
      clearSkippedItem("ai.agentModel");
      clearAiDeferred();
    },
    [clearAiDeferred, clearSkippedItem, defaultAssistant, gatewayEnabled, updateAssistant, updateSettings]
  );

  return (
    <SetupPageCard
      title={t("setupCenter.ai.gatewayAssistantNav")}
      headerRight={
        <SetupCardStatusHeader
          status={status}
          onSkip={onSkip}
          showSkip={showSkip && status === "pending"}
          skipLabel={t("setupCenter.actions.skip")}
        />
      }
    >
      <SetupCardRows>
        <SetupCardRow label={t("setupCenter.ai.gatewayAssistantNav")}>
          <Select
            value={resolveModelSelectValue(modelDraft, agentModelOptions)}
            className="h-9 min-w-[18rem] text-xs"
            disabled={agentModelOptions.length === 0 || !defaultAssistant}
            onChange={(event) => {
              const nextModelRef = resolveModelRefFromOptionValue(event.target.value, agentModelOptions);
              setModelDraft(nextModelRef);
              if (nextModelRef) {
                void applyAgentModel(nextModelRef);
              }
            }}
          >
            {agentModelOptions.length === 0 ? (
              <option value="">{t("setupCenter.ai.noModelAvailable")}</option>
            ) : (
              agentModelOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))
            )}
          </Select>
        </SetupCardRow>
      </SetupCardRows>

      {!defaultAssistant ? (
        <>
          <SetupCardSeparator />
          <SetupCardSection>
            <InlineNotice>{t("setupCenter.ai.noAssistant")}</InlineNotice>
          </SetupCardSection>
        </>
      ) : null}

      {defaultAssistant && !gatewayEnabled ? (
        <>
          <SetupCardSeparator />
          <SetupCardSection>
            <InlineNotice>{t("setupCenter.issues.gateway.description")}</InlineNotice>
          </SetupCardSection>
        </>
      ) : null}
    </SetupPageCard>
  );
}

export function SetupInlineProductModeCard({ status }: { status: SetupNavStatus }) {
  const { t } = useI18n();
  const { enabled: assistantUiEnabled, setEnabled: setAssistantUiEnabled } = useAssistantUiMode();
  const { clearDependencyDeferred } = useSetupCenter();

  return (
    <SetupPageCard
      title={t("setupCenter.dependencies.modeTitle")}
      headerRight={<SetupStatusIcon status={status} />}
    >
      <SetupCardSection>
        <div className="grid gap-3 lg:grid-cols-2">
          {(["full", "download"] as const).map((mode) => {
            const isActive = assistantUiEnabled === (mode === "full");
            return (
              <button
                key={mode}
                type="button"
                className={
                  isActive
                    ? "rounded-lg border border-primary/40 bg-primary/5 px-4 py-4 text-left transition-colors"
                    : "rounded-lg border border-border/70 bg-background/70 px-4 py-4 text-left transition-colors hover:bg-muted/60"
                }
                onClick={() => {
                  clearDependencyDeferred();
                  setAssistantUiEnabled(mode === "full");
                }}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-xs font-medium text-muted-foreground">
                      {t(`productMode.options.${mode}.title`)}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {t(`productMode.options.${mode}.description`)}
                    </div>
                  </div>
                  <SetupStatusIcon status={isActive ? "ready" : "pending"} />
                </div>
              </button>
            );
          })}
        </div>
      </SetupCardSection>
    </SetupPageCard>
  );
}

export function SetupInlineToolCard({
  name,
  status,
  showSkip = false,
  onSkip,
}: {
  name: string;
  status: SetupNavStatus;
  showSkip?: boolean;
  onSkip?: () => void;
}) {
  const { t } = useI18n();
  const { clearDependencyDeferred, clearSkippedItem } = useSetupCenter();
  const externalToolsQuery = useExternalTools();
  const installTool = useInstallExternalTool();
  const verifyTool = useVerifyExternalTool();
  const [running, setRunning] = React.useState(false);
  const [verifying, setVerifying] = React.useState(false);
  const [actionError, setActionError] = React.useState("");
  const shouldTrackInstall = running || installTool.isPending;
  const installState = useExternalToolInstallState(name, shouldTrackInstall);

  const tools = externalToolsQuery.data ?? [];
  const tool = tools.find((item) => item.name?.trim().toLowerCase() === name.toLowerCase()) ?? null;
  const bunTool = tools.find((item) => item.name?.trim().toLowerCase() === "bun") ?? null;
  const bunInstalled = isInstalledTool(bunTool);
  const toolIssue = React.useMemo(
    () => resolveToolDependencyIssues([name], tools)[0] ?? null,
    [name, tools]
  );
  const isInvalid = toolIssue?.status === "invalid";
  const isInstalled = isInstalledTool(tool);
  const canVerify = !toolIssue && isInstalled;
  const installBlockedByBun = name === "clawhub" && !bunInstalled;
  const stage = shouldTrackInstall ? String(installState.data?.stage ?? "").trim().toLowerCase() : "";
  const progress = clampProgress(installState.data?.progress);
  const stageMessage = installState.data?.message?.trim() ?? "";
  const isInstalling = shouldTrackInstall || EXTERNAL_INSTALL_STAGES.has(stage);
  const isVerifyPending = verifying && verifyTool.isPending;
  const actionLabel = isInstalling
    ? isInvalid
      ? t("setupCenter.actions.repairing")
      : t("setupCenter.actions.installing")
    : canVerify
      ? t("settings.externalTools.actions.verify")
      : resolveInstallActionLabel(tool, t);
  const ActionIcon = canVerify ? CheckCircle2 : isInvalid ? Wrench : Download;

  React.useEffect(() => {
    if (!shouldTrackInstall) {
      return;
    }
    if (stage === "done") {
      setRunning(false);
      setActionError("");
      clearSkippedItem(getToolItemId(name));
      clearDependencyDeferred();
      void externalToolsQuery.refetch();
      return;
    }
    if (stage === "error") {
      setRunning(false);
      setActionError(stageMessage || t("settings.externalTools.installDialog.error"));
      void externalToolsQuery.refetch();
    }
  }, [
    clearDependencyDeferred,
    clearSkippedItem,
    externalToolsQuery,
    name,
    shouldTrackInstall,
    stage,
    stageMessage,
    t,
  ]);

  const handleInstall = async () => {
    if (installBlockedByBun) {
      return;
    }
    setActionError("");
    setRunning(true);
    try {
      await installTool.mutateAsync({ name });
      await installState.refetch();
    } catch (error) {
      setRunning(false);
      const message =
        error instanceof Error ? error.message : t("settings.externalTools.installDialog.error");
      setActionError(message);
      messageBus.publishToast({
        intent: "warning",
        title: name.toUpperCase(),
        description: message,
      });
      await externalToolsQuery.refetch();
    }
  };

  const handleVerify = async () => {
    setActionError("");
    setVerifying(true);
    try {
      await verifyTool.mutateAsync({ name });
      await externalToolsQuery.refetch();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error ?? "");
      setActionError(message);
      messageBus.publishToast({
        intent: "warning",
        title: t("settings.externalTools.actions.verify"),
        description: message,
      });
    } finally {
      setVerifying(false);
    }
  };

  const actionFeedback = isInstalling ? (
    <div className="flex min-w-0 items-center gap-2 overflow-hidden">
      <Progress value={progress} className="h-2 w-28 shrink-0 bg-muted" />
      <span className="shrink-0 text-[11px] font-medium tabular-nums text-muted-foreground">
        {progress}%
      </span>
    </div>
  ) : actionError ? (
    <span className="block truncate text-[11px] text-destructive" title={actionError}>
      {actionError}
    </span>
  ) : null;

  return (
    <SetupPageCard
      title={name.toUpperCase()}
      titleClassName="text-xs font-semibold uppercase tracking-[0.24em]"
      headerRight={
        <SetupCardStatusHeader
          status={status}
          onSkip={onSkip}
          showSkip={showSkip && status === "pending"}
          skipLabel={t("setupCenter.actions.skip")}
        />
      }
    >
      <SetupCardRows>
        <SetupCardRow label={t("setupCenter.dependencies.toolsTitle")}>
          <span className="text-xs text-muted-foreground">
            {toolIssue ? t("setupCenter.dependencies.missing") : t("setupCenter.dependencies.installed")}
          </span>
        </SetupCardRow>
        {tool?.version?.trim() ? (
          <>
            <SetupCardSeparator />
            <SetupCardRow label={t("settings.externalTools.detail.currentVersion")}>
              <span className="text-xs text-muted-foreground">{tool.version}</span>
            </SetupCardRow>
          </>
        ) : null}
      </SetupCardRows>

      <SetupCardSeparator />
      <SetupCardSection className="flex h-9 items-center justify-between gap-3 whitespace-nowrap">
        <div className="min-w-0 flex-1 overflow-hidden">
          {installBlockedByBun && !canVerify ? (
            <span className="block truncate text-[11px] text-muted-foreground">
              {t("setupCenter.dependencies.clawhubWaitForBun")}
            </span>
          ) : (
            actionFeedback
          )}
        </div>
        <Button
          type="button"
          size="compact"
          variant="outline"
          className="shrink-0"
          disabled={installBlockedByBun || isInstalling || isVerifyPending}
          onClick={() => {
            if (canVerify) {
              void handleVerify();
              return;
            }
            void handleInstall();
          }}
        >
          {isInstalling || isVerifyPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <ActionIcon className="h-3.5 w-3.5" />
          )}
          {isVerifyPending ? t("settings.externalTools.installDialog.stage.verifying") : actionLabel}
        </Button>
      </SetupCardSection>
    </SetupPageCard>
  );
}

export function useInlineRequiredTools() {
  const { enabled: assistantUiEnabled } = useAssistantUiMode();
  return React.useMemo(
    () =>
      assistantUiEnabled ? [...FULL_MODE_REQUIRED_TOOLS] : [...DOWNLOAD_MODE_REQUIRED_TOOLS],
    [assistantUiEnabled]
  );
}
