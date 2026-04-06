import * as React from "react"
import type { ReactNode } from "react"

import { AlertTriangle, Languages, Loader2, RotateCcw, ShieldCheck, SlidersHorizontal, Sparkles } from "lucide-react"

import { useI18n } from "@/shared/i18n"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"
import {
  Dialog,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
  DashboardDialogBody,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
} from "@/shared/ui/dashboard-dialog"
import { Select } from "@/shared/ui/select"
import { Switch } from "@/shared/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip"
import { cn } from "@/lib/utils"
import type {
  WorkspaceQaCheckDefinition,
  WorkspaceQaCheckId,
  WorkspaceQaCheckSettings,
} from "../../model/workspaceQa"

import type {
  WorkspaceConstraintOption,
  WorkspaceGuidelineOption,
  WorkspaceGuidelineProfileId,
  WorkspaceImportNormalizationOptions,
  WorkspaceLanguageTaskMode,
  WorkspaceProofreadOptions,
  WorkspaceQaSummary,
  WorkspaceSelectOption,
} from "./types"

type WorkspaceActionDialogsProps = {
  cueCount: number
  importSubtitleOpen: boolean
  onImportSubtitleOpenChange: (open: boolean) => void
  useCurrentGuidelineForImport: boolean
  onUseCurrentGuidelineForImportChange: (value: boolean) => void
  importGuidelineProfileId: WorkspaceGuidelineProfileId
  onImportGuidelineProfileIdChange: (value: WorkspaceGuidelineProfileId) => void
  normalizationOptions: WorkspaceImportNormalizationOptions
  onNormalizationOptionChange: (key: keyof WorkspaceImportNormalizationOptions, value: boolean) => void
  guidelineOptions: WorkspaceGuidelineOption[]
  guidelineProfileId: WorkspaceGuidelineProfileId
  onGuidelineChange: (value: WorkspaceGuidelineProfileId) => void
  activeGuidelineLabel: string
  onConfirmImportSubtitle: () => void
  qaSummary: WorkspaceQaSummary
  qaCheckDefinitions: WorkspaceQaCheckDefinition[]
  qaCheckSettings: WorkspaceQaCheckSettings
  onQaCheckToggle: (id: WorkspaceQaCheckId, value: boolean) => void
  languageTaskOpen: boolean
  onLanguageTaskOpenChange: (open: boolean) => void
  languageTaskMode: WorkspaceLanguageTaskMode
  onLanguageTaskModeChange: (value: WorkspaceLanguageTaskMode) => void
  translateLanguageOptions: WorkspaceSelectOption[]
  translateTargetLanguage: string
  onTranslateTargetLanguageChange: (value: string) => void
  translateGlossaryOptions: WorkspaceConstraintOption[]
  translateGlossaryProfileIds: string[]
  onTranslateGlossaryProfileToggle: (value: string, checked: boolean) => void
  referenceTrackOptions: WorkspaceSelectOption[]
  translateReferenceTrackId: string
  onTranslateReferenceTrackIdChange: (value: string) => void
  translatePromptOptions: WorkspaceConstraintOption[]
  translatePromptProfileIds: string[]
  onTranslatePromptProfileToggle: (value: string, checked: boolean) => void
  translateInlinePrompt: string
  onTranslateInlinePromptChange: (value: string) => void
  translatePromptProfileName: string
  onTranslatePromptProfileNameChange: (value: string) => void
  onSaveTranslatePromptProfile: () => void
  translateReady: boolean
  translateReadinessChecking: boolean
  translateReadinessTitle: string
  translateReadinessDescription: string
  translateTaskRunning?: boolean
  translateTaskLabel?: string
  translateActionDisabled?: boolean
  translateDisabledReason?: string
  onOpenTranslateSettings: () => void
  proofreadReady: boolean
  proofreadReadinessChecking: boolean
  proofreadReadinessTitle: string
  proofreadReadinessDescription: string
  proofreadTaskRunning?: boolean
  proofreadTaskLabel?: string
  proofreadActionDisabled?: boolean
  proofreadDisabledReason?: string
  onOpenProofreadSettings: () => void
  proofreadOptions: WorkspaceProofreadOptions
  onProofreadOptionChange: (key: keyof WorkspaceProofreadOptions, value: boolean) => void
  proofreadGlossaryOptions: WorkspaceConstraintOption[]
  proofreadGlossaryProfileIds: string[]
  onProofreadGlossaryProfileToggle: (value: string, checked: boolean) => void
  proofreadPromptOptions: WorkspaceConstraintOption[]
  proofreadPromptProfileIds: string[]
  onProofreadPromptProfileToggle: (value: string, checked: boolean) => void
  proofreadInlinePrompt: string
  onProofreadInlinePromptChange: (value: string) => void
  proofreadPromptProfileName: string
  onProofreadPromptProfileNameChange: (value: string) => void
  onSaveProofreadPromptProfile: () => void
  primaryTrackOptions: WorkspaceSelectOption[]
  primaryTrackId: string
  onPrimaryTrackChange: (value: string) => void
  primaryTrackLabel: string
  onQueueTranslate: () => void
  onQueueProofread: () => void
  restoreOriginalRunning: boolean
  onRestoreOriginal: () => void
}

type TaskSectionFilter = "required" | "optional" | "all"

const DASHBOARD_FORM_CONTROL_WIDTH_CLASS = "w-full md:w-[248px]"

function SectionCard({
  title,
  description,
  headerBadge,
  children,
}: {
  title: string
  description?: string
  headerBadge?: ReactNode
  children: ReactNode
}) {
  return (
    <section className={cn("min-w-0 overflow-x-hidden p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
      <div className={cn("mb-3 flex items-start justify-between gap-3", !description && "items-center")}>
        <div className="min-w-0 space-y-1">
          <div className="text-sm font-semibold text-foreground">{title}</div>
          {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
        </div>
        {headerBadge ? <div className="shrink-0">{headerBadge}</div> : null}
      </div>
      {children}
    </section>
  )
}

function TaskSectionFilterTabs({
  value,
  onValueChange,
}: {
  value: TaskSectionFilter
  onValueChange: (value: TaskSectionFilter) => void
}) {
  const { t } = useI18n()
  const options: Array<{ value: TaskSectionFilter; label: string }> = [
    { value: "required", label: t("library.workspace.dialogs.required") },
    { value: "optional", label: t("library.workspace.dialogs.languageTask.filterOptional") },
    { value: "all", label: t("library.workspace.dialogs.languageTask.filterAll") },
  ]

  return (
    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
      <div className="inline-flex items-center gap-1.5">
        <SlidersHorizontal className="h-3.5 w-3.5" />
        <span className="uppercase tracking-[0.08em]">{t("library.workspace.dialogs.languageTask.filterShow")}</span>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        {options.map((option) => {
          const active = value === option.value
          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={active}
              onClick={() => onValueChange(option.value)}
              className={cn(
                "inline-flex h-6 items-center justify-center rounded-full border px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1",
                active
                  ? "border-primary/30 bg-primary/10 text-primary"
                  : "border-border/60 bg-card text-muted-foreground hover:bg-accent/60 hover:text-foreground",
              )}
            >
              {option.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function DashboardFormRow({
  label,
  description,
  control,
  className,
  controlClassName,
  alignTop = false,
}: {
  label: string
  description?: string
  control: ReactNode
  className?: string
  controlClassName?: string
  alignTop?: boolean
}) {
  return (
    <div
      className={cn(
        "grid gap-3 px-3 py-2.5 md:grid-cols-[minmax(0,1fr)_248px]",
        !alignTop && "md:items-center",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        className,
      )}
    >
      <div className="min-w-0 space-y-1">
        <div className="text-xs font-medium text-foreground">{label}</div>
        {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
      </div>
      <div className={cn("min-w-0 md:justify-self-end", DASHBOARD_FORM_CONTROL_WIDTH_CLASS, controlClassName)}>{control}</div>
    </div>
  )
}

function SectionBadge({ children }: { children: ReactNode }) {
  return (
    <Badge variant="outline" className="h-5 rounded-md px-2 text-xs font-medium uppercase tracking-[0.08em]">
      {children}
    </Badge>
  )
}

function HeaderMetaBadge({ children }: { children: ReactNode }) {
  return (
    <Badge variant="outline" className="h-5 min-w-0 max-w-[180px] overflow-hidden rounded-md px-2 text-xs font-medium">
      <span className="truncate">{children}</span>
    </Badge>
  )
}

function DashboardHeaderCard({
  title,
  badge,
  className,
  children,
}: {
  title: string
  badge?: ReactNode
  className?: string
  children: ReactNode
}) {
  return (
    <section
      className={cn("flex h-full min-w-0 flex-col overflow-hidden px-4 py-3", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS, className)}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 text-sm font-semibold text-foreground">{title}</div>
        {badge ? <div className="min-w-0 shrink">{badge}</div> : null}
      </div>
      <div className="mt-1.5 min-h-0 min-w-0 flex-1">{children}</div>
    </section>
  )
}

function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string
  description?: string
  checked: boolean
  onCheckedChange: (value: boolean) => void
}) {
  return (
    <DashboardFormRow
      label={label}
      description={description}
      control={
        <div className="flex h-8 items-center justify-end">
          <Switch checked={checked} onCheckedChange={onCheckedChange} />
        </div>
      }
    />
  )
}

function SummaryStat({ label, value, tone = "default" }: { label: string; value: number; tone?: "default" | "warning" | "danger" }) {
  return (
    <div
      className={cn(
        "rounded-xl border px-3 py-3",
        tone === "danger"
          ? "border-rose-500/30 bg-rose-500/[0.06]"
          : tone === "warning"
            ? "border-amber-500/30 bg-amber-500/[0.06]"
            : DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <div className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{label}</div>
      <div className="mt-1 text-lg font-semibold text-foreground">{value}</div>
    </div>
  )
}

function CompactSelectField({
  label,
  description,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string
  description?: string
  value: string
  options: WorkspaceSelectOption[]
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const selectedOption = options.find((option) => option.value === value)

  return (
    <DashboardFormRow
      label={label}
      description={[description, selectedOption?.hint].filter(Boolean).join(" ")}
      control={
        <Select
          value={value}
          onChange={(event) => onChange(event.target.value)}
          disabled={disabled}
          className="h-8 w-full border-border/70 bg-background/80"
        >
          {options.map((option) => (
            <option key={`${option.value}-${option.label}`} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
      }
    />
  )
}

function ConstraintChecklist({
  title,
  description,
  emptyLabel,
  items,
  selectedValues,
  onToggle,
}: {
  title: string
  description?: string
  emptyLabel: string
  items: WorkspaceConstraintOption[]
  selectedValues: string[]
  onToggle: (value: string, checked: boolean) => void
}) {
  const { t } = useI18n()
  const selectedSet = new Set(selectedValues)

  return (
    <div className={cn("px-3 py-2.5", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
      <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_248px] md:items-center">
        <div className="min-w-0 space-y-1">
          <div className="text-xs font-medium text-foreground">{title}</div>
          {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
        </div>
        <div className={cn("min-w-0 md:justify-self-end", DASHBOARD_FORM_CONTROL_WIDTH_CLASS)}>
          <div className="flex h-8 items-center justify-end">
            <Badge variant="outline" className="h-6 rounded-md px-2 text-xs uppercase tracking-[0.08em]">
              {t("library.workspace.dialogs.languageTask.selectedCount").replace(
                "{count}",
                String(selectedValues.length),
              )}
            </Badge>
          </div>
        </div>
      </div>
      {items.length === 0 ? (
        <div className="mt-3 rounded-lg border border-dashed border-border/60 bg-card px-3 py-3 text-xs text-muted-foreground">
          {emptyLabel}
        </div>
      ) : (
        <div className="mt-3 flex min-w-0 flex-wrap gap-2">
          {items.map((item) => (
            <button
              key={item.value}
              type="button"
              aria-pressed={selectedSet.has(item.value)}
              onClick={() => onToggle(item.value, !selectedSet.has(item.value))}
              className={cn(
                "inline-flex h-8 min-w-0 max-w-full items-center gap-2 overflow-hidden rounded-lg border px-3 text-xs transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1",
                selectedSet.has(item.value)
                  ? "border-primary/40 bg-primary/10 text-primary"
                  : "border-border/60 bg-card text-foreground hover:bg-accent/60",
              )}
            >
              <span className="block min-w-0 flex-1 truncate">{item.label}</span>
              {item.badge ? (
                <span
                  className={cn(
                    "shrink-0 rounded border px-1.5 py-0.5 text-xs uppercase tracking-[0.06em]",
                    selectedSet.has(item.value)
                      ? "border-primary/30 bg-primary/5 text-primary"
                      : "border-border/60 text-muted-foreground",
                  )}
                >
                  {item.badge}
                </span>
              ) : null}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function PromptTextarea({
  label,
  description,
  value,
  placeholder,
  onChange,
  footer,
}: {
  label: string
  description?: string
  value: string
  placeholder: string
  onChange: (value: string) => void
  footer?: ReactNode
}) {
  return (
    <div className={cn("px-3 py-2.5", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
      <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_248px] md:items-start">
        <div className="min-w-0 space-y-1">
          <div className="text-xs font-medium text-foreground">{label}</div>
          {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
        </div>
        <div className={cn("hidden md:block md:justify-self-end", DASHBOARD_FORM_CONTROL_WIDTH_CLASS)} />
      </div>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        rows={4}
        className="mt-2 w-full rounded-md border border-input bg-background px-3 py-2 text-xs leading-5 shadow-sm outline-none transition-colors placeholder:text-xs placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring"
      />
      {footer ? <div className="mt-3">{footer}</div> : null}
    </div>
  )
}

function DashboardSummaryRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 text-xs leading-5 text-muted-foreground">
      <span>{label}</span>
      <span className="text-right font-medium text-foreground">{value}</span>
    </div>
  )
}

function DashboardMetricsCard({
  items,
  columns = 3,
}: {
  items: Array<{ label: string; value: ReactNode }>
  columns?: 2 | 3
}) {
  return (
    <div
      className={cn(
        "grid min-w-0 overflow-hidden rounded-lg border border-border/70 bg-card",
        columns === 2 ? "grid-cols-2" : "grid-cols-3",
      )}
    >
      {items.map((item, index) => (
        <div key={item.label} className={cn("min-w-0 px-2.5 py-2.5 sm:px-3", index > 0 && "border-l border-border/70")}>
          <div className="overflow-hidden text-xs uppercase leading-tight tracking-[0.04em] text-muted-foreground sm:text-xs xl:text-xs">
            {item.label}
          </div>
          <div className="mt-1 truncate text-sm font-semibold text-foreground xl:text-base">{item.value}</div>
        </div>
      ))}
    </div>
  )
}

export function WorkspaceActionDialogs({
  cueCount,
  importSubtitleOpen,
  onImportSubtitleOpenChange,
  useCurrentGuidelineForImport,
  onUseCurrentGuidelineForImportChange,
  importGuidelineProfileId,
  onImportGuidelineProfileIdChange,
  normalizationOptions,
  onNormalizationOptionChange,
  guidelineOptions,
  guidelineProfileId,
  onGuidelineChange,
  activeGuidelineLabel,
  onConfirmImportSubtitle,
  qaSummary,
  qaCheckDefinitions,
  qaCheckSettings,
  onQaCheckToggle,
  languageTaskOpen,
  onLanguageTaskOpenChange,
  languageTaskMode,
  onLanguageTaskModeChange,
  translateLanguageOptions,
  translateTargetLanguage,
  onTranslateTargetLanguageChange,
  translateGlossaryOptions,
  translateGlossaryProfileIds,
  onTranslateGlossaryProfileToggle,
  referenceTrackOptions,
  translateReferenceTrackId,
  onTranslateReferenceTrackIdChange,
  translatePromptOptions,
  translatePromptProfileIds,
  onTranslatePromptProfileToggle,
  translateInlinePrompt,
  onTranslateInlinePromptChange,
  translatePromptProfileName,
  onTranslatePromptProfileNameChange,
  onSaveTranslatePromptProfile,
  translateReady,
  translateReadinessChecking,
  translateReadinessTitle,
  translateReadinessDescription,
  translateTaskRunning = false,
  translateTaskLabel = "",
  translateActionDisabled = false,
  translateDisabledReason = "",
  onOpenTranslateSettings,
  proofreadReady,
  proofreadReadinessChecking,
  proofreadReadinessTitle,
  proofreadReadinessDescription,
  proofreadTaskRunning = false,
  proofreadTaskLabel = "",
  proofreadActionDisabled = false,
  proofreadDisabledReason = "",
  onOpenProofreadSettings,
  proofreadOptions,
  onProofreadOptionChange,
  proofreadGlossaryOptions,
  proofreadGlossaryProfileIds,
  onProofreadGlossaryProfileToggle,
  proofreadPromptOptions,
  proofreadPromptProfileIds,
  onProofreadPromptProfileToggle,
  proofreadInlinePrompt,
  onProofreadInlinePromptChange,
  proofreadPromptProfileName,
  onProofreadPromptProfileNameChange,
  onSaveProofreadPromptProfile,
  primaryTrackOptions,
  primaryTrackId,
  onPrimaryTrackChange,
  primaryTrackLabel,
  onQueueTranslate,
  onQueueProofread,
  restoreOriginalRunning,
  onRestoreOriginal,
}: WorkspaceActionDialogsProps) {
  const { t } = useI18n()
  const translateLanguageDisabled = translateLanguageOptions.length === 0
  const translateSelectedLanguageLabel =
    translateLanguageOptions.find((option) => option.value === translateTargetLanguage)?.label ??
    (translateTargetLanguage
      ? translateTargetLanguage.toUpperCase()
      : t("library.workspace.dialogs.languageTask.notSelected"))
  const translateStructuredConstraintCount =
    translateGlossaryProfileIds.length + (translateReferenceTrackId ? 1 : 0)
  const translatePromptConstraintCount =
    translatePromptProfileIds.length + (translateInlinePrompt.trim() ? 1 : 0)
  const translateSelectedConstraintCount = translateStructuredConstraintCount + translatePromptConstraintCount
  const proofreadPromptConstraintCount =
    proofreadGlossaryProfileIds.length + proofreadPromptProfileIds.length + (proofreadInlinePrompt.trim() ? 1 : 0)
  const proofreadEnabledOptionCount = Object.values(proofreadOptions).filter(Boolean).length
  const enabledNormalizationCount = Object.values(normalizationOptions).filter(Boolean).length
  const languageTaskDialogRef = React.useRef<HTMLDivElement | null>(null)
  const [taskSectionFilter, setTaskSectionFilter] = React.useState<TaskSectionFilter>("required")

  React.useEffect(() => {
    if (languageTaskOpen && (languageTaskMode === "translate" || languageTaskMode === "proofread")) {
      setTaskSectionFilter("required")
    }
  }, [languageTaskMode, languageTaskOpen])

  const withDisabledTooltip = React.useCallback((content: ReactNode, tooltipLabel: string) => {
    if (!tooltipLabel.trim()) {
      return content
    }
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex">{content}</span>
        </TooltipTrigger>
        <TooltipContent className="max-w-[18rem] text-xs leading-5">{tooltipLabel}</TooltipContent>
      </Tooltip>
    )
  }, [])

  return (
    <>
      <Dialog open={importSubtitleOpen} onOpenChange={onImportSubtitleOpenChange}>
        <DashboardDialogContent size="standard" className="flex max-h-[80vh] min-h-0 flex-col gap-4 text-xs">
          <DashboardDialogHeader>
            <DialogTitle>{t("library.workspace.dialogs.importSubtitle.title")}</DialogTitle>
            <DialogDescription>
              {t("library.workspace.dialogs.importSubtitle.description")}
            </DialogDescription>
          </DashboardDialogHeader>

          <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(220px,248px)]">
            <DashboardHeaderCard
              title={t("library.task.summary")}
              badge={<SectionBadge>{t("library.workspace.dialogs.importSubtitle.title")}</SectionBadge>}
            >
              <div className="space-y-2">
                <DashboardSummaryRow
                  label={t("library.workspace.dialogs.importSubtitle.currentGuideline")}
                  value={activeGuidelineLabel}
                />
                <DashboardSummaryRow
                  label={t("library.workspace.dialogs.importSubtitle.workspaceCues")}
                  value={t("library.workspace.dialogs.shared.cueCount").replace("{count}", String(cueCount))}
                />
                <DashboardSummaryRow
                  label={t("library.workspace.dialogs.importSubtitle.guidelineSource")}
                  value={
                    useCurrentGuidelineForImport
                      ? t("library.workspace.dialogs.importSubtitle.guidelineSourceCurrent")
                      : t("library.workspace.dialogs.importSubtitle.guidelineSourceOverride")
                  }
                />
              </div>
            </DashboardHeaderCard>

            <DashboardHeaderCard
              title={t("library.task.overview")}
              badge={<HeaderMetaBadge>{t("library.workspace.dialogs.importSubtitle.subtitleOnly")}</HeaderMetaBadge>}
            >
              <DashboardMetricsCard
                columns={2}
                items={[
                  { label: t("library.workspace.dialogs.importSubtitle.metrics.normalization"), value: `${enabledNormalizationCount}` },
                  { label: t("library.workspace.dialogs.importSubtitle.metrics.cueScope"), value: `${cueCount}` },
                  {
                    label: t("library.workspace.dialogs.importSubtitle.metrics.guideline"),
                    value: useCurrentGuidelineForImport
                      ? t("library.workspace.dialogs.importSubtitle.guidelineShortCurrent")
                      : t("library.workspace.dialogs.importSubtitle.guidelineShortOverride"),
                  },
                  { label: t("library.workspace.dialogs.importSubtitle.metrics.nextStep"), value: t("library.workspace.dialogs.importSubtitle.pickFile") },
                ]}
              />
            </DashboardHeaderCard>
          </div>

          <DashboardDialogBody className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
            <SectionCard
              title={t("library.workspace.dialogs.importSubtitle.normalizationTitle")}
              description={t("library.workspace.dialogs.importSubtitle.normalizationDescription")}
            >
              <div className="space-y-3">
                <ToggleRow
                  label={t("library.workspace.dialogs.importSubtitle.normalizeLineBreaks")}
                  description={t("library.workspace.dialogs.importSubtitle.normalizeLineBreaksDescription")}
                  checked={normalizationOptions.normalizeLineBreaks}
                  onCheckedChange={(value) => onNormalizationOptionChange("normalizeLineBreaks", value)}
                />
                <ToggleRow
                  label={t("library.workspace.dialogs.importSubtitle.trimWhitespace")}
                  description={t("library.workspace.dialogs.importSubtitle.trimWhitespaceDescription")}
                  checked={normalizationOptions.trimWhitespace}
                  onCheckedChange={(value) => onNormalizationOptionChange("trimWhitespace", value)}
                />
                <ToggleRow
                  label={t("library.workspace.dialogs.importSubtitle.removeBlankLines")}
                  description={t("library.workspace.dialogs.importSubtitle.removeBlankLinesDescription")}
                  checked={normalizationOptions.removeBlankLines}
                  onCheckedChange={(value) => onNormalizationOptionChange("removeBlankLines", value)}
                />
                <ToggleRow
                  label={t("library.workspace.dialogs.importSubtitle.repairEncoding")}
                  description={t("library.workspace.dialogs.importSubtitle.repairEncodingDescription")}
                  checked={normalizationOptions.repairEncoding}
                  onCheckedChange={(value) => onNormalizationOptionChange("repairEncoding", value)}
                />
              </div>
            </SectionCard>

            <SectionCard
              title={t("library.workspace.dialogs.importSubtitle.guidelineInheritanceTitle")}
              description={t("library.workspace.dialogs.importSubtitle.guidelineInheritanceDescription")}
            >
              <div className="space-y-3">
                <ToggleRow
                  label={t("library.workspace.dialogs.importSubtitle.useCurrentGuideline")}
                  description={t("library.workspace.dialogs.importSubtitle.currentProfile").replace(
                    "{name}",
                    activeGuidelineLabel,
                  )}
                  checked={useCurrentGuidelineForImport}
                  onCheckedChange={onUseCurrentGuidelineForImportChange}
                />
                <div className={cn("px-3 py-2.5", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
                  <div className="mb-2 text-xs text-foreground">
                    {t("library.workspace.dialogs.importSubtitle.overrideProfile")}
                  </div>
                  <Select
                    value={importGuidelineProfileId}
                    onChange={(event) => onImportGuidelineProfileIdChange(event.target.value as WorkspaceGuidelineProfileId)}
                    disabled={useCurrentGuidelineForImport}
                    className="h-8 border-border/70 bg-background/80"
                  >
                    {guidelineOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
            </SectionCard>
          </DashboardDialogBody>

          <DashboardDialogFooter>
            <Button variant="ghost" size="compact" onClick={() => onImportSubtitleOpenChange(false)}>
              {t("common.close")}
            </Button>
            <Button size="compact" onClick={onConfirmImportSubtitle}>
              {t("library.workspace.dialogs.importSubtitle.continueImport")}
            </Button>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>

      <Dialog open={languageTaskOpen} onOpenChange={onLanguageTaskOpenChange}>
        <DashboardDialogContent
          size="workspace"
          ref={languageTaskDialogRef}
          tabIndex={-1}
          onOpenAutoFocus={(event) => {
            event.preventDefault()
            languageTaskDialogRef.current?.focus()
          }}
          className="grid h-[85vh] max-h-[85vh] grid-rows-[minmax(0,1fr)_auto] overflow-hidden text-xs"
        >
          <DialogHeader className="sr-only">
            <DialogTitle>{t("library.workspace.dialogs.languageTask.title")}</DialogTitle>
            <DialogDescription>
              {t("library.workspace.dialogs.languageTask.description")}
            </DialogDescription>
          </DialogHeader>

          <Tabs
            value={languageTaskMode}
            onValueChange={(value) => onLanguageTaskModeChange(value as WorkspaceLanguageTaskMode)}
            className="grid min-h-0 min-w-0 grid-rows-[auto_auto_minmax(0,1fr)] gap-4 overflow-hidden"
          >
            <TabsList className="w-fit">
              <TabsTrigger value="translate">
                <Languages className="h-3.5 w-3.5" />
                <span>{t("library.workspace.actions.translate")}</span>
                {translateTaskRunning ? (
                  <span className="ml-1 rounded-full border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                    {translateTaskLabel || t("library.workspace.task.running")}
                  </span>
                ) : null}
              </TabsTrigger>
              <TabsTrigger value="proofread">
                <Sparkles className="h-3.5 w-3.5" />
                <span>{t("library.workspace.actions.proofread")}</span>
                {proofreadTaskRunning ? (
                  <span className="ml-1 rounded-full border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                    {proofreadTaskLabel || t("library.workspace.task.running")}
                  </span>
                ) : null}
              </TabsTrigger>
              <TabsTrigger value="qa">
                <ShieldCheck className="h-3.5 w-3.5" />
                <span>{t("library.workspace.header.qa")}</span>
              </TabsTrigger>
              <TabsTrigger value="restore">
                <RotateCcw className="h-3.5 w-3.5" />
                <span>{t("library.workspace.header.restore")}</span>
              </TabsTrigger>
            </TabsList>

            <div className="grid items-stretch gap-4 lg:grid-cols-[minmax(0,4fr)_minmax(220px,1fr)]">
              {languageTaskMode === "translate" ? (
                <>
                  <DashboardHeaderCard
                    title={t("library.task.summary")}
                    badge={<SectionBadge>{t("library.workspace.actions.translate")}</SectionBadge>}
                  >
                    <div className="grid items-start gap-3 md:grid-cols-[minmax(0,1fr)_minmax(216px,248px)]">
                      <div className="space-y-2">
                        <DashboardSummaryRow
                          label={t("library.workspace.dialogs.languageTask.targetLanguage")}
                          value={translateSelectedLanguageLabel}
                        />
                        <DashboardSummaryRow
                          label={t("library.workspace.dialogs.languageTask.sourceLane")}
                          value={primaryTrackLabel}
                        />
                      </div>
                      <DashboardMetricsCard
                        items={[
                          { label: t("library.workspace.dialogs.languageTask.metrics.profiles"), value: translateStructuredConstraintCount },
                          { label: t("library.workspace.dialogs.languageTask.metrics.prompt"), value: translatePromptConstraintCount },
                          { label: t("library.workspace.dialogs.languageTask.metrics.total"), value: translateSelectedConstraintCount },
                        ]}
                      />
                    </div>
                  </DashboardHeaderCard>

                  <DashboardHeaderCard
                    title={t("library.workspace.dialogs.languageTask.modelTitle")}
                    badge={<HeaderMetaBadge>{translateReadinessTitle}</HeaderMetaBadge>}
                    className={cn(
                      translateReady
                        ? "border-emerald-500/25 bg-emerald-500/[0.06]"
                        : "border-amber-500/25 bg-amber-500/[0.06]",
                    )}
                  >
                    <div className="flex h-full min-h-0 flex-col">
                      <div className={cn("line-clamp-3 text-xs leading-5", translateReady ? "text-emerald-800" : "text-amber-800")}>
                        {translateReadinessDescription}
                      </div>
                      {!translateReady ? (
                        <Button
                          variant="outline"
                          size="compact"
                          className="mt-auto self-start border-border/70 bg-background/80"
                          onClick={onOpenTranslateSettings}
                          disabled={translateReadinessChecking}
                        >
                          {t("library.workspace.dialogs.languageTask.openSettings")}
                        </Button>
                      ) : null}
                    </div>
                  </DashboardHeaderCard>
                </>
              ) : languageTaskMode === "proofread" ? (
                <>
                  <DashboardHeaderCard
                    title={t("library.task.summary")}
                    badge={<SectionBadge>{t("library.workspace.actions.proofread")}</SectionBadge>}
                  >
                    <div className="grid items-start gap-3 md:grid-cols-[minmax(0,1fr)_minmax(216px,248px)]">
                      <div className="space-y-2">
                        <DashboardSummaryRow
                          label={t("library.workspace.dialogs.languageTask.sourceLane")}
                          value={primaryTrackLabel}
                        />
                      </div>
                      <DashboardMetricsCard
                        columns={2}
                        items={[
                          { label: t("library.workspace.dialogs.languageTask.metrics.checks"), value: proofreadEnabledOptionCount },
                          { label: t("library.workspace.dialogs.languageTask.metrics.total"), value: proofreadPromptConstraintCount },
                        ]}
                      />
                    </div>
                  </DashboardHeaderCard>

                  <DashboardHeaderCard
                    title={t("library.workspace.dialogs.languageTask.modelTitle")}
                    badge={<HeaderMetaBadge>{proofreadReadinessTitle}</HeaderMetaBadge>}
                    className={cn(
                      proofreadReady
                        ? "border-emerald-500/25 bg-emerald-500/[0.06]"
                        : "border-amber-500/25 bg-amber-500/[0.06]",
                    )}
                  >
                    <div className="flex h-full min-h-0 flex-col">
                      <div className={cn("line-clamp-3 text-xs leading-5", proofreadReady ? "text-emerald-800" : "text-amber-800")}>
                        {proofreadReadinessDescription}
                      </div>
                      {!proofreadReady ? (
                        <Button
                          variant="outline"
                          size="compact"
                          className="mt-auto self-start border-border/70 bg-background/80"
                          onClick={onOpenProofreadSettings}
                          disabled={proofreadReadinessChecking}
                        >
                          {t("library.workspace.dialogs.languageTask.openSettings")}
                        </Button>
                      ) : null}
                    </div>
                  </DashboardHeaderCard>
                </>
              ) : languageTaskMode === "qa" ? (
                <>
                  <DashboardHeaderCard
                    title={t("library.workspace.dialogs.languageTask.issueBreakdownTitle")}
                    badge={<SectionBadge>{t("library.workspace.header.qa")}</SectionBadge>}
                  >
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
                      {qaCheckDefinitions.map((item) => {
                        const value = qaSummary.issueCounts[item.id]
                        return (
                          <SummaryStat
                            key={item.id}
                            label={item.label}
                            value={value}
                            tone={value > 0 && qaCheckSettings[item.id] ? "warning" : "default"}
                          />
                        )
                      })}
                    </div>
                  </DashboardHeaderCard>

                  <DashboardHeaderCard
                    title={t("library.workspace.header.guideline")}
                    badge={<HeaderMetaBadge>{activeGuidelineLabel}</HeaderMetaBadge>}
                  >
                    <div className="flex h-full min-h-0 flex-col">
                      <div className="line-clamp-3 text-xs leading-5 text-muted-foreground">
                        {t("library.workspace.dialogs.languageTask.qaGuidelineDescription")}
                      </div>
                    </div>
                  </DashboardHeaderCard>
                </>
              ) : (
                <>
                  <DashboardHeaderCard
                    title={t("library.task.summary")}
                    badge={<SectionBadge>{t("library.workspace.header.restore")}</SectionBadge>}
                  >
                    <div className="grid items-start gap-3 md:grid-cols-[minmax(0,1fr)_minmax(216px,248px)]">
                      <div className="space-y-2">
                        <DashboardSummaryRow
                          label={t("library.workspace.dialogs.languageTask.currentLane")}
                          value={primaryTrackLabel}
                        />
                        <DashboardSummaryRow
                          label={t("library.workspace.dialogs.shared.cueScope")}
                          value={t("library.workspace.dialogs.shared.cueCount").replace("{count}", String(cueCount))}
                        />
                      </div>
                      <DashboardMetricsCard
                        columns={2}
                        items={[
                          { label: t("library.workspace.dialogs.languageTask.metrics.draft"), value: cueCount },
                          { label: t("library.workspace.dialogs.languageTask.metrics.action"), value: t("library.workspace.dialogs.languageTask.restoreAction") },
                        ]}
                      />
                    </div>
                  </DashboardHeaderCard>

                  <DashboardHeaderCard
                    title={t("library.workspace.dialogs.languageTask.warningTitle")}
                    badge={<HeaderMetaBadge>{t("library.workspace.dialogs.languageTask.destructive")}</HeaderMetaBadge>}
                    className="border-rose-500/25 bg-rose-500/[0.06]"
                  >
                    <div className="line-clamp-3 text-xs leading-5 text-rose-800">
                      {t("library.workspace.dialogs.languageTask.restoreWarningDescription")}
                    </div>
                  </DashboardHeaderCard>
                </>
              )}
            </div>

            <TabsContent value="translate" className="mt-0 min-h-0 overflow-y-auto pr-1 data-[state=inactive]:hidden">
              <div className="space-y-4">
                <TaskSectionFilterTabs value={taskSectionFilter} onValueChange={setTaskSectionFilter} />

                {(taskSectionFilter === "required" || taskSectionFilter === "all") && (
                  <SectionCard
                    title={t("library.workspace.dialogs.languageTask.targetTitle")}
                    headerBadge={<SectionBadge>{t("library.workspace.dialogs.required")}</SectionBadge>}
                  >
                    <div className="space-y-3">
                      <DashboardFormRow
                        label={t("library.workspace.dialogs.languageTask.targetLanguage")}
                        description={t("library.workspace.dialogs.languageTask.targetLanguageDescription")}
                        control={
                          <Select
                            value={translateTargetLanguage}
                            onChange={(event) => onTranslateTargetLanguageChange(event.target.value)}
                            disabled={translateLanguageDisabled}
                            className="h-8 w-full border-border/70 bg-background/80"
                          >
                            {translateLanguageDisabled ? (
                              <option value="">{t("library.workspace.dialogs.languageTask.noLanguageProfile")}</option>
                            ) : (
                              translateLanguageOptions.map((option) => (
                                <option key={option.value} value={option.value}>
                                  {option.label}
                                </option>
                              ))
                            )}
                          </Select>
                        }
                      />
                      <CompactSelectField
                        label={t("library.workspace.dialogs.languageTask.sourceLane")}
                        description={t("library.workspace.dialogs.languageTask.translateSourceLaneDescription")}
                        value={primaryTrackId}
                        options={primaryTrackOptions}
                        onChange={onPrimaryTrackChange}
                        disabled={primaryTrackOptions.length === 0}
                      />
                    </div>
                  </SectionCard>
                )}

                {(taskSectionFilter === "optional" || taskSectionFilter === "all") && (
                  <>
                    <SectionCard
                      title={t("library.workspace.dialogs.languageTask.profilesTitle")}
                      headerBadge={<SectionBadge>{t("library.workspace.dialogs.languageTask.filterOptional")}</SectionBadge>}
                    >
                      <div className="space-y-3">
                        <ConstraintChecklist
                          title={t("library.workspace.dialogs.languageTask.glossaryProfiles")}
                          description={t("library.workspace.dialogs.languageTask.glossaryProfilesDescription")}
                          emptyLabel={t("library.workspace.dialogs.languageTask.noGlossaryProfiles")}
                          items={translateGlossaryOptions}
                          selectedValues={translateGlossaryProfileIds}
                          onToggle={onTranslateGlossaryProfileToggle}
                        />
                        <CompactSelectField
                          label={t("library.task.referenceTracks")}
                          description={t("library.workspace.dialogs.languageTask.referenceTrackDescription")}
                          value={translateReferenceTrackId}
                          options={referenceTrackOptions}
                          onChange={onTranslateReferenceTrackIdChange}
                        />
                      </div>
                    </SectionCard>

                    <SectionCard
                      title={t("library.task.inlinePrompt")}
                      headerBadge={<SectionBadge>{t("library.workspace.dialogs.languageTask.filterOptional")}</SectionBadge>}
                    >
                      <div className="space-y-3">
                        <ConstraintChecklist
                          title={t("library.task.promptProfiles")}
                          description={t("library.workspace.dialogs.languageTask.promptProfilesDescription")}
                          emptyLabel={t("library.workspace.dialogs.languageTask.noPromptProfiles")}
                          items={translatePromptOptions}
                          selectedValues={translatePromptProfileIds}
                          onToggle={onTranslatePromptProfileToggle}
                        />
                        <PromptTextarea
                          label={t("library.task.inlinePrompt")}
                          description={t("library.workspace.dialogs.languageTask.inlinePromptDescription")}
                          value={translateInlinePrompt}
                          placeholder={t("library.workspace.dialogs.languageTask.translateInlinePromptPlaceholder")}
                          onChange={onTranslateInlinePromptChange}
                          footer={
                            <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end sm:items-center">
                              <Input
                                value={translatePromptProfileName}
                                onChange={(event) => onTranslatePromptProfileNameChange(event.target.value)}
                                placeholder={t("library.workspace.dialogs.languageTask.optionalProfileName")}
                                className="min-w-0 h-8 text-xs placeholder:text-xs sm:w-auto sm:max-w-[248px] sm:flex-1"
                              />
                              <Button
                                type="button"
                                variant="outline"
                                size="compact"
                                className="shrink-0"
                                disabled={translateInlinePrompt.trim().length === 0}
                                onClick={onSaveTranslatePromptProfile}
                              >
                                {t("library.workspace.dialogs.languageTask.saveAsPromptProfile")}
                              </Button>
                            </div>
                          }
                        />
                      </div>
                    </SectionCard>
                  </>
                )}
              </div>
            </TabsContent>

            <TabsContent value="proofread" className="mt-0 min-h-0 overflow-y-auto pr-1 data-[state=inactive]:hidden">
              <div className="space-y-4">
                <TaskSectionFilterTabs value={taskSectionFilter} onValueChange={setTaskSectionFilter} />

                {(taskSectionFilter === "required" || taskSectionFilter === "all") && (
                  <SectionCard
                    title={t("library.workspace.dialogs.languageTask.scopeTitle")}
                    headerBadge={<SectionBadge>{t("library.workspace.dialogs.required")}</SectionBadge>}
                  >
                    <div className="space-y-3">
                      <CompactSelectField
                        label={t("library.workspace.dialogs.languageTask.sourceLane")}
                        description={t("library.workspace.dialogs.languageTask.proofreadSourceLaneDescription")}
                        value={primaryTrackId}
                        options={primaryTrackOptions}
                        onChange={onPrimaryTrackChange}
                        disabled={primaryTrackOptions.length === 0}
                      />
                      <ToggleRow
                        label={t("library.workspace.dialogs.languageTask.proofreadChecks.spelling")}
                        description={t("library.workspace.dialogs.languageTask.proofreadChecks.spellingDescription")}
                        checked={proofreadOptions.spelling}
                        onCheckedChange={(value) => onProofreadOptionChange("spelling", value)}
                      />
                      <ToggleRow
                        label={t("library.workspace.dialogs.languageTask.proofreadChecks.punctuation")}
                        description={t("library.workspace.dialogs.languageTask.proofreadChecks.punctuationDescription")}
                        checked={proofreadOptions.punctuation}
                        onCheckedChange={(value) => onProofreadOptionChange("punctuation", value)}
                      />
                      <ToggleRow
                        label={t("library.workspace.dialogs.languageTask.proofreadChecks.terminology")}
                        description={t("library.workspace.dialogs.languageTask.proofreadChecks.terminologyDescription")}
                        checked={proofreadOptions.terminology}
                        onCheckedChange={(value) => onProofreadOptionChange("terminology", value)}
                      />
                    </div>
                  </SectionCard>
                )}

                {(taskSectionFilter === "optional" || taskSectionFilter === "all") && (
                  <SectionCard
                    title={t("library.task.inlinePrompt")}
                    headerBadge={<SectionBadge>{t("library.workspace.dialogs.languageTask.filterOptional")}</SectionBadge>}
                  >
                    <div className="space-y-3">
                      <ConstraintChecklist
                        title={t("library.workspace.dialogs.languageTask.glossaryProfiles")}
                        description={t("library.workspace.dialogs.languageTask.glossaryProfilesDescription")}
                        emptyLabel={t("library.workspace.dialogs.languageTask.noGlossaryProfiles")}
                        items={proofreadGlossaryOptions}
                        selectedValues={proofreadGlossaryProfileIds}
                        onToggle={onProofreadGlossaryProfileToggle}
                      />
                      <ConstraintChecklist
                        title={t("library.task.promptProfiles")}
                        description={t("library.workspace.dialogs.languageTask.proofreadPromptProfilesDescription")}
                        emptyLabel={t("library.workspace.dialogs.languageTask.noProofreadPromptProfiles")}
                        items={proofreadPromptOptions}
                        selectedValues={proofreadPromptProfileIds}
                        onToggle={onProofreadPromptProfileToggle}
                      />
                      <PromptTextarea
                        label={t("library.task.inlinePrompt")}
                        description={t("library.workspace.dialogs.languageTask.inlinePromptDescription")}
                        value={proofreadInlinePrompt}
                        placeholder={t("library.workspace.dialogs.languageTask.proofreadInlinePromptPlaceholder")}
                        onChange={onProofreadInlinePromptChange}
                        footer={
                          <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end sm:items-center">
                            <Input
                              value={proofreadPromptProfileName}
                              onChange={(event) => onProofreadPromptProfileNameChange(event.target.value)}
                              placeholder={t("library.workspace.dialogs.languageTask.optionalProfileName")}
                              className="min-w-0 h-8 text-xs placeholder:text-xs sm:w-auto sm:max-w-[248px] sm:flex-1"
                            />
                            <Button
                              type="button"
                              variant="outline"
                              size="compact"
                              className="shrink-0"
                              disabled={proofreadInlinePrompt.trim().length === 0}
                              onClick={onSaveProofreadPromptProfile}
                            >
                              {t("library.workspace.dialogs.languageTask.saveAsPromptProfile")}
                            </Button>
                          </div>
                        }
                      />
                    </div>
                  </SectionCard>
                )}
              </div>
            </TabsContent>

            <TabsContent value="qa" className="mt-0 min-h-0 overflow-y-auto pr-1 data-[state=inactive]:hidden">
              <div className="space-y-4">
                <SectionCard title={t("library.workspace.header.guideline")}>
                  <DashboardFormRow
                    label={t("library.workspace.dialogs.languageTask.qaGuideline")}
                    description={t("library.workspace.dialogs.languageTask.qaGuidelineDescriptionLong")}
                    control={
                      <Select
                        value={guidelineProfileId}
                        onChange={(event) => onGuidelineChange(event.target.value as WorkspaceGuidelineProfileId)}
                        className="h-8 w-full border-border/70 bg-background/80"
                      >
                        {guidelineOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    }
                  />
                </SectionCard>

                <SectionCard
                  title={t("library.workspace.dialogs.languageTask.qaRealtimeTitle")}
                  description={t("library.workspace.dialogs.languageTask.qaRealtimeDescription")}
                >
                  <div className="space-y-1">
                    {qaCheckDefinitions.map((item) => (
                      <ToggleRow
                        key={item.id}
                        label={item.label}
                        description={item.description}
                        checked={qaCheckSettings[item.id]}
                        onCheckedChange={(value) => onQaCheckToggle(item.id, value)}
                      />
                    ))}
                  </div>
                </SectionCard>
              </div>
            </TabsContent>

            <TabsContent value="restore" className="mt-0 min-h-0 overflow-y-auto pr-1 data-[state=inactive]:hidden">
              <div className="space-y-4">
                <SectionCard
                  title={t("library.workspace.dialogs.languageTask.restoreOriginalTitle")}
                  headerBadge={<SectionBadge>{t("library.workspace.dialogs.languageTask.danger")}</SectionBadge>}
                >
                  <div className="space-y-3">
                    <DashboardFormRow
                      label={t("library.workspace.dialogs.languageTask.resetCurrentDraft")}
                      description={t("library.workspace.dialogs.languageTask.resetCurrentDraftDescription")}
                      alignTop
                      control={
                        <div className="flex min-h-8 flex-wrap items-center justify-end gap-2">
                          <Button
                            variant="destructive"
                            size="compact"
                            onClick={onRestoreOriginal}
                            disabled={restoreOriginalRunning}
                          >
                            {restoreOriginalRunning ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="h-3.5 w-3.5" />}
                            <span>{t("library.workspace.dialogs.languageTask.restoreOriginalAction")}</span>
                          </Button>
                        </div>
                      }
                    />
                    <div className="flex items-start gap-2 rounded-lg border border-rose-500/20 bg-rose-500/[0.05] px-3 py-2.5 text-xs leading-5 text-rose-900">
                      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-rose-700" />
                      <span>
                        {t("library.workspace.dialogs.languageTask.restoreOriginalWarning")}
                      </span>
                    </div>
                  </div>
                </SectionCard>
              </div>
            </TabsContent>
          </Tabs>

          <DashboardDialogFooter>
            <Button
              variant="outline"
              size="compact"
              className="border-border/70 bg-background/80"
              onClick={() => onLanguageTaskOpenChange(false)}
            >
              {t("common.close")}
            </Button>
            {languageTaskMode === "translate" ? (
              withDisabledTooltip(
                <Button
                  size="compact"
                  onClick={onQueueTranslate}
                  disabled={
                    translateActionDisabled ||
                    translateTaskRunning ||
                    translateLanguageDisabled ||
                    !translateTargetLanguage ||
                    !translateReady ||
                    translateReadinessChecking
                  }
                >
                  <Languages className="h-3.5 w-3.5" />
                  <span>
                    {translateTaskRunning
                      ? translateTaskLabel || t("library.workspace.task.running")
                      : t("library.workspace.dialogs.languageTask.queueTranslation")}
                  </span>
                </Button>,
                translateActionDisabled && !translateTaskRunning ? translateDisabledReason : "",
              )
            ) : languageTaskMode === "proofread" ? (
              withDisabledTooltip(
                <Button
                  size="compact"
                  onClick={onQueueProofread}
                  disabled={
                    proofreadActionDisabled ||
                    proofreadTaskRunning ||
                    !proofreadReady ||
                    proofreadReadinessChecking
                  }
                >
                  <Sparkles className="h-3.5 w-3.5" />
                  <span>
                    {proofreadTaskRunning
                      ? proofreadTaskLabel || t("library.workspace.task.running")
                      : t("library.workspace.dialogs.languageTask.queueProofread")}
                  </span>
                </Button>,
                proofreadActionDisabled && !proofreadTaskRunning ? proofreadDisabledReason : "",
              )
            ) : null}
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>
    </>
  )
}
