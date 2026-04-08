import * as React from "react";
import {
  AlertTriangle,
  ArrowUpCircle,
  CheckCircle2,
  Circle,
  Download,
  FolderOpen,
  Loader2,
  RefreshCw,
  Search,
  Trash2,
} from "lucide-react";

import { DialogMarkdown } from "@/shared/markdown/dialog-markdown";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import { Progress } from "@/shared/ui/progress";
import { Separator } from "@/shared/ui/separator";
import {
  SETTINGS_ROW_CLASS,
  SETTINGS_ROW_LABEL_CLASS,
  SettingsSeparator,
} from "@/shared/ui/settings-layout";
import { useI18n } from "@/shared/i18n";
import {
  useExternalTools,
  useExternalToolInstallState,
  useExternalToolUpdates,
  useInstallExternalTool,
  useOpenExternalToolDirectory,
  useRemoveExternalTool,
  useVerifyExternalTool,
} from "@/shared/query/externalTools";
import type { ExternalTool, ExternalToolUpdateInfo } from "@/shared/store/externalTools";
import { cn } from "@/lib/utils";

const GENERAL_CARD_HEIGHT = "min-h-[240px]";

type InstallAction = "install" | "upgrade" | "repair";

type InstallDialogState = "idle" | "running" | "success" | "error";
type ExternalToolGroup = "runtime" | "bin";

const EXTERNAL_TOOL_GROUP_ORDER: ExternalToolGroup[] = ["runtime", "bin"];

const normalizeToolGroup = (tool: ExternalTool): ExternalToolGroup => {
  const kind = String(tool.kind ?? "").trim().toLowerCase();
  if (kind === "runtime") {
    return "runtime";
  }
  return "bin";
};

const normalizeVersion = (version?: string, toolName?: string) => {
  let value = (version ?? "").trim();
  if (!value) {
    return "";
  }
  value = value.replace(/^v/i, "");
  if (toolName?.toLowerCase() === "ffmpeg") {
    value = value.replace(/^n-/i, "");
    value = value.replace(/-tessus$/i, "");
  }
  return value;
};
const formatVersion = (version?: string) => {
  const value = (version ?? "").trim();
  if (!value) {
    return "";
  }
  return value.startsWith("v") || value.startsWith("V") ? value : `v${value}`;
};

export function ExternalToolsSection() {
  const { t } = useI18n();
  const tools = useExternalTools();
  const updates = useExternalToolUpdates();
  const installTool = useInstallExternalTool();
  const openToolDirectory = useOpenExternalToolDirectory();
  const verifyTool = useVerifyExternalTool();
  const removeTool = useRemoveExternalTool();

  const [selectedName, setSelectedName] = React.useState<string | null>(null);
  const [query, setQuery] = React.useState("");
  const [installDialogOpen, setInstallDialogOpen] = React.useState(false);
  const [installTarget, setInstallTarget] = React.useState<ExternalTool | null>(null);
  const [activeInstallName, setActiveInstallName] = React.useState<string | null>(null);
  const [installAction, setInstallAction] = React.useState<InstallAction>("install");
  const [installState, setInstallState] = React.useState<InstallDialogState>("idle");
  const [installProgress, setInstallProgress] = React.useState(0);
  const [installError, setInstallError] = React.useState("");
  const [installStage, setInstallStage] = React.useState("idle");
  const [installLastStage, setInstallLastStage] = React.useState("downloading");

  const [releaseDialogOpen, setReleaseDialogOpen] = React.useState(false);
  const [releaseNotesContent, setReleaseNotesContent] = React.useState("");
  const [releaseNotesTitle, setReleaseNotesTitle] = React.useState("");

  const items = tools.data ?? [];
  const trimmedQuery = query.trim().toLowerCase();
  const filteredItems = query.trim().length
    ? items.filter((tool) => tool.name.toLowerCase().includes(trimmedQuery))
    : items;
  const groupedItems = React.useMemo(() => {
    const buckets = new Map<ExternalToolGroup, ExternalTool[]>();
    filteredItems.forEach((tool) => {
      const group = normalizeToolGroup(tool);
      const bucket = buckets.get(group) ?? [];
      bucket.push(tool);
      buckets.set(group, bucket);
    });
    return EXTERNAL_TOOL_GROUP_ORDER
      .filter((group) => buckets.has(group))
      .map((group) => ({
        id: group,
        label: t(`settings.externalTools.group.${group}`),
        tools: (buckets.get(group) ?? []).sort((left, right) => left.name.localeCompare(right.name)),
      }));
  }, [filteredItems, t]);

  React.useEffect(() => {
    if (selectedName && !items.some((tool) => tool.name === selectedName)) {
      setSelectedName(null);
    }
  }, [items, selectedName]);

  React.useEffect(() => {
    if (!selectedName && items.length > 0) {
      setSelectedName(items[0].name);
    }
  }, [items, selectedName]);

  const updateMap = React.useMemo(() => {
    const map = new Map<string, ExternalToolUpdateInfo>();
    (updates.data ?? []).forEach((info) => {
      if (info?.name) {
        map.set(info.name, info);
      }
    });
    return map;
  }, [updates.data]);

  const selectedTool = items.find((tool) => tool.name === selectedName) ?? null;
  const selectedUpdate = selectedTool ? updateMap.get(selectedTool.name) : undefined;
  const bunTool = items.find((tool) => tool.name === "bun") ?? null;
  const bunInstalled = String(bunTool?.status ?? "").trim().toLowerCase() === "installed";

  const resolveHasUpdate = React.useCallback(
    (tool: ExternalTool, info?: ExternalToolUpdateInfo) => {
      const current = normalizeVersion(tool.version, tool.name);
      const latest = normalizeVersion(info?.latestVersion, tool.name);
      return Boolean(current && latest && current !== latest);
    },
    []
  );

  const resolveDisplayLatestVersion = React.useCallback(
    (tool: ExternalTool | null, info?: ExternalToolUpdateInfo) => {
      const latest = (info?.latestVersion ?? "").trim();
      if (latest) {
        return formatVersion(latest);
      }
      const kind = String(tool?.kind ?? "").trim().toLowerCase();
      if (kind === "runtime") {
        const runtimeVersion = (tool?.version ?? "").trim();
        if (runtimeVersion) {
          return formatVersion(runtimeVersion);
        }
      }
      return t("settings.externalTools.detail.unknown");
    },
    [t]
  );

  const resolveToolDirectory = (tool: ExternalTool | null) => {
    const execPath = tool?.execPath?.trim();
    if (!execPath) {
      return "";
    }
    const normalized = execPath.replace(/\\/g, "/");
    const lastSlash = normalized.lastIndexOf("/");
    if (lastSlash <= 0) {
      return execPath;
    }
    return normalized.slice(0, lastSlash);
  };


  const openInstallDialog = (tool: ExternalTool, action: InstallAction) => {
    if (tool.name === "clawhub" && !bunInstalled) {
      return;
    }
    setActiveInstallName(null);
    setInstallTarget(tool);
    setInstallAction(action);
    setInstallDialogOpen(true);
    setInstallState("idle");
    setInstallProgress(0);
    setInstallError("");
    setInstallStage("idle");
    setInstallLastStage("downloading");
  };

  const handleInstall = async () => {
    if (!installTarget) {
      return;
    }
    if (installTarget.name === "clawhub" && !bunInstalled) {
      setInstallStage("error");
      setInstallState("error");
      setInstallError(t("settings.externalTools.actions.installBunFirst"));
      return;
    }
    setActiveInstallName(installTarget.name);
    setInstallState("running");
    setInstallError("");
    setInstallProgress(0);
    setInstallStage("downloading");
    setInstallLastStage("downloading");
    try {
      await installTool.mutateAsync({ name: installTarget.name });
      await installStateQuery.refetch();
    } catch (error) {
      setActiveInstallName(null);
      setInstallStage("error");
      setInstallState("error");
      setInstallProgress(0);
      setInstallError(error instanceof Error ? error.message : t("settings.externalTools.installDialog.error"));
    }
  };

  const handleOpenReleaseNotes = () => {
    if (!selectedTool || !selectedUpdate?.releaseNotes) {
      return;
    }
    setReleaseNotesTitle(selectedTool.name.toUpperCase());
    setReleaseNotesContent(selectedUpdate.releaseNotes);
    setReleaseDialogOpen(true);
  };

  const handleVerify = () => {
    if (!selectedTool) {
      return;
    }
    verifyTool.mutate({ name: selectedTool.name });
  };

  const handleRemove = () => {
    if (!selectedTool) {
      return;
    }
    removeTool.mutate({ name: selectedTool.name });
    setSelectedName(null);
  };

  const isInstallRunning = installState === "running";
  const isActionBusy =
    isInstallRunning || verifyTool.isPending || removeTool.isPending || openToolDirectory.isPending;

  const installStateQuery = useExternalToolInstallState(activeInstallName ?? undefined, Boolean(activeInstallName));

  React.useEffect(() => {
    if (!activeInstallName) {
      return;
    }
    const state = installStateQuery.data;
    if (!state) {
      return;
    }
    if (stageOrder.includes(state.stage ?? "")) {
      setInstallLastStage(state.stage ?? "downloading");
    }
    if (typeof state.progress === "number") {
      setInstallProgress(state.progress);
    }
    if (state.stage) {
      setInstallStage(state.stage);
      if (state.stage === "done") {
        setActiveInstallName(null);
        setInstallState("success");
        setInstallProgress(100);
        void tools.refetch();
        void updates.refetch();
      } else if (state.stage === "error") {
        setActiveInstallName(null);
        setInstallState("error");
        setInstallError(state.message || t("settings.externalTools.installDialog.error"));
        void tools.refetch();
        void updates.refetch();
      } else if (stageOrder.includes(state.stage) && installState !== "error" && installState !== "success") {
        setInstallState("running");
      }
    }
  }, [activeInstallName, installStateQuery.data, installState, t, tools.refetch, updates.refetch]);

  const getActionLabel = (action: InstallAction) => {
    if (action === "repair") {
      return t("settings.externalTools.actions.repair");
    }
    if (action === "upgrade") {
      return t("settings.externalTools.actions.upgrade");
    }
    return t("settings.externalTools.actions.install");
  };

  const resolvePrimaryAction = (tool: ExternalTool | null, info?: ExternalToolUpdateInfo) => {
    if (!tool) {
      return null;
    }
    if (tool.status === "invalid") {
      return "repair" as const;
    }
    if (tool.status !== "installed") {
      return "install" as const;
    }
    if (resolveHasUpdate(tool, info)) {
      return "upgrade" as const;
    }
    return null;
  };

  const latestVersionLabel = resolveDisplayLatestVersion(selectedTool, selectedUpdate);

  const currentVersionLabel = selectedTool?.version
    ? formatVersion(selectedTool.version)
    : t("settings.externalTools.detail.notInstalled");

  const toolDirectory = resolveToolDirectory(selectedTool);
  const pathLabel = toolDirectory || t("settings.externalTools.detail.notInstalled");

  const selectedHasUpdate = selectedTool ? resolveHasUpdate(selectedTool, selectedUpdate) : false;
  const primaryAction = resolvePrimaryAction(selectedTool, selectedUpdate);
  const selectedToolRequiresBun = selectedTool?.name === "clawhub" && !bunInstalled;
  const installBlockedByBun = installTarget?.name === "clawhub" && !bunInstalled;

  const latestBadgeClass = selectedHasUpdate
    ? "bg-primary/10 text-primary"
    : "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-100";
  const latestBadgeIcon = selectedHasUpdate ? ArrowUpCircle : CheckCircle2;

  const installDialogTitle = `${t("settings.externalTools.installDialog.titlePrefix")} ${installTarget?.name?.toUpperCase() ?? ""}`.trim();

  const installLiveTool = installTarget
    ? items.find((tool) => tool.name === installTarget.name) ?? installTarget
    : null;
  const installUpdateInfo = installLiveTool ? updateMap.get(installLiveTool.name) : undefined;
  const installTargetVersion = resolveDisplayLatestVersion(installLiveTool, installUpdateInfo);
  const installCurrentVersion = installLiveTool?.version
    ? formatVersion(installLiveTool.version)
    : t("settings.externalTools.detail.notInstalled");

  const rowClassName = SETTINGS_ROW_CLASS;
  const stageOrder = ["downloading", "extracting", "verifying"];
  const stageIndex = stageOrder.indexOf(installStage);
  const lastStageIndex = stageOrder.indexOf(installLastStage);
  const stageLabel = t(`settings.externalTools.installDialog.stage.${installStage}`);
  const isInstallSuccess = installStage === "done";
  const isInstallError = installStage === "error";

  return (
    <div className="external-tools-card flex min-h-0 min-w-0 flex-1">
      <Card className="flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden">
        <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
              <div className="flex h-8 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder={t("settings.externalTools.searchPlaceholder")}
                  size="compact"
                  className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              {filteredItems.length === 0 ? (
                <div className="p-3 text-sm text-muted-foreground">
                  {t("settings.externalTools.searchEmpty")}
                </div>
              ) : (
                <SidebarMenu>
                  {groupedItems.map((group, groupIndex) => (
                    <React.Fragment key={group.id}>
                      <div className="px-2 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                        {group.label}
                      </div>
                      {group.tools.map((tool) => {
                        const updateInfo = updateMap.get(tool.name);
                        const hasUpdate = tool.status === "installed" && resolveHasUpdate(tool, updateInfo);
                        const isSelected = selectedName === tool.name;
                        let statusNode: React.ReactNode;

                        if (tool.status === "invalid") {
                          statusNode = (
                            <Badge
                              variant="outline"
                              className="border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-700/60 dark:bg-amber-900/40 dark:text-amber-100"
                            >
                              {t("settings.externalTools.status.repair")}
                            </Badge>
                          );
                        } else if (tool.status !== "installed") {
                          statusNode = (
                            <Badge variant="subtle">
                              {t("settings.externalTools.status.install")}
                            </Badge>
                          );
                        } else if (hasUpdate) {
                          statusNode = (
                            <span className="inline-flex items-center gap-1 text-xs font-semibold text-primary">
                              <ArrowUpCircle className="h-4 w-4" />
                              {t("settings.externalTools.status.update")}
                            </span>
                          );
                        } else {
                          statusNode = (
                            <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                              <CheckCircle2 className="h-4 w-4" />
                              {t("settings.externalTools.status.latest")}
                            </span>
                          );
                        }

                        return (
                          <SidebarMenuItem key={tool.name}>
                            <SidebarMenuButton
                              type="button"
                              isActive={isSelected}
                              className="justify-between"
                              onClick={() => setSelectedName(tool.name)}
                            >
                              <div className="flex min-w-0 items-center gap-2">
                                <span className="truncate text-xs font-semibold uppercase tracking-[0.24em]">
                                  {tool.name.toUpperCase()}
                                </span>
                              </div>
                              <div className="shrink-0">
                                {statusNode}
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
              {selectedTool ? (
                <div className={cn("flex h-full flex-col space-y-1.5", GENERAL_CARD_HEIGHT)}>
                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.externalTools.detail.currentVersion")}
                    </div>
                    <span className="whitespace-nowrap text-sm font-semibold text-foreground">{currentVersionLabel}</span>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.externalTools.detail.latestVersion")}
                    </div>
                    <span
                      className={cn(
                        "inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-1 text-xs font-medium whitespace-nowrap",
                        latestBadgeClass
                      )}
                    >
                      {React.createElement(latestBadgeIcon, { className: "h-3.5 w-3.5" })}
                      {latestVersionLabel}
                    </span>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.externalTools.detail.releaseNotes")}
                    </div>
                    {selectedUpdate?.releaseNotes ? (
                      <Button variant="outline" size="compact" onClick={handleOpenReleaseNotes} disabled={isActionBusy}>
                        {t("settings.externalTools.detail.viewReleaseNotes")}
                      </Button>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        {t("settings.externalTools.detail.noReleaseNotes")}
                      </span>
                    )}
                  </div>

                  <SettingsSeparator />

                  <div className={cn(rowClassName, "min-w-0")}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.externalTools.detail.path")}
                    </div>
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="min-w-0 flex-1 max-w-[180px] truncate whitespace-nowrap text-right text-xs text-muted-foreground">
                        {pathLabel}
                      </span>
                      <Button
                        variant="outline"
                        size="compactIcon"
                        onClick={() =>
                          selectedTool ? openToolDirectory.mutate({ name: selectedTool.name }) : undefined
                        }
                        disabled={!toolDirectory || isActionBusy}
                        aria-label={t("settings.externalTools.detail.openPath")}
                      >
                        <FolderOpen className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>

                  <SettingsSeparator />

                  <div className={rowClassName}>
                    <div className={SETTINGS_ROW_LABEL_CLASS}>
                      {t("settings.externalTools.detail.actions")}
                    </div>
                    <div className="flex items-center gap-2 overflow-x-auto">
                      {primaryAction ? (
                        <Button
                          variant="outline"
                          size="compact"
                          onClick={() => openInstallDialog(selectedTool, primaryAction)}
                          disabled={isActionBusy || selectedToolRequiresBun}
                        >
                          {isInstallRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                          {selectedToolRequiresBun
                            ? t("settings.externalTools.actions.installBunFirst")
                            : getActionLabel(primaryAction)}
                        </Button>
                      ) : null}
                      <Button variant="outline" size="compact" onClick={handleVerify} disabled={isActionBusy}>
                        <RefreshCw className="h-4 w-4" />
                        {t("settings.externalTools.actions.verify")}
                      </Button>
                      <Button variant="outline" size="compact" onClick={handleRemove} disabled={isActionBusy}>
                        <Trash2 className="h-4 w-4" />
                        {t("settings.externalTools.actions.remove")}
                      </Button>
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={installDialogOpen}
        onOpenChange={(open) => {
          setInstallDialogOpen(open);
          if (!open) {
            setActiveInstallName(null);
            setInstallState("idle");
            setInstallProgress(0);
            setInstallError("");
            setInstallStage("idle");
            setInstallLastStage("downloading");
            setInstallTarget(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{installDialogTitle}</DialogTitle>
            <DialogDescription className="sr-only">
              {t("settings.externalTools.installDialog.description")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="rounded-lg border bg-muted/40 p-3 text-sm">
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground">
                  {t("settings.externalTools.detail.currentVersion")}
                </span>
                <span className="font-medium text-foreground">
                  {installCurrentVersion}
                </span>
              </div>
              <div className="mt-2 flex items-center justify-between gap-2">
                <span className="text-muted-foreground">
                  {t("settings.externalTools.installDialog.targetVersion")}
                </span>
                <span className="font-medium text-foreground">{installTargetVersion}</span>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{t("settings.externalTools.installDialog.progress")}</span>
                <span>{Math.round(installProgress)}%</span>
              </div>
              <Progress value={installProgress} />
              <div className="space-y-1.5">
                <div className="flex flex-wrap items-center gap-4 text-xs">
                  {stageOrder.map((stage, index) => {
                    const isDone =
                      installStage === "done" ||
                      (installStage === "error" ? index < lastStageIndex : stageIndex > index);
                    const isActive = installStage === stage;
                    const isErrorStage = installStage === "error" && stage === installLastStage;
                    const iconClass = isDone
                      ? "text-emerald-600"
                      : isErrorStage
                        ? "text-destructive"
                        : isActive
                          ? "text-primary"
                          : "text-muted-foreground";
                    return (
                      <div key={stage} className="flex items-center gap-2 text-muted-foreground">
                        {isDone ? (
                          <CheckCircle2 className={cn("h-4 w-4", iconClass)} />
                        ) : isErrorStage ? (
                          <AlertTriangle className={cn("h-4 w-4", iconClass)} />
                        ) : isActive ? (
                          <Loader2 className={cn("h-4 w-4 animate-spin", iconClass)} />
                        ) : (
                          <Circle className={cn("h-3.5 w-3.5", iconClass)} />
                        )}
                        <span className={cn(isDone ? "text-foreground" : isActive ? "text-primary" : "text-muted-foreground")}>
                          {t(`settings.externalTools.installDialog.stage.${stage}`)}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
              <div className="text-[11px] text-muted-foreground">
                {t("settings.externalTools.installDialog.stageLabel")}: {stageLabel}
              </div>
            </div>

            {installState === "success" ? (
              <div className="text-sm font-medium text-emerald-600">
                {t("settings.externalTools.installDialog.success")}
              </div>
            ) : null}
            {installState === "error" ? (
              <div className="space-y-1 text-xs text-destructive">
                <div>{t("settings.externalTools.installDialog.error")}</div>
                {installError ? (
                  <div
                    className="max-w-full overflow-hidden break-all whitespace-pre-wrap text-[11px] [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:4]"
                    title={installError}
                  >
                    {installError}
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
          <DialogFooter>
            <Button
              variant="ghost"
              size="compact"
              onClick={() => setInstallDialogOpen(false)}
            >
              {isInstallSuccess ? t("settings.externalTools.installDialog.close") : t("common.cancel")}
            </Button>
            {!isInstallSuccess ? (
              <Button
                size="compact"
                onClick={handleInstall}
                disabled={isInstallRunning || !installTarget || installBlockedByBun}
              >
                {isInstallRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                {installBlockedByBun
                  ? t("settings.externalTools.actions.installBunFirst")
                  : isInstallError
                  ? t("settings.externalTools.installDialog.retry")
                  : getActionLabel(installAction)}
              </Button>
            ) : null}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={releaseDialogOpen} onOpenChange={setReleaseDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {t("settings.externalTools.releaseNotesTitle")} {releaseNotesTitle}
            </DialogTitle>
          </DialogHeader>
          <DialogMarkdown content={releaseNotesContent} />
          <DialogFooter>
            <Button variant="ghost" size="compact" onClick={() => setReleaseDialogOpen(false)}>
              {t("settings.externalTools.releaseNotesClose")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
