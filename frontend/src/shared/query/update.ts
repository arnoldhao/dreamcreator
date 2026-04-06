import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import { normalizeUpdateInfo, type UpdateInfo } from "@/shared/store/update";

const UPDATE_QUERY_KEY = ["update-state"];

export function useUpdateState() {
  return useQuery({
    queryKey: UPDATE_QUERY_KEY,
    queryFn: async (): Promise<UpdateInfo> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.UpdateHandler.GetState");
      return normalizeUpdateInfo(result as Partial<UpdateInfo>);
    },
    staleTime: 30_000,
  });
}

export function useCheckForUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (currentVersion: string): Promise<UpdateInfo> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.UpdateHandler.CheckForUpdate",
        currentVersion
      );
      return normalizeUpdateInfo(result as Partial<UpdateInfo>);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(UPDATE_QUERY_KEY, data);
    },
  });
}

export function useDownloadUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<UpdateInfo> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.UpdateHandler.DownloadUpdate");
      return normalizeUpdateInfo(result as Partial<UpdateInfo>);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(UPDATE_QUERY_KEY, data);
    },
  });
}

export function useRestartToApply() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<UpdateInfo> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.UpdateHandler.RestartToApply");
      return normalizeUpdateInfo(result as Partial<UpdateInfo>);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(UPDATE_QUERY_KEY, data);
    },
  });
}

export { UPDATE_QUERY_KEY };
