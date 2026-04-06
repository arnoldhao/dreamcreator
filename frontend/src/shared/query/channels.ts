import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { gatewayRequest } from "@/shared/api/gateway";
import type {
  ChannelLogoutRequest,
  ChannelLogoutResult,
  ChannelMenuSyncRequest,
  ChannelMenuSyncResult,
  ChannelOverview,
  ChannelPairingApproveRequest,
  ChannelPairingApproveResult,
  ChannelPairingListRequest,
  ChannelPairingListResult,
  ChannelPairingRejectRequest,
  ChannelPairingRejectResult,
  ChannelProbeRequest,
  ChannelProbeResult,
  ChannelDebugSnapshot,
} from "@/shared/store/channels";

export const CHANNELS_QUERY_KEY = ["gateway", "channels"];

export function useChannels() {
  return useQuery({
    queryKey: CHANNELS_QUERY_KEY,
    queryFn: async (): Promise<ChannelOverview[]> => {
      const result = await gatewayRequest("channels.list");
      return (result as ChannelOverview[]) ?? [];
    },
    staleTime: 5_000,
  });
}

export function useProbeChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ChannelProbeRequest): Promise<ChannelProbeResult> => {
      const result = await gatewayRequest("channels.probe", request);
      return result as ChannelProbeResult;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CHANNELS_QUERY_KEY });
    },
  });
}

export function useLogoutChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ChannelLogoutRequest): Promise<ChannelLogoutResult> => {
      const result = await gatewayRequest("channels.logout", request);
      return result as ChannelLogoutResult;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CHANNELS_QUERY_KEY });
    },
  });
}

export function useSyncChannelMenu() {
  return useMutation({
    mutationFn: async (request: ChannelMenuSyncRequest): Promise<ChannelMenuSyncResult> => {
      const result = await gatewayRequest("channels.menu.sync", request);
      return result as ChannelMenuSyncResult;
    },
  });
}

export function useChannelsDebug() {
  return useQuery({
    queryKey: ["gateway", "channels", "debug"],
    queryFn: async (): Promise<ChannelDebugSnapshot[]> => {
      const result = await gatewayRequest("channels.debug");
      return (result as ChannelDebugSnapshot[]) ?? [];
    },
    staleTime: 5_000,
  });
}

const pairingQueryKey = (request: ChannelPairingListRequest) => [
  "gateway",
  "channels",
  "pairing",
  request.channelId,
  request.accountId ?? "",
];

export function useChannelPairingList(request: ChannelPairingListRequest, enabled = true) {
  return useQuery({
    queryKey: pairingQueryKey(request),
    queryFn: async (): Promise<ChannelPairingListResult> => {
      const result = await gatewayRequest("channels.pairing.list", request);
      return result as ChannelPairingListResult;
    },
    enabled,
    staleTime: 5_000,
    refetchInterval: enabled ? 5_000 : false,
  });
}

export function useChannelPairingApprove() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ChannelPairingApproveRequest): Promise<ChannelPairingApproveResult> => {
      const result = await gatewayRequest("channels.pairing.approve", request);
      return result as ChannelPairingApproveResult;
    },
    onSuccess: (result, request) => {
      if (result.approved) {
        queryClient.setQueryData(pairingQueryKey(request), (current?: ChannelPairingListResult) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            requests: current.requests.filter((entry) => entry.code !== request.code),
          };
        });
      }
      queryClient.invalidateQueries({ queryKey: pairingQueryKey(request) });
      queryClient.invalidateQueries({ queryKey: ["config", "/channels"] });
    },
  });
}

export function useChannelPairingReject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ChannelPairingRejectRequest): Promise<ChannelPairingRejectResult> => {
      const result = await gatewayRequest("channels.pairing.reject", request);
      return result as ChannelPairingRejectResult;
    },
    onSuccess: (result, request) => {
      if (result.rejected) {
        queryClient.setQueryData(pairingQueryKey(request), (current?: ChannelPairingListResult) => {
          if (!current) {
            return current;
          }
          return {
            ...current,
            requests: current.requests.filter((entry) => entry.code !== request.code),
          };
        });
      }
      queryClient.invalidateQueries({ queryKey: pairingQueryKey(request) });
    },
  });
}
