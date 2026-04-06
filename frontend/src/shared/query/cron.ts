import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { gatewayRequest } from "@/shared/api/gateway";
import { queryKeys } from "@/shared/query/keys";
import type {
  CronCreateRequest,
  CronJob,
  CronJobsResponse,
  CronListQuery,
  CronPatchRequest,
  CronRemoveRequest,
  CronRunDetail,
  CronRunDetailRequest,
  CronRunEventsRequest,
  CronRunEventsResponse,
  CronRunRecord,
  CronRunRequest,
  CronRunsQuery,
  CronRunsResponse,
  CronStatus,
  CronUpdateRequest,
} from "@/shared/store/cron";

export function useCronStatus() {
  return useQuery({
    queryKey: queryKeys.cronStatus(),
    queryFn: async (): Promise<CronStatus> => {
      const result = await gatewayRequest("cron.status");
      return (result as CronStatus) ?? { enabled: false, jobs: 0 };
    },
    staleTime: 5_000,
  });
}

export function useCronJobs(query: CronListQuery = { includeDisabled: true }) {
  return useQuery({
    queryKey: [...queryKeys.cronJobs(), query],
    queryFn: async (): Promise<CronJob[]> => {
      const result = await gatewayRequest("cron.list", query);
      const response = result as CronJobsResponse | undefined;
      return response?.items ?? [];
    },
    staleTime: 5_000,
  });
}

export function useCronRuns(query: CronRunsQuery) {
  return useQuery({
    queryKey: queryKeys.cronRuns(query),
    queryFn: async (): Promise<CronRunsResponse> => {
      const result = await gatewayRequest("cron.runs", query);
      const response = result as CronRunsResponse | undefined;
      return {
        items: response?.items ?? [],
        total: response?.total ?? 0,
        offset: response?.offset ?? 0,
        limit: response?.limit ?? 0,
        hasMore: response?.hasMore ?? false,
        nextOffset: response?.nextOffset ?? 0,
      };
    },
    staleTime: 5_000,
  });
}

export function useCronRunDetail(request: CronRunDetailRequest, options?: { enabled?: boolean }) {
  const runID = request.runId?.trim() ?? "";
  return useQuery({
    queryKey: queryKeys.cronRunDetail(runID),
    enabled: (options?.enabled ?? true) && runID.length > 0,
    queryFn: async (): Promise<CronRunDetail> => {
      const result = await gatewayRequest("cron.runDetail", request);
      return (
        (result as CronRunDetail | undefined) ?? {
          run: {
            runId: runID,
            jobId: "",
            status: "unknown",
            startedAt: "",
          },
          events: [],
          eventsTotal: 0,
        }
      );
    },
    staleTime: 2_000,
  });
}

export function useCronRunEvents(query: CronRunEventsRequest, options?: { enabled?: boolean }) {
  const runID = query.runId?.trim() ?? "";
  return useQuery({
    queryKey: queryKeys.cronRunEvents(query),
    enabled: (options?.enabled ?? true) && runID.length > 0,
    queryFn: async (): Promise<CronRunEventsResponse> => {
      const result = await gatewayRequest("cron.runEvents", query);
      const response = result as CronRunEventsResponse | undefined;
      return {
        items: response?.items ?? [],
        total: response?.total ?? 0,
        offset: response?.offset ?? 0,
        limit: response?.limit ?? 0,
        hasMore: response?.hasMore ?? false,
        nextOffset: response?.nextOffset ?? 0,
      };
    },
    staleTime: 2_000,
  });
}

const invalidateCron = (queryClient: ReturnType<typeof useQueryClient>) => {
  queryClient.invalidateQueries({ queryKey: queryKeys.cronStatus() });
  queryClient.invalidateQueries({ queryKey: queryKeys.cronJobs() });
  queryClient.invalidateQueries({ queryKey: ["cron", "runs"] });
  queryClient.invalidateQueries({ queryKey: ["cron", "runDetail"] });
  queryClient.invalidateQueries({ queryKey: ["cron", "runEvents"] });
};

export function useAddCronJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CronCreateRequest): Promise<CronJob> => {
      const result = await gatewayRequest("cron.add", request);
      return ((result as { job?: CronJob } | undefined)?.job ?? {
        id: request.id ?? "",
        assistantId: request.assistantId,
        name: request.name,
        description: request.description,
        enabled: request.enabled,
        deleteAfterRun: Boolean(request.deleteAfterRun),
        schedule: request.schedule,
        sessionTarget: request.sessionTarget,
        wakeMode: request.wakeMode,
        payload: request.payload,
        delivery: request.delivery,
      }) as CronJob;
    },
    onSuccess: () => invalidateCron(queryClient),
  });
}

export function useUpdateCronJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CronUpdateRequest): Promise<CronJob> => {
      const result = await gatewayRequest("cron.update", request);
      return ((result as { job?: CronJob } | undefined)?.job ?? {
        id: request.id,
        assistantId: (request.patch as CronPatchRequest).assistantId ?? "",
        name: (request.patch as CronPatchRequest).name ?? request.id,
        enabled: (request.patch as CronPatchRequest).enabled ?? true,
        deleteAfterRun: Boolean((request.patch as CronPatchRequest).deleteAfterRun),
        schedule: (request.patch as CronPatchRequest).schedule ?? { kind: "every", everyMs: 60_000 },
        sessionTarget: (request.patch as CronPatchRequest).sessionTarget ?? "isolated",
        wakeMode: (request.patch as CronPatchRequest).wakeMode ?? "next-heartbeat",
        payload: (request.patch as CronPatchRequest).payload ?? { kind: "agentTurn", message: "" },
        delivery: (request.patch as CronPatchRequest).delivery,
      }) as CronJob;
    },
    onSuccess: () => invalidateCron(queryClient),
  });
}

export function useRemoveCronJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CronRemoveRequest): Promise<boolean> => {
      const result = await gatewayRequest("cron.remove", request);
      return Boolean((result as { ok?: boolean } | undefined)?.ok);
    },
    onSuccess: () => invalidateCron(queryClient),
  });
}

export function useRunCronJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CronRunRequest): Promise<CronRunRecord> => {
      const result = await gatewayRequest("cron.run", request);
      return ((result as { run?: CronRunRecord } | undefined)?.run ?? {
        runId: "",
        jobId: request.id,
        status: "failed",
        startedAt: "",
      }) as CronRunRecord;
    },
    onSuccess: () => invalidateCron(queryClient),
  });
}
