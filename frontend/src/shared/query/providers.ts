import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call, Events } from "@wailsio/runtime";

import type {
  Provider,
  ProviderModel,
  ReplaceProviderModelsRequest,
  ProviderSecret,
  ProviderWithModels,
  SyncProviderModelsRequest,
  UpdateProviderModelRequest,
  UpsertProviderRequest,
  UpsertProviderSecretRequest,
} from "@/shared/store/providers";

export const PROVIDERS_QUERY_KEY = ["providers"];
export const ENABLED_PROVIDERS_WITH_MODELS_KEY = ["providers", "enabled", "models"];

export const providerModelsKey = (providerId: string) => ["providers", providerId, "models"];
export const providerSecretKey = (providerId: string) => ["providers", providerId, "secret"];

const emitProvidersUpdated = async () => {
  try {
    await Events.Emit("providers:updated");
  } catch (error) {
    console.warn("[providers] failed to emit update event", error);
  }
};

export function useProviders() {
  return useQuery({
    queryKey: PROVIDERS_QUERY_KEY,
    queryFn: async (): Promise<Provider[]> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.ProviderHandler.ListProviders");
      return (result as Provider[]) ?? [];
    },
    staleTime: 10_000,
  });
}

export function useEnabledProvidersWithModels() {
  return useQuery({
    queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY,
    queryFn: async (): Promise<ProviderWithModels[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.ListEnabledProvidersWithModels"
      );
      return (result as ProviderWithModels[]) ?? [];
    },
    staleTime: 0,
    refetchOnWindowFocus: true,
  });
}

export function useUpsertProvider() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpsertProviderRequest): Promise<Provider> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.UpsertProvider",
        request
      );
      return result as Provider;
    },
    onMutate: async (request) => {
      await queryClient.cancelQueries({ queryKey: PROVIDERS_QUERY_KEY });
      await queryClient.cancelQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY });
      const previousProviders = queryClient.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY);
      const previousEnabled = queryClient.getQueryData<ProviderWithModels[]>(ENABLED_PROVIDERS_WITH_MODELS_KEY);
      if (request.id && previousProviders) {
        queryClient.setQueryData<Provider[]>(PROVIDERS_QUERY_KEY, (providers) =>
          (providers ?? []).map((provider) =>
            provider.id === request.id
              ? {
                  ...provider,
                  name: request.name,
                  type: request.type,
                  compatibility: request.compatibility ?? provider.compatibility,
                  endpoint: request.endpoint,
                  enabled: request.enabled,
                }
              : provider
          )
        );
      }
      if (request.id && previousEnabled) {
        queryClient.setQueryData<ProviderWithModels[]>(ENABLED_PROVIDERS_WITH_MODELS_KEY, (entries) =>
          (entries ?? []).map((entry) =>
            entry.provider.id === request.id
              ? {
                  ...entry,
                  provider: {
                    ...entry.provider,
                    name: request.name,
                    type: request.type,
                    compatibility: request.compatibility ?? entry.provider.compatibility,
                    endpoint: request.endpoint,
                    enabled: request.enabled,
                  },
                }
              : entry
          )
        );
      }
      return { previousProviders, previousEnabled };
    },
    onError: (_error, _variables, context) => {
      if (context?.previousProviders) {
        queryClient.setQueryData(PROVIDERS_QUERY_KEY, context.previousProviders);
      }
      if (context?.previousEnabled) {
        queryClient.setQueryData(ENABLED_PROVIDERS_WITH_MODELS_KEY, context.previousEnabled);
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" }),
      ]);
      await emitProvidersUpdated();
    },
  });
}

export function useDeleteProvider() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (providerId: string): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.DeleteProvider",
        providerId
      );
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" }),
      ]);
      await emitProvidersUpdated();
    },
  });
}

export function useProviderModels(providerId: string | null) {
  return useQuery({
    queryKey: providerId ? providerModelsKey(providerId) : ["providers", "models", "empty"],
    queryFn: async (): Promise<ProviderModel[]> => {
      if (!providerId) {
        return [];
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.ListProviderModels",
        providerId
      );
      return (result as ProviderModel[]) ?? [];
    },
    enabled: Boolean(providerId),
  });
}

export function useUpdateProviderModel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpdateProviderModelRequest): Promise<ProviderModel> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.UpdateProviderModel",
        request
      );
      return result as ProviderModel;
    },
    onSuccess: async (data) => {
      queryClient.setQueryData(providerModelsKey(data.providerId), (prev?: ProviderModel[]) => {
        if (!prev) {
          return [data];
        }
        return prev.map((model) => (model.id === data.id ? data : model));
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" }),
      ]);
      await emitProvidersUpdated();
    },
  });
}

export function useSyncProviderModels() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: SyncProviderModelsRequest): Promise<ProviderModel[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.SyncProviderModels",
        request
      );
      return (result as ProviderModel[]) ?? [];
    },
    onSuccess: async (data, variables) => {
      queryClient.setQueryData(providerModelsKey(variables.providerId), data);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" }),
      ]);
      await emitProvidersUpdated();
    },
  });
}

export function useReplaceProviderModels() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: ReplaceProviderModelsRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.ReplaceProviderModels",
        request
      );
    },
    onSuccess: async (_data, variables) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: providerModelsKey(variables.providerId), refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: PROVIDERS_QUERY_KEY, refetchType: "all" }),
        queryClient.invalidateQueries({ queryKey: ENABLED_PROVIDERS_WITH_MODELS_KEY, refetchType: "all" }),
      ]);
      await emitProvidersUpdated();
    },
  });
}

export function useProviderSecret(providerId: string | null) {
  return useQuery({
    queryKey: providerId ? providerSecretKey(providerId) : ["providers", "secret", "empty"],
    queryFn: async (): Promise<ProviderSecret> => {
      if (!providerId) {
        return { providerId: "", apiKey: "", orgRef: "" };
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.GetProviderSecret",
        providerId
      );
      return (result as ProviderSecret) ?? { providerId, apiKey: "", orgRef: "" };
    },
    enabled: Boolean(providerId),
  });
}

export function useUpsertProviderSecret() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpsertProviderSecretRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.ProviderHandler.UpsertProviderSecret",
        request
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: providerSecretKey(variables.providerId) });
    },
  });
}
