import type { Assistant } from "@/shared/store/assistant";
import type { GatewayHeartbeatChecklist } from "@/shared/contracts/settings";
import {
  SETTINGS_CONTROL_WIDTH_CLASS,
  SETTINGS_ROW_CLASS,
  SETTINGS_ROW_LABEL_TRUNCATE_CLASS,
} from "@/shared/ui/settings-layout";

export type DetailsSectionId =
  | "gateway"
  | "context"
  | "agentLoop"
  | "queue"
  | "cron"
  | "heartbeat"
  | "subagents"
  | "http"
  | "talkMode"
  | "voiceWakeTriggers";

export interface DetailsSection {
  id: DetailsSectionId;
  labelKey: string;
  label: string;
  descriptionKey?: string;
  description?: string;
}

export const DETAILS_SECTIONS: DetailsSection[] = [
  { id: "gateway", labelKey: "settings.gateway.detailsPanel.sections.gateway", label: "Gateway" },
  { id: "context", labelKey: "settings.gateway.detailsPanel.sections.context", label: "Context" },
  {
    id: "agentLoop",
    labelKey: "settings.gateway.detailsPanel.sections.agentLoop",
    label: "Agent loop",
  },
  { id: "queue", labelKey: "settings.gateway.detailsPanel.sections.queue", label: "Queue" },
  {
    id: "cron",
    labelKey: "settings.gateway.details.cron.item",
    label: "Cron",
    descriptionKey: "settings.gateway.details.cron.content",
    description: "",
  },
  { id: "heartbeat", labelKey: "settings.gateway.detailsPanel.sections.heartbeat", label: "Heartbeat" },
  { id: "subagents", labelKey: "settings.gateway.detailsPanel.sections.subagents", label: "Subagents" },
  { id: "http", labelKey: "settings.gateway.detailsPanel.sections.http", label: "HTTP" },
  { id: "talkMode", labelKey: "settings.gateway.detailsPanel.sections.talk", label: "Talk mode" },
  {
    id: "voiceWakeTriggers",
    labelKey: "settings.gateway.detailsPanel.sections.voiceWake",
    label: "Wake words",
  },
];

export interface GatewayDetailsPanelProps {
  assistant: Assistant;
  assistants: Assistant[];
  currentAssistantId: string | null;
  onSelectAssistant: (id: string) => void;
}

export const rowClassName = SETTINGS_ROW_CLASS;
export const panelClassName = "flex min-w-0 flex-col space-y-3";
export const rowLabelClassName = SETTINGS_ROW_LABEL_TRUNCATE_CLASS;
export const controlClassName = `${SETTINGS_CONTROL_WIDTH_CLASS} !text-xs`;
export const CRON_SESSION_RETENTION_OPTIONS = ["1h", "6h", "12h", "24h", "72h", "168h"] as const;

const EXEC_PERMISSION_MODE_KEY = "execPermissionMode";
export const EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS = "default permissions";
export const EXEC_PERMISSION_MODE_FULL_ACCESS = "full access";

export type ExecPermissionMode =
  | typeof EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS
  | typeof EXEC_PERMISSION_MODE_FULL_ACCESS;

export const toCommaList = (items?: string[]) => (items ?? []).join(", ");

export const parseCommaList = (value: string) =>
  value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

export const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

export const normalizeExecPermissionMode = (value: unknown): ExecPermissionMode => {
  if (typeof value !== "string") {
    return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
  }
  const normalized = value.trim().toLowerCase();
  if (
    normalized === EXEC_PERMISSION_MODE_FULL_ACCESS ||
    normalized === "full_access" ||
    normalized === "all-access" ||
    normalized === "all_access" ||
    normalized === "all access" ||
    normalized === "full"
  ) {
    return EXEC_PERMISSION_MODE_FULL_ACCESS;
  }
  if (
    normalized === EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS ||
    normalized === "default_permissions" ||
    normalized === "default permission" ||
    normalized === "default" ||
    normalized === "standard" ||
    normalized === "safe" ||
    normalized === "ask"
  ) {
    return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
  }
  return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
};

const resolveToolsConfig = (raw: unknown) => (isRecord(raw) ? { ...raw } : {});

export const resolveExecPermissionMode = (callsToolsRaw: unknown): ExecPermissionMode => {
  const toolsConfig = resolveToolsConfig(callsToolsRaw);
  const direct = toolsConfig[EXEC_PERMISSION_MODE_KEY];
  if (direct !== undefined) {
    return normalizeExecPermissionMode(direct);
  }
  const fallback = toolsConfig.permissionMode;
  if (fallback !== undefined) {
    return normalizeExecPermissionMode(fallback);
  }
  return EXEC_PERMISSION_MODE_DEFAULT_PERMISSIONS;
};

export const patchCallsToolsExecPermissionMode = (callsToolsRaw: unknown, mode: ExecPermissionMode) => {
  const toolsConfig = resolveToolsConfig(callsToolsRaw);
  return {
    ...toolsConfig,
    [EXEC_PERMISSION_MODE_KEY]: mode,
  };
};

export const formatHeartbeatEvery = (value?: { every?: string; everyMinutes?: number }) => {
  const direct = value?.every?.trim() ?? "";
  if (direct) {
    return direct;
  }
  const minutes = value?.everyMinutes ?? 0;
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return "";
};

export const parseHeartbeatEveryMinutes = (value: string) => {
  const trimmed = value.trim().toLowerCase();
  if (!trimmed) {
    return 0;
  }
  const match = trimmed.match(/^(\d+)\s*([smhd])?$/);
  if (!match) {
    return 0;
  }
  const amount = Number(match[1]);
  if (!Number.isFinite(amount) || amount <= 0) {
    return 0;
  }
  const unit = match[2] ?? "m";
  switch (unit) {
    case "s":
      return Math.max(1, Math.ceil(amount / 60));
    case "h":
      return amount * 60;
    case "d":
      return amount * 24 * 60;
    default:
      return amount;
  }
};

export const parseCronRunLogMaxMegabytes = (value?: string) => {
  const trimmed = (value ?? "").trim().toLowerCase();
  if (!trimmed) {
    return 0;
  }
  const match = trimmed.match(/^(\d+)\s*(b|kb|mb|gb)?$/);
  if (!match) {
    return 0;
  }
  const amount = Number(match[1]);
  if (!Number.isFinite(amount) || amount < 0) {
    return 0;
  }
  const unit = match[2] ?? "mb";
  if (unit === "gb") {
    return amount * 1024;
  }
  if (unit === "kb") {
    return Math.ceil(amount / 1024);
  }
  if (unit === "b") {
    return Math.ceil(amount / (1024 * 1024));
  }
  return amount;
};

export const formatVoiceAliases = (aliases?: Record<string, string>) =>
  aliases
    ? Object.entries(aliases)
        .map(([alias, voiceId]) => `${alias}:${voiceId}`)
        .join(", ")
    : "";

export const parseVoiceAliases = (value: string) => {
  const entries = value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
  if (entries.length === 0) {
    return undefined;
  }
  const result: Record<string, string> = {};
  for (const entry of entries) {
    const separatorIndex = entry.includes(":") ? entry.indexOf(":") : entry.indexOf("=");
    if (separatorIndex <= 0) {
      continue;
    }
    const alias = entry.slice(0, separatorIndex).trim();
    const voiceId = entry.slice(separatorIndex + 1).trim();
    if (!alias || !voiceId) {
      continue;
    }
    result[alias] = voiceId;
  }
  return Object.keys(result).length > 0 ? result : undefined;
};

export type HeartbeatSpecItemDraft = {
  id: string;
  text: string;
  done: boolean;
  priority: string;
};

export type HeartbeatSpecDraft = {
  title: string;
  notes: string;
  items: HeartbeatSpecItemDraft[];
};

export type HeartbeatTriggerResponse = {
  ok?: boolean;
  accepted?: boolean;
  executedStatus?: "queued" | "ran" | "skipped" | "failed";
  reason?: string;
};

export const normalizeStringRows = (items: string[]) =>
  items
    .map((item) => item.trim())
    .filter(Boolean);

export const buildEmptyHeartbeatSpecDraft = (): HeartbeatSpecDraft => ({
  title: "",
  notes: "",
  items: [],
});

export const buildHeartbeatSpecDraftFromChecklist = (
  checklist?: GatewayHeartbeatChecklist | null
): HeartbeatSpecDraft => {
  if (!checklist) {
    return buildEmptyHeartbeatSpecDraft();
  }
  return {
    title: typeof checklist.title === "string" ? checklist.title : "",
    notes: typeof checklist.notes === "string" ? checklist.notes : "",
    items: Array.isArray(checklist.items)
      ? checklist.items.map((item) => ({
          id: typeof item.id === "string" ? item.id : "",
          text: typeof item.text === "string" ? item.text : "",
          done: item.done === true,
          priority: typeof item.priority === "string" ? item.priority : "",
        }))
      : [],
  };
};
