export type AppearanceMode = "light" | "dark" | "auto";
export type ThemeColor = string;
export type ColorScheme = "default" | "contrast" | "slate" | "warm";
export type ProxyMode = "none" | "system" | "manual";
export type ProxyScheme = "http" | "https" | "socks5";
export type SystemProxySource = "system" | "vpn";
export type MenuBarVisibility = "always" | "whenRunning" | "never";
export type GatewayDebugMode = "off" | "basic" | "full";

export interface WindowBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ProxySettings {
  mode: ProxyMode;
  scheme: ProxyScheme;
  host: string;
  port: number;
  username: string;
  password: string;
  noProxy: string[];
  timeoutSeconds: number;
  testedAt: string;
  testSuccess: boolean;
  testMessage: string;
}

export interface SystemProxyInfo {
  address: string;
  source?: SystemProxySource;
  name?: string;
}

export interface Settings {
  appearance: AppearanceMode;
  effectiveAppearance: string;
  fontFamily: string;
  fontSize: number;
  language: string;
  themeColor: ThemeColor;
  colorScheme: ColorScheme;
  systemThemeColor?: string;
  logLevel: string;
  logMaxSizeMB: number;
  logMaxBackups: number;
  logMaxAgeDays: number;
  logCompress: boolean;
  downloadDirectory: string;
  menuBarVisibility: MenuBarVisibility;
  autoStart: boolean;
  minimizeToTrayOnStart: boolean;
  agentModelProviderId: string;
  agentModelName: string;
  agentStreamEnabled: boolean;
  chatTemperature: number;
  chatMaxTokens: number;
  mainBounds: WindowBounds;
  settingsBounds: WindowBounds;
  proxy: ProxySettings;
  version: number;
  gateway: GatewaySettings;
  memory: MemorySettings;
  tools?: Record<string, unknown>;
  skills?: Record<string, unknown>;
  channels?: Record<string, unknown>;
}

export interface UpdateSettingsRequest {
  appearance?: AppearanceMode;
  fontFamily?: string;
  fontSize?: number;
  language?: string;
  themeColor?: ThemeColor;
  colorScheme?: ColorScheme;
  logLevel?: string;
  logMaxSizeMB?: number;
  logMaxBackups?: number;
  logMaxAgeDays?: number;
  logCompress?: boolean;
  downloadDirectory?: string;
  menuBarVisibility?: MenuBarVisibility;
  autoStart?: boolean;
  minimizeToTrayOnStart?: boolean;
  agentModelProviderId?: string;
  agentModelName?: string;
  agentStreamEnabled?: boolean;
  chatTemperature?: number;
  chatMaxTokens?: number;
  mainBounds?: WindowBounds;
  settingsBounds?: WindowBounds;
  proxy?: ProxySettings;
  gateway?: UpdateGatewaySettingsRequest;
  memory?: UpdateMemorySettingsRequest;
  tools?: Record<string, unknown>;
  skills?: Record<string, unknown>;
  channels?: Record<string, unknown>;
}

export interface MemorySettings {
  enabled: boolean;
  embeddingProviderId: string;
  embeddingModel: string;
  llmProviderId: string;
  llmModel: string;
  recallTopK: number;
  vectorWeight: number;
  textWeight: number;
  recencyWeight: number;
  recencyHalfLifeDays: number;
  minScore: number;
  autoRecall: boolean;
  autoCapture: boolean;
  sessionLifecycle: boolean;
  captureMaxEntries: number;
}

export interface UpdateMemorySettingsRequest {
  enabled?: boolean;
  embeddingProviderId?: string;
  embeddingModel?: string;
  llmProviderId?: string;
  llmModel?: string;
  recallTopK?: number;
  vectorWeight?: number;
  textWeight?: number;
  recencyWeight?: number;
  recencyHalfLifeDays?: number;
  minScore?: number;
  autoRecall?: boolean;
  autoCapture?: boolean;
  sessionLifecycle?: boolean;
  captureMaxEntries?: number;
}

export interface GatewayHTTPChatCompletionsSettings {
  enabled: boolean;
}

export interface GatewayHTTPResponsesPDFSettings {
  maxPages: number;
  maxPixels: number;
  minTextChars: number;
}

export interface GatewayHTTPResponsesFilesSettings {
  allowUrl: boolean;
  urlAllowlist: string[];
  allowedMimes: string[];
  maxBytes: number;
  maxChars: number;
  maxRedirects: number;
  timeoutMs: number;
  pdf: GatewayHTTPResponsesPDFSettings;
}

export interface GatewayHTTPResponsesImagesSettings {
  allowUrl: boolean;
  urlAllowlist: string[];
  allowedMimes: string[];
  maxBytes: number;
  maxRedirects: number;
  timeoutMs: number;
}

export interface GatewayHTTPResponsesSettings {
  enabled: boolean;
  maxBodyBytes: number;
  maxUrlParts: number;
  files: GatewayHTTPResponsesFilesSettings;
  images: GatewayHTTPResponsesImagesSettings;
}

export interface GatewayHTTPEndpointsSettings {
  chatCompletions: GatewayHTTPChatCompletionsSettings;
  responses: GatewayHTTPResponsesSettings;
}

export interface GatewayHTTPSettings {
  endpoints: GatewayHTTPEndpointsSettings;
}

export interface GatewaySettings {
  controlPlaneEnabled: boolean;
  voiceEnabled: boolean;
  sandboxEnabled: boolean;
  voiceWakeEnabled: boolean;
  http: GatewayHTTPSettings;
  channelHealthCheckMinutes: number;
  runtime: GatewayRuntimeSettings;
  queue: GatewayQueueSettings;
  heartbeat: GatewayHeartbeatSettings;
  subagents: GatewaySubagentSettings;
  cron: GatewayCronSettings;
}

export interface GatewayRuntimeSettings {
  maxSteps: number;
  debugMode: GatewayDebugMode;
  recordPrompt: boolean;
  toolLoopDetection: GatewayToolLoopSettings;
  contextWindow: GatewayContextWindowSettings;
  compaction: GatewayCompactionSettings;
}

export interface GatewayToolLoopSettings {
  enabled: boolean;
  warnThreshold: number;
  criticalThreshold: number;
  globalCircuitBreakerThreshold: number;
  historySize: number;
  detectors: GatewayToolLoopDetectors;
  abortThreshold: number;
  windowSize: number;
}

export interface GatewayToolLoopDetectors {
  genericRepeat: boolean;
  knownPollNoProgress: boolean;
  pingPong: boolean;
}

export interface GatewayContextWindowSettings {
  warnTokens: number;
  hardTokens: number;
}

export interface GatewayCompactionSettings {
  mode: string;
  reserveTokens: number;
  keepRecentTokens: number;
  reserveTokensFloor: number;
  maxHistoryShare: number;
  memoryFlush: GatewayCompactionMemoryFlushSettings;
}

export interface GatewayCompactionMemoryFlushSettings {
  enabled: boolean;
  softThresholdTokens: number;
  prompt: string;
  systemPrompt: string;
}

export interface GatewayQueueSettings {
  globalConcurrency: number;
  sessionConcurrency: number;
  lanes: GatewayQueueLaneSettings;
}

export interface GatewayQueueLaneSettings {
  main: number;
  subagent: number;
  cron: number;
}

export interface GatewayHeartbeatSettings {
  enabled: boolean;
  everyMinutes: number;
  every: string;
  target: string;
  to: string;
  accountId: string;
  model: string;
  session: string;
  prompt: string;
  includeReasoning: boolean;
  suppressToolErrorWarnings: boolean;
  activeHours: GatewayHeartbeatActiveHours;
  checklist: GatewayHeartbeatChecklist;
  runSession: string;
  promptAppend: string;
  periodic: GatewayHeartbeatPeriodicSettings;
  delivery: GatewayHeartbeatDeliverySettings;
  events: GatewayHeartbeatEventSettings;
}

export interface GatewayHeartbeatActiveHours {
  start: string;
  end: string;
  timezone: string;
}

export interface GatewayHeartbeatChecklist {
  title: string;
  items: GatewayHeartbeatChecklistItem[];
  notes: string;
  version: number;
  updatedAt: string;
}

export interface GatewayHeartbeatChecklistItem {
  id: string;
  text: string;
  done: boolean;
  priority: string;
}

export interface GatewayHeartbeatPeriodicSettings {
  enabled: boolean;
  every: string;
}

export interface GatewayHeartbeatDeliverySettings {
  periodic: GatewayHeartbeatSurfacePolicy;
  eventDriven: GatewayHeartbeatSurfacePolicy;
  threadReplyMode: string;
}

export interface GatewayHeartbeatSurfacePolicy {
  center: boolean;
  popupMinSeverity: string;
  toastMinSeverity: string;
  osMinSeverity: string;
}

export interface GatewayHeartbeatEventSettings {
  cronWakeMode: string;
  execWakeMode: string;
  subagentWakeMode: string;
}

export interface GatewaySubagentSettings {
  maxDepth: number;
  maxChildren: number;
  maxConcurrent: number;
  model: string;
  thinking: string;
  tools: GatewaySubagentToolPolicy;
}

export interface GatewaySubagentToolPolicy {
  allow: string[];
  alsoAllow: string[];
  deny: string[];
}

export interface GatewayCronSettings {
  enabled: boolean;
  maxConcurrentRuns: number;
  sessionRetention: string;
  runLog: GatewayCronRunLogSetting;
}

export interface GatewayCronRunLogSetting {
  maxBytes: string;
  keepLines: number;
}

export interface UpdateGatewayHTTPChatCompletionsSettingsRequest {
  enabled?: boolean;
}

export interface UpdateGatewayHTTPResponsesPDFSettingsRequest {
  maxPages?: number;
  maxPixels?: number;
  minTextChars?: number;
}

export interface UpdateGatewayHTTPResponsesFilesSettingsRequest {
  allowUrl?: boolean;
  urlAllowlist?: string[];
  allowedMimes?: string[];
  maxBytes?: number;
  maxChars?: number;
  maxRedirects?: number;
  timeoutMs?: number;
  pdf?: UpdateGatewayHTTPResponsesPDFSettingsRequest;
}

export interface UpdateGatewayHTTPResponsesImagesSettingsRequest {
  allowUrl?: boolean;
  urlAllowlist?: string[];
  allowedMimes?: string[];
  maxBytes?: number;
  maxRedirects?: number;
  timeoutMs?: number;
}

export interface UpdateGatewayHTTPResponsesSettingsRequest {
  enabled?: boolean;
  maxBodyBytes?: number;
  maxUrlParts?: number;
  files?: UpdateGatewayHTTPResponsesFilesSettingsRequest;
  images?: UpdateGatewayHTTPResponsesImagesSettingsRequest;
}

export interface UpdateGatewayHTTPEndpointsSettingsRequest {
  chatCompletions?: UpdateGatewayHTTPChatCompletionsSettingsRequest;
  responses?: UpdateGatewayHTTPResponsesSettingsRequest;
}

export interface UpdateGatewayHTTPSettingsRequest {
  endpoints?: UpdateGatewayHTTPEndpointsSettingsRequest;
}

export interface UpdateGatewaySettingsRequest {
  controlPlaneEnabled?: boolean;
  voiceEnabled?: boolean;
  sandboxEnabled?: boolean;
  voiceWakeEnabled?: boolean;
  http?: UpdateGatewayHTTPSettingsRequest;
  channelHealthCheckMinutes?: number;
  runtime?: UpdateGatewayRuntimeSettingsRequest;
  queue?: UpdateGatewayQueueSettingsRequest;
  heartbeat?: UpdateGatewayHeartbeatSettingsRequest;
  subagents?: UpdateGatewaySubagentSettingsRequest;
  cron?: UpdateGatewayCronSettingsRequest;
}

export interface UpdateGatewayRuntimeSettingsRequest {
  maxSteps?: number;
  debugMode?: GatewayDebugMode;
  recordPrompt?: boolean;
  toolLoopDetection?: UpdateGatewayToolLoopSettingsRequest;
  contextWindow?: UpdateGatewayContextWindowSettingsRequest;
  compaction?: UpdateGatewayCompactionSettingsRequest;
}

export interface UpdateGatewayToolLoopSettingsRequest {
  enabled?: boolean;
  warnThreshold?: number;
  criticalThreshold?: number;
  globalCircuitBreakerThreshold?: number;
  historySize?: number;
  detectors?: UpdateGatewayToolLoopDetectorsRequest;
  abortThreshold?: number;
  windowSize?: number;
}

export interface UpdateGatewayToolLoopDetectorsRequest {
  genericRepeat?: boolean;
  knownPollNoProgress?: boolean;
  pingPong?: boolean;
}

export interface UpdateGatewayContextWindowSettingsRequest {
  warnTokens?: number;
  hardTokens?: number;
}

export interface UpdateGatewayCompactionSettingsRequest {
  mode?: string;
  reserveTokens?: number;
  keepRecentTokens?: number;
  reserveTokensFloor?: number;
  maxHistoryShare?: number;
  memoryFlush?: UpdateGatewayCompactionMemoryFlushSettingsRequest;
}

export interface UpdateGatewayCompactionMemoryFlushSettingsRequest {
  enabled?: boolean;
  softThresholdTokens?: number;
  prompt?: string;
  systemPrompt?: string;
}

export interface UpdateGatewayQueueSettingsRequest {
  globalConcurrency?: number;
  sessionConcurrency?: number;
  lanes?: UpdateGatewayQueueLaneSettingsRequest;
}

export interface UpdateGatewayQueueLaneSettingsRequest {
  main?: number;
  subagent?: number;
  cron?: number;
}

export interface UpdateGatewayHeartbeatSettingsRequest {
  enabled?: boolean;
  everyMinutes?: number;
  every?: string;
  target?: string;
  to?: string;
  accountId?: string;
  model?: string;
  session?: string;
  prompt?: string;
  includeReasoning?: boolean;
  suppressToolErrorWarnings?: boolean;
  activeHours?: UpdateGatewayHeartbeatActiveHoursRequest;
  checklist?: UpdateGatewayHeartbeatChecklistRequest;
  runSession?: string;
  promptAppend?: string;
  periodic?: UpdateGatewayHeartbeatPeriodicSettingsRequest;
  delivery?: UpdateGatewayHeartbeatDeliverySettingsRequest;
  events?: UpdateGatewayHeartbeatEventSettingsRequest;
}

export interface UpdateGatewayHeartbeatActiveHoursRequest {
  start?: string;
  end?: string;
  timezone?: string;
}

export interface UpdateGatewayHeartbeatChecklistRequest {
  title?: string;
  items?: UpdateGatewayHeartbeatChecklistItemRequest[];
  notes?: string;
  version?: number;
  updatedAt?: string;
}

export interface UpdateGatewayHeartbeatChecklistItemRequest {
  id?: string;
  text?: string;
  done?: boolean;
  priority?: string;
}

export interface UpdateGatewayHeartbeatPeriodicSettingsRequest {
  enabled?: boolean;
  every?: string;
}

export interface UpdateGatewayHeartbeatDeliverySettingsRequest {
  periodic?: UpdateGatewayHeartbeatSurfacePolicyRequest;
  eventDriven?: UpdateGatewayHeartbeatSurfacePolicyRequest;
  threadReplyMode?: string;
}

export interface UpdateGatewayHeartbeatSurfacePolicyRequest {
  center?: boolean;
  popupMinSeverity?: string;
  toastMinSeverity?: string;
  osMinSeverity?: string;
}

export interface UpdateGatewayHeartbeatEventSettingsRequest {
  cronWakeMode?: string;
  execWakeMode?: string;
  subagentWakeMode?: string;
}

export interface UpdateGatewaySubagentSettingsRequest {
  maxDepth?: number;
  maxChildren?: number;
  maxConcurrent?: number;
  model?: string;
  thinking?: string;
  tools?: UpdateGatewaySubagentToolPolicyRequest;
}

export interface UpdateGatewaySubagentToolPolicyRequest {
  allow?: string[];
  alsoAllow?: string[];
  deny?: string[];
}

export interface UpdateGatewayCronSettingsRequest {
  enabled?: boolean;
  maxConcurrentRuns?: number;
  sessionRetention?: string;
  runLog?: UpdateGatewayCronRunLogSettingRequest;
}

export interface UpdateGatewayCronRunLogSettingRequest {
  maxBytes?: string;
  keepLines?: number;
}
