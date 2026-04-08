import * as React from "react";
import { FolderOpen, Loader2, Pencil, RefreshCw } from "lucide-react";
import { Button } from "@/shared/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { SettingsCompactListCard, SettingsCompactRow, SettingsCompactSeparator } from "@/shared/ui/settings-layout";
import { Switch } from "@/shared/ui/switch";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { System } from "@wailsio/runtime";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useSystemProxyInfo, useTestProxy } from "@/shared/query/settings";
import type { ColorScheme, MenuBarVisibility, ProxySettings } from "@/shared/contracts/settings";
import { AppearanceSection } from "@/features/settings/appearance";

export interface GeneralSectionProps {
  language: string;
  logLevel: string;
  appearance: "light" | "dark" | "auto";
  fontFamily: string;
  fontSize: number;
  themeColor: string;
  colorScheme: ColorScheme;
  systemThemeColor?: string;
  proxy: ProxySettings;
  proxyDraft: ProxySettings;
  menuBarVisibility: MenuBarVisibility;
  autoStart: boolean;
  minimizeToTrayOnStart: boolean;
  downloadDirectory: string;
  downloadDirectoryIsBusy: boolean;
  onProxyChange: (value: ProxySettings) => void;
  onProxyTest: (value: ProxySettings) => Promise<void>;
  onProxySave: (value: ProxySettings) => void;
  proxyIsTesting: boolean;
  proxyIsSaving: boolean;
  onLanguageChange: (value: string) => void;
  onLogLevelChange: (value: string) => void;
  onAppearanceChange: (value: "light" | "dark" | "auto") => void;
  onFontFamilyChange: (value: string) => void;
  onFontSizeChange: (value: number) => void;
  onThemeColorChange: (value: string) => void;
  onColorSchemeChange: (value: ColorScheme) => void;
  onOpenLogDirectory: () => void;
  onMenuBarVisibilityChange: (value: MenuBarVisibility) => void;
  onAutoStartChange: (value: boolean) => void;
  onMinimizeToTrayOnStartChange: (value: boolean) => void;
  onSelectDownloadDirectory: () => void;
}

export function GeneralSection({
  language,
  logLevel,
  appearance,
  fontFamily,
  fontSize,
  themeColor,
  colorScheme,
  systemThemeColor,
  proxy,
  proxyDraft,
  menuBarVisibility,
  autoStart,
  minimizeToTrayOnStart,
  downloadDirectory,
  downloadDirectoryIsBusy,
  onProxyChange,
  onProxyTest,
  onProxySave,
  proxyIsTesting,
  proxyIsSaving,
  onLanguageChange,
  onLogLevelChange,
  onAppearanceChange,
  onFontFamilyChange,
  onFontSizeChange,
  onThemeColorChange,
  onColorSchemeChange,
  onOpenLogDirectory,
  onMenuBarVisibilityChange,
  onAutoStartChange,
  onMinimizeToTrayOnStartChange,
  onSelectDownloadDirectory,
}: GeneralSectionProps) {
  const isWindows = System.IsWindows();
  const { t, supportedLanguages } = useI18n();
  const [proxyDialogOpen, setProxyDialogOpen] = React.useState(false);
  const [clearConfirmOpen, setClearConfirmOpen] = React.useState(false);
  const savedProxy = proxy;
  const draftProxy = proxyDraft;
  const systemProxyQuery = useSystemProxyInfo(draftProxy.mode === "system");
  const { refetch: refetchSystemProxy, isFetching: isSystemProxyFetching } = systemProxyQuery;
  const proxyCheck = useTestProxy();
  const logLevelOptions = [
    { value: "debug", label: t("settings.general.advanced.logLevel.option.debug") },
    { value: "info", label: t("settings.general.advanced.logLevel.option.info") },
    { value: "warn", label: t("settings.general.advanced.logLevel.option.warn") },
    { value: "error", label: t("settings.general.advanced.logLevel.option.error") },
  ];
  const menuBarOptions: { value: MenuBarVisibility; label: string }[] = [
    { value: "always", label: t("settings.general.advanced.menuBarVisibility.option.always") },
    { value: "whenRunning", label: t("settings.general.advanced.menuBarVisibility.option.whenRunning") },
    { value: "never", label: t("settings.general.advanced.menuBarVisibility.option.never") },
  ];
  const filteredMenuBarOptions = isWindows
    ? menuBarOptions.filter((option) => option.value !== "never")
    : menuBarOptions;
  const menuBarVisibilityLabel = isWindows
    ? t("settings.general.advanced.trayVisibility.label")
    : t("settings.general.advanced.menuBarVisibility.label");
  const highlightShadow = "0 0 0 1px hsl(var(--border)), 0 0 0 3px hsl(var(--primary))";
  const [proxyCheckStatus, setProxyCheckStatus] = React.useState<"idle" | "checking" | "available" | "unavailable">("idle");
  const [proxyCheckKey, setProxyCheckKey] = React.useState("");
  const proxyCheckRequestRef = React.useRef(0);
  const proxyCheckInFlightKeyRef = React.useRef("");

  const resetProxyTestState = (next: ProxySettings) => ({
    ...next,
    testSuccess: false,
    testMessage: "",
    testedAt: "",
  });

  const handleProxyModeChange = (mode: ProxySettings["mode"]) => {
    const base = { ...savedProxy, mode };
    const updated = resetProxyTestState({
      ...base,
      scheme: savedProxy.scheme || "http",
    });
    onProxyChange(updated);
    if (mode !== "manual") {
      setProxyDialogOpen(false);
      onProxySave(updated);
    }
  };

  const handleProxyFieldChange = (field: keyof ProxySettings, value: string) => {
    const isNumericField = field === "port" || field === "timeoutSeconds";
    const next: ProxySettings = {
      ...draftProxy,
      [field]: isNumericField ? Number(value) || 0 : value,
    } as ProxySettings;
    onProxyChange(resetProxyTestState(next));
  };

  const handleProxyDialogOpenChange = (open: boolean) => {
    if (open) {
      onProxyChange({ ...savedProxy, mode: "manual" });
    } else {
      setClearConfirmOpen(false);
    }
    setProxyDialogOpen(open);
  };

  const handleProxyClear = () => {
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
    onProxyChange(cleared);
    setProxyDialogOpen(false);
    onProxySave(cleared);
  };

  const handleProxyClearConfirm = () => {
    setClearConfirmOpen(false);
    handleProxyClear();
  };

  const hasTested = draftProxy.testedAt && draftProxy.testedAt !== "0001-01-01T00:00:00Z";
  const testedAt = hasTested ? new Date(draftProxy.testedAt) : null;
  const hostFilled = (draftProxy.host || "").trim() !== "";
  const manualReady = draftProxy.mode === "manual" && hostFilled && draftProxy.port > 0;
  const manualButtonLabel = proxyIsSaving
    ? t("settings.general.proxy.saving")
    : proxyIsTesting
      ? t("settings.general.proxy.testing")
      : draftProxy.testSuccess
        ? t("settings.general.proxy.saved")
        : t("settings.general.proxy.test");
  const testFeedback = draftProxy.testSuccess && testedAt
    ? `${t("settings.general.proxy.testedAtLabel")}: ${testedAt.toLocaleString()}`
    : draftProxy.testMessage
      ? draftProxy.testMessage
      : "";
  const testFeedbackClass =
    draftProxy.testMessage && !draftProxy.testSuccess ? "text-destructive" : "text-muted-foreground";

  const formatHostPort = (host: string, port: number) => {
    if (!host || port <= 0) {
      return "";
    }
    const normalizedHost = host.includes(":") && !host.startsWith("[") ? `[${host}]` : host;
    return `${normalizedHost}:${port}`;
  };
  const manualProxyAddress = (() => {
    const hostPort = formatHostPort((savedProxy.host || "").trim(), savedProxy.port);
    if (!hostPort) {
      return "";
    }
    const scheme = savedProxy.scheme || "http";
    return `${scheme}://${hostPort}`;
  })();
  const systemProxyInfo = systemProxyQuery.data;
  const systemProxyRaw = (systemProxyInfo?.address || "").trim();
  const isVPNSource = systemProxyInfo?.source === "vpn";
  const vpnName = (systemProxyInfo?.name || "").trim();
  const systemProxySourceLabel = isVPNSource
    ? vpnName || t("settings.general.proxy.source.vpn")
    : t("settings.general.proxy.source.system");
  const shouldHideSystemAddress = isVPNSource && !systemProxyRaw;
  const systemProxyDisplay = shouldHideSystemAddress
    ? ""
    : systemProxyQuery.isLoading
      ? t("settings.general.proxy.statusLoading")
      : systemProxyQuery.isError
        ? t("settings.general.proxy.statusUnavailable")
        : systemProxyRaw || t("settings.general.proxy.statusNotConfigured");
  const manualProxyDisplay = manualProxyAddress || t("settings.general.proxy.statusNotConfigured");
  const statusMode = draftProxy.mode;
  const statusAddress = statusMode === "system" ? systemProxyRaw : statusMode === "manual" ? manualProxyAddress : "";
  const statusAddressDisplay =
    statusMode === "system" ? systemProxyDisplay : statusMode === "manual" ? manualProxyDisplay : "";
  const hasStatusAddress = statusAddress !== "";
  const statusKey = hasStatusAddress ? `${statusMode}:${statusAddress}` : "";
  const showSystemSourceBadge = statusMode === "system" && isVPNSource && systemProxySourceLabel;
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
  const showRefreshButton = statusMode === "system" || hasStatusAddress;
  const isStatusRefreshing = statusMode === "system" ? isSystemProxyFetching || isChecking : isChecking;

  const buildProxyTestPayload = (mode: ProxySettings["mode"]) => {
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
  };

  const runProxyCheck = React.useCallback(
    async (mode: ProxySettings["mode"], address: string) => {
      if (mode === "none" || !address) {
        return;
      }
      const key = `${mode}:${address}`;
      if (proxyCheckInFlightKeyRef.current === key) {
        return;
      }
      const payload = buildProxyTestPayload(mode);
      proxyCheckInFlightKeyRef.current = key;
      proxyCheckRequestRef.current += 1;
      const requestId = proxyCheckRequestRef.current;
      setProxyCheckKey(key);
      setProxyCheckStatus("checking");
      try {
        const result = await proxyCheck.mutateAsync(payload);
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
      } finally {
        if (proxyCheckInFlightKeyRef.current === key) {
          proxyCheckInFlightKeyRef.current = "";
        }
      }
    },
    [proxyCheck, savedProxy, t]
  );

  const handleStatusRefresh = React.useCallback(async () => {
    if (statusMode === "system") {
      try {
        const result = await refetchSystemProxy();
        const nextAddress = (result.data?.address || "").trim();
        if (nextAddress) {
          runProxyCheck("system", nextAddress);
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
      runProxyCheck(statusMode, statusAddress);
    }
  }, [hasStatusAddress, refetchSystemProxy, runProxyCheck, statusAddress, statusMode]);

  React.useEffect(() => {
    if (statusMode === "none") {
      proxyCheckInFlightKeyRef.current = "";
      setProxyCheckStatus("idle");
      setProxyCheckKey("");
      return;
    }
    if (!hasStatusAddress) {
      proxyCheckInFlightKeyRef.current = "";
      setProxyCheckStatus("idle");
      setProxyCheckKey("");
      return;
    }
    if (proxyCheckKey === statusKey && proxyCheckStatus !== "idle") {
      return;
    }
    runProxyCheck(statusMode, statusAddress);
  }, [hasStatusAddress, proxyCheckKey, proxyCheckStatus, runProxyCheck, statusAddress, statusKey, statusMode]);

  return (
    <div className="space-y-4">
      <SettingsCompactListCard>
        <div className="flex flex-col">
          <SettingsCompactRow label={t("settings.general.startup.autoStart")}>
            <Switch
              checked={autoStart}
              onCheckedChange={onAutoStartChange}
              aria-label={t("settings.general.startup.autoStart")}
            />
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={t("settings.general.startup.minimizeToTray")}>
            <Switch
              checked={minimizeToTrayOnStart}
              onCheckedChange={onMinimizeToTrayOnStartChange}
              aria-label={t("settings.general.startup.minimizeToTray")}
            />
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={menuBarVisibilityLabel}>
            <Select
              value={menuBarVisibility}
              onChange={(event) => onMenuBarVisibilityChange(event.target.value as MenuBarVisibility)}
              className="w-48"
              aria-label={menuBarVisibilityLabel}
            >
              {filteredMenuBarOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
                ))}
              </Select>
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={<label htmlFor="settings-language">{t("settings.language.label")}</label>}>
            <Select
              id="settings-language"
              value={language || "en"}
              onChange={(event) => onLanguageChange(event.target.value)}
              className="w-48"
            >
              {supportedLanguages.map((option) => (
                <option key={option.value} value={option.value}>
                  {t(`settings.language.option.${option.value}`)}
                </option>
                ))}
              </Select>
          </SettingsCompactRow>
        </div>
      </SettingsCompactListCard>

      <AppearanceSection
        appearance={appearance}
        fontFamily={fontFamily}
        fontSize={fontSize}
        themeColor={themeColor}
        colorScheme={colorScheme}
        systemThemeColor={systemThemeColor}
        onAppearanceChange={onAppearanceChange}
        onFontFamilyChange={onFontFamilyChange}
        onFontSizeChange={onFontSizeChange}
        onThemeColorChange={onThemeColorChange}
        onColorSchemeChange={onColorSchemeChange}
      />

      <SettingsCompactListCard>
        <div className="flex flex-col">
          <SettingsCompactRow label={t("settings.general.proxy.title")}>
            <div className="flex gap-2">
              {(["none", "system", "manual"] as ProxySettings["mode"][]).map((mode) => (
                <Button
                  key={mode}
                  variant="outline"
                  size="compact"
                  onClick={() => handleProxyModeChange(mode)}
                  style={draftProxy.mode === mode ? { boxShadow: highlightShadow } : undefined}
                >
                  {t(`settings.general.proxy.option.${mode}`)}
                </Button>
              ))}
            </div>
          </SettingsCompactRow>

          {draftProxy.mode !== "none" ? (
            <>
              <SettingsCompactSeparator />
              <SettingsCompactRow label={t("settings.general.proxy.status")} contentClassName="min-w-0">
                <div className="flex min-w-0 items-center justify-end gap-2">
                  {showSystemSourceBadge ? (
                    <Button asChild variant="default" size="compact" className="h-7 px-2">
                      <span>{systemProxySourceLabel}</span>
                    </Button>
                  ) : null}
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
                            onClick={handleStatusRefresh}
                            aria-label={t("settings.general.proxy.check")}
                          >
                            {isStatusRefreshing ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent side="top">
                          {t("settings.general.proxy.check")}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  ) : null}
                  {draftProxy.mode === "manual" ? (
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
              </SettingsCompactRow>
            </>
          ) : null}
        </div>

        {draftProxy.mode === "manual" ? (
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
                      value={draftProxy.scheme}
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
                      value={draftProxy.timeoutSeconds || ""}
                      onChange={(event) => handleProxyFieldChange("timeoutSeconds", event.target.value)}
                      placeholder="30"
                      size="compact"
                      className="text-sm"
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="text-sm text-muted-foreground">{t("settings.general.proxy.host")}</span>
                    <Input
                      value={draftProxy.host}
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
                      value={draftProxy.port || ""}
                      onChange={(event) => handleProxyFieldChange("port", event.target.value)}
                      placeholder="8080"
                      size="compact"
                      className="text-sm"
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="text-sm text-muted-foreground">{t("settings.general.proxy.username")}</span>
                    <Input
                      value={draftProxy.username}
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
                      value={draftProxy.password}
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
                      size="compact"
                      variant="destructive"
                      disabled={proxyIsTesting || proxyIsSaving}
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
                      size="compact"
                      variant={draftProxy.testSuccess ? "secondary" : "outline"}
                      disabled={!manualReady || proxyIsTesting || proxyIsSaving}
                      onClick={() => onProxyTest(draftProxy)}
                    >
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
                  <Button size="compact" variant="destructive" onClick={handleProxyClearConfirm}>
                    {t("settings.general.proxy.clearConfirm.confirm")}
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </>
        ) : null}
      </SettingsCompactListCard>

      <SettingsCompactListCard>
        <SettingsCompactRow label={t("settings.general.download.directory")} contentClassName="min-w-0">
          <div className="flex min-w-0 items-center justify-end gap-2">
            <span className="max-w-[260px] truncate text-right font-mono text-xs text-muted-foreground">
              {downloadDirectory}
            </span>
            <TooltipProvider delayDuration={0}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="compactIcon"
                    disabled={downloadDirectoryIsBusy}
                    onClick={onSelectDownloadDirectory}
                    aria-label={t("settings.general.download.change")}
                  >
                    {downloadDirectoryIsBusy ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Pencil className="h-4 w-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top">
                  {t("settings.general.download.change")}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </SettingsCompactRow>
      </SettingsCompactListCard>

      <SettingsCompactListCard>
        <div className="flex flex-col">
          <SettingsCompactRow label={t("settings.general.advanced.logLevel.label")}>
            <div className="flex items-center gap-2">
              <TooltipProvider delayDuration={0}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      size="compactIcon"
                      onClick={onOpenLogDirectory}
                      aria-label={t("settings.general.advanced.logLevel.openDirectory")}
                    >
                      <FolderOpen className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("settings.general.advanced.logLevel.openDirectory")}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <Select
                id="settings-log-level"
                value={logLevel || "info"}
                onChange={(event) => onLogLevelChange(event.target.value)}
                className="w-48"
              >
                {logLevelOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </div>
          </SettingsCompactRow>
        </div>
      </SettingsCompactListCard>
    </div>
  );
}
