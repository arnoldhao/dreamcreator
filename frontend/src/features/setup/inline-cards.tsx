import * as React from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle2,
  Download,
  Loader2,
  RefreshCw,
  Wrench,
} from "lucide-react";

import { messageBus } from "@/shared/message/store";
import { useI18n } from "@/shared/i18n";
import { formatTemplate } from "@/features/library/utils/i18n";
import { useAssistants, useUpdateAssistant } from "@/shared/query/assistant";
import {
  useExternalToolInstallState,
  useExternalTools,
  useExternalToolUpdates,
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
import type {
  ExternalToolInstallState,
  ExternalToolUpdateInfo,
} from "@/shared/store/externalTools";
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
import {
  buildInstallStateHandledKey,
  clampProgress,
  formatToolVersion,
  resolveInstallActionLabel,
  resolveModelRefFromOptionValue,
  resolveProviderOption,
} from "./setup-center-utils";
import { useSetupCenter } from "./store";

const EXTERNAL_INSTALL_STAGES = new Set(["downloading", "extracting", "verifying"]);

export function SetupProviderCard({
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
            <div className="flex min-w-0 items-center gap-2">
              {selectedProvider?.icon ? (
                <span
                  aria-hidden
                  className="h-4 w-4 shrink-0 bg-current text-muted-foreground"
                  style={{
                    WebkitMaskImage: `url(${selectedProvider.icon})`,
                    maskImage: `url(${selectedProvider.icon})`,
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
              <div className="truncate text-xs font-medium text-muted-foreground">
                {selectedProvider?.name ?? selectedProviderOption?.label ?? t("setupCenter.ai.noProvider")}
              </div>
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

export function SetupAgentModelCard({
  status,
  showSkip = false,
  onSkip,
  autoApplyWhenReady = false,
}: {
  status: SetupNavStatus;
  showSkip?: boolean;
  onSkip?: () => void;
  autoApplyWhenReady?: boolean;
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
  const syncKeyRef = React.useRef("");

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
      const syncKey = `${defaultAssistant.id}:${normalizedNext}:${needsGatewayEnable ? "enable" : "keep"}`;
      if (syncKeyRef.current === syncKey) {
        return;
      }
      syncKeyRef.current = syncKey;
      try {
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
      } finally {
        if (syncKeyRef.current === syncKey) {
          syncKeyRef.current = "";
        }
      }
      clearSkippedItem("ai.agentModel");
      clearAiDeferred();
    },
    [clearAiDeferred, clearSkippedItem, defaultAssistant, gatewayEnabled, updateAssistant, updateSettings]
  );

  React.useEffect(() => {
    if (!autoApplyWhenReady || !defaultAssistant || agentModelOptions.length === 0) {
      return;
    }
    const currentPrimary = defaultAssistant.model?.agent?.primary?.trim() ?? "";
    const nextModelRef = resolveUsableModelRef(currentPrimary, agentModelOptions);
    if (!nextModelRef) {
      return;
    }
    const modelReady = modelRefEquals(currentPrimary, nextModelRef);
    if (modelReady && gatewayEnabled) {
      return;
    }
    void applyAgentModel(nextModelRef);
  }, [agentModelOptions, applyAgentModel, autoApplyWhenReady, defaultAssistant, gatewayEnabled]);

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

export function SetupProductModeCard({
  status,
  headerRight,
}: {
  status: SetupNavStatus;
  headerRight?: React.ReactNode;
}) {
  const { t } = useI18n();
  const { enabled: assistantUiEnabled, setEnabled: setAssistantUiEnabled } = useAssistantUiMode();
  const { clearDependencyDeferred } = useSetupCenter();

  return (
    <SetupPageCard
      title={t("setupCenter.dependencies.modeTitle")}
      headerRight={headerRight ?? <SetupStatusIcon status={status} />}
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

export function SetupToolCard({
  name,
  status,
  showSkip = false,
  onSkip,
  activeInstallName,
  activeVerifyName,
  onActiveInstallNameChange,
  onActiveVerifyNameChange,
}: {
  name: string;
  status: SetupNavStatus;
  showSkip?: boolean;
  onSkip?: () => void;
  activeInstallName?: string | null;
  activeVerifyName?: string | null;
  onActiveInstallNameChange?: (name: string | null) => void;
  onActiveVerifyNameChange?: (name: string | null) => void;
}) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { clearDependencyDeferred, clearSkippedItem } = useSetupCenter();
  const externalToolsQuery = useExternalTools();
  const externalToolUpdatesQuery = useExternalToolUpdates();
  const installTool = useInstallExternalTool();
  const verifyTool = useVerifyExternalTool();
  const [localActiveInstallName, setLocalActiveInstallName] = React.useState<string | null>(null);
  const [localActiveVerifyName, setLocalActiveVerifyName] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState("");
  const installStateHandledRef = React.useRef("");
  const controlledInstallName = activeInstallName !== undefined;
  const controlledVerifyName = activeVerifyName !== undefined;
  const currentActiveInstallName = controlledInstallName ? activeInstallName : localActiveInstallName;
  const currentActiveVerifyName = controlledVerifyName ? activeVerifyName : localActiveVerifyName;
  const isInstallingThisTool = currentActiveInstallName === name;
  const shouldTrackInstall = isInstallingThisTool || installTool.isPending;
  const installState = useExternalToolInstallState(name, shouldTrackInstall);

  const tools = externalToolsQuery.data ?? [];
  const tool = tools.find((item) => item.name?.trim().toLowerCase() === name.toLowerCase()) ?? null;
  const toolUpdatesByName = React.useMemo(() => {
    const map = new Map<string, ExternalToolUpdateInfo>();
    for (const update of externalToolUpdatesQuery.data ?? []) {
      if (update.name) {
        map.set(update.name, update);
      }
    }
    return map;
  }, [externalToolUpdatesQuery.data]);
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
  const showVersionComparison = name === "yt-dlp" || name === "ffmpeg" || name === "bun";
  const updateInfo = toolUpdatesByName.get(name);
  const currentVersionLabel = isInstalled
    ? formatToolVersion(tool?.version, name) || t("settings.externalTools.detail.unknown")
    : t("setupCenter.dependencies.missing");
  const latestVersionLabel =
    formatToolVersion(updateInfo?.latestVersion, name) ||
    t("settings.externalTools.detail.unknown");
  const stage = shouldTrackInstall ? String(installState.data?.stage ?? "").trim().toLowerCase() : "";
  const progress = clampProgress(installState.data?.progress);
  const stageMessage = installState.data?.message?.trim() ?? "";
  const isInstalling = shouldTrackInstall || EXTERNAL_INSTALL_STAGES.has(stage);
  const isVerifyPending = currentActiveVerifyName === name && verifyTool.isPending;
  const blockedByOtherInstall = Boolean(currentActiveInstallName && currentActiveInstallName !== name);
  const blockedByOtherVerify = Boolean(currentActiveVerifyName && currentActiveVerifyName !== name);
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
      const handledKey = buildInstallStateHandledKey(name, installState.data);
      if (installStateHandledRef.current === handledKey) {
        return;
      }
      installStateHandledRef.current = handledKey;
      if (!controlledInstallName) {
        setLocalActiveInstallName(null);
      }
      onActiveInstallNameChange?.(null);
      setActionError("");
      clearSkippedItem(getToolItemId(name));
      clearDependencyDeferred();
      void externalToolsQuery.refetch();
      return;
    }
    if (stage === "error") {
      const handledKey = buildInstallStateHandledKey(name, installState.data);
      if (installStateHandledRef.current === handledKey) {
        return;
      }
      installStateHandledRef.current = handledKey;
      if (!controlledInstallName) {
        setLocalActiveInstallName(null);
      }
      onActiveInstallNameChange?.(null);
      setActionError(stageMessage || t("settings.externalTools.installDialog.error"));
      void externalToolsQuery.refetch();
    }
  }, [
    clearDependencyDeferred,
    clearSkippedItem,
    controlledInstallName,
    externalToolsQuery,
    installState.data,
    name,
    onActiveInstallNameChange,
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
    const installStateQueryKey = ["external-tools-install-state", name];
    const previousState = queryClient.getQueryData<ExternalToolInstallState>(installStateQueryKey);
    installStateHandledRef.current = buildInstallStateHandledKey(name, previousState);
    queryClient.setQueryData<ExternalToolInstallState>(installStateQueryKey, {
      name,
      stage: "downloading",
      progress: 0,
      message: "",
      updatedAt: new Date().toISOString(),
    });
    if (!controlledInstallName) {
      setLocalActiveInstallName(name);
    }
    onActiveInstallNameChange?.(name);
    try {
      await installTool.mutateAsync({ name });
      await installState.refetch();
    } catch (error) {
      if (!controlledInstallName) {
        setLocalActiveInstallName(null);
      }
      onActiveInstallNameChange?.(null);
      const message =
        error instanceof Error ? error.message : t("settings.externalTools.installDialog.error");
      queryClient.setQueryData<ExternalToolInstallState>(installStateQueryKey, {
        name,
        stage: "error",
        progress: 0,
        message,
        updatedAt: new Date().toISOString(),
      });
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
    if (!controlledVerifyName) {
      setLocalActiveVerifyName(name);
    }
    onActiveVerifyNameChange?.(name);
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
      if (!controlledVerifyName) {
        setLocalActiveVerifyName(null);
      }
      onActiveVerifyNameChange?.(null);
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
  const toolActionButton = (
    <Button
      type="button"
      size="compact"
      variant="outline"
      className="shrink-0"
      disabled={installBlockedByBun || isInstalling || isVerifyPending || blockedByOtherInstall || blockedByOtherVerify}
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
  );

  return (
    <SetupPageCard
      title={name.toUpperCase()}
      titleClassName="text-xs font-semibold uppercase tracking-[0.24em]"
      headerRight={
        <div className="flex items-center gap-2">
          <SetupCardStatusHeader
            status={status}
            onSkip={onSkip}
            showSkip={showSkip && status === "pending"}
            skipLabel={t("setupCenter.actions.skip")}
          />
          {toolActionButton}
        </div>
      }
    >
      <SetupCardRows>
        <SetupCardRow label={t("setupCenter.dependencies.status")}>
          <span className="text-xs text-muted-foreground">
            {toolIssue ? t("setupCenter.dependencies.missing") : t("setupCenter.dependencies.installed")}
          </span>
        </SetupCardRow>
        {showVersionComparison ? (
          <>
            <SetupCardSeparator />
            <SetupCardRow label={t("setupCenter.dependencies.version")}>
              <span className="text-xs text-muted-foreground">
                {formatTemplate(t("setupCenter.dependencies.versionDescription"), {
                  current: currentVersionLabel,
                  latest: latestVersionLabel,
                })}
              </span>
            </SetupCardRow>
          </>
        ) : null}
      </SetupCardRows>

      <SetupCardSeparator />
      <SetupCardSection className="flex h-9 items-center gap-3 whitespace-nowrap">
        {installBlockedByBun && !canVerify ? (
          <span className="block truncate text-[11px] text-muted-foreground">
            {t("setupCenter.dependencies.clawhubWaitForBun")}
          </span>
        ) : (
          actionFeedback
        )}
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
