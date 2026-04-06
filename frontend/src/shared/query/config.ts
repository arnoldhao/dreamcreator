import { useMutation, useQuery } from "@tanstack/react-query";

import { gatewayRequest } from "@/shared/api/gateway";

export type ConfigGetResponse = {
  config: unknown;
  version: number;
};

export type ConfigPatchOp = {
  op: "add" | "replace" | "remove" | string;
  path: string;
  value?: unknown;
};

export type ConfigPatchRequest = {
  ops: ConfigPatchOp[];
  dryRun?: boolean;
  expectedVersion?: number;
};

export type ReloadPlanStep = {
  component: string;
  action: string;
  reason?: string;
};

export type ReloadPlan = {
  mode: string;
  steps: ReloadPlanStep[];
};

export type ConfigPatchResponse = {
  preview: unknown;
  version: number;
  reloadPlan: ReloadPlan;
};

export function useConfigGet(path: string, enabled = true) {
  return useQuery({
    queryKey: ["config", path],
    queryFn: async (): Promise<ConfigGetResponse> => {
      const result = await gatewayRequest("config.get", { path });
      return result as ConfigGetResponse;
    },
    enabled,
    staleTime: 30_000,
  });
}

export function useConfigPatch() {
  return useMutation({
    mutationFn: async (request: ConfigPatchRequest): Promise<ConfigPatchResponse> => {
      const result = await gatewayRequest("config.patch", {
        dryRun: false,
        ...request,
      });
      return result as ConfigPatchResponse;
    },
  });
}
