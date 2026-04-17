import type { AssistantModelOption } from "@/features/settings/gateway/ui/assistant-parameter-panels/model-utils";
import type { ProxySettings } from "@/shared/contracts/settings";
import type { ExternalTool, ExternalToolInstallState } from "@/shared/store/externalTools";

import { PROVIDER_PRESETS } from "./constants";

type ProviderOption = {
  id: string;
  label: string;
  endpoint: string;
  type: string;
  compatibility: string;
};

type TranslateFn = (key: string) => string;

export const clampProgress = (value?: number) => {
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

export const createManualProxyDraft = (current?: ProxySettings): ProxySettings => ({
  mode: "manual",
  scheme: current?.scheme ?? "http",
  host: current?.host ?? "",
  port: current?.port ?? 0,
  username: current?.username ?? "",
  password: current?.password ?? "",
  noProxy: current?.noProxy ?? [],
  timeoutSeconds: current?.timeoutSeconds ?? 15,
  testedAt: "",
  testSuccess: false,
  testMessage: "",
});

export const resolveProviderOption = (
  allProviders: Array<{ id: string; name: string; endpoint: string; type: string; compatibility: string }>
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
      compatibility: matched?.compatibility?.trim() || preset.compatibility,
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
      compatibility: provider.compatibility,
    });
  }

  return options;
};

export const resolveModelRefFromOptionValue = (value: string, options: AssistantModelOption[]) => {
  if (!value) {
    return "";
  }
  return options.find((option) => option.value === value)?.modelRef ?? "";
};

export const resolveInstallActionLabel = (tool: ExternalTool | null, t: TranslateFn) =>
  String(tool?.status ?? "").trim().toLowerCase() === "invalid"
    ? t("settings.externalTools.actions.repair")
    : t("settings.externalTools.actions.install");

const normalizeToolVersion = (version?: string, toolName?: string) => {
  let value = (version ?? "").trim();
  if (!value) {
    return "";
  }
  value = value.replace(/^v/i, "");
  if (toolName?.toLowerCase() === "ffmpeg") {
    value = value.replace(/^n-/i, "");
    value = value.replace(/-tessus$/i, "");
  }
  return value;
};

export const formatToolVersion = (version?: string, toolName?: string) => {
  const value = normalizeToolVersion(version, toolName);
  if (!value) {
    return "";
  }
  return `v${value}`;
};

export const buildInstallStateHandledKey = (
  name: string,
  state: ExternalToolInstallState | null | undefined
) => {
  const stage = String(state?.stage ?? "").trim();
  if (stage !== "done" && stage !== "error") {
    return "";
  }
  return `${name}:${stage}:${state?.updatedAt ?? ""}:${state?.message ?? ""}`;
};
