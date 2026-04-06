import { isRecord } from "./calls-utils";

export type SkillSourceTrust = "trusted" | "community" | "local";
export type LocalSkillSourceType = "workspace" | "extra";
export type RemoteSkillSourceProvider = "clawhub";

export interface LocalSkillSourceItem {
  id: string;
  name: string;
  kind: "local";
  type: LocalSkillSourceType;
  path: string;
  enabled: boolean;
  priority: number;
  trust: SkillSourceTrust;
  builtin: boolean;
  lastScanAt?: string;
  lastScanResult?: string;
}

export interface RemoteSkillSourceItem {
  id: string;
  name: string;
  kind: "remote";
  provider: RemoteSkillSourceProvider;
  enabled: boolean;
  searchEnabled: boolean;
  installEnabled: boolean;
  builtin: boolean;
  lastSyncAt?: string;
}

export interface SkillsSourcesState {
  local: LocalSkillSourceItem[];
  remote: RemoteSkillSourceItem[];
}

export type SkillsActionMode = "allow" | "ask" | "deny";

export interface SkillsSecurityConfig {
  actionModes: Record<string, SkillsActionMode>;
  scannerMode: "off" | "warn" | "block";
  allowForceInstall: boolean;
  requireApproval: boolean;
  allowExternalToolsOnly: boolean;
  allowGenericInstaller: boolean;
  allowedBins: string[];
}

export interface SkillsAuditRecord {
  action: string;
  tool?: string;
  group?: string;
  skill?: string;
  assistantId?: string;
  providerId?: string;
  source?: string;
  ok?: boolean;
  errorCode?: string;
  error?: string;
  timestamp?: string;
}

export interface SkillsAuditConfig {
  hideUiOperationRecords: boolean;
  retentionDays: number;
}

export const SKILLS_AUDIT_RETENTION_DAY_OPTIONS = [3, 5, 7, 14, 28, 60, 90] as const;
export const DEFAULT_SKILLS_AUDIT_RETENTION_DAYS = 14;

export const defaultSkillsActionModes: Record<string, SkillsActionMode> = {
  read: "allow",
  package_write: "ask",
  deps_write: "ask",
  config_write: "ask",
  source_write: "ask",
};

const DEFAULT_LOCAL_SOURCES: LocalSkillSourceItem[] = [
  {
    id: "workspace",
    name: "DreamCreator",
    kind: "local",
    type: "workspace",
    path: "",
    enabled: true,
    priority: 0,
    trust: "local",
    builtin: true,
  },
];

const DEFAULT_REMOTE_SOURCES: RemoteSkillSourceItem[] = [
  {
    id: "clawhub",
    name: "ClawHub",
    kind: "remote",
    provider: "clawhub",
    enabled: true,
    searchEnabled: true,
    installEnabled: true,
    builtin: true,
  },
];

export function resolveSkillsSettingsState(
  rawTools?: Record<string, unknown>,
  rawSkills?: Record<string, unknown>
) {
  const toolsConfig = isRecord(rawTools) ? { ...(rawTools as Record<string, unknown>) } : {};
  const skillsConfig = isRecord(rawSkills) ? { ...(rawSkills as Record<string, unknown>) } : {};
  return {
    toolsConfig,
    skillsConfig,
  };
}

export function buildNextSettingsTools(
  nextToolsConfig: Record<string, unknown>
): Record<string, unknown> {
  return nextToolsConfig;
}

export function parseSkillsSourcesState(skillsConfig: Record<string, unknown>): SkillsSourcesState {
  const raw = skillsConfig.sources;
  if (!isRecord(raw)) {
    return {
      local: cloneLocalSources(DEFAULT_LOCAL_SOURCES),
      remote: cloneRemoteSources(DEFAULT_REMOTE_SOURCES),
    };
  }
  const local = Array.isArray(raw.local) ? parseStructuredLocalSources(raw.local) : [];
  const remote = Array.isArray(raw.remote) ? parseStructuredRemoteSources(raw.remote) : [];
  return {
    local: normalizeLocalSources(mergeLocalSources(local)),
    remote: normalizeRemoteSources(mergeRemoteSources(remote)),
  };
}

export function toSourcesPayload(state: SkillsSourcesState): Record<string, unknown> {
  return {
    local: normalizeLocalSources(state.local)
      .map((item) => ({
        id: item.id.trim(),
        name: item.name.trim(),
        kind: item.kind,
        type: item.type,
        path: item.path.trim(),
        enabled: item.enabled,
        priority: item.priority,
        trust: item.trust,
        builtin: item.builtin,
        lastScanAt: item.lastScanAt,
        lastScanResult: item.lastScanResult,
      }))
      .filter((item) => item.type !== "extra" || item.path !== ""),
    remote: normalizeRemoteSources(state.remote).map((item) => ({
      id: item.id.trim(),
      name: item.name.trim(),
      kind: item.kind,
      provider: item.provider,
      enabled: item.enabled,
      searchEnabled: item.searchEnabled,
      installEnabled: item.installEnabled,
      builtin: item.builtin,
      lastSyncAt: item.lastSyncAt,
    })),
  };
}

export function parseSkillsSecurity(skillsConfig: Record<string, unknown>): SkillsSecurityConfig {
  const security = isRecord(skillsConfig.security) ? (skillsConfig.security as Record<string, unknown>) : {};
  const actionModesRaw = isRecord(security.actionModes)
    ? (security.actionModes as Record<string, unknown>)
    : {};
  const actionModes: Record<string, SkillsActionMode> = { ...defaultSkillsActionModes };
  Object.entries(actionModesRaw).forEach(([key, value]) => {
    if (typeof value !== "string") {
      return;
    }
    const mode = value.trim().toLowerCase();
    if (mode === "allow" || mode === "ask" || mode === "deny") {
      actionModes[key] = mode;
    }
  });
  const install = isRecord(security.install) ? (security.install as Record<string, unknown>) : {};
  const deps = isRecord(security.deps) ? (security.deps as Record<string, unknown>) : {};
  const allowedBinsRaw = Array.isArray(deps.allowedBins) ? deps.allowedBins : [];
  const allowedBins = allowedBinsRaw
    .map((value) => (typeof value === "string" ? value.trim() : ""))
    .filter((value) => value !== "");
  const scannerModeRaw = typeof install.scannerMode === "string" ? install.scannerMode.trim().toLowerCase() : "warn";
  const scannerMode = scannerModeRaw === "off" || scannerModeRaw === "warn" || scannerModeRaw === "block"
    ? scannerModeRaw
    : "warn";
  return {
    actionModes,
    scannerMode,
    allowForceInstall: install.allowForceInstall !== false,
    requireApproval: install.requireApproval !== false,
    allowExternalToolsOnly: deps.allowExternalToolsOnly === true,
    allowGenericInstaller: deps.allowGenericInstaller !== false,
    allowedBins,
  };
}

export function toSecurityPayload(config: SkillsSecurityConfig): Record<string, unknown> {
  return {
    actionModes: { ...config.actionModes },
    install: {
      scannerMode: config.scannerMode,
      allowForceInstall: config.allowForceInstall,
      requireApproval: config.requireApproval,
    },
    deps: {
      allowExternalToolsOnly: config.allowExternalToolsOnly,
      allowGenericInstaller: config.allowGenericInstaller,
      allowedBins: config.allowedBins,
    },
  };
}

export function parseSkillsAudit(skillsConfig: Record<string, unknown>): SkillsAuditRecord[] {
  const raw = skillsConfig.audit;
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw
    .filter((item): item is Record<string, unknown> => isRecord(item))
    .map((item) => ({
      action: typeof item.action === "string" ? item.action : "",
      tool: typeof item.tool === "string" ? item.tool : undefined,
      group: typeof item.group === "string" ? item.group : undefined,
      skill: typeof item.skill === "string" ? item.skill : undefined,
      assistantId: typeof item.assistantId === "string" ? item.assistantId : undefined,
      providerId: typeof item.providerId === "string" ? item.providerId : undefined,
      source: typeof item.source === "string" ? item.source : undefined,
      ok: typeof item.ok === "boolean" ? item.ok : undefined,
      errorCode: typeof item.errorCode === "string" ? item.errorCode : undefined,
      error: typeof item.error === "string" ? item.error : undefined,
      timestamp: typeof item.timestamp === "string" ? item.timestamp : undefined,
    }))
    .filter((item) => item.action !== "")
    .reverse();
}

export function parseSkillsAuditConfig(skillsConfig: Record<string, unknown>): SkillsAuditConfig {
  const raw = isRecord(skillsConfig.auditConfig) ? (skillsConfig.auditConfig as Record<string, unknown>) : {};
  const hideUiOperationRecords = raw.hideUiOperationRecords !== false;
  const retentionDays = normalizeAuditRetentionDays(resolveAuditRetentionDays(raw.retentionDays));
  return {
    hideUiOperationRecords,
    retentionDays,
  };
}

export function toAuditConfigPayload(config: SkillsAuditConfig): Record<string, unknown> {
  return {
    hideUiOperationRecords: config.hideUiOperationRecords !== false,
    retentionDays: normalizeAuditRetentionDays(config.retentionDays),
  };
}

function cloneLocalSources(items: LocalSkillSourceItem[]): LocalSkillSourceItem[] {
  return items.map((item) => ({ ...item }));
}

function cloneRemoteSources(items: RemoteSkillSourceItem[]): RemoteSkillSourceItem[] {
  return items.map((item) => ({ ...item }));
}

function mergeLocalSources(items: LocalSkillSourceItem[]): LocalSkillSourceItem[] {
  const merged = cloneLocalSources(DEFAULT_LOCAL_SOURCES);
  const builtinIndex = new Map<string, number>();
  merged.forEach((item, index) => builtinIndex.set(item.type, index));
  items.forEach((item) => {
    const index = builtinIndex.get(item.type);
    if (item.builtin && typeof index === "number") {
      merged[index] = { ...merged[index], ...item, builtin: true };
      return;
    }
    merged.push({ ...item, kind: "local" });
  });
  return merged;
}

function mergeRemoteSources(items: RemoteSkillSourceItem[]): RemoteSkillSourceItem[] {
  const merged = cloneRemoteSources(DEFAULT_REMOTE_SOURCES);
  const builtinIndex = new Map<string, number>();
  merged.forEach((item, index) => builtinIndex.set(item.id, index));
  items.forEach((item) => {
    const index = builtinIndex.get(item.id);
    if (item.builtin && typeof index === "number") {
      merged[index] = { ...merged[index], ...item, builtin: true };
      return;
    }
    merged.push({ ...item, kind: "remote" });
  });
  return merged;
}

function parseStructuredLocalSources(raw: unknown[]): LocalSkillSourceItem[] {
  const result: LocalSkillSourceItem[] = [];
  raw.forEach((entry, index) => {
    if (!isRecord(entry)) {
      return;
    }
    const rawType = typeof entry.type === "string" ? entry.type.trim() : "";
    const normalizedType = normalizeLocalSourceType(rawType);
    const type: LocalSkillSourceType | "" = normalizedType || (rawType ? "" : "extra");
    if (!type) {
      return;
    }
    if (!isSupportedConfiguredLocalSourceType(type)) {
      return;
    }
    const path = typeof entry.path === "string" ? entry.path.trim() : "";
    if (type === "extra" && !path) {
      return;
    }
    const builtin = type !== "extra" || entry.builtin === true;
    result.push({
      id: normalizeSourceId(typeof entry.id === "string" ? entry.id : "", type, index),
      name: normalizeSourceName(typeof entry.name === "string" ? entry.name : "", type, path, index),
      kind: "local",
      type,
      path,
      enabled: entry.enabled !== false,
      priority: typeof entry.priority === "number" && Number.isFinite(entry.priority)
        ? Math.max(0, Math.floor(entry.priority))
        : index,
      trust: normalizeTrust(entry.trust),
      builtin,
      lastScanAt: typeof entry.lastScanAt === "string" ? entry.lastScanAt : undefined,
      lastScanResult: typeof entry.lastScanResult === "string" ? entry.lastScanResult : undefined,
    });
  });
  return result;
}

function parseStructuredRemoteSources(raw: unknown[]): RemoteSkillSourceItem[] {
  const result: RemoteSkillSourceItem[] = [];
  raw.forEach((entry) => {
    if (!isRecord(entry)) {
      return;
    }
    const provider = normalizeRemoteProvider(entry.provider);
    const id = typeof entry.id === "string" && entry.id.trim() ? entry.id.trim() : provider;
    result.push({
      id,
      name: typeof entry.name === "string" && entry.name.trim() ? entry.name.trim() : defaultRemoteSourceName(provider),
      kind: "remote",
      provider,
      enabled: entry.enabled !== false,
      searchEnabled: entry.searchEnabled !== false,
      installEnabled: entry.installEnabled !== false,
      builtin: entry.builtin !== false,
      lastSyncAt: typeof entry.lastSyncAt === "string" ? entry.lastSyncAt : undefined,
    });
  });
  return result;
}

export function normalizeLocalSources(items: LocalSkillSourceItem[]): LocalSkillSourceItem[] {
  const normalized = items
    .map((item, index) => {
      const type = normalizeLocalSourceType(item.type) || "extra";
      const path = item.path.trim();
      return {
        ...item,
        type,
        id: normalizeSourceId(item.id, type, index),
        name: normalizeSourceName(item.name, type, path, index),
        kind: "local" as const,
        path,
        enabled: item.enabled !== false,
        priority: Number.isFinite(item.priority) ? Math.max(0, Math.floor(item.priority)) : index,
        trust: normalizeTrust(item.trust),
        builtin: type !== "extra" || item.builtin === true,
      };
    })
    .filter((item) => isSupportedConfiguredLocalSourceType(item.type))
    .filter((item) => item.type !== "extra" || item.path !== "");
  normalized.sort((left, right) => {
    if (left.priority !== right.priority) {
      return left.priority - right.priority;
    }
    return left.id.localeCompare(right.id);
  });
  return normalized.map((item, index) => ({
    ...item,
    priority: index,
  }));
}

export function normalizeRemoteSources(items: RemoteSkillSourceItem[]): RemoteSkillSourceItem[] {
  const normalized = items.map((item) => ({
    ...item,
    id: (item.id || item.provider || "clawhub").trim() || "clawhub",
    name: (item.name || defaultRemoteSourceName(item.provider)).trim() || defaultRemoteSourceName(item.provider),
    kind: "remote" as const,
    provider: normalizeRemoteProvider(item.provider),
    enabled: item.enabled !== false,
    searchEnabled: item.searchEnabled !== false,
    installEnabled: item.installEnabled !== false,
    builtin: item.builtin !== false,
  }));
  normalized.sort((left, right) => left.id.localeCompare(right.id));
  return normalized;
}

export function buildLocalExtraSource(params: {
  id?: string;
  name?: string;
  path: string;
  priority?: number;
  trust?: SkillSourceTrust;
}): LocalSkillSourceItem {
  const path = params.path.trim();
  const priority = typeof params.priority === "number" && Number.isFinite(params.priority)
    ? Math.max(0, Math.floor(params.priority))
    : DEFAULT_LOCAL_SOURCES.length;
  return {
    id: normalizeSourceId(params.id ?? "", "extra", priority),
    name: normalizeSourceName(params.name ?? "", "extra", path, priority),
    kind: "local",
    type: "extra",
    path,
    enabled: true,
    priority,
    trust: normalizeTrust(params.trust),
    builtin: false,
  };
}

export function defaultBuiltinLocalSourceName(type: LocalSkillSourceType): string {
  switch (type) {
    case "workspace":
      return "DreamCreator";
    default:
      return "Local";
  }
}

export function defaultRemoteSourceName(provider: RemoteSkillSourceProvider): string {
  if (provider === "clawhub") {
    return "ClawHub";
  }
  return provider;
}

function normalizeLocalSourceType(value: unknown): LocalSkillSourceType | "" {
  const normalized = typeof value === "string" ? value.trim().toLowerCase() : "";
  if (normalized === "workspace" || normalized === "extra") {
    return normalized;
  }
  return "";
}

function isSupportedConfiguredLocalSourceType(type: LocalSkillSourceType): boolean {
  return type === "workspace" || type === "extra";
}

function normalizeRemoteProvider(value: unknown): RemoteSkillSourceProvider {
  return value === "clawhub" ? "clawhub" : "clawhub";
}

function normalizeTrust(value: unknown): SkillSourceTrust {
  const normalized = typeof value === "string" ? value.trim().toLowerCase() : "local";
  if (normalized === "trusted" || normalized === "community" || normalized === "local") {
    return normalized;
  }
  return "local";
}

function normalizeSourceId(value: string, type: LocalSkillSourceType, index: number): string {
  const trimmed = value.trim();
  if (trimmed) {
    return trimmed;
  }
  if (type !== "extra") {
    return type;
  }
  return `extra-${index + 1}`;
}

function normalizeSourceName(value: string, type: LocalSkillSourceType, path: string, index: number): string {
  const trimmed = value.trim();
  if (trimmed) {
    return trimmed;
  }
  if (type !== "extra") {
    return defaultBuiltinLocalSourceName(type);
  }
  const normalizedPath = path.trim();
  if (normalizedPath) {
    const segments = normalizedPath.split(/[\\/]+/).filter(Boolean);
    const last = segments.length > 0 ? segments[segments.length - 1]?.trim() : "";
    if (last) {
      return last;
    }
    return normalizedPath;
  }
  return `Extra ${index + 1}`;
}

function resolveAuditRetentionDays(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.floor(value);
  }
  if (typeof value === "string") {
    const parsed = Number.parseInt(value.trim(), 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return DEFAULT_SKILLS_AUDIT_RETENTION_DAYS;
}

function normalizeAuditRetentionDays(value: number): number {
  if (!Number.isFinite(value)) {
    return DEFAULT_SKILLS_AUDIT_RETENTION_DAYS;
  }
  const normalized = Math.floor(value);
  if (normalized <= 0) {
    return DEFAULT_SKILLS_AUDIT_RETENTION_DAYS;
  }
  if (normalized > 365) {
    return 365;
  }
  return normalized;
}
