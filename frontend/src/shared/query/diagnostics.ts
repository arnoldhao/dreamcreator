import { useQuery } from "@tanstack/react-query";

import { queryKeys } from "@/shared/query/keys";
import { useHttpBaseUrl } from "@/features/settings/gateway/ui/useHttpBaseUrl";
import {
  normalizeHealthSnapshot,
  normalizeLogsTail,
  normalizeStatusReport,
  type HealthSnapshotEntity,
  type LogsTailEntity,
  type StatusReportEntity,
} from "@/entities/observability";

const resolveApi = (base: string, path: string) => {
  const trimmed = base.replace(/\/+$/, "");
  return trimmed ? `${trimmed}${path}` : path;
};

export function useGatewayHealth() {
  const baseUrl = useHttpBaseUrl();
  return useQuery({
    queryKey: queryKeys.diagnosticsHealth(),
    enabled: Boolean(baseUrl),
    queryFn: async (): Promise<HealthSnapshotEntity> => {
      const res = await fetch(resolveApi(baseUrl, "/api/health"));
      const json = await res.json();
      return normalizeHealthSnapshot(json?.snapshot ?? json);
    },
  });
}

export function useGatewayStatus() {
  const baseUrl = useHttpBaseUrl();
  return useQuery({
    queryKey: queryKeys.diagnosticsStatus(),
    enabled: Boolean(baseUrl),
    queryFn: async (): Promise<StatusReportEntity> => {
      const res = await fetch(resolveApi(baseUrl, "/api/status"));
      const json = await res.json();
      return normalizeStatusReport(json?.report ?? json);
    },
  });
}

export function useGatewayLogsTail(
  params: { level?: string; component?: string; limit?: number },
  options?: { realtime?: boolean; intervalMs?: number }
) {
  const baseUrl = useHttpBaseUrl();
  return useQuery({
    queryKey: queryKeys.diagnosticsLogs(params),
    enabled: Boolean(baseUrl),
    refetchInterval: options?.realtime ? options?.intervalMs ?? 2000 : false,
    refetchOnWindowFocus: false,
    queryFn: async (): Promise<LogsTailEntity> => {
      const query = new URLSearchParams();
      if (params.level) query.set("level", params.level);
      if (params.component) query.set("component", params.component);
      if (params.limit) query.set("limit", String(params.limit));
      const res = await fetch(resolveApi(baseUrl, `/api/logs/tail?${query.toString()}`));
      const json = await res.json();
      return normalizeLogsTail(json?.logs ?? json);
    },
  });
}
