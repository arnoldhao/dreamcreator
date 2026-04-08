import * as React from "react";
import {
  AlertCircle,
  Check,
  Download,
  Github,
  Globe,
  Loader2,
  Mail,
  MessageSquare,
  RefreshCw,
  Twitter,
} from "lucide-react";

import { Select } from "@/shared/ui/select";
import { useI18n } from "@/shared/i18n";
import { DialogMarkdown } from "@/shared/markdown/dialog-markdown";
import { type DebugModeLevel, useDebugMode } from "@/shared/store/debug";
import { Button } from "@/shared/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { SettingsCompactListCard, SettingsCompactRow, SettingsCompactSeparator } from "@/shared/ui/settings-layout";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { cn } from "@/lib/utils";
import { messageBus } from "@/shared/message";
import { useSettings, useUpdateSettings as useAppSettingsUpdate } from "@/shared/query/settings";
import { useUpdateStore } from "@/shared/store/update";
import { useCheckForUpdate, useDownloadUpdate, useRestartToApply, useUpdateState } from "@/shared/query/update";
import { Browser } from "@wailsio/runtime";

export function AboutSection() {
  const { t } = useI18n();
  const { mode, setMode } = useDebugMode();
  const settingsQuery = useSettings();
  const updateAppSettings = useAppSettingsUpdate();
  const updateInfo = useUpdateStore((state) => state.info);
  const setUpdateInfo = useUpdateStore((state) => state.setInfo);
  const checkUpdate = useCheckForUpdate();
  const downloadUpdate = useDownloadUpdate();
  const restartToApply = useRestartToApply();
  const { data: serverUpdateInfo } = useUpdateState();
  const [releaseNotesOpen, setReleaseNotesOpen] = React.useState(false);
  const advancedTapCountRef = React.useRef(0);
  const [advancedUnlocked, setAdvancedUnlocked] = React.useState(false);
  const autoRefreshTriggeredRef = React.useRef(false);

  const isChecking = updateInfo.status === "checking" || checkUpdate.isPending;
  const isError = updateInfo.status === "error";
  const hasUpdate =
    updateInfo.status === "available" ||
    updateInfo.status === "downloading" ||
    updateInfo.status === "installing" ||
    updateInfo.status === "ready_to_restart";
  const isDownloading = updateInfo.status === "downloading" || updateInfo.status === "installing";
  const isReadyToRestart = updateInfo.status === "ready_to_restart";
  const releaseNotes = (updateInfo.changelog ?? "").trim();
  const hasReleaseNotes = releaseNotes.length > 0;

  const latestLabel = (() => {
    if (isError) return t("settings.about.update.latestFailed");
    if (hasUpdate) return updateInfo.latestVersion || t("settings.about.update.latestAvailable");
    return t("settings.about.update.latestOk");
  })();

  const latestBadgeClass = (() => {
    if (isError) return "bg-destructive/15 text-destructive";
    if (hasUpdate) return "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-100";
    return "bg-primary/10 text-primary";
  })();
  const latestBadgeIcon = (() => {
    if (isError) return AlertCircle;
    if (hasUpdate) return Download;
    return Check;
  })();

  React.useEffect(() => {
    if (serverUpdateInfo) {
      setUpdateInfo(serverUpdateInfo);
    }
  }, [serverUpdateInfo, setUpdateInfo]);

  React.useEffect(() => {
    if (autoRefreshTriggeredRef.current) {
      return;
    }
    const candidate = serverUpdateInfo ?? updateInfo;
    const status = candidate.status;
    if (status === "checking" || status === "downloading" || status === "installing") {
      return;
    }
    const currentVersion = (candidate.currentVersion ?? "").trim();
    if (!currentVersion) {
      return;
    }
    const checkedAt = (candidate.checkedAt ?? "").trim();
    let stale = true;
    if (checkedAt) {
      const checkedAtMs = Date.parse(checkedAt);
      if (Number.isFinite(checkedAtMs)) {
        stale = Date.now() - checkedAtMs >= 60 * 60 * 1000;
      }
    }
    if (!stale) {
      return;
    }
    autoRefreshTriggeredRef.current = true;
    void checkUpdate
      .mutateAsync(currentVersion)
      .then((next) => {
        setUpdateInfo(next);
      })
      .catch((error) => {
        console.warn(error);
      });
  }, [checkUpdate, serverUpdateInfo, setUpdateInfo, updateInfo]);

  const handleCheck = async () => {
    try {
      const next = await checkUpdate.mutateAsync(updateInfo.currentVersion);
      setUpdateInfo(next);
    } catch (error) {
      console.warn(error);
    }
  };

  const handleExternal = (url: string) => (event: React.MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault();
    Browser.OpenURL(url);
  };

  const handleDebugModeChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const nextMode = event.target.value as DebugModeLevel;
    if (nextMode !== "off" && nextMode !== "basic" && nextMode !== "full") {
      return;
    }
    if (nextMode === mode) {
      return;
    }
    const previousMode = mode;
    setMode(nextMode);

    const targetRecordPrompt = nextMode === "full";
    const currentRecordPrompt = settingsQuery.data?.gateway.runtime.recordPrompt;
    if (typeof currentRecordPrompt === "boolean" && targetRecordPrompt === currentRecordPrompt) {
      return;
    }
    updateAppSettings.mutate(
      {
        gateway: {
          runtime: {
            recordPrompt: targetRecordPrompt,
          },
        },
      },
      {
        onError: () => {
          setMode(previousMode);
        },
      }
    );
  };

  const handleAdvancedUnlock = () => {
    if (advancedUnlocked) {
      return;
    }
    advancedTapCountRef.current += 1;
    if (advancedTapCountRef.current < 10) {
      return;
    }
    setAdvancedUnlocked(true);
    messageBus.publishToast({
      intent: "success",
      title: t("settings.about.advanced.unlockedTitle"),
      description: t("settings.about.advanced.unlockedHint"),
    });
  };

  const creditGroups = [
    {
      label: t("settings.about.credits.tools"),
      items: [
        { name: "yt-dlp", url: "https://github.com/yt-dlp/yt-dlp" },
        { name: "FFmpeg", url: "https://ffmpeg.org" },
      ],
    },
  ];

  const handleInstall = async () => {
    try {
      const next = await downloadUpdate.mutateAsync();
      setUpdateInfo(next);
    } catch (error) {
      console.warn(error);
    }
  };

  const handleRestart = async () => {
    try {
      const next = await restartToApply.mutateAsync();
      setUpdateInfo(next);
    } catch (error) {
      console.warn(error);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col items-center gap-2 text-center">
        <button
          type="button"
          className="rounded-2xl transition-transform duration-200 hover:scale-[1.02] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          onClick={handleAdvancedUnlock}
          aria-label={t("settings.about.iconTap")}
        >
          <img src="/appicon.png" alt="Dream Creator" className="h-16 w-16 rounded-lg shadow-sm" />
        </button>
        <div className="text-lg font-semibold text-foreground">{t("app.name")}</div>
        <div className="max-w-2xl text-sm text-muted-foreground">
          {t("app.description")}
        </div>
      </div>

      <div className="space-y-4">
        <SettingsCompactListCard>
          <SettingsCompactRow label={t("settings.about.version")}>
            <span className="text-sm font-semibold text-foreground">
              {updateInfo.currentVersion || t("settings.about.update.current")}
            </span>
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={t("settings.about.update.latest")}>
            <div className="flex min-w-0 items-center justify-end gap-2">
              {isError && updateInfo.message ? (
                <span
                  className="max-w-[220px] truncate text-right text-sm text-destructive"
                  title={updateInfo.message}
                >
                  {updateInfo.message}
                </span>
              ) : null}
              <span className={cn("inline-flex items-center gap-1 rounded-full px-2 py-1 text-sm font-medium", latestBadgeClass)}>
                {React.createElement(latestBadgeIcon, { className: "h-3.5 w-3.5" })}
                {latestLabel}
              </span>
            </div>
          </SettingsCompactRow>

          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.about.update.changelog")}>
            {hasReleaseNotes ? (
              <Button variant="outline" size="compact" onClick={() => setReleaseNotesOpen(true)}>
                {t("settings.about.update.viewReleaseNotes")}
              </Button>
            ) : (
              <span className="text-sm text-muted-foreground">
                {t("settings.about.update.noReleaseNotes")}
              </span>
            )}
          </SettingsCompactRow>

          {isDownloading ? (
            <>
              <SettingsCompactSeparator />
              <div className="space-y-1.5 px-3 py-2">
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full bg-primary transition-[width]"
                    style={{ width: `${Math.min(Math.max(updateInfo.progress, 0), 100)}%` }}
                  />
                </div>
                <div className="text-right text-sm text-muted-foreground">{Math.round(updateInfo.progress)}%</div>
              </div>
            </>
          ) : null}

          <SettingsCompactSeparator />

          <SettingsCompactRow label={t("settings.about.update.command")}>
            <div className="flex flex-wrap items-center justify-end gap-2">
              <TooltipProvider delayDuration={0}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="compact"
                      onClick={handleCheck}
                      disabled={checkUpdate.isPending || isChecking || isDownloading}
                      aria-label={t("settings.about.update.check")}
                    >
                      {isChecking ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                      {t("settings.about.update.check")}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("settings.about.update.check")}</TooltipContent>
                </Tooltip>
              </TooltipProvider>

              {hasUpdate && !isReadyToRestart ? (
                <TooltipProvider delayDuration={0}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={handleInstall}
                        disabled={downloadUpdate.isPending || isDownloading || restartToApply.isPending}
                        aria-label={t("settings.about.update.install")}
                      >
                        {downloadUpdate.isPending || isDownloading ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Download className="h-4 w-4" />
                        )}
                        {t("settings.about.update.install")}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("settings.about.update.install")}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : null}

              {isReadyToRestart ? (
                <TooltipProvider delayDuration={0}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="compact"
                        onClick={handleRestart}
                        disabled={restartToApply.isPending}
                        aria-label={t("settings.about.update.restart")}
                      >
                        {restartToApply.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <RefreshCw className="h-4 w-4" />
                        )}
                        {t("settings.about.update.restart")}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("settings.about.update.restart")}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : null}
            </div>
          </SettingsCompactRow>
        </SettingsCompactListCard>
        <Dialog open={releaseNotesOpen} onOpenChange={setReleaseNotesOpen}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>{t("settings.about.update.changelog")}</DialogTitle>
            </DialogHeader>
            <DialogMarkdown content={releaseNotes} />
            <DialogFooter>
              <Button variant="ghost" size="compact" onClick={() => setReleaseNotesOpen(false)}>
                {t("settings.about.update.releaseNotesClose")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <SettingsCompactListCard>
          <SettingsCompactRow label={t("settings.about.meta.craftedBy")}>
            <span className="text-sm font-semibold text-foreground">Arnold HAO</span>
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={t("settings.about.meta.contact")}>
            <TooltipProvider delayDuration={0}>
              <div className="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button asChild variant="outline" size="compactIcon">
                      <a
                        href="mailto:xunruhao@gmail.com"
                        onClick={handleExternal("mailto:xunruhao@gmail.com")}
                        aria-label={t("settings.about.meta.tooltip.email")}
                      >
                        <Mail className="h-4 w-4" />
                      </a>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("settings.about.meta.tooltip.email")}
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button asChild variant="outline" size="compactIcon">
                      <a
                        href="https://x.com/ArnoldHaoCA"
                        onClick={handleExternal("https://x.com/ArnoldHaoCA")}
                        target="_blank"
                        rel="noreferrer"
                        aria-label={t("settings.about.meta.tooltip.twitter")}
                      >
                        <Twitter className="h-4 w-4" />
                      </a>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("settings.about.meta.tooltip.twitter")}
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button asChild variant="outline" size="compactIcon">
                      <a
                        href="https://dreamcreator.dreamapp.cc"
                        onClick={handleExternal("https://dreamcreator.dreamapp.cc")}
                        target="_blank"
                        rel="noreferrer"
                        aria-label={t("settings.about.meta.tooltip.website")}
                      >
                        <Globe className="h-4 w-4" />
                      </a>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("settings.about.meta.tooltip.website")}
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button asChild variant="outline" size="compactIcon">
                      <a
                        href="https://github.com/arnoldhao/dreamcreator"
                        onClick={handleExternal("https://github.com/arnoldhao/dreamcreator")}
                        target="_blank"
                        rel="noreferrer"
                        aria-label={t("settings.about.meta.tooltip.github")}
                      >
                        <Github className="h-4 w-4" />
                      </a>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("settings.about.meta.tooltip.github")}
                  </TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </SettingsCompactRow>

          <SettingsCompactSeparator />

          <SettingsCompactRow label={t("settings.about.meta.feedback")}>
            <Button asChild variant="outline" size="compact">
              <a
                href="https://github.com/arnoldhao/dreamcreator/issues"
                onClick={handleExternal("https://github.com/arnoldhao/dreamcreator/issues")}
                target="_blank"
                rel="noreferrer"
              >
                <MessageSquare className="h-4 w-4" />
                {t("settings.about.meta.sendFeedback")}
              </a>
            </Button>
          </SettingsCompactRow>
        </SettingsCompactListCard>

        <div className="space-y-2">
          <div className="pl-3 text-sm font-bold text-foreground">{t("settings.about.credits.title")}</div>
          <SettingsCompactListCard>
            {creditGroups.map((group, index) => (
              <React.Fragment key={group.label}>
                {index > 0 ? <SettingsCompactSeparator /> : null}
                <SettingsCompactRow label={group.label}>
                  <div className="flex flex-wrap justify-end gap-2">
                    {group.items.map((item) => (
                      <Button key={item.name} asChild variant="outline" size="compact">
                        <a href={item.url} onClick={handleExternal(item.url)} target="_blank" rel="noreferrer">
                          <span className="font-semibold uppercase tracking-[0.24em]">{item.name.toUpperCase()}</span>
                        </a>
                      </Button>
                    ))}
                  </div>
                </SettingsCompactRow>
              </React.Fragment>
            ))}
          </SettingsCompactListCard>
        </div>
      </div>

      {advancedUnlocked ? (
        <div className="space-y-2">
          <div className="pl-3 text-sm font-bold text-foreground">{t("settings.about.advanced.title")}</div>
          <SettingsCompactListCard>
            <SettingsCompactRow
              label={<span className="text-sm font-medium text-foreground">{t("settings.about.advanced.debug")}</span>}
              description={t("settings.about.advanced.debugHint")}
            >
              <Select
                value={mode}
                onChange={handleDebugModeChange}
                className="w-36"
                disabled={updateAppSettings.isPending}
                aria-label={t("settings.about.advanced.debug")}
              >
                <option value="off">{t("settings.about.advanced.options.off")}</option>
                <option value="basic">{t("settings.about.advanced.options.basic")}</option>
                <option value="full">{t("settings.about.advanced.options.full")}</option>
              </Select>
            </SettingsCompactRow>
          </SettingsCompactListCard>
        </div>
      ) : null}
    </div>
  );
}
