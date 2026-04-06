export type WebSearchType = "api" | "external_tools";
export type WebFetchType = "playwright" | "builtin";

export type WebSearchFormState = {
  type: WebSearchType;
  provider: string;
  apiKey: string;
  maxResults: string;
  country: string;
  searchLang: string;
  uiLang: string;
  freshness: string;
  timeoutSeconds: string;
  cacheTtlMinutes: string;
};

export type WebFetchFormState = {
  type: WebFetchType;
  playwrightMarkdown: boolean;
  acceptMarkdown: boolean;
  enableUserAgent: boolean;
  userAgent: string;
  acceptLanguage: string;
  timeoutSeconds: string;
  maxChars: string;
  maxRedirects: string;
  retryMax: string;
  headersJson: string;
};

export type BrowserControlFormState = {
  enabled: boolean;
  evaluateEnabled: boolean;
  headless: boolean;
  noSandbox: boolean;
  snapshotDefaultMode: string;
  ssrfDangerouslyAllowPrivateNetwork: boolean;
  ssrfAllowedHostnamesJson: string;
  ssrfHostnameAllowlistJson: string;
  extraArgsJson: string;
};

export type WebSearchProviderOption = {
  id: string;
  label: string;
  apiBaseUrl?: string;
  openRouterBaseUrl?: string;
};

export const DEFAULT_WEB_SEARCH_PROVIDERS: WebSearchProviderOption[] = [
  {
    id: "brave",
    label: "Brave",
    apiBaseUrl: "https://api.search.brave.com/res/v1/web/search",
  },
  {
    id: "perplexity",
    label: "Perplexity",
    apiBaseUrl: "https://api.perplexity.ai",
    openRouterBaseUrl: "https://openrouter.ai/api/v1",
  },
  {
    id: "grok",
    label: "Grok",
    apiBaseUrl: "https://api.x.ai/v1/responses",
  },
  {
    id: "tavily",
    label: "Tavily",
    apiBaseUrl: "https://api.tavily.com/search",
  },
];

const WEB_SEARCH_API_KEY_PLACEHOLDERS: Record<string, string> = {
  brave: "BRAVE_API_KEY",
  tavily: "TAVILY_API_KEY",
  perplexity: "PERPLEXITY_API_KEY",
  grok: "XAI_API_KEY",
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

export const normalizeWebSearchType = (value: string): WebSearchType => {
  const normalized = value.trim().toLowerCase();
  if (normalized === "external_tools" || normalized === "external-tools" || normalized === "external tools") {
    return "external_tools";
  }
  if (normalized === "api") {
    return "api";
  }
  return "api";
};

export const normalizeWebFetchType = (value: string): WebFetchType => {
  const normalized = value.trim().toLowerCase();
  if (normalized === "playwright") {
    return "playwright";
  }
  if (normalized === "builtin") {
    return "builtin";
  }
  return "builtin";
};

export const parseNumberInput = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined;
  }
  return parsed;
};

export const parseNonNegativeNumberInput = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return undefined;
  }
  return parsed;
};

export const resolveWebSearchAPIKeyPlaceholder = (provider: string) => {
  const normalized = provider.trim().toLowerCase();
  if (!normalized) {
    return "API_KEY";
  }
  const knownPlaceholder = WEB_SEARCH_API_KEY_PLACEHOLDERS[normalized];
  if (knownPlaceholder) {
    return knownPlaceholder;
  }
  const token = normalized
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "")
    .toUpperCase();
  return token ? `${token}_API_KEY` : "API_KEY";
};

export const readStringValue = (source: Record<string, unknown> | undefined, key: string, fallback = "") => {
  const value = source?.[key];
  if (typeof value === "string") {
    return value;
  }
  return fallback;
};

export const readBoolValue = (source: Record<string, unknown> | undefined, key: string, fallback = false) => {
  const value = source?.[key];
  if (typeof value === "boolean") {
    return value;
  }
  return fallback;
};

export const readNumberValue = (source: Record<string, unknown> | undefined, key: string) => {
  const value = source?.[key];
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  return "";
};

export const readObjectValue = (source: Record<string, unknown> | undefined, key: string) => {
  const value = source?.[key];
  if (isRecord(value)) {
    return value as Record<string, unknown>;
  }
  return undefined;
};

export const stringifyObjectValue = (source: Record<string, unknown> | undefined) => {
  if (!source) {
    return "";
  }
  const entries = Object.entries(source).sort(([left], [right]) => left.localeCompare(right));
  if (entries.length === 0) {
    return "";
  }
  return JSON.stringify(Object.fromEntries(entries));
};

export const stringifyStringArrayValue = (source: unknown) => {
  if (!Array.isArray(source)) {
    return "";
  }
  const values = source.filter((value): value is string => typeof value === "string")
    .map((value) => value.trim())
    .filter((value) => value !== "");
  if (values.length === 0) {
    return "";
  }
  return JSON.stringify(values);
};

export const parseObjectJSON = (raw: string) => {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { value: undefined as Record<string, unknown> | undefined, error: null as string | null };
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (!isRecord(parsed)) {
      return { value: undefined, error: "invalid-object" };
    }
    return { value: parsed as Record<string, unknown>, error: null };
  } catch {
    return { value: undefined, error: "invalid-json" };
  }
};

export const parseStringArrayJSON = (raw: string) => {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { value: undefined as string[] | undefined, error: null as string | null };
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) {
      return { value: undefined, error: "invalid-array" };
    }
    const values = parsed
      .filter((value): value is string => typeof value === "string")
      .map((value) => value.trim())
      .filter((value) => value !== "");
    return { value: values, error: null };
  } catch {
    return { value: undefined, error: "invalid-json" };
  }
};

export const readWebSearchProviderApiKey = (
  source: Record<string, unknown> | undefined,
  provider: string,
  fallback = ""
) => {
  const providerID = provider.trim();
  if (!providerID) {
    return fallback;
  }
  const providers = isRecord(source?.providers) ? (source?.providers as Record<string, unknown>) : undefined;
  const providerConfig = providers && isRecord(providers[providerID])
    ? (providers[providerID] as Record<string, unknown>)
    : undefined;
  return readStringValue(providerConfig, "apiKey", fallback);
};

export const readWebSearchProviderApiKeys = (source: Record<string, unknown> | undefined) => {
  const result: Record<string, string> = {};
  const providers = isRecord(source?.providers) ? (source?.providers as Record<string, unknown>) : undefined;
  if (providers) {
    Object.entries(providers).forEach(([id, value]) => {
      const providerID = id.trim();
      if (!providerID || !isRecord(value)) {
        return;
      }
      const apiKey = readStringValue(value, "apiKey", "");
      if (apiKey) {
        result[providerID] = apiKey;
      }
    });
  }
  const providerID = readStringValue(source, "provider", "").trim();
  const fallbackApiKey = readStringValue(source, "apiKey", "");
  if (providerID && fallbackApiKey && result[providerID] === undefined) {
    result[providerID] = fallbackApiKey;
  }
  return result;
};

export const serializeWebSearchProviderApiKeys = (source: Record<string, string>) => {
  const sorted = Object.entries(source)
    .map(([providerID, apiKey]) => [providerID.trim(), apiKey] as const)
    .filter(([providerID]) => providerID !== "")
    .sort(([left], [right]) => left.localeCompare(right));
  return JSON.stringify(sorted);
};
