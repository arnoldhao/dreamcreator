import * as React from "react";

import { useExternalTools } from "@/shared/query/externalTools";
import type { Assistant } from "@/shared/store/assistant";
import { useSettingsStore } from "@/shared/store/settings";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import { resolveGatewayCoreReadiness, resolveToolDependencyIssues } from "@/features/setup/readiness";

import type { ModelGroup } from "../types";

export type ChatReadinessIssueKey =
  | "gateway.disabled"
  | "assistant.selection"
  | "providers.models"
  | "model.agent.primary"
  | "external.bun"
  | "external.clawhub"
  | `assistant.${string}`;

type UseChatReadinessOptions = {
  assistant: Assistant | null;
  modelGroups: ModelGroup[];
  loading: boolean;
};

const READINESS_ISSUE_PRIORITY: Record<string, number> = {
  "providers.models": 10,
  "model.agent.primary": 20,
  "external.bun": 30,
  "external.clawhub": 40,
  "assistant.selection": 50,
  "gateway.disabled": 60,
};

export function useChatReadiness({
  assistant,
  modelGroups,
  loading,
}: UseChatReadinessOptions) {
  const gatewayEnabled = useSettingsStore((state) => state.settings?.gateway.controlPlaneEnabled ?? false);
  const settingsLoading = useSettingsStore((state) => state.isLoading);
  const { data: externalTools = [], isLoading: externalToolsLoading } = useExternalTools();
  const providersQuery = useEnabledProvidersWithModels();
  const gatewayReadiness = React.useMemo(
    () =>
      resolveGatewayCoreReadiness({
        assistant,
        providersWithModels: providersQuery.data ?? [],
        gatewayEnabled,
        checking: settingsLoading || loading || providersQuery.isLoading,
        includeGatewayDisabled: true,
        requireAssistantEnabled: false,
      }),
    [assistant, gatewayEnabled, loading, providersQuery.data, providersQuery.isLoading, settingsLoading]
  );

  const environmentIssues = React.useMemo(() => {
    const next: ChatReadinessIssueKey[] = [];

    const dependencyIssues = resolveToolDependencyIssues(["bun", "clawhub"], externalTools);
    if (dependencyIssues.some((item) => item.name === "bun")) {
      next.push("external.bun");
    }
    if (dependencyIssues.some((item) => item.name === "clawhub")) {
      next.push("external.clawhub");
    }

    return next;
  }, [externalTools]);

  const assistantIssues = gatewayReadiness.issues.filter(
    (issue) =>
      issue === "assistant.selection" ||
      issue === "providers.models" ||
      issue === "model.agent.primary" ||
      issue === "gateway.disabled" ||
      issue.startsWith("assistant.")
  ) as ChatReadinessIssueKey[];
  const isChecking = gatewayReadiness.checking || externalToolsLoading;
  const issues = React.useMemo(() => {
    return [...environmentIssues, ...assistantIssues].sort((left, right) => {
      const leftPriority = READINESS_ISSUE_PRIORITY[left] ?? 100;
      const rightPriority = READINESS_ISSUE_PRIORITY[right] ?? 100;
      if (leftPriority !== rightPriority) {
        return leftPriority - rightPriority;
      }
      return left.localeCompare(right);
    });
  }, [assistantIssues, environmentIssues]);

  return {
    assistantReady: !isChecking && assistantIssues.length === 0,
    clawhubReady: !issues.includes("external.clawhub"),
    gatewayReady: !assistantIssues.includes("gateway.disabled"),
    isChecking,
    isReady: !isChecking && issues.length === 0,
    issues,
  };
}
