export interface UsageQuery {
  startAt?: string;
  endAt?: string;
  breakdownLimit?: number;
}

export interface UsageSummary {
  totalCalls: number;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  actualCalls: number;
  estimatedCalls: number;
}

export interface UsageDaily {
  day: string;
  totalCalls: number;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

export interface UsageBreakdown {
  providerId: string;
  modelName: string;
  role: string;
  totalCalls: number;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

export interface UsageReport {
  summary: UsageSummary;
  daily: UsageDaily[];
  breakdown: UsageBreakdown[];
}
