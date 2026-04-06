export type UsageTotalsEntity = {
  requests: number;
  units: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
  costMicros: number;
};

export type UsageBucketEntity = {
  key: string;
  providerId?: string;
  modelName?: string;
  channel?: string;
  category?: string;
  requestSource?: string;
  costBasis?: string;
  requests: number;
  units: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
  reasoningTokens: number;
  costMicros: number;
};

export type UsageStatusEntity = {
  window?: string;
  totals: UsageTotalsEntity;
  buckets: UsageBucketEntity[];
};

export type UsageCostLineEntity = {
  providerId?: string;
  modelName?: string;
  channel?: string;
  category?: string;
  requestSource?: string;
  costBasis?: string;
  requests: number;
  costMicros: number;
};

export type UsageCostEntity = {
  window?: string;
  totalCostMicros: number;
  lines: UsageCostLineEntity[];
};

const numberOrZero = (value: unknown) => (typeof value === "number" && Number.isFinite(value) ? value : 0);
const stringOrEmpty = (value: unknown) => (typeof value === "string" ? value : "");

export const normalizeUsageStatus = (raw: any): UsageStatusEntity => ({
  window: stringOrEmpty(raw?.window) || undefined,
  totals: {
    requests: numberOrZero(raw?.totals?.requests),
    units: numberOrZero(raw?.totals?.units),
    inputTokens: numberOrZero(raw?.totals?.inputTokens),
    outputTokens: numberOrZero(raw?.totals?.outputTokens),
    cachedInputTokens: numberOrZero(raw?.totals?.cachedInputTokens),
    reasoningTokens: numberOrZero(raw?.totals?.reasoningTokens),
    costMicros: numberOrZero(raw?.totals?.costMicros),
  },
  buckets: Array.isArray(raw?.buckets)
    ? raw.buckets.map((item: any) => ({
        key: stringOrEmpty(item?.key),
        providerId: stringOrEmpty(item?.providerId) || undefined,
        modelName: stringOrEmpty(item?.modelName) || undefined,
        channel: stringOrEmpty(item?.channel) || undefined,
        category: stringOrEmpty(item?.category) || undefined,
        requestSource: stringOrEmpty(item?.requestSource) || undefined,
        costBasis: stringOrEmpty(item?.costBasis) || undefined,
        requests: numberOrZero(item?.requests),
        units: numberOrZero(item?.units),
        inputTokens: numberOrZero(item?.inputTokens),
        outputTokens: numberOrZero(item?.outputTokens),
        cachedInputTokens: numberOrZero(item?.cachedInputTokens),
        reasoningTokens: numberOrZero(item?.reasoningTokens),
        costMicros: numberOrZero(item?.costMicros),
      }))
    : [],
});

export const normalizeUsageCost = (raw: any): UsageCostEntity => ({
  window: stringOrEmpty(raw?.window) || undefined,
  totalCostMicros: numberOrZero(raw?.totalCostMicros),
  lines: Array.isArray(raw?.lines)
    ? raw.lines.map((item: any) => ({
        providerId: stringOrEmpty(item?.providerId) || undefined,
        modelName: stringOrEmpty(item?.modelName) || undefined,
        channel: stringOrEmpty(item?.channel) || undefined,
        category: stringOrEmpty(item?.category) || undefined,
        requestSource: stringOrEmpty(item?.requestSource) || undefined,
        costBasis: stringOrEmpty(item?.costBasis) || undefined,
        requests: numberOrZero(item?.requests),
        costMicros: numberOrZero(item?.costMicros),
      }))
    : [],
});
