import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import type {
  DeleteSkillRequest,
  InspectSkillRequest,
  InstallSkillRequest,
  ProviderSkillSpec,
  RemoveSkillRequest,
  ResolveSkillsRequest,
  SearchSkillsRequest,
  SkillDetail,
  SkillSearchResult,
  SkillsStatus,
  SkillsStatusRequest,
  SyncSkillsRequest,
  UpdateSkillRequest,
} from "@/shared/contracts/skills";

export const SKILLS_CATALOG_QUERY_KEY = ["skills", "catalog"];
export const SKILLS_SEARCH_QUERY_KEY = ["skills", "search"];
export const SKILLS_STATUS_QUERY_KEY = ["skills", "status"];
export const SKILLS_DETAIL_QUERY_KEY = ["skills", "detail"];

function invalidateSkillsQueries(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: SKILLS_CATALOG_QUERY_KEY });
  queryClient.invalidateQueries({ queryKey: SKILLS_SEARCH_QUERY_KEY });
  queryClient.invalidateQueries({ queryKey: SKILLS_STATUS_QUERY_KEY });
  queryClient.invalidateQueries({ queryKey: SKILLS_DETAIL_QUERY_KEY });
}

export function useSkillsCatalog(request?: ResolveSkillsRequest) {
  const providerId = request?.providerId?.trim() ?? "";
  return useQuery({
    queryKey: [...SKILLS_CATALOG_QUERY_KEY, providerId],
    queryFn: async (): Promise<ProviderSkillSpec[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.SkillsHandler.ResolveSkillsForProvider",
        {
          providerId,
        }
      );
      return (result as ProviderSkillSpec[]) ?? [];
    },
    staleTime: 5_000,
  });
}

export function useSkillsStatus(request?: SkillsStatusRequest) {
  const providerId = request?.providerId?.trim() ?? "";
  const assistantId = request?.assistantId?.trim() ?? "";
  const workspaceRoot = request?.workspaceRoot?.trim() ?? "";
  return useQuery({
    queryKey: [...SKILLS_STATUS_QUERY_KEY, providerId, assistantId, workspaceRoot],
    queryFn: async (): Promise<SkillsStatus> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.SkillsHandler.GetSkillsStatus",
        {
          providerId,
          assistantId,
          workspaceRoot,
        }
      );
      return (result as SkillsStatus) ?? { clawhubReady: false, reason: "clawhub_unavailable", catalogCount: 0 };
    },
    staleTime: 5_000,
  });
}

export function useSearchSkills(request: SearchSkillsRequest) {
  const query = request.query.trim();
  const limit = request.limit ?? 20;
  const assistantId = request.assistantId?.trim() ?? "";
  const workspaceRoot = request.workspaceRoot?.trim() ?? "";
  return useQuery({
    queryKey: [...SKILLS_SEARCH_QUERY_KEY, query, limit, assistantId, workspaceRoot],
    enabled: Boolean(query),
    queryFn: async (): Promise<SkillSearchResult[]> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.SearchSkills", {
        query,
        limit,
        assistantId,
        workspaceRoot,
      });
      return (result as SkillSearchResult[]) ?? [];
    },
    retry: false,
    staleTime: 5_000,
  });
}

export function useInspectSkill(request?: InspectSkillRequest) {
  const skill = request?.skill?.trim() ?? "";
  const assistantId = request?.assistantId?.trim() ?? "";
  const workspaceRoot = request?.workspaceRoot?.trim() ?? "";
  return useQuery({
    queryKey: [...SKILLS_DETAIL_QUERY_KEY, skill, assistantId, workspaceRoot],
    enabled: Boolean(skill),
    queryFn: async (): Promise<SkillDetail> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.InspectSkill", {
        skill,
        assistantId,
        workspaceRoot,
      });
      return (result as SkillDetail) ?? { id: skill, name: skill };
    },
    retry: false,
    staleTime: 10_000,
  });
}

export function useInstallSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: InstallSkillRequest): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.InstallSkill", request);
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}

export function useUpdateSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: UpdateSkillRequest): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.UpdateSkill", request);
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}

export function useSyncSkills() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: SyncSkillsRequest): Promise<ProviderSkillSpec[]> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.SyncSkills", request);
      return (result as ProviderSkillSpec[]) ?? [];
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}

export function useRemoveInstalledSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: RemoveSkillRequest): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.RemoveSkill", request);
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}

export function useToggleSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: { id: string; enabled: boolean }): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.EnableSkill", request);
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}

export function useDeleteSkill() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: DeleteSkillRequest): Promise<void> => {
      await Call.ByName("dreamcreator/internal/presentation/wails.SkillsHandler.DeleteSkill", request);
    },
    onSuccess: () => {
      invalidateSkillsQueries(queryClient);
    },
  });
}
