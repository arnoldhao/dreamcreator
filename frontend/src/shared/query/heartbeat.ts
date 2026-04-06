import { useQuery } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

export interface HeartbeatLastStatus {
  available: boolean;
  sessionKey: string;
  threadId: string;
  status: string;
  message: string;
  error: string;
  reason: string;
  indicator: string;
  createdAt: string;
}

export const heartbeatLastStatusKey = (sessionKey: string) => ["heartbeat", "last", sessionKey] as const;

export function useHeartbeatLastStatus(sessionKey: string, enabled = true) {
  return useQuery({
    queryKey: heartbeatLastStatusKey(sessionKey),
    queryFn: async (): Promise<HeartbeatLastStatus | null> => {
      if (!sessionKey.trim()) {
        return null;
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.HeartbeatHandler.GetLast",
        sessionKey,
      );
      const status = (result as Partial<HeartbeatLastStatus> | null | undefined) ?? null;
      if (!status?.available) {
        return null;
      }
      return {
        available: true,
        sessionKey: status.sessionKey ?? "",
        threadId: status.threadId ?? "",
        status: status.status ?? "",
        message: status.message ?? "",
        error: status.error ?? "",
        reason: status.reason ?? "",
        indicator: status.indicator ?? "",
        createdAt: status.createdAt ?? "",
      };
    },
    enabled: enabled && Boolean(sessionKey.trim()),
    staleTime: 5_000,
  });
}
