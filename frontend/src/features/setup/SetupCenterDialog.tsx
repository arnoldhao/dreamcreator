import * as React from "react";
import {
  Loader2,
  Pencil,
  RefreshCw,
  Sparkles,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { messageBus } from "@/shared/message/store";
import { useI18n } from "@/shared/i18n";
import { useCurrentUserProfile } from "@/shared/query/system";
import { formatTemplate } from "@/features/library/utils/i18n";
import { useExternalTools } from "@/shared/query/externalTools";
import { useSystemProxyInfo, useTestProxy, useUpdateSettings } from "@/shared/query/settings";
import type { ProxySettings } from "@/shared/contracts/settings";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import { useSettingsStore } from "@/shared/store/settings";
import { Button } from "@/shared/ui/button";
import { DashboardDialogContent } from "@/shared/ui/dashboard-dialog";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { UserAvatar, resolveUserDisplayName, resolveUserSubtitle } from "@/shared/ui/user-avatar";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/shared/ui/tooltip";

import {
  DOWNLOAD_MODE_REQUIRED_TOOLS,
  FULL_MODE_REQUIRED_TOOLS,
  type SetupStepId,
} from "./constants";
import {
  SetupCardRow,
  SetupCardRows,
  SetupCardSeparator,
  SetupCardValue,
  SetupPageCard,
  SetupStatusIcon,
  SetupValueBadge,
} from "./cards";
import {
  SetupAgentModelCard,
  SetupProductModeCard,
  SetupProviderCard,
  SetupToolCard,
} from "./inline-cards";
import {
  buildSetupNavStatusEntries,
  getToolItemId,
  isSkippableItem,
  resolveGroupStatus,
  resolveItemPage,
  resolveProxyStatus,
  resolveStepDefaultItem,
  TOOL_ITEM_PREFIX,
  type SetupNavItemId,
  type SetupNavStatus,
} from "./nav";
import { resolveToolDependencyIssues } from "./readiness";
import { createManualProxyDraft } from "./setup-center-utils";
import { useSetupCenter } from "./store";
import { useSetupStatus } from "./useSetupStatus";

type SetupCenterDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

type SetupNavItem = {
  id: SetupNavItemId;
  label: string;
  page: SetupStepId;
  status: SetupNavStatus;
  value?: string;
  iconOnly?: boolean;
};

type SetupNavGroup = {
  id: SetupStepId;
  title: string;
  status: SetupNavStatus;
  items: SetupNavItem[];
};

export function SetupCenterDialog({ open, onOpenChange }: SetupCenterDialogProps) {
  const { t, supportedLanguages } = useI18n();
  const status = useSetupStatus();
  const { enabled: assistantUiEnabled } = useAssistantUiMode();
  const {
    focusItemId,
    clearFocusItem,
    deferAi,
    deferDependencies,
    skippedItemIds,
    skipItem,
  } = useSetupCenter();
  const settings = useSettingsStore((state) => state.settings);
  const currentUserProfile = useCurrentUserProfile().data;
  const externalToolsQuery = useExternalTools();
  const updateSettings = useUpdateSettings();
  const testProxy = useTestProxy();

  const [activeItem, setActiveItem] = React.useState<SetupNavItemId>("general.language");
  const [proxyDraft, setProxyDraft] = React.useState<ProxySettings | null>(null);
  const [proxyBusy, setProxyBusy] = React.useState(false);
  const [proxyDialogOpen, setProxyDialogOpen] = React.useState(false);
  const [clearConfirmOpen, setClearConfirmOpen] = React.useState(false);
  const [proxyCheckStatus, setProxyCheckStatus] = React.useState<"idle" | "checking" | "available" | "unavailable">("idle");
  const [proxyCheckKey, setProxyCheckKey] = React.useState("");
  const [activeInstallName, setActiveInstallName] = React.useState<string | null>(null);
  const [activeVerifyName, setActiveVerifyName] = React.useState<string | null>(null);
  const contentScrollRef = React.useRef<HTMLDivElement | null>(null);
  const sectionRefs = React.useRef<Record<string, HTMLDivElement | null>>({});
  const sectionRefCallbacks = React.useRef<Record<string, (node: HTMLDivElement | null) => void>>({});
  const proxyCheckRequestRef = React.useRef(0);
  const wasOpenRef = React.useRef(false);

  const proxyMode = proxyDraft?.mode ?? settings?.proxy.mode ?? "system";
  const systemProxyQuery = useSystemProxyInfo(proxyMode === "system");
  const highlightShadow = "0 0 0 1px hsl(var(--border)), 0 0 0 3px hsl(var(--primary))";
  const { refetch: refetchSystemProxy, isFetching: isSystemProxyFetching } = systemProxyQuery;
  const resetProxyTestState = React.useCallback((next: ProxySettings): ProxySettings => ({
    ...next,
    testSuccess: false,
    testMessage: "",
    testedAt: "",
  }), []);
  const formatHostPort = React.useCallback((host: string, port: number) => {
    if (!host || port <= 0) {
      return "";
    }
    const normalizedHost = host.includes(":") && !host.startsWith("[") ? `[${host}]` : host;
    return `${normalizedHost}:${port}`;
  }, []);
  const savedProxy = settings?.proxy;
  const manualProxyAddress = React.useMemo(() => {
    const hostPort = formatHostPort((savedProxy?.host || "").trim(), savedProxy?.port ?? 0);
    if (!hostPort) {
      return "";
    }
    const scheme = savedProxy?.scheme || "http";
    return `${scheme}://${hostPort}`;
  }, [formatHostPort, savedProxy?.host, savedProxy?.port, savedProxy?.scheme]);
  const systemProxyInfo = systemProxyQuery.data;
  const systemProxyRaw = (systemProxyInfo?.address || "").trim();
  const isVPNSource = systemProxyInfo?.source === "vpn";
  const shouldHideSystemAddress = isVPNSource && !systemProxyRaw;
  const systemProxyDisplay = shouldHideSystemAddress
    ? ""
    : systemProxyQuery.isLoading
      ? t("settings.general.proxy.statusLoading")
      : systemProxyQuery.isError
        ? t("settings.general.proxy.statusUnavailable")
        : systemProxyRaw || t("settings.general.proxy.statusNotConfigured");
  const manualProxyDisplay = manualProxyAddress || t("settings.general.proxy.statusNotConfigured");
  const statusAddress = proxyMode === "system" ? systemProxyRaw : proxyMode === "manual" ? manualProxyAddress : "";
  const statusAddressDisplay =
    proxyMode === "system" ? systemProxyDisplay : proxyMode === "manual" ? manualProxyDisplay : "";
  const hasStatusAddress = statusAddress !== "";
  const statusKey = hasStatusAddress ? `${proxyMode}:${statusAddress}` : "";
  const statusLabel =
    proxyCheckStatus === "available"
      ? t("settings.general.proxy.availability.available")
      : proxyCheckStatus === "unavailable"
        ? t("settings.general.proxy.availability.unavailable")
        : t("settings.general.proxy.availability.check");
  const statusDotClass =
    proxyCheckStatus === "available"
      ? "bg-emerald-500"
      : proxyCheckStatus === "unavailable"
        ? "bg-destructive"
        : "bg-muted-foreground/40";
  const isChecking = proxyCheckStatus === "checking" && proxyCheckKey === statusKey;
  const showRefreshButton = proxyMode === "system" || hasStatusAddress;
  const isStatusRefreshing = proxyMode === "system" ? isSystemProxyFetching || isChecking : isChecking;
  const hasTested = proxyDraft?.testedAt && proxyDraft.testedAt !== "0001-01-01T00:00:00Z";
  const testedAt = hasTested ? new Date(proxyDraft?.testedAt ?? "") : null;
  const hostFilled = (proxyDraft?.host || "").trim() !== "";
  const manualReady = proxyMode === "manual" && hostFilled && (proxyDraft?.port ?? 0) > 0;
  const manualButtonLabel = proxyBusy
    ? t("settings.general.proxy.testing")
    : proxyDraft?.testSuccess
      ? t("settings.general.proxy.saved")
      : t("settings.general.proxy.test");
  const testFeedback = proxyDraft?.testSuccess && testedAt
    ? `${t("settings.general.proxy.testedAtLabel")}: ${testedAt.toLocaleString()}`
    : proxyDraft?.testMessage
      ? proxyDraft.testMessage
      : "";
  const testFeedbackClass =
    proxyDraft?.testMessage && !proxyDraft.testSuccess ? "text-destructive" : "text-muted-foreground";
  const currentLanguageLabel =
    supportedLanguages.find((option) => option.value === (settings?.language || "en"))?.label ?? (settings?.language || "en");
  const productModeReady = typeof assistantUiEnabled === "boolean";
  const currentModeLabel = t(assistantUiEnabled ? "productMode.options.full.title" : "productMode.options.download.title");
  const setupWelcomeTitle = t("setupCenter.welcomeTitle");
  const currentUserName = resolveUserDisplayName(currentUserProfile);
  const currentUserSubtitle = resolveUserSubtitle(currentUserProfile) || t("productMode.profileHint");
  const requiredTools = assistantUiEnabled ? [...FULL_MODE_REQUIRED_TOOLS] : [...DOWNLOAD_MODE_REQUIRED_TOOLS];
  const missingRequiredTools = React.useMemo(
    () => resolveToolDependencyIssues(requiredTools, externalToolsQuery.data ?? []),
    [externalToolsQuery.data, requiredTools]
  );
  const proxyReady = resolveProxyStatus(settings?.proxy, proxyDraft) === "ready";
  const missingRequiredToolNames = React.useMemo(
    () => missingRequiredTools.map((item) => item.name),
    [missingRequiredTools]
  );
  const navStatusEntries = React.useMemo(
    () => buildSetupNavStatusEntries({
      languageReady: Boolean(settings?.language?.trim()),
      proxyReady,
      providersReady: status.providersReady,
      gatewayEnabled: status.gatewayEnabled,
      agentModelReady: status.agentModelReady,
      hasChosenProductMode: productModeReady,
      requiredTools,
      missingRequiredToolNames,
      skippedItemIds,
    }),
    [
      missingRequiredToolNames,
      productModeReady,
      proxyReady,
      requiredTools,
      settings?.language,
      skippedItemIds,
      status.agentModelReady,
      status.gatewayEnabled,
      status.providersReady,
    ]
  );
  const navStatusMap = React.useMemo(
    () => new Map<SetupNavItemId, SetupNavStatus>(navStatusEntries.map((entry) => [entry.id, entry.status])),
    [navStatusEntries]
  );

  const navGroups = React.useMemo<SetupNavGroup[]>(() => {
    const generalItems: SetupNavItem[] = [
      {
        id: "general.language",
        label: t("setupCenter.general.language"),
        page: "general",
        status: navStatusMap.get("general.language") ?? "pending",
        value: currentLanguageLabel,
      },
      {
        id: "general.proxy",
        label: t("setupCenter.general.proxy"),
        page: "general",
        status: navStatusMap.get("general.proxy") ?? "pending",
        value: t(`settings.general.proxy.option.${proxyMode}`),
      },
    ];
    const aiItems: SetupNavItem[] = [
      {
        id: "ai.provider",
        label: t("setupCenter.ai.provider"),
        page: "ai",
        status: navStatusMap.get("ai.provider") ?? "pending",
        iconOnly: true,
      },
      {
        id: "ai.agentModel",
        label: t("setupCenter.ai.gatewayAssistantNav"),
        page: "ai",
        status: navStatusMap.get("ai.agentModel") ?? "pending",
        iconOnly: true,
      },
    ];

    const dependencyItems: SetupNavItem[] = [
      {
        id: "dependencies.productMode",
        label: t("setupCenter.dependencies.modeTitle"),
        page: "dependencies",
        status: navStatusMap.get("dependencies.productMode") ?? "pending",
        value: currentModeLabel,
      },
      ...requiredTools.map((name) => ({
        id: getToolItemId(name),
        label: name.toUpperCase(),
        page: "dependencies" as const,
        status: navStatusMap.get(getToolItemId(name)) ?? "pending",
        iconOnly: true,
      })),
    ];

    return [
      {
        id: "general",
        title: t("setupCenter.steps.general.title"),
        status: resolveGroupStatus(generalItems.map((item) => item.status)),
        items: generalItems,
      },
      {
        id: "ai",
        title: t("setupCenter.steps.ai.title"),
        status: resolveGroupStatus(aiItems.map((item) => item.status)),
        items: aiItems,
      },
      {
        id: "dependencies",
        title: t("setupCenter.steps.dependencies.title"),
        status: resolveGroupStatus(dependencyItems.map((item) => item.status)),
        items: dependencyItems,
      },
    ];
  }, [
    currentLanguageLabel,
    currentModeLabel,
    navStatusMap,
    proxyMode,
    requiredTools,
    t,
  ]);

  const orderedNavItems = React.useMemo(
    () => navGroups.flatMap((group) => group.items.map((item) => item.id)),
    [navGroups]
  );
  const navItems = React.useMemo(
    () => navGroups.flatMap((group) => group.items),
    [navGroups]
  );
  const activePage = resolveItemPage(activeItem);
  const pendingNavItemIds = React.useMemo(
    () => navStatusEntries.filter((entry) => entry.status === "pending").map((entry) => entry.id),
    [navStatusEntries]
  );
  const pendingCount = pendingNavItemIds.length;
  const allowClose = pendingCount === 0;
  const skippablePendingItemIds = React.useMemo(
    () => pendingNavItemIds.filter((itemId) => isSkippableItem(itemId)),
    [pendingNavItemIds]
  );
  const canSkipAndClose = pendingCount > 0 && skippablePendingItemIds.length === pendingCount;
  const firstPendingItemId = pendingNavItemIds[0] ?? orderedNavItems[0] ?? "general.language";
  const navItemMap = React.useMemo(
    () => new Map<SetupNavItemId, SetupNavItem>(navItems.map((item) => [item.id, item])),
    [navItems]
  );
  const getSectionRef = React.useCallback((itemId: SetupNavItemId) => {
    const existing = sectionRefCallbacks.current[itemId];
    if (existing) {
      return existing;
    }
    const callback = (node: HTMLDivElement | null) => {
      sectionRefs.current[itemId] = node;
    };
    sectionRefCallbacks.current[itemId] = callback;
    return callback;
  }, []);

  React.useEffect(() => {
    if (!open || orderedNavItems.includes(activeItem)) {
      return;
    }
    setActiveItem(firstPendingItemId);
  }, [activeItem, firstPendingItemId, open, orderedNavItems]);

  React.useEffect(() => {
    if (open && !wasOpenRef.current) {
      const nextActiveItem =
        focusItemId && orderedNavItems.includes(focusItemId) ? focusItemId : firstPendingItemId;
      setActiveItem(nextActiveItem);
      const frame = window.requestAnimationFrame(() => {
        sectionRefs.current[nextActiveItem]?.scrollIntoView({ behavior: "auto", block: "start" });
      });
      clearFocusItem();
      wasOpenRef.current = true;
      return () => window.cancelAnimationFrame(frame);
    }
    if (!open) {
      wasOpenRef.current = false;
    }
  }, [clearFocusItem, firstPendingItemId, focusItemId, open, orderedNavItems]);

  React.useEffect(() => {
    if (!settings?.proxy) {
      return;
    }
    setProxyDraft((current) => {
      if (settings.proxy.mode === "manual") {
        return current?.mode === "manual" ? current : settings.proxy;
      }
      return current?.mode === "manual" ? null : current;
    });
  }, [settings?.proxy]);

  React.useEffect(() => {
    if (proxyMode === "system") {
      void refetchSystemProxy();
    }
  }, [proxyMode, refetchSystemProxy]);

  React.useEffect(() => {
    if (!open) {
      return;
    }
    const container = contentScrollRef.current;
    if (!container) {
      return;
    }
    const syncActiveItem = () => {
      const containerTop = container.getBoundingClientRect().top;
      const offset = 96;
      let nextItem = orderedNavItems[0] ?? activeItem;
      for (const itemId of orderedNavItems) {
        const node = sectionRefs.current[itemId];
        if (!node) {
          continue;
        }
        const rect = node.getBoundingClientRect();
        if (rect.top - containerTop <= offset) {
          nextItem = itemId;
          continue;
        }
        break;
      }
      if (nextItem !== activeItem) {
        setActiveItem(nextItem);
      }
    };

    const frame = window.requestAnimationFrame(syncActiveItem);

    container.addEventListener("scroll", syncActiveItem, { passive: true });
    return () => {
      window.cancelAnimationFrame(frame);
      container.removeEventListener("scroll", syncActiveItem);
    };
  }, [activeItem, open, orderedNavItems]);

  const handleClose = (nextOpen: boolean) => {
    if (!nextOpen && !allowClose) {
      return;
    }
    onOpenChange(nextOpen);
  };

  const handleSelectItem = React.useCallback((itemId: SetupNavItemId) => {
    setActiveItem(itemId);
    sectionRefs.current[itemId]?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);

  const handleProxyModeChange = async (mode: ProxySettings["mode"]) => {
    if (!savedProxy) {
      return;
    }
    if (mode === "manual") {
      setProxyDraft((current) =>
        current?.mode === "manual"
          ? current
          : resetProxyTestState({
              ...createManualProxyDraft(savedProxy),
              mode: "manual",
            })
      );
      return;
    }
    await updateSettings.mutateAsync({
      proxy: {
        ...savedProxy,
        mode,
        host: "",
        port: 0,
        username: "",
        password: "",
        testedAt: "",
        testSuccess: false,
        testMessage: "",
      },
    });
    setProxyDialogOpen(false);
    setProxyDraft(null);
  };

  const handleProxyFieldChange = (field: keyof ProxySettings, value: string) => {
    const isNumericField = field === "port" || field === "timeoutSeconds";
    setProxyDraft((current) =>
      current
        ? resetProxyTestState({
            ...current,
            [field]: isNumericField ? Number(value) || 0 : value,
          } as ProxySettings)
        : current
    );
  };

  const handleProxyDialogOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      if (!savedProxy) {
        return;
      }
      setProxyDraft((current) =>
        current?.mode === "manual"
          ? current
          : resetProxyTestState({
              ...savedProxy,
              mode: "manual",
            })
      );
    } else {
      setClearConfirmOpen(false);
    }
    setProxyDialogOpen(nextOpen);
  };

  const handleProxyClear = async () => {
    if (!savedProxy) {
      return;
    }
    const cleared = resetProxyTestState({
      ...savedProxy,
      mode: "none",
      scheme: savedProxy.scheme || "http",
      host: "",
      port: 0,
      username: "",
      password: "",
      noProxy: [],
      timeoutSeconds: savedProxy.timeoutSeconds || 0,
    });
    await updateSettings.mutateAsync({ proxy: cleared });
    setProxyDraft(null);
    setProxyDialogOpen(false);
  };

  const handleProxyClearConfirm = () => {
    setClearConfirmOpen(false);
    void handleProxyClear();
  };

  const handleProxySave = async () => {
    if (!proxyDraft) {
      return;
    }
    setProxyBusy(true);
    try {
      const tested = await testProxy.mutateAsync(proxyDraft);
      await updateSettings.mutateAsync({ proxy: tested });
      setProxyDraft(tested);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error ?? "");
      setProxyDraft((current) =>
        current
          ? {
              ...current,
              testedAt: "",
              testSuccess: false,
              testMessage: message,
            }
          : current
      );
    } finally {
      setProxyBusy(false);
    }
  };

  const buildProxyTestPayload = React.useCallback((mode: ProxySettings["mode"]) => {
    if (!savedProxy) {
      return null;
    }
    const base = resetProxyTestState(savedProxy);
    if (mode === "system") {
      return {
        ...base,
        mode,
        host: "",
        port: 0,
        username: "",
        password: "",
      };
    }
    return {
      ...base,
      mode,
    };
  }, [resetProxyTestState, savedProxy]);

  const runProxyCheck = React.useCallback(
    async (mode: ProxySettings["mode"], address: string) => {
      if (mode === "none" || !address) {
        return;
      }
      const payload = buildProxyTestPayload(mode);
      if (!payload) {
        return;
      }
      const key = `${mode}:${address}`;
      proxyCheckRequestRef.current += 1;
      const requestId = proxyCheckRequestRef.current;
      setProxyCheckKey(key);
      setProxyCheckStatus("checking");
      try {
        const result = await testProxy.mutateAsync(payload);
        if (proxyCheckRequestRef.current !== requestId) {
          return;
        }
        if (result.testSuccess) {
          setProxyCheckStatus("available");
          return;
        }
        setProxyCheckStatus("unavailable");
        if (result.testMessage) {
          messageBus.publishToast({
            title: t("settings.general.proxy.checkFailed"),
            description: result.testMessage,
            intent: "warning",
          });
        }
      } catch (error) {
        if (proxyCheckRequestRef.current !== requestId) {
          return;
        }
        const message = error instanceof Error ? error.message : String(error);
        setProxyCheckStatus("unavailable");
        messageBus.publishToast({
          title: t("settings.general.proxy.checkFailed"),
          description: message,
          intent: "warning",
        });
      }
    },
    [buildProxyTestPayload, t, testProxy]
  );

  const handleStatusRefresh = React.useCallback(async () => {
    if (proxyMode === "system") {
      try {
        const result = await refetchSystemProxy();
        const nextAddress = (result.data?.address || "").trim();
        if (nextAddress) {
          void runProxyCheck("system", nextAddress);
        } else {
          setProxyCheckStatus("idle");
          setProxyCheckKey("");
        }
      } catch {
        setProxyCheckStatus("idle");
        setProxyCheckKey("");
      }
      return;
    }
    if (hasStatusAddress) {
      void runProxyCheck(proxyMode, statusAddress);
    }
  }, [hasStatusAddress, proxyMode, refetchSystemProxy, runProxyCheck, statusAddress]);

  React.useEffect(() => {
    if (proxyMode === "none") {
      setProxyCheckStatus("idle");
      setProxyCheckKey("");
      return;
    }
    if (!hasStatusAddress) {
      setProxyCheckStatus("idle");
      setProxyCheckKey("");
      return;
    }
    if (proxyCheckKey === statusKey && proxyCheckStatus !== "idle") {
      return;
    }
    void runProxyCheck(proxyMode, statusAddress);
  }, [hasStatusAddress, proxyCheckKey, proxyCheckStatus, proxyMode, runProxyCheck, statusAddress, statusKey]);

  const handleSkipItem = React.useCallback(
    (itemId: SetupNavItemId) => {
      if (!isSkippableItem(itemId)) {
        return;
      }
      skipItem(itemId);
      if (itemId.startsWith("ai.")) {
        deferAi();
      }
      if (itemId.startsWith(TOOL_ITEM_PREFIX)) {
        deferDependencies();
      }
    },
    [deferAi, deferDependencies, skipItem]
  );

  const closeDisabled = pendingCount > 0 && !canSkipAndClose;
  const closeButtonLabel = allowClose
    ? t("setupCenter.footer.enter")
    : canSkipAndClose
      ? formatTemplate(t("setupCenter.footer.skipAndEnter"), { count: pendingCount })
      : t("setupCenter.footer.close");

  const handleFooterClose = () => {
    if (allowClose) {
      onOpenChange(false);
      return;
    }
    if (!canSkipAndClose) {
      return;
    }
    for (const itemId of skippablePendingItemIds) {
      handleSkipItem(itemId);
    }
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DashboardDialogContent
        size="workspace"
        showCloseButton={false}
        className="flex h-[88vh] min-h-0 w-full max-h-[88vh] flex-col overflow-hidden p-0 outline-none focus:outline-none focus-visible:outline-none focus-visible:ring-0"
        onEscapeKeyDown={!allowClose ? (event) => event.preventDefault() : undefined}
        onPointerDownOutside={!allowClose ? (event) => event.preventDefault() : undefined}
        onInteractOutside={!allowClose ? (event) => event.preventDefault() : undefined}
        onOpenAutoFocus={(event) => event.preventDefault()}
      >
        <div className="flex h-full min-h-0 flex-col bg-background">
          <header className="flex items-center justify-between gap-4 border-b border-border/70 px-6 py-4 text-muted-foreground">
            <div className="min-w-0">
              <DialogTitle className="flex items-center gap-2 text-xl font-semibold text-muted-foreground">
                <Sparkles className="h-4 w-4 text-primary" />
                <span>{setupWelcomeTitle}</span>
              </DialogTitle>
            </div>
            <div className="flex min-w-0 items-center gap-3">
              <UserAvatar
                profile={currentUserProfile}
                className="h-8 w-8 rounded-xl"
                fallbackClassName="text-[10px]"
              />
              <div className="min-w-0 flex-1 text-left">
                <div className="truncate text-sm font-semibold leading-tight text-muted-foreground">{currentUserName}</div>
                <div className="truncate text-xs leading-tight text-muted-foreground">{currentUserSubtitle}</div>
              </div>
            </div>
          </header>

          <div className="flex min-h-0 flex-1">
            <aside className="flex w-64 shrink-0 flex-col overflow-hidden px-3 py-4 text-muted-foreground">
              <div className="min-h-0 flex-1 overflow-y-auto px-1">
                <nav className="space-y-4">
                  {navGroups.map((group) => {
                    const groupActive = activePage === group.id;
                    return (
                      <div key={group.id} className="space-y-1.5">
                        <button
                          type="button"
                          className={cn(
                            "flex w-full items-center gap-2 px-2 text-left text-lg font-medium transition-colors",
                            groupActive ? "text-muted-foreground" : "text-muted-foreground hover:text-muted-foreground"
                          )}
                          onClick={() => handleSelectItem(resolveStepDefaultItem(group.id))}
                        >
                          <SetupStatusIcon status={group.status} className="h-3.5 w-3.5" />
                          <span className="min-w-0 flex-1 truncate">{group.title}</span>
                        </button>

                        <SidebarMenu className="gap-1">
                          {group.items.map((item) => (
                            <SidebarMenuItem key={item.id}>
                              <SidebarMenuButton
                                isActive={activeItem === item.id}
                                className="justify-start gap-2 text-muted-foreground hover:text-muted-foreground data-[active=true]:text-muted-foreground"
                                onClick={() => handleSelectItem(item.id)}
                              >
                                <span
                                  className={cn(
                                    "min-w-0 flex-1 truncate text-left",
                                    item.id.startsWith(TOOL_ITEM_PREFIX) && "text-xs font-semibold uppercase tracking-[0.24em]"
                                  )}
                                >
                                  {item.label}
                                </span>
                                {item.iconOnly ? (
                                  <SetupStatusIcon status={item.status} className="shrink-0" />
                                ) : (
                                  <SetupValueBadge className="max-w-[42%] shrink-0">{item.value}</SetupValueBadge>
                                )}
                              </SidebarMenuButton>
                            </SidebarMenuItem>
                          ))}
                        </SidebarMenu>
                      </div>
                    );
                  })}
                </nav>
              </div>
            </aside>

            <Separator orientation="vertical" />

            <section ref={contentScrollRef} className="min-h-0 flex-1 overflow-y-auto px-6 py-5 text-muted-foreground">
              <div className="space-y-8">
                <SetupContentSection
                  title={t("setupCenter.steps.general.title")}
                  description={t("setupCenter.steps.general.description")}
                >
                  <div ref={getSectionRef("general.language")}>
                    <SetupPageCard
                      title={t("setupCenter.general.language")}
                      headerRight={
                        <SetupCardValue>{navItemMap.get("general.language")?.value ?? currentLanguageLabel}</SetupCardValue>
                      }
                    >
                      <SetupCardRows>
                        <SetupCardRow label={t("setupCenter.general.language")}>
                          <Select
                            value={settings?.language || "en"}
                            className="h-9 min-w-[14rem] text-xs"
                            onChange={(event) => updateSettings.mutate({ language: event.target.value })}
                          >
                            {supportedLanguages.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </Select>
                        </SetupCardRow>
                      </SetupCardRows>
                    </SetupPageCard>
                  </div>

                  <div ref={getSectionRef("general.proxy")}>
                    <SetupPageCard
                      title={t("setupCenter.general.proxy")}
                      headerRight={
                        <SetupCardValue>{navItemMap.get("general.proxy")?.value ?? t(`settings.general.proxy.option.${proxyMode}`)}</SetupCardValue>
                      }
                    >
                      <SetupCardRows>
                        <SetupCardRow label={t("settings.general.proxy.title")}>
                          <div className="flex flex-wrap justify-end gap-2">
                            {(["none", "system", "manual"] as ProxySettings["mode"][]).map((mode) => (
                              <Button
                                key={mode}
                                type="button"
                                variant="outline"
                                size="compact"
                                onClick={() => void handleProxyModeChange(mode)}
                                style={proxyMode === mode ? { boxShadow: highlightShadow } : undefined}
                              >
                                {t(`settings.general.proxy.option.${mode}`)}
                              </Button>
                            ))}
                          </div>
                        </SetupCardRow>

                        {proxyMode !== "none" ? (
                          <>
                            <SetupCardSeparator />
                            <SetupCardRow label={t("settings.general.proxy.status")} contentClassName="min-w-0">
                              <div className="flex min-w-0 items-center justify-end gap-2">
                                {statusAddressDisplay ? (
                                  <span className="max-w-[260px] truncate text-right font-mono text-xs text-muted-foreground">
                                    {statusAddressDisplay}
                                  </span>
                                ) : null}
                                {hasStatusAddress ? (
                                  <span className="inline-flex items-center">
                                    <span
                                      className={`h-2 w-2 rounded-full ${statusDotClass} ${proxyCheckStatus === "checking" ? "animate-pulse" : ""}`}
                                      aria-hidden="true"
                                    />
                                    <span className="sr-only">{statusLabel}</span>
                                  </span>
                                ) : null}
                                {showRefreshButton ? (
                                  <TooltipProvider delayDuration={0}>
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <Button
                                          type="button"
                                          variant="outline"
                                          size="compactIcon"
                                          disabled={isStatusRefreshing}
                                          onClick={() => void handleStatusRefresh()}
                                          aria-label={t("settings.general.proxy.check")}
                                        >
                                          {isStatusRefreshing ? (
                                            <Loader2 className="h-4 w-4 animate-spin" />
                                          ) : (
                                            <RefreshCw className="h-4 w-4" />
                                          )}
                                        </Button>
                                      </TooltipTrigger>
                                      <TooltipContent side="top">
                                        {t("settings.general.proxy.check")}
                                      </TooltipContent>
                                    </Tooltip>
                                  </TooltipProvider>
                                ) : null}
                                {proxyMode === "manual" ? (
                                  <TooltipProvider delayDuration={0}>
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <Button
                                          type="button"
                                          variant="outline"
                                          size="compactIcon"
                                          onClick={() => handleProxyDialogOpenChange(true)}
                                          aria-label={t("settings.general.proxy.change")}
                                        >
                                          <Pencil className="h-4 w-4" />
                                        </Button>
                                      </TooltipTrigger>
                                      <TooltipContent side="top">
                                        {t("settings.general.proxy.change")}
                                      </TooltipContent>
                                    </Tooltip>
                                  </TooltipProvider>
                                ) : null}
                              </div>
                            </SetupCardRow>
                          </>
                        ) : null}
                      </SetupCardRows>

                      {proxyMode === "manual" ? (
                        <>
                          <Dialog open={proxyDialogOpen} onOpenChange={handleProxyDialogOpenChange}>
                            <DialogContent>
                              <DialogHeader>
                                <DialogTitle>{t("settings.general.proxy.dialogTitle")}</DialogTitle>
                                <DialogDescription>{t("settings.general.proxy.testHint")}</DialogDescription>
                              </DialogHeader>
                              <div className="grid grid-cols-2 gap-3">
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.scheme")}</span>
                                  <Select
                                    value={proxyDraft?.scheme ?? "http"}
                                    onChange={(event) => handleProxyFieldChange("scheme", event.target.value)}
                                    className="w-full"
                                  >
                                    <option value="http">{t("settings.general.proxy.schemeOption.http")}</option>
                                    <option value="https">{t("settings.general.proxy.schemeOption.https")}</option>
                                    <option value="socks5">{t("settings.general.proxy.schemeOption.socks5")}</option>
                                  </Select>
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.timeout")}</span>
                                  <Input
                                    type="number"
                                    inputMode="numeric"
                                    value={proxyDraft?.timeoutSeconds || ""}
                                    onChange={(event) => handleProxyFieldChange("timeoutSeconds", event.target.value)}
                                    placeholder="30"
                                    size="compact"
                                    className="text-sm"
                                  />
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.host")}</span>
                                  <Input
                                    value={proxyDraft?.host ?? ""}
                                    onChange={(event) => handleProxyFieldChange("host", event.target.value)}
                                    placeholder="127.0.0.1"
                                    size="compact"
                                    className="text-sm"
                                  />
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.port")}</span>
                                  <Input
                                    type="number"
                                    inputMode="numeric"
                                    value={proxyDraft?.port || ""}
                                    onChange={(event) => handleProxyFieldChange("port", event.target.value)}
                                    placeholder="8080"
                                    size="compact"
                                    className="text-sm"
                                  />
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.username")}</span>
                                  <Input
                                    value={proxyDraft?.username ?? ""}
                                    onChange={(event) => handleProxyFieldChange("username", event.target.value)}
                                    placeholder={t("settings.general.proxy.usernamePlaceholder")}
                                    size="compact"
                                    className="text-sm"
                                  />
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="text-sm text-muted-foreground">{t("settings.general.proxy.password")}</span>
                                  <Input
                                    type="password"
                                    value={proxyDraft?.password ?? ""}
                                    onChange={(event) => handleProxyFieldChange("password", event.target.value)}
                                    placeholder={t("settings.general.proxy.passwordPlaceholder")}
                                    size="compact"
                                    className="text-sm"
                                  />
                                </div>
                              </div>
                              <div className="flex flex-col gap-2 pt-4 sm:flex-row sm:items-center sm:justify-between">
                                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                                  <div className={`text-sm ${testFeedback ? testFeedbackClass : ""}`}>
                                    {testFeedback}
                                  </div>
                                  <Button
                                    type="button"
                                    size="compact"
                                    variant="destructive"
                                    disabled={proxyBusy}
                                    onClick={() => setClearConfirmOpen(true)}
                                  >
                                    {t("settings.general.proxy.clear")}
                                  </Button>
                                </div>
                                <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end">
                                  <DialogClose asChild>
                                    <Button size="compact" variant="outline">
                                      {t("settings.general.proxy.close")}
                                    </Button>
                                  </DialogClose>
                                  <Button
                                    type="button"
                                    size="compact"
                                    variant={proxyDraft?.testSuccess ? "secondary" : "outline"}
                                    disabled={!manualReady || proxyBusy}
                                    onClick={() => void handleProxySave()}
                                  >
                                    {proxyBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                                    {manualButtonLabel}
                                  </Button>
                                </div>
                              </div>
                            </DialogContent>
                          </Dialog>

                          <Dialog open={clearConfirmOpen} onOpenChange={setClearConfirmOpen}>
                            <DialogContent className="max-w-sm">
                              <DialogHeader>
                                <DialogTitle>{t("settings.general.proxy.clearConfirm.title")}</DialogTitle>
                                <DialogDescription>{t("settings.general.proxy.clearConfirm.description")}</DialogDescription>
                              </DialogHeader>
                              <div className="flex items-center justify-end gap-2">
                                <DialogClose asChild>
                                  <Button size="compact" variant="outline">
                                    {t("settings.general.proxy.clearConfirm.cancel")}
                                  </Button>
                                </DialogClose>
                                <Button type="button" size="compact" variant="destructive" onClick={handleProxyClearConfirm}>
                                  {t("settings.general.proxy.clearConfirm.confirm")}
                                </Button>
                              </div>
                            </DialogContent>
                          </Dialog>
                        </>
                      ) : null}
                    </SetupPageCard>
                  </div>
                </SetupContentSection>

                <SetupContentSection
                  title={t("setupCenter.steps.ai.title")}
                  description={t("setupCenter.steps.ai.description")}
                >
                  <div ref={getSectionRef("ai.provider")}>
                    <SetupProviderCard
                      status={navItemMap.get("ai.provider")?.status ?? "pending"}
                      showSkip={(navItemMap.get("ai.provider")?.status ?? "pending") === "pending"}
                      onSkip={() => handleSkipItem("ai.provider")}
                    />
                  </div>

                  <div ref={getSectionRef("ai.agentModel")}>
                    <SetupAgentModelCard
                      status={navItemMap.get("ai.agentModel")?.status ?? "pending"}
                      showSkip={(navItemMap.get("ai.agentModel")?.status ?? "pending") === "pending"}
                      onSkip={() => handleSkipItem("ai.agentModel")}
                      autoApplyWhenReady={open}
                    />
                  </div>
                </SetupContentSection>

                <SetupContentSection
                  title={t("setupCenter.steps.dependencies.title")}
                  description={t("setupCenter.steps.dependencies.description")}
                >
                  <div ref={getSectionRef("dependencies.productMode")}>
                    <SetupProductModeCard
                      status={navItemMap.get("dependencies.productMode")?.status ?? "pending"}
                      headerRight={
                        <SetupCardValue>{navItemMap.get("dependencies.productMode")?.value ?? currentModeLabel}</SetupCardValue>
                      }
                    />
                  </div>

                  {requiredTools.map((name) => {
                    const itemId = getToolItemId(name);
                    return (
                      <div key={name} ref={getSectionRef(itemId)}>
                        <SetupToolCard
                          name={name}
                          status={navItemMap.get(itemId)?.status ?? "pending"}
                          showSkip={(navItemMap.get(itemId)?.status ?? "pending") === "pending"}
                          onSkip={() => handleSkipItem(itemId)}
                          activeInstallName={activeInstallName}
                          activeVerifyName={activeVerifyName}
                          onActiveInstallNameChange={setActiveInstallName}
                          onActiveVerifyNameChange={setActiveVerifyName}
                        />
                      </div>
                    );
                  })}
                </SetupContentSection>
              </div>
            </section>
          </div>

          <footer className="flex items-center justify-center border-t border-border/70 px-6 py-3">
            <Button
              type="button"
              size="compact"
              variant="outline"
              disabled={closeDisabled}
              onClick={handleFooterClose}
            >
              <Sparkles className="h-3.5 w-3.5" />
              {closeButtonLabel}
            </Button>
          </footer>
        </div>
      </DashboardDialogContent>
    </Dialog>
  );
}

function SetupContentSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="space-y-4">
      <div className="space-y-1 px-1">
        <div className="text-lg font-medium text-muted-foreground">{title}</div>
        {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  );
}
