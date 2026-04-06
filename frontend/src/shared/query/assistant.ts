import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call, Events } from "@wailsio/runtime";

import type {
  Assistant,
  CreateAssistantRequest,
  UpdateAssistantRequest,
  DeleteAssistantRequest,
  SetDefaultAssistantRequest,
  Assistant3DAvatarAsset,
  ImportAssistant3DAvatarFromPathRequest,
  ReadAssistant3DAvatarSourceRequest,
  ReadAssistant3DAvatarSourceResponse,
  DeleteAssistantAvatarAssetRequest,
  UpdateAssistantAvatarAssetRequest,
  AssistantMemorySummary,
  AssistantProfileOptions,
} from "@/shared/store/assistant";

export const assistantsKey = (includeDisabled: boolean) => ["assistants", { includeDisabled }];
export const assistantKey = (id: string) => ["assistant", id];
export const assistant3dAvatarAssetsKey = (kind: string) => ["assistant-3davatar-assets", { kind }];
export const assistant3dMotionAssetsKey = (kind: string) => ["assistant-3dmotion-assets", { kind }];
export const assistantMemorySummaryKey = (id: string) => ["assistant-memory-summary", id];
export const assistantProfileOptionsKey = ["assistant-profile-options"];

const emitAssistantsUpdated = () => {
  try {
    Events.Emit("assistants:updated");
  } catch (error) {
    console.warn("[assistants] failed to emit update event", error);
  }
};

export function useAssistants(includeDisabled = false) {
  return useQuery({
    queryKey: assistantsKey(includeDisabled),
    queryFn: async (): Promise<Assistant[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ListAssistants",
        includeDisabled
      );
      return (result as Assistant[]) ?? [];
    },
    staleTime: 0,
    refetchOnWindowFocus: true,
  });
}

export function useAssistant(id: string | null) {
  return useQuery({
    queryKey: id ? assistantKey(id) : ["assistant", "empty"],
    queryFn: async (): Promise<Assistant> => {
      if (!id) {
        throw new Error("assistant id is required");
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.GetAssistant",
        id
      );
      return result as Assistant;
    },
    enabled: Boolean(id),
  });
}

export function useCreateAssistant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateAssistantRequest): Promise<Assistant> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.CreateAssistant",
        request
      );
      return result as Assistant;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(assistantKey(data.id), data);
      queryClient.invalidateQueries({ queryKey: assistantsKey(false) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true) });
      emitAssistantsUpdated();
    },
  });
}

export function useUpdateAssistant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpdateAssistantRequest): Promise<Assistant> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.UpdateAssistant",
        request
      );
      return result as Assistant;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(assistantKey(data.id), data);
      queryClient.invalidateQueries({ queryKey: assistantsKey(false) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true) });
      emitAssistantsUpdated();
    },
  });
}

export function useRefreshAssistantUserLocale() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (assistantID: string): Promise<Assistant> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.RefreshAssistantUserLocale",
        assistantID
      );
      return result as Assistant;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: assistantsKey(false) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true) });
      emitAssistantsUpdated();
    },
  });
}

export function useDeleteAssistant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: DeleteAssistantRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.DeleteAssistant",
        request
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.removeQueries({ queryKey: assistantKey(variables.id) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(false) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true) });
      emitAssistantsUpdated();
    },
  });
}

export function useSetDefaultAssistant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: SetDefaultAssistantRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.SetDefaultAssistant",
        request
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: assistantsKey(false) });
      queryClient.invalidateQueries({ queryKey: assistantsKey(true) });
      emitAssistantsUpdated();
    },
  });
}

export function useAssistant3DAvatarAssets(kind: string, enabled = true) {
  return useQuery({
    queryKey: assistant3dAvatarAssetsKey(kind),
    queryFn: async (): Promise<Assistant3DAvatarAsset[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ListAvatarAssets",
        kind
      );
      return (result as Assistant3DAvatarAsset[]) ?? [];
    },
    enabled: Boolean(kind) && enabled,
    staleTime: 5_000,
  });
}

export function useAssistant3DMotionAssets(kind: string, enabled = true) {
  return useQuery({
    queryKey: assistant3dMotionAssetsKey(kind),
    queryFn: async (): Promise<Assistant3DAvatarAsset[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ListAvatarAssets",
        kind
      );
      return (result as Assistant3DAvatarAsset[]) ?? [];
    },
    enabled: Boolean(kind) && enabled,
    staleTime: 5_000,
  });
}

export function useImportAssistant3DMotionFromPath() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (
      request: ImportAssistant3DAvatarFromPathRequest
    ): Promise<Assistant3DAvatarAsset> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ImportAvatarAssetFromPath",
        request
      );
      return result as Assistant3DAvatarAsset;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: assistant3dMotionAssetsKey(data.kind) });
    },
  });
}

export function useImportAssistant3DAvatarFromPath() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (
      request: ImportAssistant3DAvatarFromPathRequest
    ): Promise<Assistant3DAvatarAsset> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ImportAvatarAssetFromPath",
        request
      );
      return result as Assistant3DAvatarAsset;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: assistant3dAvatarAssetsKey(data.kind) });
    },
  });
}

export function useDeleteAssistant3DAvatarAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: DeleteAssistantAvatarAssetRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.DeleteAvatarAsset",
        request
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: assistant3dAvatarAssetsKey(variables.kind) });
    },
  });
}

export function useDeleteAssistant3DMotionAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: DeleteAssistantAvatarAssetRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.DeleteAvatarAsset",
        request
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: assistant3dMotionAssetsKey(variables.kind) });
    },
  });
}

export function useUpdateAssistantAvatarAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (
      request: UpdateAssistantAvatarAssetRequest
    ): Promise<Assistant3DAvatarAsset> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.UpdateAvatarAsset",
        request
      );
      return result as Assistant3DAvatarAsset;
    },
    onSuccess: (data) => {
      if (data.kind === "vrma") {
        queryClient.invalidateQueries({ queryKey: assistant3dMotionAssetsKey(data.kind) });
      } else {
        queryClient.invalidateQueries({ queryKey: assistant3dAvatarAssetsKey(data.kind) });
      }
    },
  });
}

export function useReadAssistant3DAvatarSource() {
  return useMutation({
    mutationFn: async (
      request: ReadAssistant3DAvatarSourceRequest
    ): Promise<ReadAssistant3DAvatarSourceResponse> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.ReadAvatarSource",
        request
      );
      return result as ReadAssistant3DAvatarSourceResponse;
    },
  });
}

export function useAssistantMemorySummary(id: string | null) {
  return useQuery({
    queryKey: id ? assistantMemorySummaryKey(id) : ["assistant-memory-summary", "empty"],
    queryFn: async (): Promise<AssistantMemorySummary> => {
      if (!id) {
        throw new Error("assistant id is required");
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.GetAssistantMemorySummary",
        id
      );
      return result as AssistantMemorySummary;
    },
    enabled: Boolean(id),
    staleTime: 5_000,
  });
}

export function useAssistantProfileOptions() {
  return useQuery({
    queryKey: assistantProfileOptionsKey,
    queryFn: async (): Promise<AssistantProfileOptions> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AssistantHandler.GetAssistantProfileOptions"
      );
      return (
        (result as AssistantProfileOptions) ?? {
          roles: [],
          defaultRole: "",
          vibes: [],
          defaultVibe: "",
        }
      );
    },
    staleTime: 300_000,
  });
}
