import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import type {
  ExternalTool,
  ExternalToolInstallState,
  ExternalToolUpdateInfo,
  GetExternalToolInstallStateRequest,
  InstallExternalToolRequest,
  OpenExternalToolDirectoryRequest,
  RemoveExternalToolRequest,
  SetExternalToolPathRequest,
  VerifyExternalToolRequest,
} from "@/shared/store/externalTools";

export const EXTERNAL_TOOLS_QUERY_KEY = ["external-tools"];

type UseExternalToolsOptions = {
  refetchInterval?: number | false;
  staleTime?: number;
};

export function useExternalTools(options?: UseExternalToolsOptions) {
  return useQuery({
    queryKey: EXTERNAL_TOOLS_QUERY_KEY,
    queryFn: async (): Promise<ExternalTool[]> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.ExternalToolsHandler.ListTools");
      return (result as ExternalTool[]) ?? [];
    },
    staleTime: options?.staleTime ?? 5_000,
    refetchInterval: options?.refetchInterval,
  });
}

export function useExternalToolUpdates() {
  return useQuery({
    queryKey: ["external-tools-updates"],
    queryFn: async (): Promise<ExternalToolUpdateInfo[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.ListToolUpdates"
      );
      return (result as ExternalToolUpdateInfo[]) ?? [];
    },
    staleTime: 60 * 60 * 1_000,
  });
}

export function useExternalToolInstallState(name?: string, enabled = false) {
  return useQuery({
    queryKey: ["external-tools-install-state", name],
    queryFn: async (): Promise<ExternalToolInstallState> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.GetToolInstallState",
        { name } as GetExternalToolInstallStateRequest
      );
      return result as ExternalToolInstallState;
    },
    enabled: Boolean(name) && enabled,
    staleTime: 0,
    refetchInterval: enabled ? 500 : false,
  });
}

export function useInstallExternalTool() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: InstallExternalToolRequest): Promise<ExternalTool> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.InstallTool",
        request
      );
      return result as ExternalTool;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: EXTERNAL_TOOLS_QUERY_KEY });
    },
  });
}

export function useVerifyExternalTool() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: VerifyExternalToolRequest): Promise<ExternalTool> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.VerifyTool",
        request
      );
      return result as ExternalTool;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: EXTERNAL_TOOLS_QUERY_KEY });
    },
  });
}

export function useSetExternalToolPath() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: SetExternalToolPathRequest): Promise<ExternalTool> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.SetToolPath",
        request
      );
      return result as ExternalTool;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: EXTERNAL_TOOLS_QUERY_KEY });
    },
  });
}

export function useRemoveExternalTool() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: RemoveExternalToolRequest): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.ExternalToolsHandler.RemoveTool", request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: EXTERNAL_TOOLS_QUERY_KEY });
    },
  });
}

export function useOpenExternalToolDirectory() {
  return useMutation({
    mutationFn: async (request: OpenExternalToolDirectoryRequest): Promise<void> => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.ExternalToolsHandler.OpenToolDirectory",
        request
      );
    },
  });
}
