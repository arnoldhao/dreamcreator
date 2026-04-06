export type HealthComponentEntity = {
  name: string;
  status: string;
  latencyMs?: number;
  detail?: string;
};

export type HealthSnapshotEntity = {
  overall: string;
  version: number;
  updatedAt: string;
  components: HealthComponentEntity[];
};

export type StatusReportEntity = {
  appVersion: string;
  uptimeSec: number;
  activeSessions: number;
  activeRuns: number;
  queueDepth: number;
  connectedNodes: number;
  channels?: any[];
};

export type LogRecordEntity = {
  ts: string;
  level: string;
  component?: string;
  message: string;
  fields?: Record<string, unknown>;
};

export type LogsTailEntity = {
  records: LogRecordEntity[];
};

const numberOrZero = (value: unknown) => (typeof value === "number" && Number.isFinite(value) ? value : 0);
const stringOrEmpty = (value: unknown) => (typeof value === "string" ? value : "");

export const normalizeHealthSnapshot = (raw: any): HealthSnapshotEntity => ({
  overall: stringOrEmpty(raw?.overall),
  version: numberOrZero(raw?.version),
  updatedAt: stringOrEmpty(raw?.updatedAt),
  components: Array.isArray(raw?.components)
    ? raw.components.map((item: any) => ({
        name: stringOrEmpty(item?.name),
        status: stringOrEmpty(item?.status),
        latencyMs: numberOrZero(item?.latencyMs) || undefined,
        detail: stringOrEmpty(item?.detail) || undefined,
      }))
    : [],
});

export const normalizeStatusReport = (raw: any): StatusReportEntity => ({
  appVersion: stringOrEmpty(raw?.appVersion),
  uptimeSec: numberOrZero(raw?.uptimeSec),
  activeSessions: numberOrZero(raw?.activeSessions),
  activeRuns: numberOrZero(raw?.activeRuns),
  queueDepth: numberOrZero(raw?.queueDepth),
  connectedNodes: numberOrZero(raw?.connectedNodes),
  channels: Array.isArray(raw?.channels) ? raw.channels : [],
});

export const normalizeLogsTail = (raw: any): LogsTailEntity => ({
  records: Array.isArray(raw?.records)
    ? raw.records.map((item: any) => ({
        ts: stringOrEmpty(item?.ts),
        level: stringOrEmpty(item?.level),
        component: stringOrEmpty(item?.component) || undefined,
        message: stringOrEmpty(item?.message),
        fields:
          item?.fields && typeof item.fields === "object" && !Array.isArray(item.fields)
            ? (item.fields as Record<string, unknown>)
            : undefined,
      }))
    : [],
});
