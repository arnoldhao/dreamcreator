import type { Assistant } from "@/shared/store/assistant";
import type { ExternalTool } from "@/shared/store/externalTools";
import type { ProviderWithModels } from "@/shared/store/providers";

export type SharedGatewayIssueCode =
  | "gateway.disabled"
  | "assistant.selection"
  | "assistant.disabled"
  | "providers.models"
  | "model.agent.primary"
  | "model.reference.invalid"
  | "provider.unavailable"
  | "model.unavailable"
  | "provider.apiKey.missing"
  | `assistant.${string}`;

export type ToolDependencyIssue = {
  name: string;
  status: "missing" | "invalid";
};

const SHARED_GATEWAY_PRIORITY: Record<string, number> = {
  "providers.models": 10,
  "model.agent.primary": 20,
  "model.reference.invalid": 30,
  "provider.unavailable": 40,
  "model.unavailable": 50,
  "provider.apiKey.missing": 60,
  "assistant.selection": 70,
  "assistant.disabled": 80,
  "gateway.disabled": 90,
};

export const parseModelRef = (value: string) => {
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

export const findToolByName = (tools: ExternalTool[], name: string) =>
  tools.find((tool) => tool.name?.trim().toLowerCase() === name.toLowerCase()) ?? null;

export const isInstalledTool = (tool: ExternalTool | null | undefined) =>
  Boolean(tool && String(tool.status ?? "").trim().toLowerCase() === "installed" && tool.execPath?.trim());

export function resolveToolDependencyIssues(required: readonly string[], tools: ExternalTool[]): ToolDependencyIssue[] {
  return required.flatMap((name) => {
    const tool = findToolByName(tools, name);
    const status = String(tool?.status ?? "").trim().toLowerCase();
    const execPath = tool?.execPath?.trim();
    if (tool && status === "installed" && execPath) {
      return [];
    }
    return [{ name, status: status === "invalid" ? "invalid" : "missing" as const }];
  });
}

type ResolveGatewayCoreReadinessOptions = {
  assistant: Assistant | null;
  providersWithModels: ProviderWithModels[];
  gatewayEnabled?: boolean;
  hasProviderApiKey?: boolean | null;
  checking?: boolean;
  includeGatewayDisabled?: boolean;
  requireAssistantEnabled?: boolean;
};

export function resolveGatewayCoreReadiness({
  assistant,
  providersWithModels,
  gatewayEnabled = true,
  hasProviderApiKey = null,
  checking = false,
  includeGatewayDisabled = true,
  requireAssistantEnabled = false,
}: ResolveGatewayCoreReadinessOptions) {
  if (checking) {
    return {
      checking: true,
      ready: false,
      issues: [] as SharedGatewayIssueCode[],
      parsedModelRef: { providerId: "", modelName: "" },
      providerEntry: null as ProviderWithModels | null,
      modelExists: false,
    };
  }

  const issues = new Set<SharedGatewayIssueCode>();

  if (!assistant) {
    issues.add("assistant.selection");
  }

  if (assistant && requireAssistantEnabled && !assistant.enabled) {
    issues.add("assistant.disabled");
  }

  if (assistant) {
    for (const item of assistant.readiness?.missing ?? []) {
      if (item === "providers.models" || item === "model.agent.primary") {
        issues.add(item);
        continue;
      }
      const normalized = item?.trim();
      if (normalized) {
        issues.add(`assistant.${normalized}`);
      }
    }
  }

  if (includeGatewayDisabled && !gatewayEnabled) {
    issues.add("gateway.disabled");
  }

  const totalModels = providersWithModels.reduce((count, item) => count + item.models.length, 0);
  if (totalModels === 0) {
    issues.add("providers.models");
  }

  const primaryModelRef = assistant?.model?.agent?.primary?.trim() ?? "";
  if (!primaryModelRef) {
    issues.add("model.agent.primary");
  }

  const parsedModelRef = parseModelRef(primaryModelRef);
  let providerEntry: ProviderWithModels | null = null;
  let modelExists = false;

  if (primaryModelRef) {
    if (!parsedModelRef.providerId || !parsedModelRef.modelName) {
      issues.add("model.reference.invalid");
    } else {
      providerEntry =
        providersWithModels.find(
          (item) => item.provider.id.trim().toLowerCase() === parsedModelRef.providerId.trim().toLowerCase()
        ) ?? null;
      if (!providerEntry) {
        issues.add("provider.unavailable");
      } else {
        modelExists = providerEntry.models.some(
          (item) => item.name.trim().toLowerCase() === parsedModelRef.modelName.trim().toLowerCase()
        );
        if (!modelExists) {
          issues.add("model.unavailable");
        }
      }
    }
  }

  if (hasProviderApiKey === false && parsedModelRef.providerId) {
    issues.add("provider.apiKey.missing");
  }

  const sortedIssues = Array.from(issues).sort((left, right) => {
    const leftPriority = SHARED_GATEWAY_PRIORITY[left] ?? 999;
    const rightPriority = SHARED_GATEWAY_PRIORITY[right] ?? 999;
    if (leftPriority !== rightPriority) {
      return leftPriority - rightPriority;
    }
    return left.localeCompare(right);
  });

  return {
    checking: false,
    ready: sortedIssues.length === 0,
    issues: sortedIssues,
    parsedModelRef,
    providerEntry,
    modelExists,
  };
}
