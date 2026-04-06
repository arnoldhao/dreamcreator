import { parseModelMeta } from "@/shared/utils/modelMeta";
import type { ProviderModel } from "@/shared/store/providers";

export const MODEL_MISSING_VALUE_PREFIX = "__model_ref__:";

export type AssistantModelOption = {
  value: string;
  label: string;
  providerId: string;
  modelName: string;
  modelRef: string;
  model: ProviderModel;
  meta: ReturnType<typeof parseModelMeta>;
};

export const parseModelRef = (value?: string) => {
  const trimmed = (value ?? "").trim();
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

export const buildModelRef = (providerId: string, modelName: string) => {
  const provider = providerId.trim();
  const model = modelName.trim();
  if (!provider || !model) {
    return "";
  }
  return `${provider}/${model}`;
};

export const isEmbeddingModel = (model: ProviderModel, meta?: ReturnType<typeof parseModelMeta>) => {
  if (meta?.capabilities?.includes("image")) {
    return false;
  }
  const haystack = `${model.name} ${model.displayName} ${model.capabilitiesJson}`.toLowerCase();
  return (
    haystack.includes("embed") ||
    haystack.includes("embedding") ||
    haystack.includes("vector") ||
    haystack.includes("bge") ||
    haystack.includes("e5") ||
    haystack.includes("gte")
  );
};

export const isImageModel = (model: ProviderModel, meta?: ReturnType<typeof parseModelMeta>) => {
  if (meta?.capabilities?.includes("image")) {
    return true;
  }
  if (model.supportsVision === true) {
    return true;
  }
  const haystack = `${model.name} ${model.displayName} ${model.capabilitiesJson}`.toLowerCase();
  return (
    haystack.includes("image") ||
    haystack.includes("vision") ||
    haystack.includes("dall") ||
    haystack.includes("stable-diffusion") ||
    haystack.includes("sdxl") ||
    haystack.includes("flux") ||
    haystack.includes("midjourney")
  );
};

export const modelRefEquals = (left: string, right: string) => {
  const parsedLeft = parseModelRef(left);
  const parsedRight = parseModelRef(right);
  return (
    parsedLeft.providerId.trim().toLowerCase() === parsedRight.providerId.trim().toLowerCase() &&
    parsedLeft.modelName.trim().toLowerCase() === parsedRight.modelName.trim().toLowerCase()
  );
};

export const modelRefKey = (value: string) => {
  const parsed = parseModelRef(value);
  const provider = parsed.providerId.trim().toLowerCase();
  const model = parsed.modelName.trim().toLowerCase();
  if (!provider || !model) {
    return "";
  }
  return `${provider}::${model}`;
};

export const resolveModelSelectValue = (modelRef: string, options: AssistantModelOption[]) => {
  const trimmed = modelRef.trim();
  if (!trimmed) {
    return "";
  }
  const matched = options.find((option) => modelRefEquals(option.modelRef, trimmed));
  return matched?.value ?? `${MODEL_MISSING_VALUE_PREFIX}${trimmed}`;
};

export const resolveUsableModelRef = (modelRef: string, options: AssistantModelOption[]) => {
  const trimmed = modelRef.trim();
  if (!trimmed) {
    return options[0]?.modelRef ?? "";
  }
  const matched = options.find((option) => modelRefEquals(option.modelRef, trimmed));
  return matched?.modelRef ?? options[0]?.modelRef ?? "";
};
