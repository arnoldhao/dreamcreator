import type { ReactNode } from "react"

import {
  Columns2,
  Film,
  Languages,
  Loader2,
  PanelRight,
  RotateCcw,
  ShieldCheck,
  Sparkles,
} from "lucide-react"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"
import { DASHBOARD_CONTROL_GROUP_CLASS } from "@/shared/ui/dashboard"
import { Select } from "@/shared/ui/select"
import { Switch } from "@/shared/ui/switch"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip"
import { cn } from "@/lib/utils"

import type { LibraryWorkspaceEditor } from "../../model/workspaceStore"
import { ModeSwitcher } from "./ModeSwitcher"
import type {
  WorkspaceDisplayMode,
  WorkspaceGuidelineOption,
  WorkspaceGuidelineProfileId,
  WorkspaceSubtitleTrackOption,
  WorkspaceVideoOption,
} from "./types"
import {
  WORKSPACE_CONTROL_FIELD_CLASS,
  WORKSPACE_CONTROL_LABEL_CLASS,
  WORKSPACE_CONTROL_SELECT_CLASS,
  WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS,
} from "./controlStyles"

type WorkspaceHeaderMetaItem = {
  label: string
  value: string
  truncate?: boolean
}

type WorkspaceHeaderProps = {
  libraryName: string
  metaItems: WorkspaceHeaderMetaItem[]
  hasPendingReview?: boolean
  pendingReviewButtonLabel?: string
  pendingReviewMenuDescription?: string
  reviewCompletionReady?: boolean
  reviewApplying?: boolean
  reviewDiscarding?: boolean
  activeEditor: LibraryWorkspaceEditor
  onEditorChange: (value: LibraryWorkspaceEditor) => void
  activeVideoId: string
  videoOptions: WorkspaceVideoOption[]
  onVideoChange: (value: string) => void
  primarySubtitleTrackId: string
  subtitleTrackOptions: WorkspaceSubtitleTrackOption[]
  onPrimarySubtitleTrackChange: (value: string) => void
  secondarySubtitleTrackId: string
  comparisonTrackOptions: WorkspaceSubtitleTrackOption[]
  onSecondarySubtitleTrackChange: (value: string) => void
  displayMode: WorkspaceDisplayMode
  canUseDualDisplay?: boolean
  dualDisplayDisabledReason?: string
  onDisplayModeChange: (value: WorkspaceDisplayMode) => void
  autoFollow?: boolean
  onAutoFollowChange?: (value: boolean) => void
  subtitleStyleSidebarOpen?: boolean
  onSubtitleStyleSidebarOpenChange?: (value: boolean) => void
  guidelineProfileId: WorkspaceGuidelineProfileId
  guidelineOptions: WorkspaceGuidelineOption[]
  onGuidelineChange: (value: WorkspaceGuidelineProfileId) => void
  canSubtitleActions: boolean
  canTranslateAction?: boolean
  canProofreadAction?: boolean
  canQaAction?: boolean
  canRestoreAction?: boolean
  translateDisabledReason?: string
  proofreadDisabledReason?: string
  qaDisabledReason?: string
  translateButtonLabel?: string
  proofreadButtonLabel?: string
  qaButtonLabel?: string
  restoreButtonLabel?: string
  translateRunning?: boolean
  proofreadRunning?: boolean
  qaRunning?: boolean
  onRunQa: () => void
  onTranslate: () => void
  onProofread: () => void
  onRestore: () => void
  onCompleteReview?: () => void
  onBatchAcceptReview?: () => void
  onBatchRejectReview?: () => void
}

function HeaderMetric({ label, value, truncate = false }: WorkspaceHeaderMetaItem) {
  const resolvedValue = value.trim() || "-"
  return (
    <div className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
      <span className="text-xs uppercase tracking-[0.14em] text-muted-foreground/80">{label}</span>
      <span
        className={cn("font-medium text-foreground/80", truncate && "max-w-[16rem] truncate")}
        title={truncate ? resolvedValue : undefined}
      >
        {resolvedValue}
      </span>
    </div>
  )
}

function ContextField({
  label,
  icon: Icon,
  children,
  className,
}: {
  label: string
  icon?: typeof Film
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn(WORKSPACE_CONTROL_FIELD_CLASS, WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS, className)}>
      {Icon ? <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground/80" /> : null}
      <span className={WORKSPACE_CONTROL_LABEL_CLASS}>{label}</span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}

function DisplayModeControl({
  options,
  value,
  canUseDualDisplay = true,
  dualDisplayDisabledReason = "",
  onChange,
}: {
  options: Array<{ value: WorkspaceDisplayMode; label: string; icon: typeof Film }>
  value: WorkspaceDisplayMode
  canUseDualDisplay?: boolean
  dualDisplayDisabledReason?: string
  onChange: (value: WorkspaceDisplayMode) => void
}) {
  return (
    <div className={cn(DASHBOARD_CONTROL_GROUP_CLASS, "shrink-0")}>
      {options.map((option, index) => {
        const Icon = option.icon
        const active = option.value === value
        const disabled = option.value === "dual" && !canUseDualDisplay
        const content = (
          <Button
            key={option.value}
            type="button"
            variant={active ? "secondary" : "ghost"}
            size="compact"
            className={cn("gap-1.5 rounded-none border-0 px-2.5", index > 0 && "border-l border-border/70")}
            disabled={disabled}
            onClick={() => onChange(option.value)}
            aria-label={option.label}
          >
            <Icon className="h-3.5 w-3.5" />
            <span className="text-xs">{option.label}</span>
          </Button>
        )
        if (!(disabled && dualDisplayDisabledReason.trim())) {
          return content
        }
        return (
          <Tooltip key={option.value}>
            <TooltipTrigger asChild>
              <span className="inline-flex">{content}</span>
            </TooltipTrigger>
            <TooltipContent className="max-w-[18rem] text-xs leading-5">
              {dualDisplayDisabledReason}
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}

function withDisabledTooltip(content: ReactNode, tooltipLabel: string) {
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
}

export function WorkspaceHeader({
  libraryName,
  metaItems,
  hasPendingReview = false,
  pendingReviewButtonLabel = "",
  pendingReviewMenuDescription = "",
  reviewCompletionReady = false,
  reviewApplying = false,
  reviewDiscarding = false,
  activeEditor,
  onEditorChange,
  activeVideoId,
  videoOptions,
  onVideoChange,
  primarySubtitleTrackId,
  subtitleTrackOptions,
  onPrimarySubtitleTrackChange,
  secondarySubtitleTrackId,
  comparisonTrackOptions,
  onSecondarySubtitleTrackChange,
  displayMode,
  canUseDualDisplay = true,
  dualDisplayDisabledReason = "",
  onDisplayModeChange,
  autoFollow = true,
  onAutoFollowChange,
  subtitleStyleSidebarOpen = false,
  onSubtitleStyleSidebarOpenChange,
  guidelineProfileId,
  guidelineOptions,
  onGuidelineChange,
  canSubtitleActions,
  canTranslateAction = true,
  canProofreadAction = true,
  canQaAction = true,
  canRestoreAction = true,
  translateDisabledReason = "",
  proofreadDisabledReason = "",
  qaDisabledReason = "",
  translateButtonLabel,
  proofreadButtonLabel,
  qaButtonLabel,
  restoreButtonLabel,
  translateRunning = false,
  proofreadRunning = false,
  qaRunning = false,
  onRunQa,
  onTranslate,
  onProofread,
  onRestore,
  onCompleteReview,
  onBatchAcceptReview,
  onBatchRejectReview,
}: WorkspaceHeaderProps) {
  const { t } = useI18n()
  const displayOptions: Array<{ value: WorkspaceDisplayMode; label: string; icon: typeof Languages }> = [
    { value: "single", label: t("library.workspace.header.displaySingle"), icon: Languages },
    { value: "dual", label: t("library.workspace.header.displayDual"), icon: Columns2 },
  ]
  const showSecondaryTrack = displayMode === "dual" && canUseDualDisplay
  const reviewSubmitting = reviewApplying || reviewDiscarding

  return (
    <header className="sticky top-0 z-20 shrink-0 border-b border-border/70 bg-background/95 px-4 py-2 backdrop-blur">
      <div className="flex flex-col gap-2.5">
        <div className="grid gap-2.5 xl:grid-cols-[minmax(0,1fr)_auto] xl:items-center">
          <div className="min-w-0 space-y-1">
            <div className="truncate text-xs font-semibold tracking-[0.01em] text-foreground">{libraryName}</div>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              {metaItems.map((item) => (
                <HeaderMetric key={`${item.label}-${item.value}`} {...item} />
              ))}
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2">
            <ModeSwitcher value={activeEditor} onChange={onEditorChange} />
          </div>
        </div>

        <div className="flex flex-col gap-2 border-t border-border/60 pt-2 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
            {activeEditor === "video" ? (
              <ContextField label={t("library.workspace.header.video")} icon={Film}>
                <Select
                  value={activeVideoId}
                  onChange={(event) => onVideoChange(event.target.value)}
                  disabled={videoOptions.length === 0}
                  className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
                >
                  {videoOptions.length === 0 ? (
                    <option value="">{t("library.workspace.header.noVideoVersion")}</option>
                  ) : (
                    videoOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))
                  )}
                </Select>
              </ContextField>
            ) : null}

            <ContextField label={t("library.workspace.header.track")} icon={Languages}>
              <Select
                value={primarySubtitleTrackId}
                onChange={(event) => onPrimarySubtitleTrackChange(event.target.value)}
                disabled={subtitleTrackOptions.length === 0}
                className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
              >
                {subtitleTrackOptions.length === 0 ? (
                  <option value="">{t("library.workspace.header.noSubtitleTrack")}</option>
                ) : (
                  subtitleTrackOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))
                )}
              </Select>
            </ContextField>

            {showSecondaryTrack ? (
              <ContextField label={t("library.workspace.header.secondary")}>
                <Select
                  value={secondarySubtitleTrackId}
                  onChange={(event) => onSecondarySubtitleTrackChange(event.target.value)}
                  className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
                >
                  {comparisonTrackOptions.map((option) => (
                    <option key={option.value || "none"} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </ContextField>
            ) : null}

            {activeEditor === "video" ? (
              <div
                className={cn(
                  WORKSPACE_CONTROL_FIELD_CLASS,
                  "w-fit min-w-0 max-w-[14.5rem] shrink justify-between gap-2",
                )}
              >
                <span
                  className={cn(WORKSPACE_CONTROL_LABEL_CLASS, "min-w-0 flex-1 truncate")}
                  title={t("library.workspace.toolbar.autoFollow")}
                >
                  {t("library.workspace.toolbar.autoFollow")}
                </span>
                <Switch checked={autoFollow} onCheckedChange={onAutoFollowChange} className="shrink-0" />
              </div>
            ) : null}

            {activeEditor === "subtitle" ? (
              <ContextField label={t("library.workspace.header.guideline")} icon={ShieldCheck}>
                <Select
                  value={guidelineProfileId}
                  onChange={(event) => onGuidelineChange(event.target.value as WorkspaceGuidelineProfileId)}
                  className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
                >
                  {guidelineOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </ContextField>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2">
            <DisplayModeControl
              options={displayOptions}
              value={displayMode}
              canUseDualDisplay={canUseDualDisplay}
              dualDisplayDisabledReason={dualDisplayDisabledReason}
              onChange={onDisplayModeChange}
            />

            {activeEditor === "video" ? (
              <Button
                type="button"
                variant={subtitleStyleSidebarOpen ? "secondary" : "outline"}
                size="compact"
                className="gap-1.5"
                onClick={() => onSubtitleStyleSidebarOpenChange?.(!subtitleStyleSidebarOpen)}
              >
                <PanelRight className="h-3.5 w-3.5" />
                <span>{t("library.config.subtitleStyles.exportProfileStyleDocument")}</span>
              </Button>
            ) : null}
            {activeEditor === "subtitle" ? (
              <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                {hasPendingReview ? (
                  reviewCompletionReady ? (
                    <Button
                      size="compact"
                      className="gap-1.5 rounded-none rounded-l-lg border-0 px-3 shadow-none"
                      onClick={onCompleteReview}
                      disabled={reviewSubmitting}
                    >
                      <Sparkles className="h-3.5 w-3.5" />
                      <span>{pendingReviewButtonLabel || t("library.workspace.review.complete")}</span>
                    </Button>
                  ) : (
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          size="compact"
                          className="gap-1.5 rounded-none rounded-l-lg border-0 px-3 shadow-none"
                          disabled={reviewSubmitting}
                        >
                          <Sparkles className="h-3.5 w-3.5" />
                          <span>{pendingReviewButtonLabel || t("library.workspace.review.pendingButton")}</span>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="start" className="w-[20rem]">
                        <DropdownMenuLabel>
                          {pendingReviewButtonLabel || t("library.workspace.review.pendingButton")}
                        </DropdownMenuLabel>
                        {pendingReviewMenuDescription ? (
                          <>
                            <div className="px-2 py-1.5 text-xs leading-5 text-muted-foreground">
                              {pendingReviewMenuDescription}
                            </div>
                            <DropdownMenuSeparator />
                          </>
                        ) : null}
                        <DropdownMenuItem onSelect={onBatchAcceptReview} disabled={reviewSubmitting}>
                          <span>{t("library.workspace.review.bulkAccept")}</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-xs text-rose-700 focus:text-rose-700"
                          onSelect={onBatchRejectReview}
                          disabled={reviewSubmitting}
                        >
                          <span>{t("library.workspace.review.bulkReject")}</span>
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )
                ) : null}
                {withDisabledTooltip(
                  <Button
                    variant={translateRunning ? "secondary" : "ghost"}
                    size="compact"
                    className={cn(
                      "gap-1.5 rounded-none border-0",
                      hasPendingReview && "border-l border-border/70",
                      !translateRunning && "bg-transparent",
                    )}
                    onClick={onTranslate}
                    disabled={!canSubtitleActions || !canTranslateAction}
                  >
                    {translateRunning ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Languages className="h-3.5 w-3.5" />
                    )}
                    <span>{translateButtonLabel || t("library.workspace.actions.translate")}</span>
                  </Button>,
                  !canTranslateAction ? translateDisabledReason : "",
                )}
                {withDisabledTooltip(
                  <Button
                    variant={proofreadRunning ? "secondary" : "ghost"}
                    size="compact"
                    className={cn(
                      "gap-1.5 rounded-none border-0 border-l border-border/70",
                      !proofreadRunning && "bg-transparent",
                    )}
                    onClick={onProofread}
                    disabled={!canSubtitleActions || (!canProofreadAction && !proofreadRunning)}
                  >
                    {proofreadRunning ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Sparkles className="h-3.5 w-3.5" />
                    )}
                    <span>{proofreadButtonLabel || t("library.workspace.header.proofread")}</span>
                  </Button>,
                  !canProofreadAction && !proofreadRunning ? proofreadDisabledReason : "",
                )}
                {withDisabledTooltip(
                  <Button
                    variant={qaRunning ? "secondary" : "ghost"}
                    size="compact"
                    className={cn(
                      "gap-1.5 rounded-none border-0 border-l border-border/70",
                      !qaRunning && "bg-transparent",
                    )}
                    onClick={onRunQa}
                    disabled={!canSubtitleActions || (!canQaAction && !qaRunning)}
                  >
                    <ShieldCheck className="h-3.5 w-3.5" />
                    <span>{qaButtonLabel || t("library.workspace.header.qa")}</span>
                  </Button>,
                  !canQaAction && !qaRunning ? qaDisabledReason : "",
                )}
                <Button
                  variant="ghost"
                  size="compact"
                  className="gap-1.5 rounded-none border-0 border-l border-border/70 bg-transparent"
                  onClick={onRestore}
                  disabled={!canSubtitleActions || !canRestoreAction}
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  <span>{restoreButtonLabel || t("library.workspace.header.restore")}</span>
                </Button>
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </header>
  )
}
