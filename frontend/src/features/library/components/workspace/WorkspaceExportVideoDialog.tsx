import * as React from "react";

import { Clapperboard, Columns2, Languages } from "lucide-react";

import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import type { TranscodePreset } from "@/shared/contracts/library";
import { Button } from "@/shared/ui/button";
import { DASHBOARD_CONTROL_GROUP_CLASS } from "@/shared/ui/dashboard";
import { Dialog, DialogDescription, DialogTitle } from "@/shared/ui/dialog";
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
  DashboardDialogBody,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
} from "@/shared/ui/dashboard-dialog";
import { Select } from "@/shared/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";

import type { WorkspaceDisplayMode, WorkspaceSelectOption } from "./types";

type SubtitleHandling = "none" | "embed" | "burnin";

type WorkspaceExportVideoDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resourceName: string;
  cueCount: number;
  subtitleFormat: string;
  presetOptions: TranscodePreset[];
  presetId: string;
  onPresetIdChange: (value: string) => void;
  subtitleHandling: SubtitleHandling;
  onSubtitleHandlingChange: (value: SubtitleHandling) => void;
  displayMode: WorkspaceDisplayMode;
  canUseDualDisplay?: boolean;
  dualDisplayDisabledReason?: string;
  onDisplayModeChange: (value: WorkspaceDisplayMode) => void;
  primaryTrackOptions: WorkspaceSelectOption[];
  primaryTrackId: string;
  onPrimaryTrackIdChange: (value: string) => void;
  secondaryTrackOptions: WorkspaceSelectOption[];
  secondaryTrackId: string;
  onSecondaryTrackIdChange: (value: string) => void;
  embeddedSubtitleFormat: string;
  embeddedSubtitleLabel: string;
  isSubmitting: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onSubmit: () => void;
};

const HANDLING_OPTIONS: Array<{
  value: SubtitleHandling;
  labelKey: string;
  defaultLabel: string;
  descriptionKey: string;
  defaultDescription: string;
}> = [
  {
    value: "none",
    labelKey: "library.workspace.dialogs.exportVideo.handling.none",
    defaultLabel: "No subtitles",
    descriptionKey:
      "library.workspace.dialogs.exportVideo.handling.noneDescription",
    defaultDescription:
      "Only export the video output. Subtitle embedding and burn-in stay disabled.",
  },
  {
    value: "embed",
    labelKey: "library.workspace.dialogs.exportVideo.handling.embed",
    defaultLabel: "Embed subtitle track",
    descriptionKey:
      "library.workspace.dialogs.exportVideo.handling.embedDescription",
    defaultDescription:
      "Mux a soft subtitle track into the exported video so the viewer can toggle it on demand.",
  },
  {
    value: "burnin",
    labelKey: "library.workspace.dialogs.exportVideo.handling.burnin",
    defaultLabel: "Burn-in ASS",
    descriptionKey:
      "library.workspace.dialogs.exportVideo.handling.burninDescription",
    defaultDescription:
      "Render the current subtitle view directly into the exported video with the selected subtitle style.",
  },
];

const BUILTIN_RESOLUTION_TAB_ORDER = [
  "original",
  "2160p",
  "1080p",
  "720p",
  "480p",
] as const;

const BUILTIN_RESOLUTION_TAB_ORDER_MAP = new Map<string, number>(
  BUILTIN_RESOLUTION_TAB_ORDER.map((value, index) => [value, index]),
);

function resolvePresetResolutionKey(preset: TranscodePreset) {
  const normalizedScale = (preset.scale ?? "").trim().toLowerCase();
  if (normalizedScale && BUILTIN_RESOLUTION_TAB_ORDER_MAP.has(normalizedScale)) {
    return normalizedScale;
  }
  const width =
    typeof preset.width === "number" && Number.isFinite(preset.width)
      ? Math.round(preset.width)
      : 0;
  const height =
    typeof preset.height === "number" && Number.isFinite(preset.height)
      ? Math.round(preset.height)
      : 0;
  if (normalizedScale === "custom") {
    if (width > 0 && height > 0) {
      return `custom:${width}x${height}`;
    }
    return "custom";
  }
  if (width > 0 && height > 0) {
    return `${width}x${height}`;
  }
  return "original";
}

function resolvePresetResolutionLabel(
  key: string,
  t: (key: string) => string,
) {
  if (key === "original") {
    return t("library.workspace.scale.original");
  }
  if (key === "custom") {
    return t("library.workspace.scale.custom");
  }
  if (key.startsWith("custom:")) {
    return key.slice("custom:".length).replace("x", "×");
  }
  if (BUILTIN_RESOLUTION_TAB_ORDER_MAP.has(key)) {
    return key.toUpperCase();
  }
  const [width, height] = key.split("x");
  if (!width || !height) {
    return key;
  }
  return `${width}×${height}`;
}

function resolvePresetResolutionOrder(key: string) {
  const builtinOrder = BUILTIN_RESOLUTION_TAB_ORDER_MAP.get(key);
  if (typeof builtinOrder === "number") {
    return builtinOrder - 100;
  }
  if (key === "custom") {
    return 1000;
  }
  if (key.startsWith("custom:")) {
    return 900;
  }
  const [widthRaw, heightRaw] = key.split("x");
  const width = Number(widthRaw);
  const height = Number(heightRaw);
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    return 1001;
  }
  return 100 - width * height;
}

function CompactSummaryStrip({
  children,
  className,
  watermark,
}: {
  children: React.ReactNode;
  className?: string;
  watermark?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-1.5 rounded-xl border border-border/70 bg-card px-3 py-2",
        className,
      )}
    >
      {watermark ? (
        <span className="shrink-0 select-none pr-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground/55">
          {watermark}
        </span>
      ) : null}
      {children}
    </div>
  );
}

function CompactMeta({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: React.ReactNode;
  tone?: "default" | "muted";
}) {
  return (
    <span
      className={cn(
        "inline-flex min-w-0 items-center gap-1.5 rounded-md border px-2 py-1 text-[11px]",
        tone === "muted"
          ? "border-border/50 bg-background/70 text-muted-foreground"
          : "border-border/60 bg-background/90",
      )}
    >
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span
        className={cn(
          "min-w-0 truncate font-medium",
          tone === "muted" ? "text-muted-foreground" : "text-foreground",
        )}
      >
        {value}
      </span>
    </span>
  );
}

function CompactInlineField({
  label,
  control,
  className,
  hint,
  controlClassName,
}: {
  label: string;
  control: React.ReactNode;
  className?: string;
  hint?: string;
  controlClassName?: string;
}) {
  return (
    <div
      className={cn(
        "grid gap-2 px-2.5 py-2 md:grid-cols-2 md:items-center",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        className,
      )}
    >
      <div className="min-w-0">
        <div className="truncate text-[11px] font-medium text-foreground">
          {label}
        </div>
        {hint ? (
          <div className="truncate text-[11px] text-muted-foreground">
            {hint}
          </div>
        ) : null}
      </div>
      <div className={cn("min-w-0 md:justify-self-stretch", controlClassName)}>
        {control}
      </div>
    </div>
  );
}

const COMPACT_SELECT_CLASS =
  "h-8 w-full min-w-0 border-border/70 bg-background/80 text-xs";

const TRACK_MAPPING_ROW_CLASS =
  "grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] md:items-center";

export function WorkspaceExportVideoDialog({
  open,
  onOpenChange,
  resourceName,
  cueCount,
  subtitleFormat,
  presetOptions,
  presetId,
  onPresetIdChange,
  subtitleHandling,
  onSubtitleHandlingChange,
  displayMode,
  canUseDualDisplay = true,
  dualDisplayDisabledReason = "",
  onDisplayModeChange,
  primaryTrackOptions,
  primaryTrackId,
  onPrimaryTrackIdChange,
  secondaryTrackOptions,
  secondaryTrackId,
  onSecondaryTrackIdChange,
  embeddedSubtitleFormat,
  embeddedSubtitleLabel,
  isSubmitting,
  disabled = false,
  disabledReason = "",
  onSubmit,
}: WorkspaceExportVideoDialogProps) {
  const { t } = useI18n();
  const withDisabledTooltip = React.useCallback(
    (content: React.ReactNode, tooltipLabel: string) => {
      if (!tooltipLabel.trim()) {
        return content;
      }
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex">{content}</span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[18rem] text-xs leading-5">
            {tooltipLabel}
          </TooltipContent>
        </Tooltip>
      );
    },
    [],
  );
  const selectedPreset = React.useMemo(
    () =>
      presetOptions.find((option) => option.id === presetId) ??
      presetOptions[0] ??
      null,
    [presetId, presetOptions],
  );
  const selectedHandling =
    HANDLING_OPTIONS.find((option) => option.value === subtitleHandling) ??
    HANDLING_OPTIONS[0];
  const monoModeLabel = t("library.workspace.table.modeMono");
  const bilingualModeLabel = t("library.workspace.table.modeBilingual");
  const displayModeOptions = React.useMemo(
    () => [
      {
        value: "mono" as const,
        label: monoModeLabel,
        icon: Languages,
      },
      {
        value: "bilingual" as const,
        label: bilingualModeLabel,
        icon: Columns2,
      },
    ],
    [bilingualModeLabel, monoModeLabel],
  );
  const trackEmptyLabel = t("library.workspace.dialogs.exportVideo.trackEmpty");
  const primaryTrackLabel = React.useMemo(
    () =>
      primaryTrackOptions.find((option) => option.value === primaryTrackId)
        ?.label || trackEmptyLabel,
    [primaryTrackId, primaryTrackOptions, trackEmptyLabel],
  );
  const secondaryTrackLabel = React.useMemo(
    () =>
      secondaryTrackOptions.find((option) => option.value === secondaryTrackId)
        ?.label || trackEmptyLabel,
    [secondaryTrackId, secondaryTrackOptions, trackEmptyLabel],
  );
  const subtitlesEnabled = subtitleHandling !== "none";
  const embeddedSupportsAss = embeddedSubtitleFormat === "ass";
  const resolutionGroups = React.useMemo(() => {
    const grouped = new Map<string, TranscodePreset[]>();
    for (const preset of presetOptions) {
      const key = resolvePresetResolutionKey(preset);
      const group = grouped.get(key);
      if (group) {
        group.push(preset);
      } else {
        grouped.set(key, [preset]);
      }
    }
    return Array.from(grouped.entries())
      .map(([key, presets]) => ({
        key,
        label: resolvePresetResolutionLabel(key, t),
        order: resolvePresetResolutionOrder(key),
        presets,
      }))
      .sort((left, right) => {
        if (left.order !== right.order) {
          return left.order - right.order;
        }
        return left.label.localeCompare(right.label);
      });
  }, [presetOptions, t]);
  const selectedResolutionKey = React.useMemo(() => {
    if (selectedPreset) {
      return resolvePresetResolutionKey(selectedPreset);
    }
    return resolutionGroups[0]?.key ?? "";
  }, [resolutionGroups, selectedPreset]);
  const visiblePresetOptions = React.useMemo(
    () =>
      resolutionGroups.find((group) => group.key === selectedResolutionKey)
        ?.presets ?? [],
    [resolutionGroups, selectedResolutionKey],
  );
  const handleResolutionChange = React.useCallback(
    (nextResolution: string) => {
      const nextGroup = resolutionGroups.find(
        (group) => group.key === nextResolution,
      );
      if (!nextGroup || nextGroup.presets.length === 0) {
        return;
      }
      if (!nextGroup.presets.some((preset) => preset.id === presetId)) {
        onPresetIdChange(nextGroup.presets[0].id);
      }
    },
    [onPresetIdChange, presetId, resolutionGroups],
  );
  const selectedPresetResolutionLabel = selectedPreset
    ? resolvePresetResolutionLabel(resolvePresetResolutionKey(selectedPreset), t)
    : t("common.notAvailable");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DashboardDialogContent
        size="detail"
        className="flex max-h-[84vh] min-h-0 flex-col gap-3 text-xs"
      >
        <DashboardDialogHeader>
          <DialogTitle>
            {t("library.workspace.exportVideo")}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {t("library.workspace.dialogs.exportVideo.dialogDescription")}
          </DialogDescription>
        </DashboardDialogHeader>

        <div className="space-y-2">
          <CompactSummaryStrip
            watermark={t("library.workspace.dialogs.exportVideo.inputSection")}
          >
            <CompactMeta
              label={t("library.columns.taskName")}
              value={
                resourceName ||
                t("library.workspace.dialogs.exportVideo.currentVideoFallback")
              }
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.primaryTrack")}
              value={
                primaryTrackLabel ||
                t("library.workspace.dialogs.exportVideo.trackEmpty")
              }
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.secondaryTrack")}
              value={
                secondaryTrackLabel ||
                t("library.workspace.dialogs.exportVideo.trackEmpty")
              }
              tone={secondaryTrackLabel ? "default" : "muted"}
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.metrics.subtitleDraft")}
              value={subtitleFormat.toUpperCase()}
              tone="muted"
            />
          </CompactSummaryStrip>

          <CompactSummaryStrip
            watermark={t("library.workspace.dialogs.exportVideo.outputSection")}
          >
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.metrics.cueScope")}
              value={t("library.workspace.dialogs.exportVideo.cueCount").replace(
                "{count}",
                String(cueCount),
              )}
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.metrics.subtitleMode")}
              value={t(selectedHandling.labelKey)}
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportVideo.resolution")}
              value={selectedPresetResolutionLabel}
            />
            <CompactMeta
              label={t("library.workspace.fields.exportFormat")}
              value={selectedPreset?.container?.toUpperCase() || "-"}
            />
            <CompactMeta
              label={t("library.workspace.transcode.selectPreset")}
              value={
                selectedPreset?.name ||
                t("library.workspace.dialogs.exportVideo.presetEmpty")
              }
            />
          </CompactSummaryStrip>
        </div>

        <Tabs
          value={selectedResolutionKey}
          onValueChange={handleResolutionChange}
          className="flex min-h-0 flex-1 flex-col gap-3"
        >
          <TabsList className="flex h-auto w-full flex-wrap gap-1.5">
            {resolutionGroups.length === 0 ? (
              <TabsTrigger value="__empty__" disabled>
                {t("library.workspace.dialogs.exportVideo.presetEmpty")}
              </TabsTrigger>
            ) : (
              resolutionGroups.map((group) => (
                <TabsTrigger
                  key={group.key}
                  value={group.key}
                  disabled={isSubmitting}
                >
                  {group.label}
                </TabsTrigger>
              ))
            )}
          </TabsList>

          <DashboardDialogBody className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto pr-1">
            <div
              className={cn(
                "min-w-0 space-y-2 p-3",
                DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
              )}
            >
              <CompactInlineField
                label={t("library.workspace.transcode.selectPreset")}
                hint={t("library.workspace.dialogs.exportVideo.outputHint")}
                control={
                  <Select
                    value={selectedPreset?.id ?? ""}
                    onChange={(event) => onPresetIdChange(event.target.value)}
                    disabled={presetOptions.length === 0 || isSubmitting}
                    className={COMPACT_SELECT_CLASS}
                  >
                    {visiblePresetOptions.length === 0 ? (
                      <option value="">
                        {t("library.workspace.dialogs.exportVideo.presetEmpty")}
                      </option>
                    ) : null}
                    {visiblePresetOptions.map((preset) => (
                      <option key={preset.id} value={preset.id}>
                        {`${preset.container.toUpperCase()} · ${preset.name}`}
                      </option>
                    ))}
                  </Select>
                }
              />

              <CompactInlineField
                label={t("library.workspace.dialogs.exportVideo.subtitleHandlingTitle")}
                hint={t("library.workspace.dialogs.exportVideo.subtitleHandlingDescription")}
                className="md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"
                control={
                  <div className="grid gap-1.5 sm:grid-cols-3">
                    {HANDLING_OPTIONS.map((option) => {
                      const active = option.value === subtitleHandling;
                      return (
                        <button
                          key={option.value}
                          type="button"
                          onClick={() => onSubtitleHandlingChange(option.value)}
                          disabled={isSubmitting}
                          className={cn(
                            "rounded-md border px-2 py-1.5 text-[11px] transition-colors",
                            active
                              ? "border-primary/40 bg-primary/[0.08] font-medium text-foreground shadow-[inset_0_0_0_1px_hsl(var(--primary)/0.18)]"
                              : "border-border/60 bg-card text-muted-foreground hover:bg-muted/60",
                          )}
                        >
                          {t(option.labelKey)}
                        </button>
                      );
                    })}
                  </div>
                }
              />

              <CompactInlineField
                label={t("library.workspace.dialogs.exportVideo.displayMode")}
                className="md:grid-cols-[minmax(0,1fr)_max-content]"
                controlClassName="md:justify-self-end"
                control={
                  <div
                    className={cn(
                      DASHBOARD_CONTROL_GROUP_CLASS,
                      "w-auto shrink-0 justify-start overflow-hidden",
                    )}
                  >
                    {displayModeOptions.map((option, index) => {
                      const Icon = option.icon;
                      const active = option.value === displayMode;
                      const disabled =
                        isSubmitting ||
                        (option.value === "bilingual" && !canUseDualDisplay);
                      const button = (
                        <Button
                          key={option.value}
                          type="button"
                          variant={active ? "secondary" : "ghost"}
                          size="compact"
                          disabled={disabled}
                          className={cn(
                            "gap-1.5 rounded-none border-0 px-2.5",
                            index > 0 && "border-l border-border/70",
                          )}
                          onClick={() => onDisplayModeChange(option.value)}
                          aria-label={option.label}
                        >
                          <Icon className="h-3.5 w-3.5" />
                          <span className="text-xs">{option.label}</span>
                        </Button>
                      );
                      if (!(option.value === "bilingual" && !canUseDualDisplay && !isSubmitting && dualDisplayDisabledReason.trim())) {
                        return button;
                      }
                      return (
                        <Tooltip key={option.value}>
                          <TooltipTrigger asChild>
                            <span className="inline-flex">{button}</span>
                          </TooltipTrigger>
                          <TooltipContent className="max-w-[18rem] text-xs leading-5">
                            {dualDisplayDisabledReason}
                          </TooltipContent>
                        </Tooltip>
                      );
                    })}
                  </div>
                }
              />
              <div
                className={cn(
                  "min-w-0 space-y-1.5 px-2.5 py-2",
                  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                )}
              >
                <div className="text-[11px] font-medium text-foreground">
                  {t("library.workspace.dialogs.exportVideo.trackMappingTitle")}
                </div>
                {displayMode === "bilingual" && canUseDualDisplay ? (
                  <div className="space-y-2">
                    <div className={TRACK_MAPPING_ROW_CLASS}>
                      <div className="truncate text-[11px] text-muted-foreground">
                        {t("library.workspace.header.track")}
                      </div>
                      <div className="min-w-0">
                        <Select
                          value={primaryTrackId}
                          onChange={(event) =>
                            onPrimaryTrackIdChange(event.target.value)
                          }
                          disabled={primaryTrackOptions.length === 0 || isSubmitting}
                          className={COMPACT_SELECT_CLASS}
                        >
                          {primaryTrackOptions.length === 0 ? (
                            <option value="">
                              {t("library.workspace.header.noSubtitleTrack")}
                            </option>
                          ) : null}
                          {primaryTrackOptions.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                    </div>
                    <div className={TRACK_MAPPING_ROW_CLASS}>
                      <div className="truncate text-[11px] text-muted-foreground">
                        {t("library.workspace.header.secondary")}
                      </div>
                      <div className="min-w-0">
                        <Select
                          value={secondaryTrackId}
                          onChange={(event) =>
                            onSecondaryTrackIdChange(event.target.value)
                          }
                          disabled={isSubmitting}
                          className={COMPACT_SELECT_CLASS}
                        >
                          {secondaryTrackOptions.map((option) => (
                            <option
                              key={option.value || "none"}
                              value={option.value}
                            >
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className={TRACK_MAPPING_ROW_CLASS}>
                    <div className="truncate text-[11px] text-muted-foreground">
                      {t("library.workspace.header.track")}
                    </div>
                    <div className="min-w-0">
                      <Select
                        value={primaryTrackId}
                        onChange={(event) =>
                          onPrimaryTrackIdChange(event.target.value)
                        }
                        disabled={primaryTrackOptions.length === 0 || isSubmitting}
                        className={COMPACT_SELECT_CLASS}
                      >
                        {primaryTrackOptions.length === 0 ? (
                          <option value="">
                            {t("library.workspace.header.noSubtitleTrack")}
                          </option>
                        ) : null}
                        {primaryTrackOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    </div>
                  </div>
                )}
              </div>

              {subtitlesEnabled && subtitleHandling === "embed" && !embeddedSupportsAss ? (
                <div
                  className={cn(
                    "min-w-0 space-y-1.5 px-2.5 py-2",
                    DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                  )}
                >
                  <div className="text-[11px] font-medium text-foreground">
                    {t("library.workspace.header.style")}
                  </div>
                  <div className="text-[11px] leading-5 text-muted-foreground">
                    {t("library.workspace.dialogs.exportVideo.embedStyleUnsupported").replace("{route}", embeddedSubtitleLabel)}
                  </div>
                </div>
              ) : null}

            </div>
          </DashboardDialogBody>
        </Tabs>

        <DashboardDialogFooter className="sm:justify-end">
          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center">
            <Button
              variant="outline"
              size="compact"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              {t("common.cancel")}
            </Button>
            {withDisabledTooltip(
              <Button
                size="compact"
                onClick={onSubmit}
                disabled={!selectedPreset || isSubmitting || disabled}
              >
                <Clapperboard className="h-4 w-4" />
                {isSubmitting
                  ? t("library.workspace.dialogs.exportVideo.queueing")
                  : t("library.workspace.dialogs.exportVideo.submit")}
              </Button>,
              disabled && !isSubmitting ? disabledReason : "",
            )}
          </div>
        </DashboardDialogFooter>
      </DashboardDialogContent>
    </Dialog>
  );
}
