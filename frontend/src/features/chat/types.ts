import type { Provider, ProviderModel } from "@/shared/store/providers";
import type { ModelMeta } from "@/shared/utils/modelMeta";

export type ProviderGroup = {
  provider: Provider;
  models: Array<{ model: ProviderModel; meta: ModelMeta }>;
};

export type EditingTarget = { id: string; text: string };

export type ModelGroup = {
  provider: Provider;
  models: ProviderModel[];
};
