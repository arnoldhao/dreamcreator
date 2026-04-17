import type { ExternalTool } from "@/shared/store/externalTools";

export type SetupStepId = "general" | "ai" | "dependencies";
export type SetupSeverity = "blocking" | "recommended";
export type SetupIssueCode =
  | "general.language"
  | "ai.gateway"
  | "ai.providers"
  | "ai.model"
  | "dependency.productMode"
  | "dependency.downloadTools"
  | "dependency.fullTools";

export interface SetupIssue {
  code: SetupIssueCode;
  severity: SetupSeverity;
  step: SetupStepId;
  meta?: Record<string, string | number | boolean>;
}

export interface ProviderPreset {
  id: string;
  label: string;
  endpoint: string;
  type: "openai" | "anthropic";
  compatibility: "openai" | "anthropic" | "deepseek" | "openrouter" | "google";
}

export const SETUP_STEP_ORDER: SetupStepId[] = ["general", "ai", "dependencies"];

export const PROVIDER_PRESETS: ProviderPreset[] = [
  { id: "deepseek", label: "DeepSeek", endpoint: "https://api.deepseek.com", type: "openai", compatibility: "deepseek" },
  { id: "openrouter", label: "OpenRouter", endpoint: "https://openrouter.ai/api/v1", type: "openai", compatibility: "openrouter" },
  { id: "openai", label: "OpenAI", endpoint: "https://api.openai.com/v1", type: "openai", compatibility: "openai" },
  { id: "anthropic", label: "Anthropic", endpoint: "https://api.anthropic.com/v1", type: "anthropic", compatibility: "anthropic" },
  { id: "google", label: "Google Gemini", endpoint: "https://generativelanguage.googleapis.com/v1beta/openai", type: "openai", compatibility: "google" },
  { id: "moonshotai", label: "Moonshot AI", endpoint: "https://api.moonshot.ai/v1", type: "openai", compatibility: "openai" },
  { id: "zai", label: "Z.AI", endpoint: "https://api.z.ai/api/paas/v4", type: "openai", compatibility: "openai" },
];

export const DOWNLOAD_MODE_REQUIRED_TOOLS = ["yt-dlp", "ffmpeg", "bun"] as const;
export const FULL_MODE_REQUIRED_TOOLS = ["yt-dlp", "ffmpeg", "bun", "clawhub"] as const;

export const isToolInstalled = (tool: ExternalTool | null | undefined) =>
  Boolean(tool && String(tool.status ?? "").trim().toLowerCase() === "installed" && tool.execPath?.trim());
