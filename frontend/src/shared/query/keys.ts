const serialize = (value: unknown) => {
  try {
    return JSON.stringify(value ?? {});
  } catch {
    return String(value ?? "");
  }
};

export const queryKeys = {
  settings: () => ["settings"],
  cronStatus: () => ["cron", "status"],
  cronJobs: () => ["cron", "jobs"],
  cronRuns: (params?: unknown) => ["cron", "runs", serialize(params)],
  cronRunDetail: (runId?: string) => ["cron", "runDetail", runId ?? ""],
  cronRunEvents: (params?: unknown) => ["cron", "runEvents", serialize(params)],
  usageStatus: (params?: unknown) => ["usage", "status", serialize(params)],
  usageCost: (params?: unknown) => ["usage", "cost", serialize(params)],
  voiceStatus: () => ["voice", "status"],
  talkConfig: () => ["voice", "talk", "config"],
  voiceWake: () => ["voice", "wake"],
  diagnosticsHealth: () => ["diagnostics", "health"],
  diagnosticsStatus: () => ["diagnostics", "status"],
  diagnosticsLogs: (params?: unknown) => ["diagnostics", "logs", serialize(params)],
};
