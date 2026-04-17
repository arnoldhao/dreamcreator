import * as React from "react";
import { CircleOff, ExternalLink, Eye, Link2, Loader2, Plug2, RefreshCw, Search, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import {
  SETTINGS_ROW_CLASS,
  SETTINGS_ROW_LABEL_CLASS,
  SettingsSeparator,
} from "@/shared/ui/settings-layout";
import { useI18n } from "@/shared/i18n";
import {
  useCancelConnectorConnect,
  useClearConnector,
  useConnectorConnectSession,
  useFinishConnectorConnect,
  useConnectors,
  useOpenConnectorSite,
  useStartConnectorConnect,
} from "@/shared/query/connectors";
import { messageBus } from "@/shared/message";
import type { Connector, ConnectorConnectSession, FinishConnectorConnectResult } from "@/shared/contracts/connectors";
import { cn } from "@/lib/utils";

const STATUS_META: Record<string, { label: string; className: string; icon: React.ComponentType<{ className?: string }> }> = {
  connected: {
    label: "Connected",
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-100",
    icon: Plug2,
  },
  expired: {
    label: "Expired",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-100",
    icon: RefreshCw,
  },
  disconnected: {
    label: "Disconnected",
    className: "bg-muted text-muted-foreground",
    icon: CircleOff,
  },
};

type ConnectorGroup = "search_engine" | "community" | "video" | "developer" | "other";

type ConnectorMeta = {
  group: ConnectorGroup;
  labelKey: string;
  fallbackLabel: string;
};

const CONNECTOR_GROUP_ORDER: ConnectorGroup[] = ["search_engine", "community", "video", "developer", "other"];

const CONNECTOR_META: Record<string, ConnectorMeta> = {
  google: { group: "search_engine", labelKey: "settings.connectors.item.google", fallbackLabel: "Google" },
  github: { group: "developer", labelKey: "settings.connectors.item.github", fallbackLabel: "GitHub" },
  reddit: { group: "community", labelKey: "settings.connectors.item.reddit", fallbackLabel: "Reddit" },
  zhihu: { group: "search_engine", labelKey: "settings.connectors.item.zhihu", fallbackLabel: "Zhihu" },
  x: { group: "community", labelKey: "settings.connectors.item.x", fallbackLabel: "X" },
  xiaohongshu: { group: "community", labelKey: "settings.connectors.item.xiaohongshu", fallbackLabel: "Xiaohongshu" },
  bilibili: { group: "video", labelKey: "settings.connectors.item.bilibili", fallbackLabel: "Bilibili" },
};

const GENERAL_CARD_HEIGHT = "min-h-[240px]";

const formatCookieExpires = (expires?: number) => {
  if (!expires || expires <= 0) {
    return "-";
  }
  const date = new Date(expires * 1000);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString();
};

const resolveConnectorMeta = (connectorType: string): ConnectorMeta | null => {
  const normalized = connectorType.trim().toLowerCase();
  if (!normalized) {
    return null;
  }
  return CONNECTOR_META[normalized] ?? null;
};

const resolveConnectorGroup = (connector: Connector): ConnectorGroup => {
  const rawGroup = connector.group?.trim().toLowerCase();
  if (rawGroup === "search_engine" || rawGroup === "community" || rawGroup === "video" || rawGroup === "developer" || rawGroup === "other") {
    return rawGroup;
  }
  const meta = resolveConnectorMeta(connector.type);
  return meta?.group ?? "other";
};

export function ConnectorsSection() {
  const { t } = useI18n();
  const connectors = useConnectors();
  const startConnectorConnect = useStartConnectorConnect();
  const finishConnectorConnect = useFinishConnectorConnect();
  const cancelConnectorConnect = useCancelConnectorConnect();
  const clearConnector = useClearConnector();
  const openConnectorSite = useOpenConnectorSite();

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [query, setQuery] = React.useState("");
  const [loginDialogOpen, setLoginDialogOpen] = React.useState(false);
  const [loginTarget, setLoginTarget] = React.useState<Connector | null>(null);
  const [loginSessionId, setLoginSessionId] = React.useState("");
  const [loginResult, setLoginResult] = React.useState<FinishConnectorConnectResult | null>(null);
  const [loginError, setLoginError] = React.useState("");
  const [cookiesDialogOpen, setCookiesDialogOpen] = React.useState(false);
  const loginSession = useConnectorConnectSession({ sessionId: loginSessionId }, loginDialogOpen && loginSessionId.trim().length > 0);

  const items = connectors.data ?? [];
  const resolveConnectorLabel = React.useCallback(
    (connector: Connector) => {
      const meta = resolveConnectorMeta(connector.type);
      if (!meta) {
        return connector.type;
      }
      return t(meta.labelKey);
    },
    [t]
  );

  const resolveConnectorGroupLabel = React.useCallback(
    (group: ConnectorGroup) => {
      switch (group) {
        case "search_engine":
          return t("settings.connectors.group.searchEngine");
        case "community":
          return t("settings.connectors.group.community");
        case "video":
          return t("settings.connectors.group.video");
        case "developer":
          return t("settings.connectors.group.developer");
        default:
          return t("settings.connectors.group.other");
      }
    },
    [t]
  );

  const trimmedQuery = query.trim().toLowerCase();
  const filteredItems = React.useMemo(() => {
    if (!trimmedQuery) {
      return items;
    }
    return items.filter((connector) => {
      const label = resolveConnectorLabel(connector).toLowerCase();
      const type = connector.type.toLowerCase();
      const groupLabel = resolveConnectorGroupLabel(resolveConnectorGroup(connector)).toLowerCase();
      return (
        label.includes(trimmedQuery) ||
        type.includes(trimmedQuery) ||
        groupLabel.includes(trimmedQuery)
      );
    });
  }, [items, resolveConnectorGroupLabel, resolveConnectorLabel, trimmedQuery]);

  const groupedItems = React.useMemo(() => {
    const buckets = new Map<ConnectorGroup, Connector[]>();
    filteredItems.forEach((connector) => {
      const group = resolveConnectorGroup(connector);
      const bucket = buckets.get(group) ?? [];
      bucket.push(connector);
      buckets.set(group, bucket);
    });

    const order = new Map<ConnectorGroup, number>();
    CONNECTOR_GROUP_ORDER.forEach((group, index) => {
      order.set(group, index);
    });

    return Array.from(buckets.entries())
      .sort((left, right) => {
        const leftOrder = order.get(left[0]) ?? Number.MAX_SAFE_INTEGER;
        const rightOrder = order.get(right[0]) ?? Number.MAX_SAFE_INTEGER;
        return leftOrder - rightOrder;
      })
      .map(([group, connectorsInGroup]) => {
        const sorted = [...connectorsInGroup].sort((left, right) =>
          resolveConnectorLabel(left).localeCompare(resolveConnectorLabel(right))
        );
        return {
          id: group,
          label: resolveConnectorGroupLabel(group),
          connectors: sorted,
        };
      });
  }, [filteredItems, resolveConnectorGroupLabel, resolveConnectorLabel]);

  React.useEffect(() => {
    if (selectedId && !items.some((item) => item.id === selectedId)) {
      setSelectedId(null);
    }
  }, [items, selectedId]);

  React.useEffect(() => {
    if (selectedId && filteredItems.some((item) => item.id === selectedId)) {
      return;
    }
    if (filteredItems.length > 0) {
      setSelectedId(filteredItems[0].id);
      return;
    }
    setSelectedId(null);
  }, [filteredItems, selectedId]);

  const selected = items.find((item) => item.id === selectedId) ?? null;
  const status = STATUS_META[selected?.status ?? "disconnected"] ?? STATUS_META.disconnected;

  const isBusy =
    startConnectorConnect.isPending ||
    finishConnectorConnect.isPending ||
    cancelConnectorConnect.isPending ||
    openConnectorSite.isPending ||
    clearConnector.isPending;
  const isLoginRunning =
    startConnectorConnect.isPending ||
    finishConnectorConnect.isPending ||
    cancelConnectorConnect.isPending;
  const isOpenRunning = openConnectorSite.isPending;

  const resolveLoginError = React.useCallback((error: unknown) => {
    const message = error instanceof Error ? error.message : String(error);
    if (message.toLowerCase().includes("no supported browser detected")) {
      return t("settings.connectors.browserMissing");
    }
    if (message.toLowerCase().includes("connector browser session ended")) {
      return t("settings.connectors.browserSessionEnded");
    }
    if (message.toLowerCase().includes("connector session not found")) {
      return t("settings.connectors.loginSessionMissing");
    }
    return error instanceof Error ? error.message : t("settings.connectors.loginError");
  }, [t]);

  const toLoginResult = React.useCallback((session: ConnectorConnectSession): FinishConnectorConnectResult => {
    return {
      sessionId: session.sessionId,
      saved: session.saved,
      rawCookiesCount: session.rawCookiesCount,
      filteredCookiesCount: session.filteredCookiesCount,
      domains: session.domains,
      reason: session.reason,
      connector: session.connector,
    };
  }, []);

  const disposeLoginSession = React.useCallback(async (sessionId: string) => {
    const trimmed = sessionId.trim();
    if (!trimmed) {
      return;
    }
    try {
      await cancelConnectorConnect.mutateAsync({ sessionId: trimmed });
    } catch {
      // ignore disposal failures; a fresh connect attempt will replace stale sessions
    }
  }, [cancelConnectorConnect]);

  const resetLoginState = React.useCallback(() => {
    setLoginDialogOpen(false);
    setLoginTarget(null);
    setLoginSessionId("");
    setLoginResult(null);
    setLoginError("");
  }, []);

  const handleCancelLogin = React.useCallback(async () => {
    const sessionId = loginSessionId.trim();
    if (sessionId) {
      try {
        await disposeLoginSession(sessionId);
      } catch (error) {
        messageBus.publishToast({
          intent: "danger",
          title: t("settings.connectors.loginTitle"),
          description: resolveLoginError(error),
        });
      }
    }
    resetLoginState();
  }, [disposeLoginSession, loginSessionId, resetLoginState, resolveLoginError, t]);

  const handleConnect = async (connector: Connector) => {
    setLoginTarget(connector);
    setLoginDialogOpen(true);
    setLoginSessionId("");
    setLoginResult(null);
    setLoginError("");
    try {
      const result = await startConnectorConnect.mutateAsync({ id: connector.id });
      setLoginSessionId(result.sessionId);
    } catch (error) {
      setLoginError(resolveLoginError(error));
    }
  };

  const handleFinishLogin = async () => {
    const sessionId = loginSessionId.trim();
    if (!sessionId) {
      setLoginError(t("settings.connectors.loginSessionMissing"));
      return;
    }
    setLoginError("");
    try {
      const result = await finishConnectorConnect.mutateAsync({ sessionId });
      setLoginResult(result);
      await disposeLoginSession(sessionId);
      setLoginSessionId("");
      if (!result.saved) {
        setLoginError(t("settings.connectors.noCookiesRead"));
        return;
      }
      resetLoginState();
    } catch (error) {
      setLoginError(resolveLoginError(error));
    }
  };

  React.useEffect(() => {
    const session = loginSession.data;
    if (!session || loginSessionId.trim().length === 0 || isLoginRunning) {
      return;
    }
    if (session.state === "running") {
      return;
    }

    const sessionId = session.sessionId;
    setLoginResult(toLoginResult(session));
    void disposeLoginSession(sessionId);
    setLoginSessionId("");

    if (session.state === "completed" && session.saved) {
      resetLoginState();
      return;
    }

    if (session.state === "completed") {
      setLoginError(t("settings.connectors.noCookiesRead"));
      return;
    }

    if (session.error) {
      setLoginError(session.error);
      return;
    }

    setLoginError(t("settings.connectors.loginError"));
  }, [disposeLoginSession, isLoginRunning, loginSession.data, loginSessionId, resetLoginState, t, toLoginResult]);

  const resolveOpenError = (error: unknown) => {
    const message = error instanceof Error ? error.message : String(error);
    if (message.toLowerCase().includes("no cookies")) {
      return t("settings.connectors.noCookies");
    }
    if (message.toLowerCase().includes("no supported browser detected")) {
      return t("settings.connectors.browserMissing");
    }
    return error instanceof Error ? error.message : t("settings.connectors.openSiteError");
  };

  const handleOpenSite = async (connector: Connector) => {
    try {
      await openConnectorSite.mutateAsync({ id: connector.id });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("settings.connectors.openSite"),
        description: resolveOpenError(error),
      });
    }
  };

  const rowClassName = SETTINGS_ROW_CLASS;
  const dialogStatus = startConnectorConnect.isPending
    ? t("settings.connectors.loginLaunching")
    : finishConnectorConnect.isPending
      ? t("settings.connectors.loginReadingCookies")
      : cancelConnectorConnect.isPending
        ? t("settings.connectors.loginClosingBrowser")
        : loginResult
          ? t("settings.connectors.loginCompleted")
        : loginSessionId
          ? t("settings.connectors.loginReady")
          : t("settings.connectors.loginIdle");
  const selectedLabel = selected ? resolveConnectorLabel(selected) : "";
  const cookiesCount = selected?.cookiesCount ?? selected?.cookies?.length ?? 0;
  const cookiesList = selected?.cookies ?? [];
  const isConnected = (selected?.status ?? "disconnected") === "connected";
  const loginDomainsLabel = loginResult?.domains && loginResult.domains.length > 0 ? loginResult.domains.join(", ") : "-";

  return (
    <div className="connectors-card flex min-h-0 min-w-0 flex-1">
      <Card className="flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden">
        <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
              <div className="flex h-8 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder={t("settings.connectors.searchPlaceholder")}
                  size="compact"
                  className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              {filteredItems.length === 0 ? (
                <div className="p-3 text-sm text-muted-foreground">
                  {t("settings.connectors.searchEmpty")}
                </div>
              ) : (
                <SidebarMenu>
                  {groupedItems.map((group, groupIndex) => (
                    <React.Fragment key={group.id}>
                      <div className="px-2 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                        {group.label}
                      </div>
                      {group.connectors.map((connector) => {
                        const statusMeta = STATUS_META[connector.status ?? "disconnected"] ?? STATUS_META.disconnected;
                        const isSelected = connector.id === selectedId;
                        return (
                          <SidebarMenuItem key={connector.id}>
                            <SidebarMenuButton
                              type="button"
                              isActive={isSelected}
                              className="justify-between"
                              onClick={() => setSelectedId(connector.id)}
                            >
                              <div className="flex min-w-0 items-center gap-2">
                                <span className="truncate text-sm font-medium">
                                  {resolveConnectorLabel(connector)}
                                </span>
                              </div>
                              <div className="shrink-0">
                                <span
                                  className={cn(
                                    "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                                    statusMeta.className
                                  )}
                                >
                                  {React.createElement(statusMeta.icon, { className: "h-3.5 w-3.5" })}
                                </span>
                              </div>
                            </SidebarMenuButton>
                          </SidebarMenuItem>
                        );
                      })}
                      {groupIndex < groupedItems.length - 1 ? (
                        <div className="px-2 py-2">
                          <Separator />
                        </div>
                      ) : null}
                    </React.Fragment>
                  ))}
                </SidebarMenu>
              )}
            </div>
          </div>

          <Separator orientation="vertical" className="self-stretch" />

          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto px-3 py-1.5">
              {selected ? (
                <div className={cn("flex h-full flex-col space-y-1.5", GENERAL_CARD_HEIGHT)}>
                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.connectors.detail.status")}
                    </div>
                    <span
                      className={cn(
                        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                        status.className
                      )}
                    >
                      {React.createElement(status.icon, { className: "h-3.5 w-3.5" })}
                      {t(`settings.connectors.status.${status.label.toLowerCase()}`)}
                    </span>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.connectors.detail.data")}
                    </div>
                    <div className="flex min-w-0 items-center justify-end gap-2">
                      <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                        {cookiesCount}
                      </span>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={() => setCookiesDialogOpen(true)}
                        disabled={cookiesCount === 0}
                      >
                        <Eye className="h-4 w-4" />
                        {t("settings.connectors.viewCookies")}
                      </Button>
                    </div>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.connectors.detail.scope")}
                    </div>
                    <div className="max-w-[60%] text-right text-xs text-muted-foreground">
                      {selected.domains && selected.domains.length > 0 ? selected.domains.join(", ") : "-"}
                    </div>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.connectors.detail.actions")}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={() => handleConnect(selected)}
                        disabled={isBusy}
                      >
                        {isLoginRunning ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Link2 className="h-4 w-4" />
                        )}
                        {isConnected
                          ? t("settings.connectors.reconnect")
                          : t("settings.connectors.connect")}
                      </Button>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={() => handleOpenSite(selected)}
                        disabled={isBusy || cookiesCount === 0}
                      >
                        {isOpenRunning ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <ExternalLink className="h-4 w-4" />
                        )}
                        {t("settings.connectors.openSite")}
                      </Button>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={() => clearConnector.mutate({ id: selected.id })}
                        disabled={isBusy}
                      >
                        <Trash2 className="h-4 w-4" />
                        {t("settings.connectors.clear")}
                      </Button>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="p-4 text-sm text-muted-foreground">
                  {t("settings.connectors.empty")}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={loginDialogOpen}
        onOpenChange={(open) => {
          if (open) {
            setLoginDialogOpen(true);
            return;
          }
          if (!isLoginRunning) {
            void handleCancelLogin();
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t("settings.connectors.loginTitle")}
            </DialogTitle>
            <DialogDescription>
              {t("settings.connectors.loginDescription")}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-2 text-sm text-muted-foreground">
            <div>
              {t("settings.connectors.loginTarget")}: {loginTarget ? resolveConnectorLabel(loginTarget) : "-"}
            </div>
            <div>{dialogStatus}</div>
            {loginResult ? (
              <div className="rounded-md border border-border/70 bg-muted/40 p-3 text-xs text-muted-foreground">
                <div>
                  {t("settings.connectors.cookiesRead")}: {loginResult.rawCookiesCount}
                </div>
                <div>
                  {t("settings.connectors.cookiesSaved")}: {loginResult.filteredCookiesCount}
                </div>
                <div>
                  {t("settings.connectors.cookiesDomains")}: {loginDomainsLabel}
                </div>
              </div>
            ) : null}
            {loginError ? (
              <div className="rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                {loginError}
              </div>
            ) : null}
            <div className="rounded-md border border-border/70 bg-muted/40 p-3 text-xs text-muted-foreground">
              {t("settings.connectors.loginHint")}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" className="h-7" onClick={() => void handleCancelLogin()} disabled={isLoginRunning}>
              {t("common.cancel")}
            </Button>
            <Button className="h-7" onClick={() => void handleFinishLogin()} disabled={isLoginRunning || !loginSessionId}>
              {finishConnectorConnect.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Link2 className="h-4 w-4" />}
              {t("settings.connectors.loginFinish")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={cookiesDialogOpen} onOpenChange={setCookiesDialogOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle className="text-left">
              {selectedLabel ? `${selectedLabel} Cookies` : t("settings.connectors.cookiesTitle")}
            </DialogTitle>
          </DialogHeader>
          <div className="rounded-md border">
            {cookiesList.length === 0 ? (
              <div className="p-4 text-sm text-muted-foreground">
                {t("settings.connectors.cookiesEmpty")}
              </div>
            ) : (
              <>
                <div className="bg-card">
                  <table className="w-full table-fixed text-xs">
                    <colgroup>
                      <col className="w-[120px]" />
                      <col />
                      <col />
                      <col className="w-[60px]" />
                      <col className="w-[160px]" />
                      <col className="w-[60px]" />
                    </colgroup>
                    <thead>
                      <tr className="border-b">
                        <th className="w-[120px] px-3 py-2 text-left font-medium text-muted-foreground">Name</th>
                        <th className="px-3 py-2 text-left font-medium text-muted-foreground">Value</th>
                        <th className="px-3 py-2 text-left font-medium text-muted-foreground">Domain</th>
                        <th className="w-[60px] px-3 py-2 text-left font-medium text-muted-foreground">Path</th>
                        <th className="w-[160px] px-3 py-2 text-left font-medium text-muted-foreground">Expires</th>
                        <th className="w-[60px] px-3 py-2 text-left font-medium text-muted-foreground">Secure</th>
                      </tr>
                    </thead>
                  </table>
                </div>
                <div className="max-h-[360px] overflow-y-auto overflow-x-hidden">
                  <table className="w-full table-fixed text-xs">
                    <colgroup>
                      <col className="w-[120px]" />
                      <col />
                      <col />
                      <col className="w-[60px]" />
                      <col className="w-[160px]" />
                      <col className="w-[60px]" />
                    </colgroup>
                    <tbody>
                      {cookiesList.map((cookie, index) => (
                        <tr key={`${cookie.name}-${cookie.domain}-${index}`} className="border-b last:border-b-0">
                          <td className="truncate px-3 py-2 font-medium text-foreground">{cookie.name}</td>
                          <td className="truncate px-3 py-2 text-muted-foreground">{cookie.value}</td>
                          <td className="truncate px-3 py-2 text-muted-foreground">{cookie.domain}</td>
                          <td className="truncate px-3 py-2 text-muted-foreground">{cookie.path}</td>
                          <td className="truncate px-3 py-2 text-muted-foreground">
                            {formatCookieExpires(cookie.expires)}
                          </td>
                          <td className="truncate px-3 py-2 text-muted-foreground">{cookie.secure ? "Yes" : "No"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" className="h-7" onClick={() => setCookiesDialogOpen(false)}>
              {t("common.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
