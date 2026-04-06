import * as React from "react";

import { Captions } from "lucide-react";

import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { Button } from "@/shared/ui/button";
import { Dialog, DialogTitle } from "@/shared/ui/dialog";
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
  DashboardDialogBody,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
} from "@/shared/ui/dashboard-dialog";
import { Select } from "@/shared/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";

import type {
  LibrarySubtitleExportPresetDTO,
  SubtitleExportConfig,
} from "@/shared/contracts/library";
import {
  DEFAULT_SUBTITLE_EXPORT_ASS_TITLE,
  DEFAULT_SUBTITLE_EXPORT_EVENT_NAME,
  DEFAULT_SUBTITLE_EXPORT_LIBRARY_NAME,
  DEFAULT_SUBTITLE_EXPORT_PROJECT_NAME,
  FCPXML_FRAME_DURATION_PRESETS,
  ITT_FRAME_RATE_MULTIPLIER_PRESETS,
  normalizeFCPXMLFrameDuration,
  normalizeITTFrameRate,
  normalizeITTFrameRateMultiplier,
  resolveFCPXMLFrameDurationLabel,
  resolveITTFrameRateLabel,
} from "../../utils/subtitleStyles";

const FORMAT_OPTIONS = ["srt", "vtt", "ass", "ssa", "itt", "fcpxml"] as const;

type WorkspaceExportSubtitleDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resourceName: string;
  trackLabel: string;
  cueCount: number;
  currentFormat: string;
  targetFormat: string;
  onTargetFormatChange: (value: string) => void;
  presetOptions: LibrarySubtitleExportPresetDTO[];
  selectedPresetId: string;
  onSelectedPresetIdChange: (value: string) => void;
  exportConfig: SubtitleExportConfig;
  onExportConfigChange: (next: SubtitleExportConfig) => void;
  ittLanguageOptions: Array<{ value: string; label: string }>;
  hasUnsavedChanges: boolean;
  isSubmitting: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onSubmit: () => void;
};

function resolvePresetFormatKey(value: string) {
  const normalized = value.trim().toLowerCase();
  return normalized === "ssa" ? "ass" : normalized;
}

function resolveExportPresetFormat(preset: LibrarySubtitleExportPresetDTO) {
  return resolvePresetFormatKey(
    (preset.format ?? preset.targetFormat ?? "").trim(),
  );
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

function CompactFieldPanel({
  label,
  hint,
  children,
  className,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "min-w-0 space-y-1.5 px-2.5 py-2",
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
      <div className="min-w-0">{children}</div>
    </div>
  );
}

function CompactInlineField({
  label,
  control,
  className,
}: {
  label: string;
  control: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid gap-2 px-2.5 py-2 md:grid-cols-[minmax(0,1fr)_320px] md:items-center",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        className,
      )}
    >
      <div className="truncate text-[11px] font-medium text-foreground">
        {label}
      </div>
      <div className="min-w-0 md:justify-self-end">{control}</div>
    </div>
  );
}

function resolveExportSubtitleVTTKindLabel(
  value: string | undefined,
  t: (key: string) => string,
) {
  switch ((value ?? "").trim().toLowerCase()) {
    case "captions":
      return t("library.workspace.dialogs.exportSubtitle.vttKindCaptions");
    case "descriptions":
      return t("library.workspace.dialogs.exportSubtitle.vttKindDescriptions");
    default:
      return t("library.workspace.dialogs.exportSubtitle.vttKindSubtitles");
  }
}

function buildResolvedOutputSummary(
  format: string,
  exportConfig: SubtitleExportConfig,
  t: (key: string) => string,
) {
  switch (resolvePresetFormatKey(format)) {
    case "srt":
      return `${t("library.workspace.dialogs.exportSubtitle.encoding")} ${exportConfig.srt?.encoding || "utf-8"}`;
    case "vtt":
      return `${resolveExportSubtitleVTTKindLabel(exportConfig.vtt?.kind, t)} · ${exportConfig.vtt?.language || "en-US"}`;
    case "ass":
      return `${exportConfig.ass?.playResX ?? 1920}×${exportConfig.ass?.playResY ?? 1080} · ${exportConfig.ass?.title || DEFAULT_SUBTITLE_EXPORT_ASS_TITLE}`;
    case "itt":
      return `${resolveITTFrameRateLabel(exportConfig.itt?.frameRate, exportConfig.itt?.frameRateMultiplier)} · ${exportConfig.itt?.language || "en-US"}`;
    case "fcpxml":
      return `${exportConfig.fcpxml?.width ?? 1920}×${exportConfig.fcpxml?.height ?? 1080} · ${resolveFCPXMLFrameDurationLabel(exportConfig.fcpxml?.frameDuration)} · ${exportConfig.fcpxml?.colorSpace || "1-1-1 (Rec. 709)"}`;
    default:
      return "-";
  }
}

export function WorkspaceExportSubtitleDialog({
  open,
  onOpenChange,
  resourceName,
  trackLabel,
  cueCount,
  currentFormat,
  targetFormat,
  onTargetFormatChange,
  presetOptions,
  selectedPresetId,
  onSelectedPresetIdChange,
  exportConfig,
  onExportConfigChange,
  ittLanguageOptions,
  hasUnsavedChanges,
  isSubmitting,
  disabled = false,
  disabledReason = "",
  onSubmit,
}: WorkspaceExportSubtitleDialogProps) {
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
  const normalizedTargetFormat = targetFormat.toLowerCase();
  const presetFormatKey = resolvePresetFormatKey(normalizedTargetFormat);
  const srtConfig = exportConfig.srt ?? {};
  const vttConfig = exportConfig.vtt ?? {};
  const assConfig = exportConfig.ass ?? {};
  const ittConfig = exportConfig.itt ?? {};
  const fcpxmlConfig = exportConfig.fcpxml ?? {};
  const defaultEnglishUSLabel = React.useMemo(
    () => `${t("settings.language.option.en")} (en-US)`,
    [t],
  );
  const vttKindOptions = React.useMemo(
    () => [
      {
        value: "subtitles",
        label: t("library.workspace.dialogs.exportSubtitle.vttKindSubtitles"),
      },
      {
        value: "captions",
        label: t("library.workspace.dialogs.exportSubtitle.vttKindCaptions"),
      },
      {
        value: "descriptions",
        label: t("library.workspace.dialogs.exportSubtitle.vttKindDescriptions"),
      },
    ],
    [t],
  );
  const effectiveVTTLanguageOptions = React.useMemo(() => {
    const seen = new Set<string>();
    const options: Array<{ value: string; label: string }> = [];
    for (const option of ittLanguageOptions) {
      const value = option.value.trim();
      if (!value || seen.has(value)) {
        continue;
      }
      seen.add(value);
      options.push({ value, label: option.label || value });
    }
    if (!seen.has("en-US")) {
      options.unshift({ value: "en-US", label: defaultEnglishUSLabel });
      seen.add("en-US");
    }
    const current = (vttConfig.language ?? "").trim();
    if (current && !seen.has(current)) {
      options.push({ value: current, label: current });
    }
    return options;
  }, [defaultEnglishUSLabel, ittLanguageOptions, vttConfig.language]);
  const effectiveITTFrameRate = normalizeITTFrameRate(ittConfig.frameRate);
  const effectiveITTFrameRateMultiplier = normalizeITTFrameRateMultiplier(
    ittConfig.frameRateMultiplier,
  );
  const effectiveITTFrameRateMultiplierOptions = React.useMemo(() => {
    const options = [...ITT_FRAME_RATE_MULTIPLIER_PRESETS];
    if (
      !options.some((item) => item.value === effectiveITTFrameRateMultiplier)
    ) {
      options.push({
        value: effectiveITTFrameRateMultiplier,
        label: effectiveITTFrameRateMultiplier.replace(" ", "/"),
      });
    }
    return options;
  }, [effectiveITTFrameRateMultiplier]);
  const effectiveITTLanguageOptions = React.useMemo(() => {
    const seen = new Set<string>();
    const options: Array<{ value: string; label: string }> = [];
    for (const option of ittLanguageOptions) {
      const value = option.value.trim();
      if (!value || seen.has(value)) {
        continue;
      }
      seen.add(value);
      options.push({ value, label: option.label || value });
    }
    if (!seen.has("en-US")) {
      options.unshift({ value: "en-US", label: defaultEnglishUSLabel });
      seen.add("en-US");
    }
    const current = (ittConfig.language ?? "").trim();
    if (current && !seen.has(current)) {
      options.push({ value: current, label: current });
    }
    return options;
  }, [defaultEnglishUSLabel, ittConfig.language, ittLanguageOptions]);
  const filteredPresets = presetOptions.filter(
    (preset) => resolveExportPresetFormat(preset) === presetFormatKey,
  );
  const visibleSelectedPreset =
    filteredPresets.find((preset) => preset.id === selectedPresetId) ?? null;
  const resolvedSummary = buildResolvedOutputSummary(
    normalizedTargetFormat,
    exportConfig,
    t,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DashboardDialogContent
        size="detail"
        className="flex max-h-[84vh] min-h-0 flex-col gap-3 text-xs"
      >
        <DashboardDialogHeader>
          <DialogTitle>
            {t("library.workspace.exportSubtitle")}
          </DialogTitle>
        </DashboardDialogHeader>

        <div className="space-y-2">
          <CompactSummaryStrip
            watermark={t("library.workspace.dialogs.exportSubtitle.inputSection")}
          >
            <CompactMeta
              label={t("library.columns.taskName")}
              value={
                resourceName ||
                t("library.workspace.dialogs.exportSubtitle.fileFallback")
              }
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportSubtitle.track")}
              value={
                trackLabel ||
                t("library.workspace.dialogs.exportSubtitle.trackEmpty")
              }
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportSubtitle.currentFormat")}
              value={currentFormat.toUpperCase()}
            />
          </CompactSummaryStrip>

          <CompactSummaryStrip
            watermark={t("library.workspace.dialogs.exportSubtitle.outputSection")}
          >
            <CompactMeta
              label={t("library.workspace.dialogs.exportSubtitle.metrics.cueScope")}
              value={t("library.workspace.dialogs.exportVideo.cueCount").replace("{count}", String(cueCount))}
            />
            <CompactMeta
              label={t("library.workspace.fields.exportFormat")}
              value={targetFormat.toUpperCase()}
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportSubtitle.metrics.draftStatus")}
              value={
                hasUnsavedChanges
                  ? t("library.workspace.dialogs.exportSubtitle.draftDirty")
                  : t("library.workspace.dialogs.exportSubtitle.draftSaved")
              }
            />
            <CompactMeta
              label={t("library.workspace.dialogs.exportSubtitle.outputTitle")}
              value={resolvedSummary}
            />
          </CompactSummaryStrip>
        </div>

        <Tabs
          value={normalizedTargetFormat}
          onValueChange={onTargetFormatChange}
          className="flex min-h-0 flex-1 flex-col gap-3"
        >
          <TabsList className="grid w-full grid-cols-3 gap-1.5 sm:grid-cols-6">
            {FORMAT_OPTIONS.map((format) => (
              <TabsTrigger
                key={format}
                value={format}
                disabled={isSubmitting}
                className="uppercase"
              >
                {format.toUpperCase()}
              </TabsTrigger>
            ))}
          </TabsList>

          <DashboardDialogBody className="min-h-0 flex-1 overflow-y-auto pr-1">
            <div
              className={cn(
                "min-w-0 space-y-2 p-3",
                DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
              )}
            >
              <div className="space-y-2">
                <div className="grid gap-2">
                  <CompactInlineField
                    label={t("library.config.subtitleStyles.subtitleExportPresets")}
                    control={
                      <Select
                        value={visibleSelectedPreset?.id ?? ""}
                        onChange={(event) =>
                          onSelectedPresetIdChange(event.target.value)
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        <option value="">
                          {t("library.workspace.dialogs.exportSubtitle.profileManual")}
                        </option>
                        {filteredPresets.map((preset) => (
                          <option key={preset.id} value={preset.id}>
                            {preset.name || preset.id}
                          </option>
                        ))}
                      </Select>
                    }
                  />
                </div>

                <TabsContent
                  value="srt"
                  className="mt-0 data-[state=inactive]:hidden"
                >
                  <div className="grid gap-2 md:grid-cols-2">
                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.encoding")}
                    >
                      <Select
                        value={srtConfig.encoding || "utf-8"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            srt: {
                              ...(exportConfig.srt ?? {}),
                              encoding: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        <option value="utf-8">UTF-8</option>
                        <option value="gbk">GBK</option>
                        <option value="big5">Big5</option>
                      </Select>
                    </CompactFieldPanel>
                  </div>
                </TabsContent>

                <TabsContent
                  value="vtt"
                  className="mt-0 data-[state=inactive]:hidden"
                >
                  <div className="grid gap-2 md:grid-cols-2">
                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.kind")}
                    >
                      <Select
                        value={vttConfig.kind || "subtitles"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            vtt: {
                              ...(exportConfig.vtt ?? {}),
                              kind: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {vttKindOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.language")}
                    >
                      <Select
                        value={(vttConfig.language ?? "").trim() || "en-US"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            vtt: {
                              ...(exportConfig.vtt ?? {}),
                              language: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {effectiveVTTLanguageOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    </CompactFieldPanel>
                  </div>
                </TabsContent>

                {(["ass", "ssa"] as const).map((format) => (
                  <TabsContent
                    key={format}
                    value={format}
                    className="mt-0 data-[state=inactive]:hidden"
                  >
                    <div className="grid gap-2 md:grid-cols-2">
                      <CompactFieldPanel
                        label={t("library.workspace.dialogs.exportSubtitle.title")}
                      >
                        <input
                          value={assConfig.title || ""}
                          onChange={(event) =>
                            onExportConfigChange({
                              ...exportConfig,
                              ass: {
                                ...(exportConfig.ass ?? {}),
                                title: event.target.value,
                              },
                            })
                          }
                          disabled={isSubmitting}
                          className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                          placeholder={DEFAULT_SUBTITLE_EXPORT_ASS_TITLE}
                        />
                      </CompactFieldPanel>

                      <CompactFieldPanel
                        label={t("library.workspace.dialogs.exportSubtitle.playRes")}
                      >
                        <div className="grid grid-cols-2 gap-2">
                          <input
                            type="number"
                            min={1}
                            value={assConfig.playResX ?? 1920}
                            onChange={(event) =>
                              onExportConfigChange({
                                ...exportConfig,
                                ass: {
                                  ...(exportConfig.ass ?? {}),
                                  playResX:
                                    Number(event.target.value) > 0
                                      ? Number(event.target.value)
                                      : 1920,
                                },
                              })
                            }
                            disabled={isSubmitting}
                            className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                            placeholder="1920"
                          />
                          <input
                            type="number"
                            min={1}
                            value={assConfig.playResY ?? 1080}
                            onChange={(event) =>
                              onExportConfigChange({
                                ...exportConfig,
                                ass: {
                                  ...(exportConfig.ass ?? {}),
                                  playResY:
                                    Number(event.target.value) > 0
                                      ? Number(event.target.value)
                                      : 1080,
                                },
                              })
                            }
                            disabled={isSubmitting}
                            className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                            placeholder="1080"
                          />
                        </div>
                      </CompactFieldPanel>
                    </div>
                  </TabsContent>
                ))}

                <TabsContent
                  value="itt"
                  className="mt-0 data-[state=inactive]:hidden"
                >
                  <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.frameRate")}
                    >
                      <input
                        type="number"
                        min={1}
                        step="1"
                        value={effectiveITTFrameRate}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            itt: {
                              ...(exportConfig.itt ?? {}),
                              frameRate: normalizeITTFrameRate(
                                Number(event.target.value),
                              ),
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                        placeholder="30"
                      />
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.frameRateMultiplier")}
                    >
                      <Select
                        value={effectiveITTFrameRateMultiplier}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            itt: {
                              ...(exportConfig.itt ?? {}),
                              frameRateMultiplier:
                                normalizeITTFrameRateMultiplier(
                                  event.target.value,
                                ),
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {effectiveITTFrameRateMultiplierOptions.map(
                          (preset) => (
                            <option key={preset.value} value={preset.value}>
                              {preset.label}
                            </option>
                          ),
                        )}
                      </Select>
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.language")}
                    >
                      <Select
                        value={(ittConfig.language ?? "").trim() || "en-US"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            itt: {
                              ...(exportConfig.itt ?? {}),
                              language: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {effectiveITTLanguageOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    </CompactFieldPanel>
                  </div>
                </TabsContent>

                <TabsContent
                  value="fcpxml"
                  className="mt-0 data-[state=inactive]:hidden"
                >
                  <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.resolution")}
                      className="xl:col-span-2"
                    >
                      <div className="grid grid-cols-2 gap-2">
                        <input
                          type="number"
                          min={1}
                          value={fcpxmlConfig.width ?? 1920}
                          onChange={(event) =>
                            onExportConfigChange({
                              ...exportConfig,
                              fcpxml: {
                                ...(exportConfig.fcpxml ?? {}),
                                width:
                                  Number(event.target.value) > 0
                                    ? Number(event.target.value)
                                    : 1920,
                              },
                            })
                          }
                          disabled={isSubmitting}
                          className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                          placeholder="1920"
                        />
                        <input
                          type="number"
                          min={1}
                          value={fcpxmlConfig.height ?? 1080}
                          onChange={(event) =>
                            onExportConfigChange({
                              ...exportConfig,
                              fcpxml: {
                                ...(exportConfig.fcpxml ?? {}),
                                height:
                                  Number(event.target.value) > 0
                                    ? Number(event.target.value)
                                    : 1080,
                              },
                            })
                          }
                          disabled={isSubmitting}
                          className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                          placeholder="1080"
                        />
                      </div>
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.frameDuration")}
                    >
                      <Select
                        value={normalizeFCPXMLFrameDuration(
                          fcpxmlConfig.frameDuration,
                        )}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              frameDuration: normalizeFCPXMLFrameDuration(
                                event.target.value,
                              ),
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {FCPXML_FRAME_DURATION_PRESETS.map((preset) => (
                          <option key={preset.value} value={preset.value}>
                            {preset.label} ({preset.value})
                          </option>
                        ))}
                      </Select>
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.colorSpace")}
                    >
                      <Select
                        value={fcpxmlConfig.colorSpace || "1-1-1 (Rec. 709)"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              colorSpace: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        <option value="1-1-1 (Rec. 709)">Rec. 709</option>
                        <option value="Rec. 2020">Rec. 2020</option>
                        <option value="P3-D65">P3-D65</option>
                      </Select>
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.projectName")}
                      className="xl:col-span-2"
                    >
                      <input
                        value={fcpxmlConfig.projectName || ""}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              projectName: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                        placeholder={DEFAULT_SUBTITLE_EXPORT_PROJECT_NAME}
                      />
                    </CompactFieldPanel>
                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.libraryName")}
                      className="xl:col-span-2"
                    >
                      <input
                        value={fcpxmlConfig.libraryName || ""}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              libraryName: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                        placeholder={DEFAULT_SUBTITLE_EXPORT_LIBRARY_NAME}
                      />
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.eventName")}
                      className="xl:col-span-2"
                    >
                      <input
                        value={fcpxmlConfig.eventName || ""}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              eventName: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                        placeholder={DEFAULT_SUBTITLE_EXPORT_EVENT_NAME}
                      />
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.version")}
                    >
                      <input
                        value={fcpxmlConfig.version ?? "1.11"}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              version: event.target.value,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                      />
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.defaultLane")}
                    >
                      <input
                        type="number"
                        value={String(fcpxmlConfig.defaultLane ?? 1)}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              defaultLane:
                                Number(event.target.value) >= 0
                                  ? Number(event.target.value)
                                  : 1,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                      />
                    </CompactFieldPanel>

                    <CompactFieldPanel
                      label={t("library.workspace.dialogs.exportSubtitle.startTimecodeSeconds")}
                    >
                      <input
                        value={String(
                          fcpxmlConfig.startTimecodeSeconds ?? 3600,
                        )}
                        onChange={(event) =>
                          onExportConfigChange({
                            ...exportConfig,
                            fcpxml: {
                              ...(exportConfig.fcpxml ?? {}),
                              startTimecodeSeconds:
                                Number(event.target.value) > 0
                                  ? Number(event.target.value)
                                  : 3600,
                            },
                          })
                        }
                        disabled={isSubmitting}
                        className="h-8 w-full rounded-md border border-border/70 bg-background/80 px-2 text-xs"
                      />
                    </CompactFieldPanel>
                  </div>
                </TabsContent>
              </div>
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
              <Button size="compact" onClick={onSubmit} disabled={isSubmitting || disabled}>
                <Captions className="h-4 w-4" />
                {isSubmitting
                  ? t("library.workspace.dialogs.exportSubtitle.exporting")
                  : t("library.workspace.exportSubtitle")}
              </Button>,
              disabled && !isSubmitting ? disabledReason : "",
            )}
          </div>
        </DashboardDialogFooter>
      </DashboardDialogContent>
    </Dialog>
  );
}
