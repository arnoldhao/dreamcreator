export type TTSProviderEntity = {
  providerId: string;
  displayName: string;
  available: boolean;
  requiresAuth?: boolean;
};

export type TTSConfigEntity = {
  providerId?: string;
  voiceId?: string;
  modelId?: string;
  format?: string;
};

export type TTSStatusEntity = {
  enabled: boolean;
  providers: TTSProviderEntity[];
  config: TTSConfigEntity;
};

export type TTSArtifactEntity = {
  artifactId: string;
  providerId: string;
  voiceId?: string;
  format: string;
  contentType: string;
  path?: string;
  sizeBytes: number;
  durationMs?: number;
};

export type TTSConvertEntity = {
  jobId: string;
  costMicros: number;
  artifact: TTSArtifactEntity;
};

export type TalkConfigEntity = {
  voiceId?: string;
  voiceAliases?: Record<string, string>;
  modelId?: string;
  outputFormat?: string;
  apiKey?: string;
  interruptOnSpeech?: boolean;
};

export type TalkStateEntity = {
  enabled: boolean;
  phase?: string;
  voiceLocked: boolean;
  lockedVoiceId?: string;
  updatedAt?: string;
};

export type VoiceWakeEntity = {
  version: number;
  triggers: string[];
};

const boolOrFalse = (value: unknown) => (typeof value === "boolean" ? value : false);
const numberOrZero = (value: unknown) => (typeof value === "number" && Number.isFinite(value) ? value : 0);
const stringOrEmpty = (value: unknown) => (typeof value === "string" ? value : "");
const recordOfStrings = (value: unknown) => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  const entries = Object.entries(value as Record<string, unknown>);
  const result: Record<string, string> = {};
  for (const [key, raw] of entries) {
    if (typeof raw !== "string") {
      continue;
    }
    const trimmedKey = key.trim();
    const trimmedValue = raw.trim();
    if (!trimmedKey || !trimmedValue) {
      continue;
    }
    result[trimmedKey] = trimmedValue;
  }
  return Object.keys(result).length > 0 ? result : undefined;
};

export const normalizeTTSStatus = (raw: any): TTSStatusEntity => ({
  enabled: boolOrFalse(raw?.enabled),
  providers: Array.isArray(raw?.providers)
    ? raw.providers.map((item: any) => ({
        providerId: stringOrEmpty(item?.providerId),
        displayName: stringOrEmpty(item?.displayName),
        available: boolOrFalse(item?.available),
        requiresAuth: typeof item?.requiresAuth === "boolean" ? item.requiresAuth : undefined,
      }))
    : [],
  config: {
    providerId: stringOrEmpty(raw?.config?.providerId) || undefined,
    voiceId: stringOrEmpty(raw?.config?.voiceId) || undefined,
    modelId: stringOrEmpty(raw?.config?.modelId) || undefined,
    format: stringOrEmpty(raw?.config?.format) || undefined,
  },
});

export const normalizeTTSConvert = (raw: any): TTSConvertEntity => ({
  jobId: stringOrEmpty(raw?.jobId),
  costMicros: numberOrZero(raw?.costMicros),
  artifact: {
    artifactId: stringOrEmpty(raw?.artifact?.artifactId),
    providerId: stringOrEmpty(raw?.artifact?.providerId),
    voiceId: stringOrEmpty(raw?.artifact?.voiceId) || undefined,
    format: stringOrEmpty(raw?.artifact?.format),
    contentType: stringOrEmpty(raw?.artifact?.contentType),
    path: stringOrEmpty(raw?.artifact?.path) || undefined,
    sizeBytes: numberOrZero(raw?.artifact?.sizeBytes),
    durationMs: numberOrZero(raw?.artifact?.durationMs) || undefined,
  },
});

export const normalizeTalkConfig = (raw: any): TalkConfigEntity => {
  const talk = raw?.config?.talk ?? raw?.talk ?? raw?.config ?? raw ?? {};
  return {
    voiceId: stringOrEmpty(talk?.voiceId) || undefined,
    voiceAliases: recordOfStrings(talk?.voiceAliases),
    modelId: stringOrEmpty(talk?.modelId) || undefined,
    outputFormat: stringOrEmpty(talk?.outputFormat) || undefined,
    apiKey: stringOrEmpty(talk?.apiKey) || undefined,
    interruptOnSpeech:
      typeof talk?.interruptOnSpeech === "boolean" ? talk.interruptOnSpeech : undefined,
  };
};

export const normalizeTalkState = (raw: any): TalkStateEntity => ({
  enabled: boolOrFalse(raw?.enabled),
  phase: stringOrEmpty(raw?.phase) || undefined,
  voiceLocked: boolOrFalse(raw?.voiceLocked),
  lockedVoiceId: stringOrEmpty(raw?.lockedVoiceId) || undefined,
  updatedAt: stringOrEmpty(raw?.updatedAt) || undefined,
});

export const normalizeVoiceWake = (raw: any): VoiceWakeEntity => ({
  version: numberOrZero(raw?.version),
  triggers: Array.isArray(raw?.triggers) ? raw.triggers.map((item: any) => stringOrEmpty(item)).filter(Boolean) : [],
});
