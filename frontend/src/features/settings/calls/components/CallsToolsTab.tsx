import * as React from "react";
import { Call } from "@wailsio/runtime";
import {
  ArrowRight,
  Ban,
  Check,
  HelpCircle,
} from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useSettings, useUpdateSettings } from "@/shared/query/settings";
import { useGatewayTools } from "@/shared/query/tools";
import type { GatewayToolMethodSpec } from "@/shared/store/gatewayTools";
import { CallsCard } from "./CallsCard";
import { ToolMethodIOPanel } from "./ToolMethodIOPanel";
import {
  ToolConfigCard,
  ToolConfigEmptyState,
  ToolConfigTabPanel,
  ToolContentTabs,
  type ToolDependencyStatus,
  ToolDetailLayout,
  ToolIOTabPanel,
  ToolOverviewCard,
} from "./tool-detail-layout";
import type { ToolItem } from "../types";
import {
  CATEGORY_ORDER,
  baseToolIds,
  isRecord,
  mapGatewayTool,
  normalizeToolId,
  resolveCategoryId,
  resolveToolStatus,
} from "../utils/calls-utils";
import {
  buildToolInputExample,
  buildToolOutputExample,
  findGatewayToolMethod,
  parseGatewayToolMethods,
  toPrettyJSON,
} from "../utils/gateway-tool-utils";
import {
  DEFAULT_WEB_SEARCH_PROVIDERS,
  normalizePreferredBrowser,
  normalizeWebSearchType,
  parseNumberInput,
  parseStringArrayJSON,
  readBoolValue,
  readNumberValue,
  readObjectValue,
  readStringValue,
  readWebSearchProviderApiKey,
  readWebSearchProviderApiKeys,
  resolveWebSearchAPIKeyPlaceholder,
  serializeWebSearchProviderApiKeys,
  stringifyStringArrayValue,
  type BrowserControlFormState,
  type WebFetchFormState,
  type WebSearchFormState,
  type WebSearchProviderOption,
} from "../utils/web-tool-settings-utils";

type RuntimeBrowserCandidate = {
  id: string;
  label: string;
  available: boolean;
  execPath: string;
  error: string;
};

type RuntimeDetectionRow = {
  label: string;
  value: string;
  badge?: "not_installed" | "not_detected";
};

const BROWSER_LABELS: Record<string, string> = {
  chrome: "Chrome",
  chromium: "Chromium",
  edge: "Edge",
  brave: "Brave",
};

const normalizeRuntimeBrowserCandidates = (value: unknown): RuntimeBrowserCandidate[] => {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!isRecord(item)) {
      return [];
    }
    const id = readStringValue(item, "id", "").trim().toLowerCase();
    const fallbackLabel = id ? (BROWSER_LABELS[id] ?? id) : "Browser";
    return [{
      id,
      label: readStringValue(item, "label", fallbackLabel).trim() || fallbackLabel,
      available: readBoolValue(item, "available", false),
      execPath: readStringValue(item, "execPath", "").trim(),
      error: readStringValue(item, "error", "").trim(),
    }];
  });
};

export function CallsToolsTab() {
  const { t } = useI18n();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();

  const gatewayToolsQuery = useGatewayTools();
  const gatewayTools = React.useMemo(
    () =>
      (gatewayToolsQuery.data ?? [])
        .map((tool) => mapGatewayTool(tool))
        .filter((tool): tool is ToolItem => Boolean(tool)),
    [gatewayToolsQuery.data]
  );
  const toolItems = React.useMemo(() => [...gatewayTools], [gatewayTools]);
  const toolLookup = React.useMemo(() => {
    const map = new Map<string, ToolItem>();
    toolItems.forEach((tool) => {
      map.set(normalizeToolId(tool.id), tool);
    });
    return map;
  }, [toolItems]);
  const resolveToolLabel = React.useCallback(
    (toolID: string) => {
      const item = toolLookup.get(normalizeToolId(toolID));
      if (!item) {
        return toolID;
      }
      return t(item.labelKey);
    },
    [toolLookup, t]
  );
  const resolveCategoryLabel = React.useCallback(
    (category: string) => t(`settings.tools.category.${category}`),
    [t]
  );
  const resolveRequirementName = React.useCallback(
    (requirementID: string, fallbackName: string) => {
      switch (requirementID) {
        case "browser.cdp_runtime":
        case "web_fetch.local_browser":
          return t("settings.tools.requirements.localCDPBrowser");
        default:
          return fallbackName;
      }
    },
    [t]
  );
  const resolveRequirementReason = React.useCallback(
    (reason: string) => {
      const normalized = reason.trim().toLowerCase();
      switch (normalized) {
        case "browser executable not found":
          return t("settings.tools.runtimeDetection.notInstalled");
        case "no supported browser detected":
          return t("settings.tools.runtimeDetection.noneDetected");
        case "browser process exited":
          return t("settings.tools.reason.browserProcessExited");
        default:
          return reason || t("settings.tools.reason.unavailable");
      }
    },
    [t]
  );
  const resolveToolDependencies = React.useCallback(
    (tool: ToolItem) => {
      const dependencies: ToolDependencyStatus[] =
        !tool.requirements || tool.requirements.length === 0
          ? []
          : tool.requirements.map((requirement) => {
              const fallbackName = requirement.name || requirement.id;
              const fallbackReason = resolveRequirementReason(requirement.reason || "");
              return {
                id: requirement.id,
                name: resolveRequirementName(requirement.id, fallbackName),
                ok: requirement.available,
                reason: fallbackReason,
              };
            });
      const riskLevel = (tool.riskLevel ?? "").trim().toLowerCase();
      const needsApproval = tool.requiresApproval === true || riskLevel === "high";
      const needsSandbox = tool.requiresSandbox === true || riskLevel === "high";
      const permissionBadges: string[] = [];
      if (needsApproval) {
        permissionBadges.push(t("settings.tools.detail.permissions.badges.approval"));
      }
      if (needsSandbox) {
        permissionBadges.push(t("settings.tools.detail.permissions.badges.sandbox"));
      }
      if (permissionBadges.length === 0) {
        permissionBadges.push(t("settings.tools.detail.permissions.badges.none"));
      }
      dependencies.push({
        id: "__permission__",
        name: t("settings.tools.detail.permissions.label"),
        ok: true,
        reason: "",
        badges: permissionBadges,
      });
      return dependencies;
    },
    [resolveRequirementName, resolveRequirementReason, t]
  );
  const renderToolStatusBadge = React.useCallback(
    (status: ReturnType<typeof resolveToolStatus> | null) => {
      if (!status) {
        return null;
      }
      const label = status.allowed
        ? t("settings.tools.status.allowed")
        : t("settings.tools.status.blocked");
      const Icon = status.allowed ? Check : Ban;
      const variant = status.allowed ? "secondary" : "outline";
      return (
        <Badge
          variant={variant}
          className="h-5 w-5 justify-center p-0"
          title={label}
          aria-label={label}
        >
          <Icon className="h-3 w-3" />
          <span className="sr-only">{label}</span>
        </Badge>
      );
    },
    [t]
  );
  const filteredTools = React.useMemo(() => {
    const base = toolItems.filter((tool) => baseToolIds.includes(normalizeToolId(tool.id)));
    return base.length > 0 ? base : toolItems;
  }, [toolItems]);
  const groupedTools = React.useMemo(() => {
    const groups = new Map<string, ToolItem[]>();
    filteredTools.forEach((tool) => {
      const category = resolveCategoryId(tool.category);
      const list = groups.get(category) ?? [];
      list.push(tool);
      groups.set(category, list);
    });
    const entries = Array.from(groups.entries()).map(([category, items]) => {
      const sorted = [...items].sort((left, right) =>
        resolveToolLabel(left.id).localeCompare(resolveToolLabel(right.id))
      );
      return { id: category, label: resolveCategoryLabel(category), tools: sorted };
    });
    const order = new Map<string, number>();
    CATEGORY_ORDER.forEach((key, index) => {
      order.set(key, index);
    });
    entries.sort((left, right) => {
      const leftIndex = order.get(left.id);
      const rightIndex = order.get(right.id);
      if (leftIndex !== undefined && rightIndex !== undefined) {
        return leftIndex - rightIndex;
      }
      if (leftIndex !== undefined) {
        return -1;
      }
      if (rightIndex !== undefined) {
        return 1;
      }
      return left.label.localeCompare(right.label);
    });
    return entries;
  }, [filteredTools, resolveCategoryLabel, resolveToolLabel]);

  const [selectedToolId, setSelectedToolId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (selectedToolId && filteredTools.some((tool) => tool.id === selectedToolId)) {
      return;
    }
    if (filteredTools.length > 0) {
      setSelectedToolId(filteredTools[0].id);
    }
  }, [filteredTools, selectedToolId]);

  const selectedTool = filteredTools.find((tool) => tool.id === selectedToolId) ?? null;
  const [gatewayPanelTab, setGatewayPanelTab] = React.useState<"config" | "io">("config");
  const [gatewayActionDraft, setGatewayActionDraft] = React.useState<string>("");
  const [gatewayToolEnablePending, setGatewayToolEnablePending] = React.useState<boolean>(false);
  const gatewayMethods = React.useMemo(() => {
    if (!selectedTool) {
      return [] as GatewayToolMethodSpec[];
    }
    const parsed = parseGatewayToolMethods(selectedTool.methods, selectedTool.schemaJson);
    if (parsed.length > 0) {
      return parsed;
    }
    const fallbackName = selectedTool.id?.trim();
    if (!fallbackName) {
      return [] as GatewayToolMethodSpec[];
    }
    return [{ name: fallbackName }];
  }, [selectedTool?.id, selectedTool?.methods, selectedTool?.schemaJson]);
  const gatewayActions = React.useMemo(() => gatewayMethods.map((method) => method.name), [gatewayMethods]);
  const activeGatewayAction = React.useMemo(() => {
    if (gatewayActionDraft && gatewayActions.includes(gatewayActionDraft)) {
      return gatewayActionDraft;
    }
    return gatewayActions[0] ?? "";
  }, [gatewayActionDraft, gatewayActions]);
  const activeGatewayMethod = React.useMemo(
    () => findGatewayToolMethod(gatewayMethods, activeGatewayAction),
    [activeGatewayAction, gatewayMethods]
  );

  React.useEffect(() => {
    setGatewayPanelTab("config");
    setGatewayActionDraft("");
  }, [selectedTool?.id]);

  React.useEffect(() => {
    if (gatewayActions.length === 0) {
      if (gatewayActionDraft !== "") {
        setGatewayActionDraft("");
      }
      return;
    }
    if (!gatewayActionDraft || !gatewayActions.includes(gatewayActionDraft)) {
      setGatewayActionDraft(gatewayActions[0]);
    }
  }, [gatewayActionDraft, gatewayActions]);

  const handleToggleToolEnabled = React.useCallback(
    async (tool: ToolItem, enabled: boolean) => {
      if (!tool?.id) {
        return;
      }
      setGatewayToolEnablePending(true);
      try {
        await Call.ByName("dreamcreator/internal/presentation/wails.ToolsHandler.EnableTool", {
          id: tool.id,
          enabled,
        });
        await gatewayToolsQuery.refetch();
      } catch (error) {
        messageBus.publishToast({
          intent: "warning",
          title: t("settings.tools.status.updateFailed"),
          description: String(error),
        });
      } finally {
        setGatewayToolEnablePending(false);
      }
    },
    [gatewayToolsQuery, t]
  );

  const callsToolsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.tools;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.tools]);
  const toolsConfig = React.useMemo(() => ({ ...(callsToolsRaw ?? {}) }), [callsToolsRaw]);
  const webConfig = React.useMemo(
    () => (isRecord(toolsConfig.web) ? (toolsConfig.web as Record<string, unknown>) : undefined),
    [toolsConfig.web]
  );
  const webSearchConfig = React.useMemo(() => {
    return isRecord(webConfig?.search) ? (webConfig?.search as Record<string, unknown>) : undefined;
  }, [webConfig]);
  const webFetchConfig = React.useMemo(() => {
    return isRecord(toolsConfig.web_fetch) ? (toolsConfig.web_fetch as Record<string, unknown>) : undefined;
  }, [toolsConfig.web_fetch]);
  const browserConfig = React.useMemo(
    () => (isRecord(toolsConfig.browser) ? (toolsConfig.browser as Record<string, unknown>) : undefined),
    [toolsConfig.browser]
  );
  const initialWebSearchType = React.useMemo(
    () => {
      const configuredType = normalizeWebSearchType(readStringValue(webSearchConfig, "type", ""));
      if (configuredType === "api" || configuredType === "external_tools") {
        return configuredType;
      }
      if (readBoolValue(webSearchConfig, "enabled", false)) {
        return "api";
      }
      return "api";
    },
    [webSearchConfig]
  );
  const initialWebSearchProvider = React.useMemo(
    () => readStringValue(webSearchConfig, "provider", "brave"),
    [webSearchConfig]
  );
  const initialWebSearchProviderApiKeys = React.useMemo(
    () => readWebSearchProviderApiKeys(webSearchConfig),
    [webSearchConfig]
  );
  const initialWebSearchAPIKey = React.useMemo(
    () => {
      const providerID = initialWebSearchProvider.trim();
      if (providerID && initialWebSearchProviderApiKeys[providerID] !== undefined) {
        return initialWebSearchProviderApiKeys[providerID];
      }
      return readWebSearchProviderApiKey(webSearchConfig, initialWebSearchProvider, "");
    },
    [initialWebSearchProvider, initialWebSearchProviderApiKeys, webSearchConfig]
  );
  const initialWebSearchForm = React.useMemo<WebSearchFormState>(
    () => {
      return {
        type: initialWebSearchType,
        provider: initialWebSearchProvider,
        apiKey: initialWebSearchAPIKey,
        maxResults: readNumberValue(webSearchConfig, "maxResults"),
        country: readStringValue(webSearchConfig, "country", ""),
        searchLang: readStringValue(webSearchConfig, "search_lang", ""),
        uiLang: readStringValue(webSearchConfig, "ui_lang", ""),
        freshness: readStringValue(webSearchConfig, "freshness", ""),
        timeoutSeconds: readNumberValue(webSearchConfig, "timeoutSeconds"),
        cacheTtlMinutes: readNumberValue(webSearchConfig, "cacheTtlMinutes"),
      };
    },
    [initialWebSearchAPIKey, initialWebSearchProvider, initialWebSearchType, webSearchConfig]
  );
  const [webSearchForm, setWebSearchForm] = React.useState<WebSearchFormState>(initialWebSearchForm);
  const [webSearchProviderAPIKeys, setWebSearchProviderAPIKeys] = React.useState<Record<string, string>>(
    initialWebSearchProviderApiKeys
  );
  const skipNextWebSearchBlurSaveRef = React.useRef(false);

  const initialWebFetchForm = React.useMemo<WebFetchFormState>(() => ({
    headless: readBoolValue(webFetchConfig, "headless", true),
    preferredBrowser: normalizePreferredBrowser(readStringValue(webFetchConfig, "preferredBrowser", "chrome")),
    timeoutSeconds: readNumberValue(webFetchConfig, "timeoutSeconds"),
    maxChars: readNumberValue(webFetchConfig, "maxChars"),
  }), [webFetchConfig]);
  const [webFetchForm, setWebFetchForm] = React.useState<WebFetchFormState>(initialWebFetchForm);
  const skipNextWebFetchBlurSaveRef = React.useRef(false);
  const initialBrowserForm = React.useMemo<BrowserControlFormState>(() => {
    const ssrfPolicy = readObjectValue(browserConfig, "ssrfPolicy");
    return {
      enabled: readBoolValue(browserConfig, "enabled", true),
      headless: readBoolValue(browserConfig, "headless", true),
      preferredBrowser: normalizePreferredBrowser(readStringValue(browserConfig, "preferredBrowser", "chrome")),
      ssrfDangerouslyAllowPrivateNetwork: readBoolValue(
        ssrfPolicy,
        "dangerouslyAllowPrivateNetwork",
        false
      ),
      ssrfAllowedHostnamesJson: stringifyStringArrayValue(ssrfPolicy?.allowedHostnames),
      ssrfHostnameAllowlistJson: stringifyStringArrayValue(ssrfPolicy?.hostnameAllowlist),
    };
  }, [browserConfig]);
  const [browserForm, setBrowserForm] = React.useState<BrowserControlFormState>(initialBrowserForm);
  const skipNextBrowserBlurSaveRef = React.useRef(false);

  React.useEffect(() => {
    setWebSearchForm(initialWebSearchForm);
  }, [initialWebSearchForm]);

  React.useEffect(() => {
    setWebSearchProviderAPIKeys(initialWebSearchProviderApiKeys);
  }, [initialWebSearchProviderApiKeys]);

  React.useEffect(() => {
    setWebFetchForm(initialWebFetchForm);
  }, [initialWebFetchForm]);

  React.useEffect(() => {
    setBrowserForm(initialBrowserForm);
  }, [initialBrowserForm]);

  const resolveWebSearchDraftApiKey = React.useCallback(
    (provider: string) => {
      const providerID = provider.trim();
      if (!providerID) {
        return "";
      }
      if (webSearchProviderAPIKeys[providerID] !== undefined) {
        return webSearchProviderAPIKeys[providerID];
      }
      return "";
    },
    [webSearchProviderAPIKeys]
  );

  const updateWebSearchProviderApiKey = React.useCallback((provider: string, apiKey: string) => {
    const providerID = provider.trim();
    if (!providerID) {
      return;
    }
    setWebSearchProviderAPIKeys((prev) => {
      if (prev[providerID] === apiKey) {
        return prev;
      }
      return { ...prev, [providerID]: apiKey };
    });
  }, []);

  const webSearchProviders = React.useMemo<WebSearchProviderOption[]>(() => {
    const rawProviders = isRecord(webSearchConfig?.providers)
      ? (webSearchConfig?.providers as Record<string, unknown>)
      : undefined;
    const options: WebSearchProviderOption[] = [];
    if (rawProviders) {
      Object.entries(rawProviders).forEach(([id, value]) => {
        if (!id.trim()) {
          return;
        }
        if (isRecord(value)) {
          options.push({
            id,
            label: readStringValue(value, "label", id),
            apiBaseUrl: readStringValue(value, "apiBaseUrl", ""),
            openRouterBaseUrl: readStringValue(value, "openRouterBaseUrl", ""),
          });
          return;
        }
        options.push({ id, label: id });
      });
    }
    if (options.length === 0) {
      options.push(...DEFAULT_WEB_SEARCH_PROVIDERS);
    }
    const currentProvider = webSearchForm.provider.trim();
    if (currentProvider && !options.some((option) => option.id === currentProvider)) {
      options.push({ id: currentProvider, label: currentProvider });
    }
    return options.sort((left, right) => left.label.localeCompare(right.label));
  }, [webSearchConfig?.providers, webSearchForm.provider]);

  const selectedWebSearchProvider = React.useMemo(
    () => webSearchProviders.find((option) => option.id === webSearchForm.provider) ?? webSearchProviders[0],
    [webSearchForm.provider, webSearchProviders]
  );
  const webSearchAPIKeyPlaceholder = React.useMemo(
    () => resolveWebSearchAPIKeyPlaceholder(webSearchForm.provider),
    [webSearchForm.provider]
  );

  React.useEffect(() => {
    if (!webSearchForm.provider && webSearchProviders.length > 0) {
      const nextProvider = webSearchProviders[0].id;
      setWebSearchForm((prev) => ({
        ...prev,
        provider: nextProvider,
        apiKey: resolveWebSearchDraftApiKey(nextProvider),
      }));
    }
  }, [resolveWebSearchDraftApiKey, webSearchForm.provider, webSearchProviders]);

  const isWebSearchDirty = React.useMemo(
    () =>
      JSON.stringify(webSearchForm) !== JSON.stringify(initialWebSearchForm) ||
      serializeWebSearchProviderApiKeys(webSearchProviderAPIKeys) !==
        serializeWebSearchProviderApiKeys(initialWebSearchProviderApiKeys),
    [webSearchForm, initialWebSearchForm, webSearchProviderAPIKeys, initialWebSearchProviderApiKeys]
  );
  const isWebFetchDirty = React.useMemo(
    () => JSON.stringify(webFetchForm) !== JSON.stringify(initialWebFetchForm),
    [initialWebFetchForm, webFetchForm]
  );
  const isBrowserDirty = React.useMemo(
    () => JSON.stringify(browserForm) !== JSON.stringify(initialBrowserForm),
    [browserForm, initialBrowserForm]
  );
  const webSearchDisabled = settingsQuery.isLoading || updateSettings.isPending;
  const webFetchDisabled = settingsQuery.isLoading || updateSettings.isPending;
  const browserDisabled = settingsQuery.isLoading || updateSettings.isPending;

  const handleResetWebSearch = React.useCallback(() => {
    setWebSearchForm(initialWebSearchForm);
    setWebSearchProviderAPIKeys(initialWebSearchProviderApiKeys);
    skipNextWebSearchBlurSaveRef.current = false;
  }, [initialWebSearchForm, initialWebSearchProviderApiKeys]);

  const handleResetWebFetch = React.useCallback(() => {
    setWebFetchForm(initialWebFetchForm);
    skipNextWebFetchBlurSaveRef.current = false;
  }, [initialWebFetchForm]);

  const handleResetBrowser = React.useCallback(() => {
    setBrowserForm(initialBrowserForm);
    skipNextBrowserBlurSaveRef.current = false;
  }, [initialBrowserForm]);

  const handleSaveWebSearch = React.useCallback((
    formState?: WebSearchFormState,
    providerAPIKeysState?: Record<string, string>
  ) => {
    const currentForm = formState ?? webSearchForm;
    const currentProviderAPIKeys = providerAPIKeysState ?? webSearchProviderAPIKeys;
    const nextToolsConfig: Record<string, unknown> = { ...toolsConfig };
    const nextWeb = isRecord(nextToolsConfig.web)
      ? { ...(nextToolsConfig.web as Record<string, unknown>) }
      : {};
    const nextSearch = isRecord(nextWeb.search)
      ? { ...(nextWeb.search as Record<string, unknown>) }
      : {};
    const setOrDelete = (key: string, value: unknown) => {
      if (value === undefined || value === "") {
        delete nextSearch[key];
        return;
      }
      nextSearch[key] = value;
    };
    const normalizedType = normalizeWebSearchType(currentForm.type);
    setOrDelete("type", normalizedType);
    if (normalizedType === "api") {
      const providerID = currentForm.provider.trim();
      const apiKey = currentForm.apiKey.trim();
      setOrDelete("provider", providerID);
      setOrDelete("apiKey", apiKey);
      const providerApiKeys = { ...currentProviderAPIKeys };
      if (providerID) {
        providerApiKeys[providerID] = currentForm.apiKey;
      }
      const providersRaw = isRecord(nextSearch.providers)
        ? { ...(nextSearch.providers as Record<string, unknown>) }
        : {};
      Object.entries(providerApiKeys).forEach(([rawProviderID, rawApiKey]) => {
        const normalizedProviderID = rawProviderID.trim();
        if (!normalizedProviderID) {
          return;
        }
        const normalizedAPIKey = rawApiKey.trim();
        const providerRaw = isRecord(providersRaw[normalizedProviderID])
          ? { ...(providersRaw[normalizedProviderID] as Record<string, unknown>) }
          : {};
        if (normalizedAPIKey) {
          providerRaw.apiKey = normalizedAPIKey;
        } else {
          delete providerRaw.apiKey;
        }
        if (Object.keys(providerRaw).length > 0) {
          providersRaw[normalizedProviderID] = providerRaw;
        } else {
          delete providersRaw[normalizedProviderID];
        }
      });
      if (Object.keys(providersRaw).length > 0) {
        nextSearch.providers = providersRaw;
      } else {
        delete nextSearch.providers;
      }
      setOrDelete("maxResults", parseNumberInput(currentForm.maxResults));
      setOrDelete("country", currentForm.country.trim());
      setOrDelete("search_lang", currentForm.searchLang.trim());
      setOrDelete("ui_lang", currentForm.uiLang.trim());
      setOrDelete("freshness", currentForm.freshness.trim());
      setOrDelete("timeoutSeconds", parseNumberInput(currentForm.timeoutSeconds));
      setOrDelete("cacheTtlMinutes", parseNumberInput(currentForm.cacheTtlMinutes));
      delete nextSearch.external_tools;
    } else {
      delete nextSearch.provider;
      delete nextSearch.apiKey;
      delete nextSearch.maxResults;
      delete nextSearch.country;
      delete nextSearch.search_lang;
      delete nextSearch.ui_lang;
      delete nextSearch.freshness;
      delete nextSearch.timeoutSeconds;
      delete nextSearch.cacheTtlMinutes;
      nextSearch.external_tools = {};
    }
    delete nextSearch["enabled"];
    if (Object.keys(nextSearch).length > 0) {
      nextWeb.search = nextSearch;
    } else {
      delete nextWeb.search;
    }
    if (Object.keys(nextWeb).length > 0) {
      nextToolsConfig.web = nextWeb;
    } else {
      delete nextToolsConfig.web;
    }
    const payload = nextToolsConfig;
    updateSettings.mutate(
      { tools: payload },
      {
        onSuccess: () => {
          void gatewayToolsQuery.refetch();
        },
      }
    );
    return true;
  }, [
    gatewayToolsQuery,
    toolsConfig,
    updateSettings,
    webSearchForm,
    webSearchProviderAPIKeys,
  ]);

  const handleSaveWebFetch = React.useCallback((formState?: WebFetchFormState) => {
    const currentForm = formState ?? webFetchForm;
    const nextToolsConfig: Record<string, unknown> = { ...toolsConfig };
    const nextWeb = isRecord(nextToolsConfig.web)
      ? { ...(nextToolsConfig.web as Record<string, unknown>) }
      : {};
    const nextTopLevelFetch = isRecord(nextToolsConfig.web_fetch)
      ? { ...(nextToolsConfig.web_fetch as Record<string, unknown>) }
      : {};
    const applyWebFetchValues = (target: Record<string, unknown>) => {
      const setOrDelete = (key: string, value: unknown) => {
        if (value === undefined || value === "") {
          delete target[key];
          return;
        }
        target[key] = value;
      };
      target.headless = currentForm.headless;
      target.preferredBrowser = normalizePreferredBrowser(currentForm.preferredBrowser);
      setOrDelete("timeoutSeconds", parseNumberInput(currentForm.timeoutSeconds));
      setOrDelete("maxChars", parseNumberInput(currentForm.maxChars));
      delete target.acceptMarkdown;
      delete target.enableUserAgent;
      delete target.userAgent;
      delete target.acceptLanguage;
      delete target.headers;
      delete target.maxRedirects;
      delete target.retryMax;
      delete target.enabled;
    };
    applyWebFetchValues(nextTopLevelFetch);
    delete nextWeb.fetch;
    if (Object.keys(nextWeb).length > 0) {
      nextToolsConfig.web = nextWeb;
    } else {
      delete nextToolsConfig.web;
    }
    nextToolsConfig.web_fetch = nextTopLevelFetch;
    const payload = nextToolsConfig;
    updateSettings.mutate(
      { tools: payload },
      {
        onSuccess: () => {
          void gatewayToolsQuery.refetch();
        },
      }
    );
    return true;
  }, [gatewayToolsQuery, toolsConfig, updateSettings, webFetchForm]);

  const handleSaveBrowser = React.useCallback((formState?: BrowserControlFormState) => {
    const currentForm = formState ?? browserForm;
    const parsedAllowedHostnames = parseStringArrayJSON(currentForm.ssrfAllowedHostnamesJson);
    if (parsedAllowedHostnames.error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.tools.browserControl.arrayInvalid"),
        description: t("settings.tools.browserControl.arrayInvalidDesc"),
      });
      return false;
    }
    const parsedHostnameAllowlist = parseStringArrayJSON(currentForm.ssrfHostnameAllowlistJson);
    if (parsedHostnameAllowlist.error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.tools.browserControl.arrayInvalid"),
        description: t("settings.tools.browserControl.arrayInvalidDesc"),
      });
      return false;
    }
    const nextToolsConfig: Record<string, unknown> = { ...toolsConfig };
    const nextBrowser = isRecord(nextToolsConfig.browser)
      ? { ...(nextToolsConfig.browser as Record<string, unknown>) }
      : {};

    nextBrowser.enabled = currentForm.enabled;
    nextBrowser.headless = currentForm.headless;
    nextBrowser.preferredBrowser = normalizePreferredBrowser(currentForm.preferredBrowser);
    delete nextBrowser.executablePath;
    delete nextBrowser.evaluateEnabled;
    delete nextBrowser.noSandbox;
    delete nextBrowser.snapshotDefaults;
    delete nextBrowser.extraArgs;

    const nextSSRFRules: Record<string, unknown> = {
      dangerouslyAllowPrivateNetwork: currentForm.ssrfDangerouslyAllowPrivateNetwork,
    };
    if (parsedAllowedHostnames.value && parsedAllowedHostnames.value.length > 0) {
      nextSSRFRules.allowedHostnames = parsedAllowedHostnames.value;
    }
    if (parsedHostnameAllowlist.value && parsedHostnameAllowlist.value.length > 0) {
      nextSSRFRules.hostnameAllowlist = parsedHostnameAllowlist.value;
    }
    nextBrowser.ssrfPolicy = nextSSRFRules;

    nextToolsConfig.browser = nextBrowser;
    const payload = nextToolsConfig;
    updateSettings.mutate(
      { tools: payload },
      {
        onSuccess: () => {
          void gatewayToolsQuery.refetch();
        },
      }
    );
    return true;
  }, [browserForm, gatewayToolsQuery, t, toolsConfig, updateSettings]);

  const handleToggleBrowserEnabled = React.useCallback((enabled: boolean) => {
    setBrowserForm((prev) => ({ ...prev, enabled }));
    const nextToolsConfig: Record<string, unknown> = { ...toolsConfig };
    const nextBrowser = isRecord(nextToolsConfig.browser)
      ? { ...(nextToolsConfig.browser as Record<string, unknown>) }
      : {};
    nextBrowser.enabled = enabled;
    delete nextBrowser.executablePath;
    nextToolsConfig.browser = nextBrowser;
    const payload = nextToolsConfig;
    updateSettings.mutate(
      { tools: payload },
      {
        onSuccess: () => {
          void gatewayToolsQuery.refetch();
        },
      }
    );
  }, [gatewayToolsQuery, toolsConfig, updateSettings]);

  const handleWebSearchFieldBlur = React.useCallback(() => {
    if (skipNextWebSearchBlurSaveRef.current) {
      skipNextWebSearchBlurSaveRef.current = false;
      return;
    }
    if (!isWebSearchDirty || webSearchDisabled) {
      return;
    }
    handleSaveWebSearch();
  }, [handleSaveWebSearch, isWebSearchDirty, webSearchDisabled]);

  const handleWebFetchFieldBlur = React.useCallback(() => {
    if (skipNextWebFetchBlurSaveRef.current) {
      skipNextWebFetchBlurSaveRef.current = false;
      return;
    }
    if (!isWebFetchDirty || webFetchDisabled) {
      return;
    }
    handleSaveWebFetch();
  }, [handleSaveWebFetch, isWebFetchDirty, webFetchDisabled]);

  const handleBrowserFieldBlur = React.useCallback(() => {
    if (skipNextBrowserBlurSaveRef.current) {
      skipNextBrowserBlurSaveRef.current = false;
      return;
    }
    if (!isBrowserDirty || browserDisabled) {
      return;
    }
    handleSaveBrowser();
  }, [browserDisabled, handleSaveBrowser, isBrowserDirty]);

  const webSearchRowClassName =
    "flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4";
  const webSearchLabelClassName = "min-w-0 truncate text-sm font-medium text-muted-foreground";
  const webSearchControlClassName = "w-full sm:w-60 min-w-0";
  const webSearchTabsControlClassName = "w-full shrink-0 sm:w-auto sm:min-w-fit";
  const handleWebSearchTypeChange = React.useCallback(
    (nextType: string) => {
      const normalizedType = normalizeWebSearchType(nextType);
      if (webSearchForm.type === normalizedType) {
        return;
      }
      const nextForm: WebSearchFormState = { ...webSearchForm, type: normalizedType };
      setWebSearchForm(nextForm);
      if (webSearchDisabled) {
        return;
      }
      handleSaveWebSearch(nextForm, webSearchProviderAPIKeys);
    },
    [handleSaveWebSearch, webSearchDisabled, webSearchForm, webSearchProviderAPIKeys]
  );
  const handleWebSearchProviderChange = React.useCallback(
    (nextProvider: string) => {
      const nextProviderApiKey = resolveWebSearchDraftApiKey(nextProvider);
      const nextForm: WebSearchFormState = {
        ...webSearchForm,
        provider: nextProvider,
        apiKey: nextProviderApiKey,
      };
      setWebSearchForm(nextForm);
      if (webSearchDisabled) {
        return;
      }
      handleSaveWebSearch(nextForm, webSearchProviderAPIKeys);
    },
    [
      handleSaveWebSearch,
      resolveWebSearchDraftApiKey,
      webSearchDisabled,
      webSearchForm,
      webSearchProviderAPIKeys,
    ]
  );
  const handleBrowserPreferredBrowserChange = React.useCallback(
    (value: string) => {
      const nextForm: BrowserControlFormState = {
        ...browserForm,
        preferredBrowser: normalizePreferredBrowser(value),
      };
      setBrowserForm(nextForm);
      if (browserDisabled) {
        return;
      }
      handleSaveBrowser(nextForm);
    },
    [browserDisabled, browserForm, handleSaveBrowser]
  );
  const handleBrowserHeadlessChange = React.useCallback(
    (checked: boolean) => {
      const nextForm: BrowserControlFormState = {
        ...browserForm,
        headless: Boolean(checked),
      };
      setBrowserForm(nextForm);
      if (browserDisabled) {
        return;
      }
      handleSaveBrowser(nextForm);
    },
    [browserDisabled, browserForm, handleSaveBrowser]
  );
  const handleBrowserPrivateNetworkChange = React.useCallback(
    (checked: boolean) => {
      const nextForm: BrowserControlFormState = {
        ...browserForm,
        ssrfDangerouslyAllowPrivateNetwork: Boolean(checked),
      };
      setBrowserForm(nextForm);
      if (browserDisabled) {
        return;
      }
      handleSaveBrowser(nextForm);
    },
    [browserDisabled, browserForm, handleSaveBrowser]
  );
  const handleWebFetchPreferredBrowserChange = React.useCallback(
    (value: string) => {
      const nextForm: WebFetchFormState = {
        ...webFetchForm,
        preferredBrowser: normalizePreferredBrowser(value),
      };
      setWebFetchForm(nextForm);
      if (webFetchDisabled) {
        return;
      }
      handleSaveWebFetch(nextForm);
    },
    [handleSaveWebFetch, webFetchDisabled, webFetchForm]
  );
  const handleWebFetchHeadlessChange = React.useCallback(
    (checked: boolean) => {
      const nextForm: WebFetchFormState = {
        ...webFetchForm,
        headless: Boolean(checked),
      };
      setWebFetchForm(nextForm);
      if (webFetchDisabled) {
        return;
      }
      handleSaveWebFetch(nextForm);
    },
    [handleSaveWebFetch, webFetchDisabled, webFetchForm]
  );
  const renderWebSearchFieldLabel = React.useCallback((label: string, description?: string) => {
    return (
      <div className="flex min-w-0 items-center gap-1.5">
        <span className={webSearchLabelClassName}>{label}</span>
        {description ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                aria-label={label}
              >
                <HelpCircle className="h-3.5 w-3.5" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-80 whitespace-pre-line text-xs">
              {description}
            </TooltipContent>
          </Tooltip>
        ) : null}
      </div>
    );
  }, []);
  const renderRuntimeDetectionCard = React.useCallback((rows: RuntimeDetectionRow[]) => {
    return (
      <div className="rounded-md border border-border/60 bg-muted/25 p-3 text-xs text-muted-foreground">
        <div className="divide-y divide-border/60">
          {rows.map((item, index) => {
            const rowSpacingClass = rows.length === 1
              ? ""
              : index === 0
                ? "pb-2"
                : index === rows.length - 1
                  ? "pt-2"
                  : "py-2";
            return (
              <div
                key={`${item.label}-${item.value}-${index}`}
                className={`flex min-w-0 items-center justify-between gap-4 ${rowSpacingClass}`}
              >
                <span className="min-w-0 flex-1 text-foreground/80">{item.label}</span>
                {item.badge ? (
                  <Badge
                    variant="outline"
                    className="max-w-[60%] shrink whitespace-nowrap border-border/70 bg-background/80"
                    title={item.value}
                  >
                    <Ban className="mr-1 h-3 w-3 shrink-0" />
                    <span className="truncate">{item.value}</span>
                  </Badge>
                ) : (
                  <span
                    className="min-w-0 max-w-[60%] shrink truncate whitespace-nowrap text-right"
                    title={item.value}
                  >
                    {item.value}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      </div>
    );
  }, []);
  return (
    <CallsCard
      leftList={
        filteredTools.length === 0 ? (
          <div className="p-3 text-sm text-muted-foreground">
            {t("settings.tools.list.empty")}
          </div>
        ) : (
          <SidebarMenu>
            {groupedTools.map((group, groupIndex) => (
              <React.Fragment key={group.id}>
                <div className="px-2 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                  {group.label}
                </div>
                {group.tools.map((tool) => {
                  const status = resolveToolStatus(tool);
                  return (
                    <SidebarMenuItem key={tool.id}>
                      <SidebarMenuButton
                        isActive={tool.id === selectedToolId}
                        onClick={() => setSelectedToolId(tool.id)}
                        className="justify-between"
                      >
                        <span className="truncate text-sm font-medium">
                          {t(tool.labelKey)}
                        </span>
                        {renderToolStatusBadge(status)}
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
                {groupIndex < groupedTools.length - 1 ? (
                  <div className="px-2 py-2">
                    <Separator />
                  </div>
                ) : null}
              </React.Fragment>
            ))}
          </SidebarMenu>
        )
      }
      rightContent={
        !selectedTool ? (
          <div className="p-4 text-sm text-muted-foreground">
            {t("settings.tools.list.empty")}
          </div>
        ) : (
          (() => {
            const status = resolveToolStatus(selectedTool);
            const isWebSearchTool = selectedTool.id === "web_search";
            const isWebFetchTool = selectedTool.id === "web_fetch";
            const isGatewayTool = selectedTool.id === "gateway";
            const isBrowserTool = selectedTool.id === "browser";
            const gatewayConfig = isGatewayTool && isRecord(toolsConfig.gateway)
              ? (toolsConfig.gateway as Record<string, unknown>)
              : undefined;
            const hasGatewayConfig = Boolean(gatewayConfig && Object.keys(gatewayConfig).length > 0);
            const toolDependencies = resolveToolDependencies(selectedTool);
            const browserRuntimeRequirement =
              isBrowserTool || isWebFetchTool
                ? (selectedTool.requirements ?? []).find((requirement) =>
                    requirement.id === (isBrowserTool ? "browser.cdp_runtime" : "web_fetch.local_browser")
                  )
                : undefined;
            const browserRuntimeData = browserRuntimeRequirement && isRecord(browserRuntimeRequirement.data)
              ? (browserRuntimeRequirement.data as Record<string, unknown>)
              : undefined;
            const browserCandidates = normalizeRuntimeBrowserCandidates(browserRuntimeData?.candidates);
            const availableBrowserCandidates = browserCandidates.filter((candidate) => candidate.available);
            const browserSelectOptions = availableBrowserCandidates;
            const webFetchPreferredBrowserValue = browserSelectOptions.some(
              (candidate) => candidate.id === webFetchForm.preferredBrowser
            )
              ? webFetchForm.preferredBrowser
              : (browserSelectOptions[0]?.id ?? "");
            const browserPreferredBrowserValue = browserSelectOptions.some(
              (candidate) => candidate.id === browserForm.preferredBrowser
            )
              ? browserForm.preferredBrowser
              : (browserSelectOptions[0]?.id ?? "");
            const runtimeDetectionRows: RuntimeDetectionRow[] = browserCandidates.map((candidate) => {
              const normalizedError = candidate.error.trim().toLowerCase();
              if (candidate.available) {
                return {
                  label: candidate.label,
                  value: candidate.execPath || t("settings.tools.runtimeDetection.detected"),
                };
              }
              if (normalizedError.includes("browser executable not found")) {
                return {
                  label: candidate.label,
                  value: t("settings.tools.runtimeDetection.notInstalled"),
                  badge: "not_installed",
                };
              }
              return {
                label: candidate.label,
                value: t("settings.tools.runtimeDetection.notDetected"),
                badge: "not_detected",
              };
            });
            const webSearchProviderMeta = [
              selectedWebSearchProvider?.apiBaseUrl
                ? `${t("settings.tools.webSearch.apiBase")}: ${selectedWebSearchProvider.apiBaseUrl}`
                : "",
              selectedWebSearchProvider?.openRouterBaseUrl
                ? `${t("settings.tools.webSearch.openRouterBase")}: ${selectedWebSearchProvider.openRouterBaseUrl}`
                : "",
            ]
              .filter((entry) => entry !== "")
              .join("\n");
            return (
              <div className="flex h-full flex-col space-y-3">
                {!isWebSearchTool && !isWebFetchTool && !isGatewayTool && !isBrowserTool ? (
                  <ToolDetailLayout
                    overview={
                      <ToolOverviewCard
                        title={t(selectedTool.labelKey)}
                        description={t(selectedTool.descriptionKey)}
                        descriptionLabel={t("settings.tools.detail.descriptionLabel")}
                        statusBadge={renderToolStatusBadge(status)}
                        enabledLabel={t("settings.tools.detail.enabled")}
                        enabled={selectedTool.available}
                        enabledDisabled={gatewayToolEnablePending}
                        onEnabledChange={(enabled) => {
                          void handleToggleToolEnabled(selectedTool, enabled);
                        }}
                        dependencies={toolDependencies}
                      />
                    }
                    content={
                      <ToolContentTabs
                        value={gatewayPanelTab}
                        onValueChange={setGatewayPanelTab}
                        configLabel={t("settings.tools.detail.tabs.config")}
                        ioLabel={t("settings.tools.detail.tabs.io")}
                      >
                        <ToolConfigTabPanel>
                          <ToolConfigEmptyState
                            title={t("settings.tools.detail.config.noneTitle")}
                            description={t("settings.tools.detail.config.noneDescription")}
                          />
                        </ToolConfigTabPanel>
                        <ToolIOTabPanel>
                          <ToolMethodIOPanel
                              rowClassName={webSearchRowClassName}
                              labelClassName={webSearchLabelClassName}
                              controlClassName={webSearchControlClassName}
                              methodLabel={renderWebSearchFieldLabel(
                                t("settings.tools.detail.io.method"),
                                t("settings.tools.detail.io.methodDesc")
                              )}
                              actionValue={activeGatewayAction}
                              actions={gatewayActions}
                              onActionChange={setGatewayActionDraft}
                              emptyMethodLabel={t("settings.tools.detail.io.empty")}
                              inputTitle={t("settings.tools.detail.io.input")}
                              outputTitle={t("settings.tools.detail.io.output")}
                              inputPayload={toPrettyJSON(
                                buildToolInputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                              outputPayload={toPrettyJSON(
                                buildToolOutputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                            />
                        </ToolIOTabPanel>
                      </ToolContentTabs>
                    }
                  />
                ) : isWebSearchTool ? (
                  <ToolDetailLayout
                    overview={
                      <ToolOverviewCard
                        title={t(selectedTool.labelKey)}
                        description={t(selectedTool.descriptionKey)}
                        descriptionLabel={t("settings.tools.detail.descriptionLabel")}
                        statusBadge={renderToolStatusBadge(status)}
                        enabledLabel={t("settings.tools.detail.enabled")}
                        enabled={selectedTool.available}
                        enabledDisabled={gatewayToolEnablePending}
                        onEnabledChange={(enabled) => {
                          void handleToggleToolEnabled(selectedTool, enabled);
                        }}
                        dependencies={toolDependencies}
                      />
                    }
                    content={
                      <ToolContentTabs
                        value={gatewayPanelTab}
                        onValueChange={setGatewayPanelTab}
                        configLabel={t("settings.tools.detail.tabs.config")}
                        ioLabel={t("settings.tools.detail.tabs.io")}
                      >
                        <ToolConfigTabPanel>
                          <TooltipProvider delayDuration={0}>
                            <div className="space-y-3">
                      <div className={webSearchRowClassName}>
                        {renderWebSearchFieldLabel(
                          t("settings.tools.webSearch.type"),
                          t("settings.tools.webSearch.typeDesc")
                        )}
                        <Tabs
                          value={webSearchForm.type}
                          onValueChange={handleWebSearchTypeChange}
                          className={webSearchTabsControlClassName}
                        >
                          <TabsList className="w-full justify-start sm:w-auto">
                            <TabsTrigger value="api">
                              {t("settings.tools.webSearch.typeValue.api")}
                            </TabsTrigger>
                            <TabsTrigger value="external_tools">
                              {t("settings.tools.webSearch.typeValue.externalTools")}
                            </TabsTrigger>
                          </TabsList>
                        </Tabs>
                      </div>
                      {webSearchForm.type === "api" ? (
                        <>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(
                              t("settings.tools.webSearch.provider"),
                              webSearchProviderMeta || undefined
                            )}
                            <Select
                              value={webSearchForm.provider}
                              onChange={(event) => {
                                handleWebSearchProviderChange(event.target.value);
                              }}
                              className={webSearchControlClassName}
                              disabled={webSearchDisabled}
                            >
                              {webSearchProviders.map((option) => (
                                <option key={option.id} value={option.id}>
                                  {option.label}
                                </option>
                              ))}
                            </Select>
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(
                              t("settings.tools.webSearch.apiKey"),
                              t("settings.tools.webSearch.apiKeyHint")
                            )}
                            <Input
                              type="password"
                              value={webSearchForm.apiKey}
                              onChange={(event) => {
                                const nextApiKey = event.target.value;
                                setWebSearchForm((prev) => ({ ...prev, apiKey: nextApiKey }));
                                updateWebSearchProviderApiKey(webSearchForm.provider, nextApiKey);
                              }}
                              onBlur={handleWebSearchFieldBlur}
                              placeholder={webSearchAPIKeyPlaceholder}
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.maxResults"))}
                            <Input
                              type="number"
                              min={1}
                              max={10}
                              value={webSearchForm.maxResults}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({
                                  ...prev,
                                  maxResults: event.target.value,
                                }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="5"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                        </>
                      ) : (
                        <>
                          <Separator />
                          <div className="rounded-md border border-border/60 bg-muted/25 p-3 text-xs text-muted-foreground">
                            <p>
                              {t("settings.tools.webSearch.externalToolsHint")}
                            </p>
                          </div>
                        </>
                      )}
                      {webSearchForm.type === "api" ? (
                        <>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.country"))}
                            <Input
                              value={webSearchForm.country}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({ ...prev, country: event.target.value }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="US"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.searchLang"))}
                            <Input
                              value={webSearchForm.searchLang}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({ ...prev, searchLang: event.target.value }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="en"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.uiLang"))}
                            <Input
                              value={webSearchForm.uiLang}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({ ...prev, uiLang: event.target.value }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="en"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.freshness"))}
                            <Input
                              value={webSearchForm.freshness}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({
                                  ...prev,
                                  freshness: event.target.value,
                                }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="pd / pw / pm / py"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.timeoutSeconds"))}
                            <Input
                              type="number"
                              min={1}
                              value={webSearchForm.timeoutSeconds}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({
                                  ...prev,
                                  timeoutSeconds: event.target.value,
                                }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="30"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <Separator />
                          <div className={webSearchRowClassName}>
                            {renderWebSearchFieldLabel(t("settings.tools.webSearch.cacheTtlMinutes"))}
                            <Input
                              type="number"
                              min={1}
                              value={webSearchForm.cacheTtlMinutes}
                              onChange={(event) =>
                                setWebSearchForm((prev) => ({
                                  ...prev,
                                  cacheTtlMinutes: event.target.value,
                                }))
                              }
                              onBlur={handleWebSearchFieldBlur}
                              placeholder="15"
                              className={webSearchControlClassName}
                              size="compact"
                              disabled={webSearchDisabled}
                            />
                          </div>
                          <div className="flex justify-center pt-2">
                            <Button
                              variant="destructive"
                              size="compact"
                              onPointerDown={() => {
                                skipNextWebSearchBlurSaveRef.current = true;
                              }}
                              onClick={handleResetWebSearch}
                              disabled={webSearchDisabled}
                            >
                              {t("common.reset")}
                            </Button>
                          </div>
                        </>
                      ) : null}
                            </div>
                          </TooltipProvider>
                        </ToolConfigTabPanel>
                        <ToolIOTabPanel>
                          <ToolMethodIOPanel
                              rowClassName={webSearchRowClassName}
                              labelClassName={webSearchLabelClassName}
                              controlClassName={webSearchControlClassName}
                              methodLabel={renderWebSearchFieldLabel(
                                t("settings.tools.detail.io.method"),
                                t("settings.tools.detail.io.methodDesc")
                              )}
                              actionValue={activeGatewayAction}
                              actions={gatewayActions}
                              onActionChange={setGatewayActionDraft}
                              emptyMethodLabel={t("settings.tools.detail.io.empty")}
                              inputTitle={t("settings.tools.detail.io.input")}
                              outputTitle={t("settings.tools.detail.io.output")}
                              inputPayload={toPrettyJSON(
                                buildToolInputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                              outputPayload={toPrettyJSON(
                                buildToolOutputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                            />
                        </ToolIOTabPanel>
                      </ToolContentTabs>
                    }
                  />
                ) : isGatewayTool ? (
                  <ToolDetailLayout
                    overview={
                      <ToolOverviewCard
                        title={t(selectedTool.labelKey)}
                        description={t(selectedTool.descriptionKey)}
                        descriptionLabel={t("settings.tools.detail.descriptionLabel")}
                        statusBadge={renderToolStatusBadge(status)}
                        enabledLabel={t("settings.tools.detail.enabled")}
                        enabled={selectedTool.available}
                        enabledDisabled={gatewayToolEnablePending}
                        onEnabledChange={(enabled) => {
                          void handleToggleToolEnabled(selectedTool, enabled);
                        }}
                        dependencies={toolDependencies}
                      />
                    }
                    content={
                      <ToolContentTabs
                        value={gatewayPanelTab}
                        onValueChange={setGatewayPanelTab}
                        configLabel={t("settings.tools.detail.tabs.config")}
                        ioLabel={t("settings.tools.detail.tabs.io")}
                      >
                        <ToolConfigTabPanel>
                          {hasGatewayConfig ? (
                            <ToolConfigCard
                              title={t("settings.tools.detail.config.title")}
                            >
                              <pre className="overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-3 text-xs">
                                {toPrettyJSON(gatewayConfig)}
                              </pre>
                            </ToolConfigCard>
                          ) : (
                            <ToolConfigEmptyState
                              title={t("settings.tools.detail.config.noneTitle")}
                              description={t("settings.tools.detail.config.noneDescription")}
                            />
                          )}
                        </ToolConfigTabPanel>
                        <ToolIOTabPanel>
                          <ToolMethodIOPanel
                              rowClassName={webSearchRowClassName}
                              labelClassName={webSearchLabelClassName}
                              controlClassName={webSearchControlClassName}
                              methodLabel={renderWebSearchFieldLabel(
                                t("settings.tools.detail.io.method"),
                                t("settings.tools.detail.io.methodDesc")
                              )}
                              actionValue={activeGatewayAction}
                              actions={gatewayActions}
                              onActionChange={setGatewayActionDraft}
                              emptyMethodLabel={t("settings.tools.detail.io.empty")}
                              inputTitle={t("settings.tools.detail.io.input")}
                              outputTitle={t("settings.tools.detail.io.output")}
                              inputPayload={toPrettyJSON(
                                buildToolInputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                              outputPayload={toPrettyJSON(
                                buildToolOutputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                            />
                        </ToolIOTabPanel>
                      </ToolContentTabs>
                    }
                  />
                ) : isBrowserTool ? (
                  <ToolDetailLayout
                    overview={
                      <ToolOverviewCard
                        title={t(selectedTool.labelKey)}
                        description={t(selectedTool.descriptionKey)}
                        descriptionLabel={t("settings.tools.detail.descriptionLabel")}
                        statusBadge={renderToolStatusBadge(status)}
                        enabledLabel={t("settings.tools.browserControl.enabled")}
                        enabled={browserForm.enabled}
                        enabledDisabled={browserDisabled}
                        onEnabledChange={handleToggleBrowserEnabled}
                        dependencies={toolDependencies}
                      />
                    }
                    content={
                      <ToolContentTabs
                        value={gatewayPanelTab}
                        onValueChange={setGatewayPanelTab}
                        configLabel={t("settings.tools.detail.tabs.config")}
                        ioLabel={t("settings.tools.detail.tabs.io")}
                      >
                        <ToolConfigTabPanel>
                          <TooltipProvider delayDuration={0}>
                            <div className="space-y-3">
                              {(selectedTool.requirements ?? []).some((requirement) => !requirement.available) ? (
                                <div className="space-y-1 rounded-md border border-destructive/40 bg-destructive/5 p-3 text-xs text-destructive">
                                  {(selectedTool.requirements ?? [])
                                    .filter((requirement) => !requirement.available)
                                    .map((requirement) => {
                                      const fallbackName = resolveRequirementName(
                                        requirement.id,
                                        requirement.name || requirement.id
                                      );
                                      const fallbackReason = resolveRequirementReason(requirement.reason || "");
                                      return (
                                        <p key={requirement.id}>
                                          <span className="font-medium">
                                            {fallbackName}:
                                          </span>{" "}
                                          {fallbackReason}
                                        </p>
                                      );
                                    })}
                                  </div>
                              ) : null}
                              <Separator />
                              <div className={webSearchRowClassName}>
                                {renderWebSearchFieldLabel(
                                  t("settings.tools.browserControl.preferredBrowser"),
                                  t("settings.tools.browserControl.preferredBrowserDesc")
                                )}
                                <Select
                                  value={browserPreferredBrowserValue}
                                  onChange={(event) => {
                                    handleBrowserPreferredBrowserChange(event.target.value);
                                  }}
                                  className={webSearchControlClassName}
                                  disabled={browserDisabled || browserSelectOptions.length === 0}
                                >
                                  {browserSelectOptions.length === 0 ? (
                                    <option value="">{t("settings.tools.runtimeDetection.noneDetected")}</option>
                                  ) : browserSelectOptions.map((candidate) => (
                                    <option key={candidate.id || candidate.label} value={candidate.id}>
                                      {candidate.label}
                                    </option>
                                  ))}
                                </Select>
                              </div>
                              <Separator />
                              {renderRuntimeDetectionCard(runtimeDetectionRows)}
                              <Separator />
                              <div className={webSearchRowClassName}>
                                {renderWebSearchFieldLabel(
                                  t("settings.tools.browserControl.headless"),
                                  t("settings.tools.browserControl.headlessDesc")
                                )}
                                <Switch
                                  checked={browserForm.headless}
                                  onCheckedChange={handleBrowserHeadlessChange}
                                  disabled={browserDisabled}
                                />
                              </div>
                              <Separator />
                              <details className="group rounded-md border border-border/70">
                                <summary className="flex cursor-pointer list-none items-center justify-between px-3 py-2 [&::-webkit-details-marker]:hidden">
                                  <div className="min-w-0">
                                    <p className="text-sm font-medium">
                                      {t("settings.tools.browserControl.ssrfSection")}
                                    </p>
                                    <p className="text-xs text-muted-foreground">
                                      {t("settings.tools.browserControl.ssrfSectionDesc")}
                                    </p>
                                  </div>
                                  <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-90" />
                                </summary>
                                <div className="space-y-3 border-t border-border/70 px-3 py-3">
                                  <div className={webSearchRowClassName}>
                                    {renderWebSearchFieldLabel(
                                      t("settings.tools.browserControl.ssrfDangerouslyAllowPrivateNetwork"),
                                      t("settings.tools.browserControl.ssrfDangerouslyAllowPrivateNetworkDesc")
                                    )}
                                    <Switch
                                      checked={browserForm.ssrfDangerouslyAllowPrivateNetwork}
                                      onCheckedChange={handleBrowserPrivateNetworkChange}
                                      disabled={browserDisabled}
                                    />
                                  </div>
                                  <Separator />
                                  <div className={webSearchRowClassName}>
                                    {renderWebSearchFieldLabel(
                                      t("settings.tools.browserControl.ssrfAllowedHostnames")
                                    )}
                                    <Input
                                      value={browserForm.ssrfAllowedHostnamesJson}
                                      onChange={(event) =>
                                        setBrowserForm((prev) => ({
                                          ...prev,
                                          ssrfAllowedHostnamesJson: event.target.value,
                                        }))
                                      }
                                      onBlur={handleBrowserFieldBlur}
                                      placeholder='["localhost","metadata.internal"]'
                                      className={webSearchControlClassName}
                                      size="compact"
                                      disabled={browserDisabled}
                                    />
                                  </div>
                                  <Separator />
                                  <div className={webSearchRowClassName}>
                                    {renderWebSearchFieldLabel(
                                      t("settings.tools.browserControl.ssrfHostnameAllowlist")
                                    )}
                                    <Input
                                      value={browserForm.ssrfHostnameAllowlistJson}
                                      onChange={(event) =>
                                        setBrowserForm((prev) => ({
                                          ...prev,
                                          ssrfHostnameAllowlistJson: event.target.value,
                                        }))
                                      }
                                      onBlur={handleBrowserFieldBlur}
                                      placeholder='["*.example.com"]'
                                      className={webSearchControlClassName}
                                      size="compact"
                                      disabled={browserDisabled}
                                    />
                                  </div>
                                </div>
                              </details>
                              <div className="flex justify-center pt-2">
                                <Button
                                  variant="destructive"
                                  size="compact"
                                  onPointerDown={() => {
                                    skipNextBrowserBlurSaveRef.current = true;
                                  }}
                                  onClick={handleResetBrowser}
                                  disabled={browserDisabled}
                                >
                                  {t("common.reset")}
                                </Button>
                              </div>
                            </div>
                          </TooltipProvider>
                        </ToolConfigTabPanel>
                        <ToolIOTabPanel>
                          <ToolMethodIOPanel
                              rowClassName={webSearchRowClassName}
                              labelClassName={webSearchLabelClassName}
                              controlClassName={webSearchControlClassName}
                              methodLabel={renderWebSearchFieldLabel(
                                t("settings.tools.detail.io.method"),
                                t("settings.tools.detail.io.methodDesc")
                              )}
                              actionValue={activeGatewayAction}
                              actions={gatewayActions}
                              onActionChange={setGatewayActionDraft}
                              emptyMethodLabel={t("settings.tools.detail.io.empty")}
                              inputTitle={t("settings.tools.detail.io.input")}
                              outputTitle={t("settings.tools.detail.io.output")}
                              inputPayload={toPrettyJSON(
                                buildToolInputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                              outputPayload={toPrettyJSON(
                                buildToolOutputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                            />
                        </ToolIOTabPanel>
                      </ToolContentTabs>
                    }
                  />
                ) : (
                  <ToolDetailLayout
                    overview={
                      <ToolOverviewCard
                        title={t(selectedTool.labelKey)}
                        description={t(selectedTool.descriptionKey)}
                        descriptionLabel={t("settings.tools.detail.descriptionLabel")}
                        statusBadge={renderToolStatusBadge(status)}
                        enabledLabel={t("settings.tools.detail.enabled")}
                        enabled={selectedTool.available}
                        enabledDisabled={gatewayToolEnablePending}
                        onEnabledChange={(enabled) => {
                          void handleToggleToolEnabled(selectedTool, enabled);
                        }}
                        dependencies={toolDependencies}
                      />
                    }
                    content={
                      <ToolContentTabs
                        value={gatewayPanelTab}
                        onValueChange={setGatewayPanelTab}
                        configLabel={t("settings.tools.detail.tabs.config")}
                        ioLabel={t("settings.tools.detail.tabs.io")}
                      >
                        <ToolConfigTabPanel>
                          <TooltipProvider delayDuration={0}>
                            <div className="space-y-3">
                      <div className={webSearchRowClassName}>
                        {renderWebSearchFieldLabel(
                          t("settings.tools.webFetch.preferredBrowser"),
                          t("settings.tools.webFetch.preferredBrowserDesc")
                        )}
                        <Select
                          value={webFetchPreferredBrowserValue}
                          onChange={(event) => {
                            handleWebFetchPreferredBrowserChange(event.target.value);
                          }}
                          className={webSearchControlClassName}
                          disabled={webFetchDisabled || browserSelectOptions.length === 0}
                        >
                          {browserSelectOptions.length === 0 ? (
                            <option value="">{t("settings.tools.runtimeDetection.noneDetected")}</option>
                          ) : browserSelectOptions.map((candidate) => (
                            <option key={candidate.id || candidate.label} value={candidate.id}>
                              {candidate.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <Separator />
                      {renderRuntimeDetectionCard(runtimeDetectionRows)}
                      <Separator />
                      <div className={webSearchRowClassName}>
                        {renderWebSearchFieldLabel(
                          t("settings.tools.webFetch.headless"),
                          t("settings.tools.webFetch.headlessDesc")
                        )}
                        <Switch
                          checked={webFetchForm.headless}
                          onCheckedChange={handleWebFetchHeadlessChange}
                          disabled={webFetchDisabled}
                        />
                      </div>
                      <Separator />
                      <div className={webSearchRowClassName}>
                        {renderWebSearchFieldLabel(t("settings.tools.webFetch.timeoutSeconds"))}
                        <Input
                          type="number"
                          min={1}
                          value={webFetchForm.timeoutSeconds}
                          onChange={(event) =>
                            setWebFetchForm((prev) => ({ ...prev, timeoutSeconds: event.target.value }))
                          }
                          onBlur={handleWebFetchFieldBlur}
                          placeholder="20"
                          className={webSearchControlClassName}
                          size="compact"
                          disabled={webFetchDisabled}
                        />
                      </div>
                      <Separator />
                      <div className={webSearchRowClassName}>
                        {renderWebSearchFieldLabel(t("settings.tools.webFetch.maxChars"))}
                        <Input
                          type="number"
                          min={1}
                          value={webFetchForm.maxChars}
                          onChange={(event) =>
                            setWebFetchForm((prev) => ({ ...prev, maxChars: event.target.value }))
                          }
                          onBlur={handleWebFetchFieldBlur}
                          placeholder="50000"
                          className={webSearchControlClassName}
                          size="compact"
                          disabled={webFetchDisabled}
                        />
                      </div>
                      <div className="flex justify-center pt-2">
                        <Button
                          variant="destructive"
                          size="compact"
                          onPointerDown={() => {
                            skipNextWebFetchBlurSaveRef.current = true;
                          }}
                          onClick={handleResetWebFetch}
                          disabled={webFetchDisabled}
                        >
                          {t("common.reset")}
                        </Button>
                      </div>
                            </div>
                          </TooltipProvider>
                        </ToolConfigTabPanel>
                        <ToolIOTabPanel>
                          <ToolMethodIOPanel
                              rowClassName={webSearchRowClassName}
                              labelClassName={webSearchLabelClassName}
                              controlClassName={webSearchControlClassName}
                              methodLabel={renderWebSearchFieldLabel(
                                t("settings.tools.detail.io.method"),
                                t("settings.tools.detail.io.methodDesc")
                              )}
                              actionValue={activeGatewayAction}
                              actions={gatewayActions}
                              onActionChange={setGatewayActionDraft}
                              emptyMethodLabel={t("settings.tools.detail.io.empty")}
                              inputTitle={t("settings.tools.detail.io.input")}
                              outputTitle={t("settings.tools.detail.io.output")}
                              inputPayload={toPrettyJSON(
                                buildToolInputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                              outputPayload={toPrettyJSON(
                                buildToolOutputExample(selectedTool.id, activeGatewayMethod, activeGatewayAction)
                              )}
                            />
                        </ToolIOTabPanel>
                      </ToolContentTabs>
                    }
                  />
                )}
              </div>
            );
          })()
        )
      }
    />
  );
}
