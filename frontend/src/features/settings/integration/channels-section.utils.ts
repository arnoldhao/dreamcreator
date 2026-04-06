import { AlertTriangle, CircleOff, HelpCircle, Plug2, RefreshCw } from "lucide-react";
import type { ComponentType } from "react";

import type { ChannelOverview } from "@/shared/store/channels";

export const STATUS_META: Record<
  string,
  { labelKey: string; label: string; className: string; icon: ComponentType<{ className?: string }> }
> = {
  online: {
    labelKey: "settings.integration.channels.status.online",
    label: "Online",
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-100",
    icon: Plug2,
  },
  offline: {
    labelKey: "settings.integration.channels.status.offline",
    label: "Offline",
    className: "bg-muted text-muted-foreground",
    icon: CircleOff,
  },
  degraded: {
    labelKey: "settings.integration.channels.status.degraded",
    label: "Degraded",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-100",
    icon: AlertTriangle,
  },
  reconnecting: {
    labelKey: "settings.integration.channels.status.reconnecting",
    label: "Reconnecting",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-100",
    icon: RefreshCw,
  },
  disabled: {
    labelKey: "settings.integration.channels.status.disabled",
    label: "Disabled",
    className: "bg-muted text-muted-foreground",
    icon: CircleOff,
  },
  unknown: {
    labelKey: "settings.integration.channels.status.unknown",
    label: "Unknown",
    className: "bg-muted text-muted-foreground",
    icon: HelpCircle,
  },
  not_configured: {
    labelKey: "settings.integration.channels.status.notConfigured",
    label: "Not configured",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-100",
    icon: AlertTriangle,
  },
};

export type ChannelField = {
  key: string;
  type: "text" | "secret" | "select" | "toggle";
  labelKey: string;
  label: string;
  placeholder?: string;
  options?: Array<{ value: string; labelKey: string; label: string }>;
  allowDefault?: boolean;
};

export type CommandSettingValue = "on" | "off";
export type TelegramCustomCommand = { command: string; description: string };
export type GroupAllowEntry = { id: string; requireMention: boolean };

export const telegramCommandNamePattern = /^[a-z0-9_]{1,32}$/;

const parseListInput = (value: string) =>
  value
    .split(/[\n,;]+/g)
    .map((entry) => entry.trim())
    .filter(Boolean);


const GROUP_POLICY_OPTIONS: ChannelField["options"] = [
  {
    value: "allowlist",
    labelKey: "settings.integration.channels.config.options.groupPolicy.allowlist",
    label: "Allowlist",
  },
  { value: "open", labelKey: "settings.integration.channels.config.options.groupPolicy.open", label: "Open" },
  {
    value: "disabled",
    labelKey: "settings.integration.channels.config.options.groupPolicy.disabled",
    label: "Disabled",
  },
];

const DM_POLICY_OPTIONS: ChannelField["options"] = [
  {
    value: "pairing",
    labelKey: "settings.integration.channels.config.options.dmPolicy.pairing",
    label: "Pairing",
  },
  {
    value: "allowlist",
    labelKey: "settings.integration.channels.config.options.dmPolicy.allowlist",
    label: "Allowlist",
  },
  { value: "open", labelKey: "settings.integration.channels.config.options.dmPolicy.open", label: "Open" },
  {
    value: "disabled",
    labelKey: "settings.integration.channels.config.options.dmPolicy.disabled",
    label: "Disabled",
  },
];

const TELEGRAM_STREAM_OPTIONS: ChannelField["options"] = [
  {
    value: "off",
    labelKey: "settings.integration.channels.config.options.streamMode.off",
    label: "Off",
  },
  {
    value: "block",
    labelKey: "settings.integration.channels.config.options.streamMode.block",
    label: "Block",
  },
  {
    value: "partial",
    labelKey: "settings.integration.channels.config.options.streamMode.partial",
    label: "Partial",
  },
];

const SLACK_STREAM_OPTIONS: ChannelField["options"] = [
  {
    value: "replace",
    labelKey: "settings.integration.channels.config.options.streamMode.replace",
    label: "Replace",
  },
  {
    value: "status_final",
    labelKey: "settings.integration.channels.config.options.streamMode.status_final",
    label: "Status final",
  },
  {
    value: "append",
    labelKey: "settings.integration.channels.config.options.streamMode.append",
    label: "Append",
  },
];

const FIELD_TOKEN: ChannelField = {
  key: "token",
  type: "secret",
  labelKey: "settings.integration.channels.config.fields.token",
  label: "Token",
};

export const FIELD_BOT_TOKEN: ChannelField = {
  key: "botToken",
  type: "secret",
  labelKey: "settings.integration.channels.config.fields.botToken",
  label: "Bot token",
};

const FIELD_BEARER_TOKEN: ChannelField = {
  key: "bearerToken",
  type: "secret",
  labelKey: "settings.integration.channels.config.fields.bearerToken",
  label: "Bearer token",
};

export const FIELD_GROUP_POLICY: ChannelField = {
  key: "groupPolicy",
  type: "select",
  labelKey: "settings.integration.channels.config.fields.groupPolicy",
  label: "Group policy",
  options: GROUP_POLICY_OPTIONS,
};

export const FIELD_DM_POLICY: ChannelField = {
  key: "dmPolicy",
  type: "select",
  labelKey: "settings.integration.channels.config.fields.dmPolicy",
  label: "DM policy",
  options: DM_POLICY_OPTIONS,
};

export const FIELD_STREAM_MODE_TELEGRAM: ChannelField = {
  key: "streamMode",
  type: "select",
  labelKey: "settings.integration.channels.config.fields.streamMode",
  label: "Stream mode",
  options: TELEGRAM_STREAM_OPTIONS,
  allowDefault: false,
};

const FIELD_STREAM_MODE_SLACK: ChannelField = {
  key: "streamMode",
  type: "select",
  labelKey: "settings.integration.channels.config.fields.streamMode",
  label: "Stream mode",
  options: SLACK_STREAM_OPTIONS,
};

export const COMMAND_SETTING_OPTIONS: Array<{ value: CommandSettingValue; labelKey: string; label: string }> = [
  { value: "on", labelKey: "settings.integration.channels.menu.commands.on", label: "On" },
  { value: "off", labelKey: "settings.integration.channels.menu.commands.off", label: "Off" },
];

const CHANNEL_FIELDS: Record<string, ChannelField[]> = {
  telegram: [
    FIELD_BOT_TOKEN,
    FIELD_DM_POLICY,
    FIELD_GROUP_POLICY,
    FIELD_STREAM_MODE_TELEGRAM,
  ],
  discord: [FIELD_TOKEN, FIELD_DM_POLICY, FIELD_GROUP_POLICY],
  whatsapp: [FIELD_DM_POLICY, FIELD_GROUP_POLICY],
  web: [FIELD_BEARER_TOKEN],
  slack: [FIELD_BOT_TOKEN, FIELD_DM_POLICY, FIELD_GROUP_POLICY, FIELD_STREAM_MODE_SLACK],
};

const DEFAULT_FIELDS: ChannelField[] = [FIELD_TOKEN, FIELD_DM_POLICY, FIELD_GROUP_POLICY];

const REQUIRED_FIELDS_BY_CHANNEL: Record<string, string[]> = {
  telegram: ["botToken"],
  discord: ["token"],
  web: ["bearerToken"],
  slack: ["botToken"],
};

export const REQUIRED_FIELD_LABELS: Record<string, ChannelField> = {
  token: FIELD_TOKEN,
  botToken: FIELD_BOT_TOKEN,
  bearerToken: FIELD_BEARER_TOKEN,
};

export const resolveChannelFields = (channelId?: string | null) => {
  if (!channelId) {
    return [];
  }
  return CHANNEL_FIELDS[channelId] ?? DEFAULT_FIELDS;
};

export const resolveRecord = (value: unknown): Record<string, any> => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, any>;
};

export const resolveConfigEnabled = (channel: ChannelOverview | null, config: Record<string, any>) => {
  if (typeof config.enabled === "boolean") {
    return config.enabled;
  }
  if (typeof channel?.enabled === "boolean") {
    return channel.enabled;
  }
  return true;
};

export const resolveMissingRequiredFields = (channelId: string | null, config: Record<string, any>) => {
  if (!channelId) {
    return [];
  }
  const required = REQUIRED_FIELDS_BY_CHANNEL[channelId] ?? [];
  if (required.length === 0) {
    return [];
  }
  return required.filter((field) => {
    if (channelId === "telegram" && field === "botToken") {
      return resolveTelegramBotToken(config) === "";
    }
    const value = config[field];
    if (value == null) {
      return true;
    }
    if (typeof value === "string") {
      return value.trim() === "";
    }
    return false;
  });
};

export const resolveCommandSettingValue = (value: unknown): CommandSettingValue => {
  if (value === false) {
    return "off";
  }
  return "on";
};

export const resolveCustomCommandList = (value: unknown): TelegramCustomCommand[] => {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((entry) => {
      if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
        return null;
      }
      const record = entry as Record<string, any>;
      return {
        command: typeof record.command === "string" ? record.command : "",
        description: typeof record.description === "string" ? record.description : "",
      };
    })
    .filter((entry): entry is TelegramCustomCommand => Boolean(entry));
};

export const resolveAllowFromList = (value: unknown): string[] => {
  if (!value) {
    return [];
  }
  if (Array.isArray(value)) {
    return value
      .map((entry) => (typeof entry === "string" || typeof entry === "number" ? String(entry) : ""))
      .map((entry) => entry.trim())
      .filter(Boolean);
  }
  if (typeof value === "string") {
    return parseListInput(value);
  }
  return [];
};

export const resolveGroupAllowList = (value: unknown): GroupAllowEntry[] => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [];
  }
  const record = value as Record<string, any>;
  const entries = Object.entries(record)
    .map(([id, raw]) => {
      const entry = raw && typeof raw === "object" && !Array.isArray(raw) ? (raw as Record<string, any>) : {};
      const requireMention = typeof entry.requireMention === "boolean" ? entry.requireMention : true;
      return { id, requireMention };
    })
    .filter((entry) => entry.id.trim() !== "");
  entries.sort((left, right) => left.id.localeCompare(right.id));
  return entries;
};

export const sanitizeAllowFromDraft = (entries: string[]) =>
  entries
    .map((entry) => entry.trim())
    .filter(Boolean);

export const normalizeTelegramCommandName = (value: string) => {
  let trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }
  if (trimmed.startsWith("/")) {
    trimmed = trimmed.slice(1);
  }
  trimmed = trimmed.trim().toLowerCase();
  return trimmed.replace(/-/g, "_");
};

export const sanitizeGroupAllowDraft = (entries: GroupAllowEntry[]) =>
  entries
    .map((entry) => ({
      id: entry.id.trim(),
      requireMention: entry.requireMention !== false,
    }))
    .filter((entry) => entry.id !== "");

export const sanitizeCustomCommandDraft = (entries: TelegramCustomCommand[]) =>
  entries
    .map((entry) => ({
      command: entry.command.trim(),
      description: entry.description.trim(),
    }))
    .filter((entry) => entry.command !== "" || entry.description !== "");

export const areListsEqual = (left: string[], right: string[]) => {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((entry, index) => entry === right[index]);
};

export const areCustomCommandsEqual = (left: TelegramCustomCommand[], right: TelegramCustomCommand[]) => {
  if (left.length !== right.length) {
    return false;
  }
  return left.every(
    (entry, index) =>
      entry.command === right[index].command && entry.description === right[index].description
  );
};

export const areGroupEntriesEqual = (left: GroupAllowEntry[], right: GroupAllowEntry[]) => {
  if (left.length !== right.length) {
    return false;
  }
  return left.every(
    (entry, index) =>
      entry.id === right[index].id && entry.requireMention === right[index].requireMention
  );
};

export const stableStringify = (value: unknown) => {
  const seen = new WeakSet();
  return JSON.stringify(value, (_key, raw) => {
    if (!raw || typeof raw !== "object") {
      return raw;
    }
    if (seen.has(raw as object)) {
      return raw;
    }
    seen.add(raw as object);
    if (Array.isArray(raw)) {
      return raw;
    }
    const entries = Object.entries(raw as Record<string, unknown>).sort(([a], [b]) =>
      a.localeCompare(b)
    );
    const ordered: Record<string, unknown> = {};
    for (const [key, val] of entries) {
      ordered[key] = val;
    }
    return ordered;
  });
};

export const resolveTelegramBotToken = (config: Record<string, any>) => {
  const botToken = typeof config.botToken === "string" ? config.botToken.trim() : "";
  return botToken;
};

export const resolveChannelStatus = (
  channelId: string | null,
  channel: ChannelOverview | null,
  config: Record<string, any>,
  configReady = true
) => {
  if (!channelId) {
    return "unknown";
  }
  const enabled = resolveConfigEnabled(channel, config);
  if (!enabled) {
    return "disabled";
  }
  if (!configReady) {
    return channel?.state || "unknown";
  }
  const missing = resolveMissingRequiredFields(channelId, config);
  if (missing.length > 0) {
    return "not_configured";
  }
  return channel?.state || "unknown";
};

export const formatGatewayError = (error: unknown) => {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  if (error && typeof error === "object") {
    const errorValue = (error as { error?: unknown }).error;
    if (typeof errorValue === "string") {
      return errorValue;
    }
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string") {
      return message;
    }
    try {
      return JSON.stringify(error);
    } catch {
      return String(error);
    }
  }
  return String(error);
};
