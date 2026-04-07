import { useQuery } from "@tanstack/react-query";

import { gatewayRequest } from "@/shared/api/gateway";
import { queryKeys } from "@/shared/query/keys";
import type { UsageCostEntity, UsageStatusEntity } from "@/entities/usage";
import { normalizeUsageCost, normalizeUsageStatus } from "@/entities/usage";

export type UsageStatusQuery = {
  window?: string;
  startAt?: string;
  endAt?: string;
  groupBy?: string[];
  providerId?: string;
  modelName?: string;
  channel?: string;
  category?: string;
  requestSource?: string;
  costBasis?: string;
  timezoneOffsetMinutes?: number;
};

export type UsageCostQuery = UsageStatusQuery;

export function useUsageStatus(query: UsageStatusQuery) {
  return useQuery({
    queryKey: queryKeys.usageStatus(query),
    queryFn: async (): Promise<UsageStatusEntity> => {
      const result = await gatewayRequest("usage.status", query);
      return normalizeUsageStatus(result);
    },
  });
}

export function useUsageCost(query: UsageCostQuery) {
  return useQuery({
    queryKey: queryKeys.usageCost(query),
    queryFn: async (): Promise<UsageCostEntity> => {
      const result = await gatewayRequest("usage.cost", query);
      return normalizeUsageCost(result);
    },
  });
}
