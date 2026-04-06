import type { ProviderModel } from "@/shared/store/providers";

type ModelCapability = "tools" | "reasoning" | "image" | "audio" | "video";

const ORDERED_MODEL_CAPABILITIES: ModelCapability[] = [
  "tools",
  "reasoning",
  "image",
  "audio",
  "video",
];

export interface ModelMeta {
  description: string;
  contextWindow?: number;
  capabilities: ModelCapability[];
}

export function parseModelMeta(model: ProviderModel): ModelMeta {
  const capabilities = new Set<ModelCapability>();
  let contextWindow: number | undefined;
  let description = "";
  const allowJsonTools = model.supportsTools === undefined;
  const allowJsonReasoning = model.supportsReasoning === undefined;
  const allowJsonVision = model.supportsVision === undefined;
  const allowJsonAudio = model.supportsAudio === undefined;
  const allowJsonVideo = model.supportsVideo === undefined;

  if (typeof model.contextWindowTokens === "number" && model.contextWindowTokens > 0) {
    contextWindow = model.contextWindowTokens;
  }
  if (model.supportsTools) {
    capabilities.add("tools");
  }
  if (model.supportsReasoning) {
    capabilities.add("reasoning");
  }
  if (model.supportsVision) {
    capabilities.add("image");
  }
  if (model.supportsAudio) {
    capabilities.add("audio");
  }
  if (model.supportsVideo) {
    capabilities.add("video");
  }

  const raw = model.capabilitiesJson?.trim();
  if (raw) {
    try {
      const data = JSON.parse(raw) as Record<string, any>;
      description =
        pickString(
          data?.description,
          data?.summary,
          data?.display_name,
          data?.displayName,
          data?.name
        ) ?? "";

      if (allowJsonTools) {
        const supportsTools = resolveSupportFlag(
          data?.supports?.tools ??
            data?.supports?.tool_calling ??
            data?.supports?.toolCalling ??
            data?.supports?.function_calling ??
            data?.supports?.functionCalling ??
            data?.supports?.functions ??
            data?.tool_call ??
            data?.tool_calling ??
            data?.toolCalling ??
            data?.function_calling ??
            data?.functionCalling ??
            data?.tools ??
            data?.functions
        );
        if (supportsTools) {
          capabilities.add("tools");
        }
      }

      if (allowJsonReasoning) {
        const supportsReasoning = resolveSupportFlag(
          data?.supports?.reasoning ??
            data?.supports?.chain_of_thought ??
            data?.supports?.cot ??
            data?.reasoning ??
            data?.chain_of_thought ??
            data?.cot
        );
        if (supportsReasoning) {
          capabilities.add("reasoning");
        }
      }

      if (allowJsonVision || allowJsonAudio || allowJsonVideo) {
        const collected = new Set<ModelCapability>();
        collectModalities(collected, data?.modalities);
        collectModalities(collected, data?.modalities?.input);
        collectModalities(collected, data?.modalities?.output);
        collectModalities(collected, data?.input_modalities);
        collectModalities(collected, data?.output_modalities);
        collectModalities(collected, data?.inputModalities);
        collectModalities(collected, data?.outputModalities);
        if (allowJsonVision && collected.has("image")) {
          capabilities.add("image");
        }
        if (allowJsonAudio && collected.has("audio")) {
          capabilities.add("audio");
        }
        if (allowJsonVideo && collected.has("video")) {
          capabilities.add("video");
        }
      }

      if (allowJsonVision && (resolveSupportFlag(data?.supports?.vision) || resolveSupportFlag(data?.vision))) {
        capabilities.add("image");
      }
      if (allowJsonAudio && (resolveSupportFlag(data?.supports?.audio) || resolveSupportFlag(data?.audio))) {
        capabilities.add("audio");
      }
      if (allowJsonVideo && (resolveSupportFlag(data?.supports?.video) || resolveSupportFlag(data?.video))) {
        capabilities.add("video");
      }

      if (contextWindow === undefined) {
        contextWindow = resolveContextWindow(data);
      }
    } catch {
      // ignore malformed capability JSON
    }
  }

  if (!description || description.toLowerCase() === model.name.toLowerCase()) {
    description = model.displayName?.trim() ?? "";
  }
  if (description.toLowerCase() === model.name.toLowerCase()) {
    description = "";
  }

  const orderedCapabilities = ORDERED_MODEL_CAPABILITIES.filter((item) => capabilities.has(item));

  return {
    description,
    contextWindow,
    capabilities: orderedCapabilities,
  };
}

export function formatContextWindow(value: number): string {
  if (value >= 1000) {
    return `${Math.round(value / 1000)}k`;
  }
  return `${value}`;
}

function pickString(...values: Array<string | undefined | null>): string | undefined {
  for (const value of values) {
    const trimmed = value?.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return undefined;
}

function resolveSupportFlag(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "number") {
    return value > 0;
  }
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "yes" || normalized === "1";
  }
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  if (value && typeof value === "object") {
    return Object.keys(value as Record<string, unknown>).length > 0;
  }
  return false;
}

function collectModalities(target: Set<ModelCapability>, value: unknown) {
  if (!value) {
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((entry) => {
      const mapped = mapModality(entry);
      if (mapped) {
        target.add(mapped);
      }
    });
    return;
  }
  if (typeof value === "string") {
    const mapped = mapModality(value);
    if (mapped) {
      target.add(mapped);
    }
    return;
  }
  if (typeof value === "object") {
    Object.keys(value as Record<string, unknown>).forEach((key) => {
      const mapped = mapModality(key);
      if (mapped) {
        target.add(mapped);
      }
    });
  }
}

function mapModality(value: unknown): ModelCapability | null {
  if (typeof value !== "string") {
    return null;
  }
  const normalized = value.trim().toLowerCase();
  if (normalized === "text") {
    return null;
  }
  if (normalized === "vision" || normalized === "image" || normalized === "images") {
    return "image";
  }
  if (normalized === "audio" || normalized === "speech") {
    return "audio";
  }
  if (normalized === "video") {
    return "video";
  }
  return null;
}

function resolveContextWindow(data: Record<string, any>): number | undefined {
  const candidates = [
    data?.context_window,
    data?.contextWindow,
    data?.context_length,
    data?.contextLength,
    data?.max_context_length,
    data?.maxContextLength,
    data?.limit?.context,
    data?.limits?.context,
    data?.context?.window,
    data?.context?.length,
    data?.context?.max,
  ];
  for (const candidate of candidates) {
    const parsed = parseContextValue(candidate);
    if (parsed) {
      return parsed;
    }
  }
  return undefined;
}

function parseContextValue(value: unknown): number | undefined {
  if (typeof value === "number" && !Number.isNaN(value)) {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim().toLowerCase();
    const match = trimmed.match(/(\d+(\.\d+)?)(k)?/);
    if (!match) {
      return undefined;
    }
    const numeric = Number(match[1]);
    if (Number.isNaN(numeric)) {
      return undefined;
    }
    return match[3] ? Math.round(numeric * 1000) : Math.round(numeric);
  }
  return undefined;
}
