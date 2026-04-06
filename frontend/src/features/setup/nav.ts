import type { ProxySettings } from "@/shared/contracts/settings";

import type { SetupStepId } from "./constants";

export type StaticSetupNavItemId =
  | "general.language"
  | "general.proxy"
  | "ai.provider"
  | "ai.agentModel"
  | "dependencies.productMode";

export type SetupNavItemId = StaticSetupNavItemId | `dependencies.tool.${string}`;
export type SetupNavStatus = "ready" | "pending" | "skipped";

export type SetupNavStatusEntry = {
  id: SetupNavItemId;
  status: SetupNavStatus;
};

type BuildSetupNavStatusEntriesOptions = {
  languageReady: boolean;
  proxyReady: boolean;
  providersReady: boolean;
  gatewayEnabled: boolean;
  agentModelReady: boolean;
  hasChosenProductMode: boolean;
  requiredTools: readonly string[];
  missingRequiredToolNames: readonly string[];
  skippedItemIds: readonly string[];
};

export const TOOL_ITEM_PREFIX = "dependencies.tool.";

export const getToolItemId = (name: string) => `${TOOL_ITEM_PREFIX}${name}` as SetupNavItemId;

export const isSkippableItem = (itemId: SetupNavItemId) =>
  itemId.startsWith("ai.") || itemId.startsWith(TOOL_ITEM_PREFIX);

export const resolveStepDefaultItem = (step: SetupStepId): SetupNavItemId => {
  switch (step) {
    case "general":
      return "general.language";
    case "ai":
      return "ai.provider";
    case "dependencies":
      return "dependencies.productMode";
    default:
      return "general.language";
  }
};

export const resolveItemPage = (itemId: SetupNavItemId): SetupStepId => {
  if (itemId.startsWith("general.")) {
    return "general";
  }
  if (itemId.startsWith("ai.")) {
    return "ai";
  }
  return "dependencies";
};

export const resolveGroupStatus = (statuses: readonly SetupNavStatus[]): SetupNavStatus => {
  if (statuses.length === 0) {
    return "pending";
  }
  if (statuses.every((status) => status === "skipped")) {
    return "skipped";
  }
  if (statuses.every((status) => status === "ready" || status === "skipped")) {
    return "ready";
  }
  return "pending";
};

export const resolveProxyStatus = (
  settings: ProxySettings | undefined,
  draft: ProxySettings | null
): SetupNavStatus => {
  const next = draft ?? settings;
  if (!next || next.mode === "none" || next.mode === "system") {
    return "ready";
  }
  return next.testSuccess ? "ready" : "pending";
};

export const buildSetupNavStatusEntries = ({
  languageReady,
  proxyReady,
  providersReady,
  gatewayEnabled,
  agentModelReady,
  hasChosenProductMode,
  requiredTools,
  missingRequiredToolNames,
  skippedItemIds,
}: BuildSetupNavStatusEntriesOptions): SetupNavStatusEntry[] => {
  const skippedItemIdSet = new Set(skippedItemIds);
  const missingRequiredToolNameSet = new Set(missingRequiredToolNames);
  const resolveStatus = (itemId: SetupNavItemId, ready: boolean): SetupNavStatus => {
    if (ready) {
      return "ready";
    }
    return skippedItemIdSet.has(itemId) ? "skipped" : "pending";
  };

  return [
    { id: "general.language", status: resolveStatus("general.language", languageReady) },
    { id: "general.proxy", status: resolveStatus("general.proxy", proxyReady) },
    { id: "ai.provider", status: resolveStatus("ai.provider", providersReady) },
    {
      id: "ai.agentModel",
      status: resolveStatus("ai.agentModel", gatewayEnabled && agentModelReady),
    },
    {
      id: "dependencies.productMode",
      status: resolveStatus("dependencies.productMode", hasChosenProductMode),
    },
    ...requiredTools.map((name) => ({
      id: getToolItemId(name),
      status: resolveStatus(getToolItemId(name), !missingRequiredToolNameSet.has(name)),
    })),
  ];
};
