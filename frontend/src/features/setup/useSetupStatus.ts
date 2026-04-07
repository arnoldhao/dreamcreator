import * as React from "react";

import { useI18n } from "@/shared/i18n";
import { useAssistants } from "@/shared/query/assistant";
import { useExternalTools } from "@/shared/query/externalTools";
import { useEnabledProvidersWithModels, useProviderSecret } from "@/shared/query/providers";
import { useSettingsStore } from "@/shared/store/settings";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import { formatTemplate } from "@/features/library/utils/i18n";

import {
  DOWNLOAD_MODE_REQUIRED_TOOLS,
  FULL_MODE_REQUIRED_TOOLS,
  type SetupIssue,
} from "./constants";
import { buildSetupNavStatusEntries, resolveProxyStatus } from "./nav";
import { parseModelRef, resolveGatewayCoreReadiness, resolveToolDependencyIssues } from "./readiness";
import { useSetupCenter } from "./store";

type SetupEntryKind = "setup" | "notifications";

export function useSetupStatus(unreadNoticeCount = 0) {
  const { t } = useI18n();
  const { enabled: assistantUiEnabled } = useAssistantUiMode();
  const { aiDeferred, dependencyDeferred, skippedItemIds } = useSetupCenter();
  const settings = useSettingsStore((state) => state.settings);
  const settingsLoading = useSettingsStore((state) => state.isLoading);
  const assistantsQuery = useAssistants(true);
  const providersQuery = useEnabledProvidersWithModels();
  const externalToolsQuery = useExternalTools();

  const defaultAssistant = React.useMemo(() => {
    const assistants = assistantsQuery.data ?? [];
    return assistants.find((item) => item.isDefault) ?? assistants[0] ?? null;
  }, [assistantsQuery.data]);
  const parsedDefaultModel = React.useMemo(
    () => parseModelRef(defaultAssistant?.model?.agent?.primary?.trim() ?? ""),
    [defaultAssistant?.model?.agent?.primary]
  );
  const providerSecretQuery = useProviderSecret(parsedDefaultModel.providerId || null);

  const requiredTools = React.useMemo(
    () => (assistantUiEnabled ? [...FULL_MODE_REQUIRED_TOOLS] : [...DOWNLOAD_MODE_REQUIRED_TOOLS]),
    [assistantUiEnabled]
  );

  const missingRequiredTools = React.useMemo(
    () => resolveToolDependencyIssues(requiredTools, externalToolsQuery.data ?? []),
    [externalToolsQuery.data, requiredTools]
  );
  const proxyReady = resolveProxyStatus(settings?.proxy, null) === "ready";

  const languageReady = Boolean(settings?.language?.trim());
  const gatewayEnabled = Boolean(settings?.gateway.controlPlaneEnabled);
  const gatewayReadiness = React.useMemo(
    () =>
      resolveGatewayCoreReadiness({
        assistant: defaultAssistant,
        providersWithModels: providersQuery.data ?? [],
        gatewayEnabled,
        hasProviderApiKey: parsedDefaultModel.providerId ? Boolean(providerSecretQuery.data?.apiKey?.trim()) : null,
        checking: assistantsQuery.isLoading || providersQuery.isLoading || providerSecretQuery.isLoading,
        includeGatewayDisabled: true,
        requireAssistantEnabled: false,
      }),
    [
      assistantsQuery.isLoading,
      defaultAssistant,
      gatewayEnabled,
      parsedDefaultModel.providerId,
      providerSecretQuery.data?.apiKey,
      providerSecretQuery.isLoading,
      providersQuery.data,
      providersQuery.isLoading,
    ]
  );
  const providersReady = !gatewayReadiness.issues.includes("providers.models");
  const agentModelReady = !gatewayReadiness.issues.includes("model.agent.primary");
  const productModeReady = typeof assistantUiEnabled === "boolean";
  const dependencyReady = productModeReady && missingRequiredTools.length === 0;
  const generalReady = languageReady;
  const aiReady = gatewayReadiness.ready;
  const aiDegraded = !aiReady && aiDeferred;
  const dependencyDegraded = !dependencyReady && dependencyDeferred;

  const issues = React.useMemo<SetupIssue[]>(() => {
    const next: SetupIssue[] = [];

    if (!languageReady) {
      next.push({ code: "general.language", severity: "blocking", step: "general" });
    }
    if (!productModeReady) {
      next.push({
        code: "dependency.productMode",
        severity: dependencyDeferred ? "recommended" : "blocking",
        step: "dependencies",
      });
    }
    if (missingRequiredTools.length > 0) {
      next.push({
        code: assistantUiEnabled ? "dependency.fullTools" : "dependency.downloadTools",
        severity: dependencyDeferred ? "recommended" : "blocking",
        step: "dependencies",
        meta: { count: missingRequiredTools.length, names: missingRequiredTools.map((item) => item.name).join(", ") },
      });
    }
    if (gatewayReadiness.issues.includes("gateway.disabled")) {
      next.push({
        code: "ai.gateway",
        severity: aiDeferred ? "recommended" : "blocking",
        step: "ai",
      });
    }
    if (
      gatewayReadiness.issues.includes("providers.models") ||
      gatewayReadiness.issues.includes("provider.unavailable") ||
      gatewayReadiness.issues.includes("provider.apiKey.missing")
    ) {
      next.push({
        code: "ai.providers",
        severity: aiDeferred ? "recommended" : "blocking",
        step: "ai",
      });
    }
    if (
      gatewayReadiness.issues.includes("model.agent.primary") ||
      gatewayReadiness.issues.includes("model.reference.invalid") ||
      gatewayReadiness.issues.includes("model.unavailable")
    ) {
      next.push({
        code: "ai.model",
        severity: aiDeferred ? "recommended" : "blocking",
        step: "ai",
      });
    }

    return next;
  }, [
    aiDeferred,
    assistantUiEnabled,
    dependencyDeferred,
    gatewayReadiness.issues,
    productModeReady,
    languageReady,
    missingRequiredTools,
  ]);

  const blockingIssues = issues.filter((item) => item.severity === "blocking");
  const recommendedIssues = issues.filter((item) => item.severity === "recommended");
  const currentIssue = blockingIssues[0] ?? recommendedIssues[0] ?? null;
  const providerCardReady = providersReady;
  const modelCardReady = gatewayEnabled && agentModelReady;
  const configuredToolCount = requiredTools.length - missingRequiredTools.length;
  const configuredChecks = [
    generalReady,
    proxyReady,
    providerCardReady,
    modelCardReady,
    productModeReady,
    ...requiredTools.map((name) => !missingRequiredTools.some((item) => item.name === name)),
  ].filter(Boolean).length;
  const totalChecks = 5 + requiredTools.length;
  const completedChecks = [
    generalReady,
    proxyReady,
    providerCardReady || aiDeferred,
    modelCardReady || aiDeferred,
    productModeReady || dependencyDeferred,
    ...requiredTools.map((name) => {
      const installed = !missingRequiredTools.some((item) => item.name === name);
      return installed || dependencyDeferred;
    }),
  ].filter(Boolean).length;
  const progress = Math.round((completedChecks / totalChecks) * 100);
  const checking =
    settingsLoading ||
    gatewayReadiness.checking ||
    externalToolsQuery.isLoading;
  const setupComplete = generalReady && (dependencyReady || dependencyDeferred) && (aiReady || aiDeferred);
  const navStatusEntries = React.useMemo(
    () =>
      buildSetupNavStatusEntries({
        languageReady,
        proxyReady,
        providersReady,
        gatewayEnabled,
        agentModelReady,
        hasChosenProductMode: productModeReady,
        requiredTools,
        missingRequiredToolNames: missingRequiredTools.map((item) => item.name),
        skippedItemIds,
      }),
    [
      agentModelReady,
      gatewayEnabled,
      productModeReady,
      languageReady,
      missingRequiredTools,
      providersReady,
      proxyReady,
      requiredTools,
      skippedItemIds,
    ]
  );
  const pendingNavItemCount = navStatusEntries.filter((entry) => entry.status === "pending").length;
  const shouldAutoOpen = !checking && pendingNavItemCount > 0;

  const currentTitle = currentIssue
    ? resolveIssueTitle(currentIssue.code, t)
    : unreadNoticeCount > 0
      ? t("sidebar.footer.menu.notifications")
      : t("setupCenter.footer.readyTitle");
  const currentDescription = currentIssue
    ? resolveIssueDescription(currentIssue, t)
    : unreadNoticeCount > 0
      ? formatTemplate(t("setupCenter.footer.notificationsDescription"), { count: unreadNoticeCount })
      : t("setupCenter.footer.readyDescription");

  return {
    checking,
    issues,
    blockingIssues,
    recommendedIssues,
    currentIssue,
    currentTitle,
    currentDescription,
    progress,
    totalChecks,
    configuredChecks,
    completedChecks,
    configuredToolCount,
    generalReady,
    dependencyReady,
    dependencyDegraded,
    aiReady,
    aiDegraded,
    setupComplete,
    shouldAutoOpen,
    pendingNavItemCount,
    hasChosenProductMode: productModeReady,
    assistantUiEnabled,
    missingRequiredTools,
    defaultAssistant,
    providersReady,
    agentModelReady,
    gatewayEnabled,
    footerEntryKind: currentIssue ? ("setup" as SetupEntryKind) : ("notifications" as SetupEntryKind),
  };
}

function resolveIssueTitle(code: SetupIssue["code"], t: (key: string) => string) {
  switch (code) {
    case "general.language":
      return t("setupCenter.issues.language.title");
    case "ai.gateway":
      return t("setupCenter.issues.gateway.title");
    case "ai.providers":
      return t("setupCenter.issues.providers.title");
    case "ai.model":
      return t("setupCenter.issues.model.title");
    case "dependency.productMode":
      return t("setupCenter.issues.productMode.title");
    case "dependency.downloadTools":
      return t("setupCenter.issues.downloadTools.title");
    case "dependency.fullTools":
      return t("setupCenter.issues.fullTools.title");
    default:
      return t("setupCenter.footer.action");
  }
}

function resolveIssueDescription(
  issue: SetupIssue,
  t: (key: string) => string
) {
  const count = Number(issue.meta?.count ?? 0);

  switch (issue.code) {
    case "general.language":
      return t("setupCenter.issues.language.description");
    case "ai.gateway":
      return issue.severity === "recommended"
        ? t("setupCenter.issues.aiSkippedDescription")
        : t("setupCenter.issues.gateway.description");
    case "ai.providers":
      return issue.severity === "recommended"
        ? t("setupCenter.issues.aiSkippedDescription")
        : t("setupCenter.issues.providers.description");
    case "ai.model":
      return issue.severity === "recommended"
        ? t("setupCenter.issues.aiSkippedDescription")
        : t("setupCenter.issues.model.description");
    case "dependency.productMode":
      return issue.severity === "recommended"
        ? t("setupCenter.issues.dependenciesSkippedDescription")
        : t("setupCenter.issues.productMode.description");
    case "dependency.downloadTools":
    case "dependency.fullTools":
      return issue.severity === "recommended"
        ? t("setupCenter.issues.dependenciesSkippedDescription")
        : formatTemplate(t("setupCenter.issues.tools.description"), { count });
    default:
      return t("setupCenter.footer.action");
  }
}
