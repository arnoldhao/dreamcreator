import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { useVirtualizer } from "@tanstack/react-virtual"
import { AlertTriangle, CheckCircle2, CircleDot, XCircle } from "lucide-react"

import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"
import { DASHBOARD_CONTROL_GROUP_CLASS, DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS } from "@/shared/ui/dashboard"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table"
import { cn } from "@/lib/utils"
import {
  DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
  hasWorkspaceQaMetricColumn,
  type WorkspaceQaCheckSettings,
} from "../../model/workspaceQa"

import type {
  WorkspaceDensity,
  WorkspaceDisplayMode,
  WorkspaceResolvedSubtitleRow,
  WorkspaceReviewDecision,
} from "./types"

type SubtitleTablePaneProps = {
  mode: "video" | "subtitle"
  title: string
  chrome?: "card" | "plain"
  compressed?: boolean
  rows: WorkspaceResolvedSubtitleRow[]
  selectedRowId: string
  editingRowId?: string
  currentRowId: string
  hoveredRowId: string
  displayMode: WorkspaceDisplayMode
  density: WorkspaceDensity
  autoFollow: boolean
  isLoading: boolean
  errorMessage?: string
  qaCheckSettings?: WorkspaceQaCheckSettings
  reviewPending?: boolean
  reviewApplying?: boolean
  onSelectRow: (rowId: string) => void
  onEditingRowChange?: (rowId: string) => void
  onHoverRow: (rowId: string) => void
  onEditSourceText?: (rowId: string, value: string) => void
  onReviewDecisionChange?: (cueIndex: number, decision: WorkspaceReviewDecision) => void
}

const COLUMN_WIDTHS: Record<"video" | "subtitle", Record<string, string>> = {
  video: {
    index: "w-[68px] min-w-[68px] max-w-[68px]",
    time: "w-[136px] min-w-[136px] max-w-[136px]",
    duration: "w-[78px] min-w-[78px] max-w-[78px]",
    text: "",
    suggestion: "w-[148px] min-w-[148px] max-w-[168px]",
    decision: "w-[128px] min-w-[128px] max-w-[148px]",
  },
  subtitle: {
    index: "w-[68px] min-w-[68px] max-w-[68px]",
    start: "w-[108px] min-w-[108px] max-w-[108px]",
    end: "w-[108px] min-w-[108px] max-w-[108px]",
    time: "w-[140px] min-w-[140px] max-w-[140px]",
    duration: "w-[86px] min-w-[86px] max-w-[86px]",
    cps: "w-[74px] min-w-[74px] max-w-[74px]",
    cpl: "w-[74px] min-w-[74px] max-w-[74px]",
    text: "w-[460px] min-w-[460px]",
    suggestion: "w-[180px] min-w-[180px] max-w-[220px]",
    decision: "w-[156px] min-w-[156px] max-w-[176px]",
  },
}

const VIDEO_TIMELINE_VIRTUAL_OVERSCAN = 8
type VirtualTimelineRow = {
  key: React.Key
  start: number
}

function SuggestionCell({ row, mode }: { row: WorkspaceResolvedSubtitleRow; mode: "video" | "subtitle" }) {
  const activeSuggestion = row.reviewSuggestion
    ? {
        sourceLabel: row.reviewSuggestion.sourceLabel,
        detailLabel: row.reviewSuggestion.detailLabel,
        reason: row.reviewSuggestion.reason,
        severity: row.reviewSuggestion.severity,
      }
    : row.qaIssues[0]
      ? {
          sourceLabel: row.qaIssues[0].sourceLabel,
          detailLabel:
            row.qaIssues.length > 1
              ? `${row.qaIssues[0].detailLabel} +${row.qaIssues.length - 1}`
              : row.qaIssues[0].detailLabel,
          reason: row.qaIssues[0].reason,
          severity: row.qaIssues[0].severity,
        }
      : null

  if (!activeSuggestion) {
    return null
  }

  const toneClassName =
    activeSuggestion.severity === "error"
      ? "text-rose-700"
      : activeSuggestion.severity === "warning"
        ? "text-amber-700"
        : "text-foreground"

  return (
    <div className="space-y-0.5">
      <div className={cn("text-xs font-medium", toneClassName, mode === "video" && "text-xs")}>
        {activeSuggestion.sourceLabel} / {activeSuggestion.detailLabel}
      </div>
      <div className={cn("line-clamp-3 text-xs leading-5 text-muted-foreground", mode === "video" && "text-xs")}>
        {activeSuggestion.reason}
      </div>
    </div>
  )
}

function ReviewDecisionCell({
  row,
  disabled,
  onSelectRow,
  onReviewDecisionChange,
}: {
  row: WorkspaceResolvedSubtitleRow
  disabled: boolean
  onSelectRow: (rowId: string) => void
  onReviewDecisionChange?: (cueIndex: number, decision: WorkspaceReviewDecision) => void
}) {
  const { t } = useI18n()
  if (!row.reviewSuggestion) {
    return null
  }
  const decision = row.reviewDecision ?? "undecided"
  if (decision === "undecided") {
    return (
      <div data-subtitle-row-action="true" className="flex min-h-[2.25rem] items-center">
        <div className={cn(DASHBOARD_CONTROL_GROUP_CLASS, "w-full min-w-0 overflow-hidden")}>
          <Button
            type="button"
            size="compact"
            variant="ghost"
            className="min-w-0 flex-1 rounded-none border-0 bg-transparent px-2"
            disabled={disabled}
            onPointerDown={(event) => event.stopPropagation()}
            onClick={(event) => {
              event.stopPropagation()
              onSelectRow(row.id)
              onReviewDecisionChange?.(row.reviewSuggestion!.cueIndex, "accept")
            }}
          >
            <span className="truncate">{t("library.workspace.review.accept")}</span>
          </Button>
          <Button
            type="button"
            size="compact"
            variant="ghost"
            className="min-w-0 flex-1 rounded-none border-0 border-l border-border/70 bg-transparent px-2"
            disabled={disabled}
            onPointerDown={(event) => event.stopPropagation()}
            onClick={(event) => {
              event.stopPropagation()
              onSelectRow(row.id)
              onReviewDecisionChange?.(row.reviewSuggestion!.cueIndex, "reject")
            }}
          >
            <span className="truncate">{t("library.workspace.review.reject")}</span>
          </Button>
        </div>
      </div>
    )
  }
  return (
    <div data-subtitle-row-action="true" className="flex min-h-[2.25rem] items-center">
      <div className={cn(DASHBOARD_CONTROL_GROUP_CLASS, "w-full min-w-0 overflow-hidden")}>
        <Button
          type="button"
          size="compact"
          variant={decision === "accept" ? "secondary" : "outline"}
          className={cn(
            "min-w-0 flex-[2_1_0%] gap-1 rounded-none border-0 px-2",
            decision === "accept"
              ? "bg-emerald-500/12 text-emerald-800 hover:bg-emerald-500/12"
              : "bg-amber-500/10 text-amber-800 hover:bg-amber-500/10",
          )}
          disabled={disabled}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
        >
          {decision === "accept" ? (
            <CheckCircle2 className="h-3 w-3 shrink-0" />
          ) : (
            <XCircle className="h-3 w-3 shrink-0" />
          )}
          <span className="truncate">
            {decision === "accept"
              ? t("library.workspace.review.accepted")
              : t("library.workspace.review.rejected")}
          </span>
        </Button>
        <Button
          type="button"
          size="compact"
          variant="ghost"
          className="min-w-0 flex-[1_1_0%] gap-1 rounded-none border-0 border-l border-border/70 bg-transparent px-2"
          disabled={disabled}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            onSelectRow(row.id)
            onReviewDecisionChange?.(row.reviewSuggestion!.cueIndex, "undecided")
          }}
        >
          <span className="truncate">{t("library.workspace.review.undo")}</span>
        </Button>
      </div>
    </div>
  )
}

function MetricCell({
  row,
  metric,
}: {
  row: WorkspaceResolvedSubtitleRow
  metric: "cps" | "cpl"
}) {
  const error = row.qaIssues.some((issue) => issue.code === `${metric}-error`)
  const warning = row.qaIssues.some((issue) => issue.code === `${metric}-warning`)
  const value = metric === "cps" ? row.metrics.cps.toFixed(1) : String(row.metrics.cpl)

  return (
    <div
      className={cn(
        "font-mono text-2xs font-medium",
        error ? "text-rose-600" : warning ? "text-amber-700" : "text-foreground/80",
      )}
    >
      {value}
    </div>
  )
}

function TextCell({
  row,
  mode,
  displayMode,
  editing,
  textColumnWidth,
  onEditSourceText,
}: {
  row: WorkspaceResolvedSubtitleRow
  mode: "video" | "subtitle"
  displayMode: WorkspaceDisplayMode
  editing: boolean
  textColumnWidth?: number
  onEditSourceText?: (rowId: string, value: string) => void
}) {
  const editable = mode === "subtitle" && editing && onEditSourceText
  const sourceTextClassName =
    mode === "video"
      ? "line-clamp-1 break-words text-xs font-medium leading-5 tracking-[0.01em] text-foreground"
      : "whitespace-pre-wrap break-words text-xs leading-5 text-foreground"
  const translationTextClassName =
    mode === "video"
      ? "line-clamp-1 break-words text-2xs leading-5 text-muted-foreground"
      : "whitespace-pre-wrap break-words text-2xs leading-5 text-muted-foreground"
  const showSecondaryTrack = mode === "video" && displayMode === "bilingual" && row.translationText.trim().length > 0

  return (
    <div
      className={cn(
        mode === "video"
          ? showSecondaryTrack
            ? "space-y-0.5"
            : ""
          : "min-w-[460px] space-y-1",
      )}
      style={mode === "video" && textColumnWidth ? { minWidth: textColumnWidth } : undefined}
    >
      {editable ? (
        <textarea
          data-subtitle-text-editor="true"
          data-subtitle-row-id={row.id}
          value={row.sourceText}
          onChange={(event) => onEditSourceText?.(row.id, event.target.value)}
          onClick={(event) => event.stopPropagation()}
          onPointerDown={(event) => event.stopPropagation()}
          className="min-h-[72px] w-full resize-y rounded-md border border-border/70 bg-background px-2.5 py-2 text-xs leading-5 text-foreground outline-none focus:border-ring"
          autoFocus
        />
      ) : (
        row.reviewSuggestion && mode !== "video" ? (
          <ReviewDiffTextCell row={row} />
        ) : (
          <div className={sourceTextClassName}>{row.sourceText}</div>
        )
      )}

      {showSecondaryTrack ? (
        <div className={translationTextClassName}>
          {row.translationText}
        </div>
      ) : null}
    </div>
  )
}

function ReviewDiffTextCell({ row }: { row: WorkspaceResolvedSubtitleRow }) {
  const suggestion = row.reviewSuggestion
  if (!suggestion) {
    return <div className="whitespace-pre-wrap break-words text-xs leading-5 text-foreground">{row.sourceText}</div>
  }
  const diff = buildInlineDiff(suggestion.originalText, suggestion.suggestedText)
  return (
    <div className="space-y-1.5">
      <div className="flex items-start gap-2 rounded-md border border-rose-500/20 bg-rose-500/[0.06] px-2.5 py-1.5">
        <div className="shrink-0 pt-0.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-rose-700">-</div>
        <div className="min-w-0 whitespace-pre-wrap break-words text-xs leading-5 text-rose-900">
          <span>{diff.prefix}</span>
          {diff.beforeDiff ? <span className="rounded bg-rose-500/18 px-0.5">{diff.beforeDiff}</span> : null}
          <span>{diff.suffix}</span>
        </div>
      </div>
      <div className="flex items-start gap-2 rounded-md border border-emerald-500/20 bg-emerald-500/[0.07] px-2.5 py-1.5">
        <div className="shrink-0 pt-0.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-emerald-700">+</div>
        <div className="min-w-0 whitespace-pre-wrap break-words text-xs leading-5 text-emerald-900">
          <span>{diff.prefix}</span>
          {diff.afterDiff ? <span className="rounded bg-emerald-500/20 px-0.5">{diff.afterDiff}</span> : null}
          <span>{diff.suffix}</span>
        </div>
      </div>
    </div>
  )
}

function buildInlineDiff(before: string, after: string) {
  let prefixIndex = 0
  while (
    prefixIndex < before.length &&
    prefixIndex < after.length &&
    before[prefixIndex] === after[prefixIndex]
  ) {
    prefixIndex += 1
  }

  let beforeTail = before.length - 1
  let afterTail = after.length - 1
  while (
    beforeTail >= prefixIndex &&
    afterTail >= prefixIndex &&
    before[beforeTail] === after[afterTail]
  ) {
    beforeTail -= 1
    afterTail -= 1
  }

  const sharedSuffix =
    beforeTail + 1 < before.length ? before.slice(beforeTail + 1) : afterTail + 1 < after.length ? after.slice(afterTail + 1) : ""

  return {
    prefix: before.slice(0, prefixIndex),
    beforeDiff: before.slice(prefixIndex, beforeTail + 1),
    afterDiff: after.slice(prefixIndex, afterTail + 1),
    suffix: sharedSuffix,
  }
}

export function SubtitleTablePane({
  mode,
  title,
  chrome = "card",
  compressed = false,
  rows,
  selectedRowId,
  editingRowId = "",
  currentRowId,
  hoveredRowId,
  displayMode,
  density,
  autoFollow,
  isLoading,
  errorMessage,
  qaCheckSettings = DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
  reviewPending = false,
  reviewApplying = false,
  onSelectRow,
  onEditingRowChange,
  onHoverRow,
  onEditSourceText,
  onReviewDecisionChange,
}: SubtitleTablePaneProps) {
  const { t } = useI18n()
  const rowRefs = React.useRef<Record<string, HTMLElement | null>>({})
  const scrollParentRef = React.useRef<HTMLDivElement | null>(null)
  const columnWidths = COLUMN_WIDTHS[mode]
  const longestVisibleTextLength = React.useMemo(() => {
    if (mode !== "video") {
      return 0
    }
    return rows.reduce((maxLength, row) => {
      const visibleParts: string[] = [row.sourceText]
      if (displayMode === "bilingual") {
        visibleParts.push(row.translationText)
      }
      const rowMaxLength = visibleParts.reduce((partMax, part) => {
        const lineMax = part.split("\n").reduce((lineLength, line) => Math.max(lineLength, line.trim().length), 0)
        return Math.max(partMax, lineMax)
      }, 0)
      return Math.max(maxLength, rowMaxLength)
    }, 0)
  }, [displayMode, mode, rows])
  const videoTextColumnWidth = React.useMemo(() => {
    if (mode !== "video") {
      return undefined
    }
    return Math.max(260, Math.min(680, longestVisibleTextLength * 8 + 56))
  }, [longestVisibleTextLength, mode])
  const videoGridMinWidth = React.useMemo(() => {
    if (mode !== "video") {
      return 0
    }
    return 68 + 136 + 78 + (videoTextColumnWidth ?? 260)
  }, [mode, videoTextColumnWidth])
  const videoGridTemplateColumns = React.useMemo(() => {
    if (mode !== "video") {
      return ""
    }
    return `68px 136px 78px minmax(${videoTextColumnWidth ?? 260}px, 1fr)`
  }, [mode, videoTextColumnWidth])
  const estimatedVideoRowHeight = React.useMemo(() => {
    if (compressed) {
      return displayMode === "bilingual" ? 76 : 58
    }
    if (density === "compact") {
      return displayMode === "bilingual" ? 92 : 74
    }
    return displayMode === "bilingual" ? 104 : 86
  }, [compressed, density, displayMode])
  const showCpsColumn = mode === "subtitle" && hasWorkspaceQaMetricColumn(qaCheckSettings, "cps")
  const showCplColumn = mode === "subtitle" && hasWorkspaceQaMetricColumn(qaCheckSettings, "cpl")
  const tableWrapperClassName = mode === "video" ? "min-w-full w-max" : "min-w-[1320px]"
  const formatTimelineColumnTime = React.useCallback((valueMs: number) => {
    const totalSeconds = Math.round(Math.max(0, valueMs) / 1000)
    const hours = Math.floor(totalSeconds / 3600)
    const minutes = Math.floor((totalSeconds % 3600) / 60)
    const seconds = totalSeconds % 60
    if (hours > 0) {
      return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`
    }
    return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`
  }, [])

  const columns = React.useMemo<ColumnDef<WorkspaceResolvedSubtitleRow>[]>(
    () => {
      const indexColumn: ColumnDef<WorkspaceResolvedSubtitleRow> = {
        id: "index",
        header: () => <span>#</span>,
        cell: ({ row }) => {
          const active = row.original.id === currentRowId
          const warned = row.original.qaIssues.length > 0
          return (
            <div className="flex h-full items-center gap-1.5 text-2xs font-medium leading-5 text-muted-foreground">
              {active ? <CircleDot className="h-3.5 w-3.5 text-sky-600" /> : <span className="h-3.5 w-3.5 rounded-full border border-border/70" />}
              <span>{String(row.original.index).padStart(3, "0")}</span>
              {mode === "video" && warned ? <AlertTriangle className="h-3 w-3 text-amber-600" /> : null}
            </div>
          )
        },
      }
      const durationColumn: ColumnDef<WorkspaceResolvedSubtitleRow> = {
        id: "duration",
        header: () => <span>{t("library.workspace.table.columnDuration")}</span>,
        cell: ({ row }) => <div className="font-mono text-2xs leading-5 text-foreground/80">{row.original.durationLabel}</div>,
      }
      const textColumn: ColumnDef<WorkspaceResolvedSubtitleRow> = {
        id: "text",
        header: () => <span>{t("library.workspace.table.columnText")}</span>,
        cell: ({ row }) => (
          <TextCell
            row={row.original}
            mode={mode}
            displayMode={displayMode}
            editing={row.original.id === editingRowId}
            textColumnWidth={videoTextColumnWidth}
            onEditSourceText={onEditSourceText}
          />
        ),
      }

      if (mode === "video") {
        return [
          indexColumn,
          {
            id: "time",
            header: () => <span>{t("library.workspace.table.columnTime")}</span>,
            cell: ({ row }) => (
              <div className="font-mono text-2xs leading-5 text-foreground/80">
                {`${formatTimelineColumnTime(row.original.startMs)} - ${formatTimelineColumnTime(row.original.endMs)}`}
              </div>
            ),
          },
          durationColumn,
          textColumn,
        ]
      }

      return [
        indexColumn,
        {
          id: "start",
          header: () => <span>{t("library.workspace.table.columnStart")}</span>,
          cell: ({ row }) => <div className="font-mono text-2xs leading-5 text-foreground/80">{row.original.start}</div>,
        },
        {
          id: "end",
          header: () => <span>{t("library.workspace.table.columnEnd")}</span>,
          cell: ({ row }) => <div className="font-mono text-2xs leading-5 text-foreground/80">{row.original.end}</div>,
        },
        durationColumn,
        ...(showCpsColumn
          ? [
              {
                id: "cps",
                header: () => <span>CPS</span>,
                cell: ({ row }) => <MetricCell row={row.original} metric="cps" />,
              } satisfies ColumnDef<WorkspaceResolvedSubtitleRow>,
            ]
          : []),
        ...(showCplColumn
          ? [
              {
                id: "cpl",
                header: () => <span>CPL</span>,
                cell: ({ row }) => <MetricCell row={row.original} metric="cpl" />,
              } satisfies ColumnDef<WorkspaceResolvedSubtitleRow>,
            ]
          : []),
        textColumn,
        {
          id: "suggestion",
          header: () => <span>{t("library.workspace.table.columnSuggestion")}</span>,
          cell: ({ row }) => <SuggestionCell row={row.original} mode={mode} />,
        },
        {
          id: "decision",
          header: () => <span>{t("library.workspace.table.columnDecision")}</span>,
          cell: ({ row }) => (
            <ReviewDecisionCell
              row={row.original}
              disabled={reviewApplying}
              onSelectRow={onSelectRow}
              onReviewDecisionChange={onReviewDecisionChange}
            />
          ),
        },
      ]
    },
    [
      currentRowId,
      displayMode,
      editingRowId,
      mode,
      onSelectRow,
      onEditSourceText,
      onReviewDecisionChange,
      qaCheckSettings,
      reviewApplying,
      reviewPending,
      showCplColumn,
      showCpsColumn,
      t,
      formatTimelineColumnTime,
    ],
  )

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  const rowModelRows = table.getRowModel().rows
  const rowIndexById = React.useMemo(
    () => new Map(rows.map((row, index) => [row.id, index])),
    [rows],
  )
  const videoRowVirtualizer = useVirtualizer({
    count: mode === "video" ? rowModelRows.length : 0,
    getScrollElement: () => scrollParentRef.current,
    estimateSize: () => estimatedVideoRowHeight,
    overscan: VIDEO_TIMELINE_VIRTUAL_OVERSCAN,
  })
  const virtualVideoRows = mode === "video" ? videoRowVirtualizer.getVirtualItems() : []

  React.useEffect(() => {
    if (mode !== "video") {
      return
    }
    videoRowVirtualizer.measure()
  }, [compressed, density, displayMode, mode, videoRowVirtualizer, videoTextColumnWidth])

  React.useEffect(() => {
    if (!autoFollow || !currentRowId) {
      return
    }
    if (mode === "video") {
      const currentIndex = rowIndexById.get(currentRowId)
      if (typeof currentIndex !== "number") {
        return
      }
      videoRowVirtualizer.scrollToIndex(currentIndex, { align: "auto" })
      return
    }
    const row = rowRefs.current[currentRowId]
    row?.scrollIntoView({ block: "nearest" })
  }, [autoFollow, currentRowId, mode, rowIndexById, videoRowVirtualizer])

  React.useEffect(() => {
    if (mode !== "subtitle" || !selectedRowId) {
      return
    }
    const row = rowRefs.current[selectedRowId]
    row?.scrollIntoView({ block: "nearest" })
  }, [mode, selectedRowId])

  const reviewCount = reviewPending
    ? rows.filter(
        (row) =>
          row.reviewSuggestion &&
          (row.reviewDecision ?? "undecided") === "undecided",
      ).length
    : rows.filter((row) => row.qaIssues.length > 0).length

  const setMeasuredRowRef = React.useCallback(
    (rowId: string, index: number, node: HTMLElement | null) => {
      if (node) {
        node.dataset.index = String(index)
        rowRefs.current[rowId] = node
        videoRowVirtualizer.measureElement(node)
        return
      }
      delete rowRefs.current[rowId]
    },
    [videoRowVirtualizer],
  )

  const renderVideoCompressedRow = React.useCallback(
    (row: WorkspaceResolvedSubtitleRow, virtualRow: VirtualTimelineRow & { index: number }) => {
      const selected = row.id === selectedRowId
      const current = row.id === currentRowId
      const hovered = row.id === hoveredRowId
      return (
        <button
          key={row.id}
          type="button"
          ref={(node) => setMeasuredRowRef(row.id, virtualRow.index, node)}
          data-index={virtualRow.index}
          className={cn(
            "absolute left-0 right-0 top-0 w-full rounded-md border border-border/60 px-2.5 py-2 text-left transition-colors",
            "bg-background/70 hover:bg-muted/50",
            selected && "border-primary/40 bg-primary/[0.08]",
            current && "shadow-[inset_2px_0_0_0_rgba(56,189,248,0.8)]",
            hovered && "bg-muted/60",
          )}
          style={{ transform: `translateY(${virtualRow.start}px)` }}
          onClick={() => onSelectRow(row.id)}
          onMouseEnter={() => onHoverRow(row.id)}
          onMouseLeave={() => onHoverRow("")}
        >
          <div className="flex items-center gap-2 text-2xs text-muted-foreground">
            <span>#{row.index}</span>
            <span>{`${formatTimelineColumnTime(row.startMs)} - ${formatTimelineColumnTime(row.endMs)}`}</span>
            <span>{row.durationLabel}</span>
          </div>
          <div className="mt-1 line-clamp-1 whitespace-pre-wrap break-words text-xs leading-5 text-foreground">
            {row.sourceText}
          </div>
          {displayMode === "bilingual" ? (
            <div className="line-clamp-1 whitespace-pre-wrap break-words text-2xs leading-5 text-muted-foreground">
              {row.translationText}
            </div>
          ) : null}
        </button>
      )
    },
    [currentRowId, displayMode, formatTimelineColumnTime, hoveredRowId, onHoverRow, onSelectRow, selectedRowId, setMeasuredRowRef],
  )

  const renderVideoTableRow = React.useCallback(
    (row: (typeof rowModelRows)[number], virtualRow: VirtualTimelineRow & { index: number }) => {
      const original = row.original
      const selected = original.id === selectedRowId
      const current = original.id === currentRowId
      const hovered = original.id === hoveredRowId
      const warned = original.qaIssues.length > 0
      return (
        <div
          key={row.id}
          ref={(node) => setMeasuredRowRef(original.id, virtualRow.index, node)}
          data-subtitle-row-id={original.id}
          data-index={virtualRow.index}
          className={cn(
            "absolute left-0 right-0 top-0 grid cursor-pointer border-b border-border/60 align-top transition-colors",
            density === "compact" ? "py-1.5" : "py-2",
            "border-border/50 bg-background",
            warned && "bg-amber-500/[0.035] hover:bg-amber-500/[0.08]",
            current &&
              "bg-sky-500/[0.08] shadow-[inset_3px_0_0_0_rgba(56,189,248,0.72)] hover:bg-sky-500/[0.12]",
            hovered && "bg-muted/45",
            selected &&
              "bg-primary/[0.06] shadow-[inset_0_0_0_1px_hsl(var(--primary)/0.22)] hover:bg-primary/[0.1]",
          )}
          style={{
            transform: `translateY(${virtualRow.start}px)`,
            gridTemplateColumns: videoGridTemplateColumns,
            minWidth: videoGridMinWidth,
          }}
          onClick={() => onSelectRow(original.id)}
          onMouseEnter={() => onHoverRow(original.id)}
          onMouseLeave={() => onHoverRow("")}
        >
          {row.getVisibleCells().map((cell) => (
            <div
              key={cell.id}
              className={cn(
                "px-3 text-2xs",
                cell.column.id === "text"
                  ? "whitespace-normal py-1 pr-4 text-left"
                  : "flex h-full items-center whitespace-nowrap text-2xs leading-5 text-foreground/76",
                columnWidths[cell.column.id] ?? "",
              )}
              style={
                cell.column.id === "text" && videoTextColumnWidth
                  ? { width: videoTextColumnWidth, minWidth: videoTextColumnWidth }
                  : undefined
              }
            >
              {flexRender(cell.column.columnDef.cell, cell.getContext())}
            </div>
          ))}
        </div>
      )
    },
    [
      columnWidths,
      currentRowId,
      density,
      hoveredRowId,
      onHoverRow,
      onSelectRow,
      selectedRowId,
      setMeasuredRowRef,
      videoGridMinWidth,
      videoGridTemplateColumns,
      videoTextColumnWidth,
    ],
  )

  if (mode === "video" && compressed) {
    const compressedContent = (
      <div
        ref={scrollParentRef}
        className={cn(
          "min-h-0 overflow-x-hidden overflow-y-auto overscroll-contain px-2 py-2",
          chrome === "plain" ? "h-full" : "flex-1",
        )}
      >
        <div style={{ height: videoRowVirtualizer.getTotalSize(), position: "relative" }}>
          {virtualVideoRows.map((virtualRow) =>
            renderVideoCompressedRow(rows[virtualRow.index], virtualRow),
          )}
        </div>
        {!isLoading && rows.length === 0 ? (
          <div className="flex min-h-[180px] items-center justify-center px-4 text-center text-xs text-muted-foreground">
            {errorMessage || t("library.workspace.table.emptyFiltered")}
          </div>
        ) : null}
        {isLoading ? (
          <div className="flex min-h-[180px] items-center justify-center px-4 text-xs text-muted-foreground">
            {t("library.workspace.table.loading")}
          </div>
        ) : null}
      </div>
    )

    if (chrome === "plain") {
      return compressedContent
    }
    return (
      <div className={`flex h-full min-h-0 flex-col overflow-hidden ${DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS}`}>
        <div className="flex items-center justify-between gap-3 border-b border-border/70 px-3 py-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-3">
            <span className="font-medium text-foreground">{title}</span>
            <span>{t("library.workspace.table.visibleCount").replace("{count}", String(rows.length))}</span>
          </div>
          <div className="flex items-center gap-3">
            <span>{t("library.workspace.table.current")} {currentRowId ? currentRowId.replace("cue-", "") : "-"}</span>
          </div>
        </div>
        {compressedContent}
      </div>
    )
  }

  const tableContent = (
    <div
      ref={mode === "video" ? scrollParentRef : undefined}
      className={cn(
        "min-h-0 overflow-x-auto overflow-y-auto overscroll-contain",
        chrome === "card" ? "flex-1" : "h-full",
      )}
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onEditingRowChange?.("")
        }
      }}
    >
      <div className={tableWrapperClassName}>
        {mode === "video" ? (
          <div className="min-w-full w-max" style={{ minWidth: videoGridMinWidth }}>
            {table.getHeaderGroups().map((headerGroup) => (
              <div
                key={headerGroup.id}
                className="sticky top-0 z-10 grid bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85"
                style={{ gridTemplateColumns: videoGridTemplateColumns, minWidth: videoGridMinWidth }}
              >
                {headerGroup.headers.map((header) => (
                  <div
                    key={header.id}
                    className={cn(
                      "h-8 whitespace-nowrap px-3 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground/80",
                      "flex items-center border-b border-border/70",
                      columnWidths[header.column.id] ?? "",
                    )}
                    style={
                      header.column.id === "text" && videoTextColumnWidth
                        ? { width: videoTextColumnWidth, minWidth: videoTextColumnWidth }
                        : undefined
                    }
                  >
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </div>
                ))}
              </div>
            ))}
            <div style={{ height: videoRowVirtualizer.getTotalSize(), position: "relative" }}>
              {virtualVideoRows.map((virtualRow) => renderVideoTableRow(rowModelRows[virtualRow.index], virtualRow))}
            </div>
          </div>
        ) : (
          <Table className="table-auto w-full">
            <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="hover:bg-transparent">
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={cn(
                        "h-9 whitespace-nowrap px-3 text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground",
                        columnWidths[header.column.id] ?? "",
                      )}
                    >
                      {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {rowModelRows.map((row) => {
                const original = row.original
                const selected = original.id === selectedRowId
                const current = original.id === currentRowId
                const hovered = original.id === hoveredRowId
                const warned = original.qaIssues.length > 0
                return (
                  <TableRow
                    key={row.id}
                    ref={(node) => {
                      rowRefs.current[original.id] = node
                    }}
                    data-subtitle-row-id={original.id}
                    className={cn(
                      "cursor-pointer border-b border-border/60 align-top transition-colors",
                      density === "compact"
                        ? "[&_td]:py-2"
                        : "[&_td]:py-3",
                      warned && "bg-amber-500/[0.035] hover:bg-amber-500/[0.08]",
                      current &&
                        "bg-sky-500/[0.08] hover:bg-sky-500/[0.12]",
                      hovered && "bg-muted/45",
                      selected && "bg-primary/[0.08] shadow-[inset_0_0_0_1px_hsl(var(--primary)/0.22)] hover:bg-primary/[0.1]",
                    )}
                    onClick={() => onSelectRow(original.id)}
                    onMouseEnter={() => onHoverRow(original.id)}
                    onMouseLeave={() => onHoverRow("")}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cn(
                          "px-3 text-2xs",
                          cell.column.id === "text" ? "align-top" : "align-middle",
                          cell.column.id === "text" ? "whitespace-normal text-left" : "whitespace-nowrap",
                          columnWidths[cell.column.id] ?? "",
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {!isLoading && rows.length === 0 ? (
        <div className="flex h-full min-h-[220px] items-center justify-center px-6 text-center text-sm text-muted-foreground">
          {errorMessage || t("library.workspace.table.emptyFiltered")}
        </div>
      ) : null}

      {isLoading ? (
        <div className="flex h-full min-h-[220px] items-center justify-center px-6 text-sm text-muted-foreground">
          {t("library.workspace.table.loading")}
        </div>
      ) : null}
    </div>
  )

  if (chrome === "plain") {
    return tableContent
  }

  return (
    <div className={`flex h-full min-h-0 flex-col overflow-hidden ${DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS}`}>
      <div className="flex items-center justify-between gap-3 border-b border-border/70 px-3 py-2 text-xs text-muted-foreground">
        <div className="flex items-center gap-3">
          <span className="font-medium text-foreground">{title}</span>
          <span>{t("library.workspace.table.visibleCount").replace("{count}", String(rows.length))}</span>
          <span>{t("library.workspace.table.needReviewCount").replace("{count}", String(reviewCount))}</span>
        </div>
        <div className="flex items-center gap-3">
          <span>{t("library.workspace.table.current")} {currentRowId ? currentRowId.replace("cue-", "") : "-"}</span>
          <span>
            {displayMode === "bilingual"
              ? t("library.workspace.table.modeBilingual")
              : t("library.workspace.table.modeMono")}
          </span>
        </div>
      </div>
      {tableContent}
    </div>
  )
}
