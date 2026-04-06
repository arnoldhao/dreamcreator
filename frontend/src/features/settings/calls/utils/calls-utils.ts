import type { GatewayToolRequirement, GatewayToolSpec } from "@/shared/store/gatewayTools";
import type { SkillRuntimeInstallSpec } from "@/shared/contracts/skills";

import type { SkillsActionErrorInfo, ToolItem, ToolRequirementStatus } from "../types";

export const normalizeToolId = (value: string) => value.trim().toLowerCase();

export const baseToolIds: string[] = [];

export const CATEGORY_ORDER = [
  "fs",
  "runtime",
  "web",
  "ui",
  "media",
  "messaging",
  "automation",
  "sessions",
  "external_tools",
  "skills",
  "nodes",
  "voice",
  "memory",
  "library",
  "other",
] as const;

export const CATEGORY_LABELS: Record<string, string> = {
  fs: "File System",
  runtime: "Runtime",
  web: "Web",
  ui: "UI",
  media: "Media",
  messaging: "Messaging",
  automation: "Automation",
  sessions: "Sessions",
  external_tools: "External Tools",
  skills: "Skills",
  nodes: "Nodes",
  voice: "Voice",
  memory: "Memory",
  library: "Library",
  other: "Other",
};

export const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

export const resolveCategoryId = (value?: string) => {
  const normalized = value?.trim().toLowerCase();
  return normalized || "other";
};

const parseSemverCore = (value: string): { major: number; minor: number; patch: number; pre: string } | null => {
  const normalized = value.trim().replace(/^v/i, "");
  if (!normalized) {
    return null;
  }
  const [core, pre = ""] = normalized.split("-", 2);
  const segments = core.split(".");
  if (segments.length === 0 || segments.length > 3) {
    return null;
  }
  const numbers = [0, 0, 0];
  for (let index = 0; index < segments.length; index += 1) {
    const segment = segments[index];
    if (!/^\d+$/.test(segment)) {
      return null;
    }
    numbers[index] = Number.parseInt(segment, 10);
  }
  return {
    major: numbers[0],
    minor: numbers[1],
    patch: numbers[2],
    pre: pre.trim().toLowerCase(),
  };
};

const compareSemverLike = (leftRaw: string, rightRaw: string): number | null => {
  const left = parseSemverCore(leftRaw);
  const right = parseSemverCore(rightRaw);
  if (!left || !right) {
    return null;
  }
  if (left.major !== right.major) {
    return left.major < right.major ? -1 : 1;
  }
  if (left.minor !== right.minor) {
    return left.minor < right.minor ? -1 : 1;
  }
  if (left.patch !== right.patch) {
    return left.patch < right.patch ? -1 : 1;
  }
  if (left.pre === right.pre) {
    return 0;
  }
  if (left.pre === "") {
    return 1;
  }
  if (right.pre === "") {
    return -1;
  }
  return left.pre < right.pre ? -1 : 1;
};

export const isSkillVersionOutdated = (currentVersion: string, latestVersion: string) => {
  const current = currentVersion.trim();
  const latest = latestVersion.trim();
  if (!current || !latest) {
    return false;
  }
  const compared = compareSemverLike(current, latest);
  if (compared === null) {
    return current.toLowerCase() !== latest.toLowerCase();
  }
  return compared < 0;
};

export const formatByteSize = (value?: number) => {
  if (!value || !Number.isFinite(value) || value <= 0) {
    return "";
  }
  const units = ["B", "KB", "MB", "GB"];
  let amount = value;
  let unitIndex = 0;
  while (amount >= 1024 && unitIndex < units.length - 1) {
    amount /= 1024;
    unitIndex += 1;
  }
  const fixed = amount >= 100 || unitIndex === 0 ? amount.toFixed(0) : amount.toFixed(1);
  return `${fixed} ${units[unitIndex]}`;
};

export const formatRuntimeInstallSpec = (spec: SkillRuntimeInstallSpec) => {
  const head = spec.label?.trim() || spec.id?.trim() || spec.kind?.trim() || "install";
  const details: string[] = [];
  if (spec.package?.trim()) {
    details.push(`package=${spec.package.trim()}`);
  }
  if (spec.formula?.trim()) {
    details.push(`formula=${spec.formula.trim()}`);
  }
  if (spec.module?.trim()) {
    details.push(`module=${spec.module.trim()}`);
  }
  if (spec.tap?.trim()) {
    details.push(`tap=${spec.tap.trim()}`);
  }
  if (spec.bins && spec.bins.length > 0) {
    details.push(`bins=${spec.bins.join(", ")}`);
  }
  if (details.length === 0) {
    return head;
  }
  return `${head} (${details.join(" · ")})`;
};

export const stripMarkdownFrontmatter = (content: string) => {
  const normalized = content.replace(/\r\n/g, "\n").replace(/^\uFEFF/, "");
  if (!normalized.startsWith("---\n")) {
    return content.trim();
  }
  const lines = normalized.split("\n");
  let endIndex = -1;
  for (let i = 1; i < lines.length; i += 1) {
    const line = lines[i].trim();
    if (line === "---" || line === "...") {
      endIndex = i;
      break;
    }
  }
  if (endIndex <= 0) {
    return content.trim();
  }
  return lines.slice(endIndex + 1).join("\n").trim();
};

const unwrapErrorMessage = (error: unknown) => {
  const readMessageField = (value: unknown): string | null => {
    if (!value || typeof value !== "object") {
      return null;
    }
    const candidate = (value as { message?: unknown }).message;
    return typeof candidate === "string" ? candidate : null;
  };

  if (error instanceof Error) {
    const trimmed = error.message.trim();
    if (trimmed !== "") {
      return trimmed;
    }
  }

  const directMessage = readMessageField(error);
  if (directMessage && directMessage.trim() !== "") {
    return directMessage.trim();
  }

  const fallback = String(error ?? "").trim();
  if (!fallback.startsWith("{") || !fallback.endsWith("}")) {
    return fallback;
  }
  try {
    const parsed = JSON.parse(fallback) as unknown;
    const parsedMessage = readMessageField(parsed);
    if (parsedMessage && parsedMessage.trim() !== "") {
      return parsedMessage.trim();
    }
  } catch {
    return fallback;
  }
  return fallback;
};

export const resolveSkillsActionErrorInfo = (error: unknown): SkillsActionErrorInfo => {
  const message = unwrapErrorMessage(error);
  const normalized = message.toLowerCase();
  const rateLimited = normalized.includes("rate limit exceeded") || normalized.includes("too many requests");
  const requiresForce =
    normalized.includes("use --force to install suspicious skills in non-interactive mode") ||
    (normalized.includes("flagged as suspicious") && normalized.includes("--force"));
  return {
    message,
    rateLimited,
    requiresForce,
  };
};

export const resolveToolStatus = (tool: ToolItem) => {
  if (!tool.available) {
    return { allowed: false, reasonKey: "settings.tools.reason.unavailable" };
  }
  return { allowed: true };
};

const mapGatewayToolRequirement = (requirement: unknown): ToolRequirementStatus | null => {
  if (!requirement || typeof requirement !== "object") {
    return null;
  }
  const source = requirement as GatewayToolRequirement;
  const id = typeof source.id === "string" ? source.id.trim() : "";
  if (!id) {
    return null;
  }
  return {
    id,
    name: typeof source.name === "string" && source.name.trim() !== "" ? source.name.trim() : id,
    available: source.available !== false,
    reason: typeof source.reason === "string" ? source.reason.trim() : "",
  };
};

export const mapGatewayTool = (tool: GatewayToolSpec): ToolItem | null => {
  const id = (tool.id || tool.name || "").trim();
  if (!id) {
    return null;
  }
  const label = (tool.name || id).trim();
  return {
    id,
    source: "gateway",
    available: tool.enabled !== false,
    requirements: Array.isArray(tool.requirements)
      ? tool.requirements
          .map((requirement) => mapGatewayToolRequirement(requirement))
          .filter((requirement): requirement is ToolRequirementStatus => Boolean(requirement))
      : [],
    category: tool.category,
    riskLevel: tool.riskLevel,
    schemaJson: tool.schemaJson,
    methods: Array.isArray(tool.methods) ? tool.methods : undefined,
    requiresSandbox: tool.requiresSandbox === true,
    requiresApproval: tool.requiresApproval === true,
    labelKey: `settings.tools.builtin.${id}.name`,
    label,
    descriptionKey: `settings.tools.builtin.${id}.description`,
    description: tool.description || "Gateway tool.",
  };
};
