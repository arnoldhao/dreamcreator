import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { gatewayRequest } from "@/shared/api/gateway";
import { queryKeys } from "@/shared/query/keys";
import {
  normalizeTalkConfig,
  normalizeTalkState,
  normalizeTTSStatus,
  normalizeVoiceWake,
  type TalkConfigEntity,
  type TalkStateEntity,
  type TTSConfigEntity,
  type TTSStatusEntity,
  type VoiceWakeEntity,
} from "@/entities/voice";

export function useTTSStatus() {
  return useQuery({
    queryKey: queryKeys.voiceStatus(),
    queryFn: async (): Promise<TTSStatusEntity> => {
      const result = await gatewayRequest("tts.status");
      return normalizeTTSStatus(result);
    },
  });
}

export function useSetTTSConfig() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (config: TTSConfigEntity) => {
      const result = await gatewayRequest("tts.config.set", config);
      return normalizeTTSStatus(result);
    },
    onSuccess: (data) => {
      client.setQueryData(queryKeys.voiceStatus(), data);
    },
  });
}

export function useTalkConfig() {
  return useQuery({
    queryKey: queryKeys.talkConfig(),
    queryFn: async (): Promise<TalkConfigEntity> => {
      const result = await gatewayRequest("talk.config", { includeSecrets: true });
      return normalizeTalkConfig(result);
    },
  });
}

export function useSetTalkConfig() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (config: TalkConfigEntity) => {
      const result = await gatewayRequest("talk.config.set", { config: { talk: config } });
      return normalizeTalkConfig(result);
    },
    onSuccess: (data) => {
      client.setQueryData(queryKeys.talkConfig(), data);
    },
  });
}

export function useTalkMode() {
  return useMutation({
    mutationFn: async (payload: { enabled: boolean; phase?: string }): Promise<TalkStateEntity> => {
      const result = await gatewayRequest("talk.mode", payload);
      return normalizeTalkState(result);
    },
  });
}

export function useVoiceWake() {
  return useQuery({
    queryKey: queryKeys.voiceWake(),
    queryFn: async (): Promise<VoiceWakeEntity> => {
      const result = await gatewayRequest("voicewake.get");
      return normalizeVoiceWake(result);
    },
  });
}

export function useSetVoiceWake() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { triggers: string[] }) => {
      const result = await gatewayRequest("voicewake.set", payload);
      return normalizeVoiceWake(result);
    },
    onSuccess: (data) => {
      client.setQueryData(queryKeys.voiceWake(), data);
    },
  });
}
