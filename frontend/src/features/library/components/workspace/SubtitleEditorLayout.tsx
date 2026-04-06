import * as React from "react"

import { useI18n } from "@/shared/i18n"
import type { WorkspaceQaCheckSettings } from "../../model/workspaceQa"

import type {
  WorkspaceDensity,
  WorkspaceDisplayMode,
  WorkspaceFilter,
  WorkspaceQaFilter,
  WorkspaceReviewDecision,
  WorkspaceResolvedSubtitleRow,
} from "./types"
import {
  DASHBOARD_WORKSPACE_META_BAR_CLASS,
  DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS,
} from "@/shared/ui/dashboard"
import { cn } from "@/lib/utils"
import { SubtitleTablePane } from "./SubtitleTablePane"
import { WorkspaceToolbar } from "./WorkspaceToolbar"

type SubtitleEditorLayoutProps = {
  rows: WorkspaceResolvedSubtitleRow[]
  primaryTrackLabel: string
  secondaryTrackLabel: string
  reviewPending?: boolean
  reviewApplying?: boolean
  saveState?: "saved" | "saving" | "dirty" | "editing" | "error"
  saveErrorMessage?: string
  selectedRowId: string
  editingRowId: string
  currentRowId: string
  hoveredRowId: string
  displayMode: WorkspaceDisplayMode
  density: WorkspaceDensity
  searchValue: string
  replaceValue: string
  filterValue: WorkspaceFilter
  qaFilter: WorkspaceQaFilter
  isLoading: boolean
  errorMessage?: string
  editingEnabled?: boolean
  qaCheckSettings: WorkspaceQaCheckSettings
  onReviewDecisionChange?: (cueIndex: number, decision: WorkspaceReviewDecision) => void
  onSearchChange: (value: string) => void
  onReplaceValueChange: (value: string) => void
  onApplyReplace: () => void
  onFilterChange: (value: WorkspaceFilter) => void
  onQaFilterChange: (value: WorkspaceQaFilter) => void
  onDensityChange: (value: WorkspaceDensity) => void
  onSelectRow: (rowId: string) => void
  onEditingRowChange: (rowId: string) => void
  onHoverRow: (rowId: string) => void
  onEditSourceText: (rowId: string, value: string) => void
}

export function SubtitleEditorLayout({
  rows,
  primaryTrackLabel,
  secondaryTrackLabel,
  reviewPending = false,
  reviewApplying = false,
  saveState = "saved",
  saveErrorMessage = "",
  selectedRowId,
  editingRowId,
  currentRowId,
  hoveredRowId,
  displayMode,
  density,
  searchValue,
  replaceValue,
  filterValue,
  qaFilter,
  isLoading,
  errorMessage,
  editingEnabled = true,
  qaCheckSettings,
  onReviewDecisionChange,
  onSearchChange,
  onReplaceValueChange,
  onApplyReplace,
  onFilterChange,
  onQaFilterChange,
  onDensityChange,
  onSelectRow,
  onEditingRowChange,
  onHoverRow,
  onEditSourceText,
}: SubtitleEditorLayoutProps) {
  const { t } = useI18n()
  const rootRef = React.useRef<HTMLDivElement | null>(null)
  const reviewCount = rows.filter((row) => row.qaIssues.length > 0).length
  const displayModeLabel =
    displayMode === "dual"
      ? t("library.workspace.table.modeDual")
      : t("library.workspace.table.modeSingle")
  const draftStatusLabel =
    saveState === "saving"
      ? t("library.workspace.table.draftSaving")
      : saveState === "error"
        ? t("library.workspace.table.draftSaveFailed")
        : saveState === "editing"
          ? t("library.workspace.table.draftEditing")
          : saveState === "dirty"
            ? t("library.workspace.table.draftDirty")
            : t("library.workspace.table.draftSaved")
  const draftStatusClassName =
    saveState === "saving"
      ? "border-sky-500/30 bg-sky-500/10 text-sky-800"
      : saveState === "error"
        ? "border-rose-500/30 bg-rose-500/10 text-rose-800"
        : saveState === "editing" || saveState === "dirty"
          ? "border-amber-500/30 bg-amber-500/10 text-amber-800"
          : "border-emerald-500/30 bg-emerald-500/10 text-emerald-800"

  React.useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target
      if (!(target instanceof Node)) {
        return
      }
      const root = rootRef.current
      if (!root) {
        return
      }
      if (!root.contains(target)) {
        if (editingRowId) {
          onEditingRowChange("")
        }
        return
      }
      if (!(target instanceof Element)) {
        return
      }

      if (target.closest("[data-subtitle-row-action='true']")) {
        return
      }

      const rowElement = target.closest<HTMLElement>("[data-subtitle-row-id]")
      const rowId = rowElement?.dataset.subtitleRowId ?? ""
      const insideActiveEditor =
        target.closest("[data-subtitle-text-editor='true']") !== null && rowId === editingRowId

      if (rowId) {
        onSelectRow(rowId)
        if (insideActiveEditor) {
          return
        }
        if (!editingEnabled) {
          onEditingRowChange("")
          return
        }
        if (!editingRowId) {
          onEditingRowChange(rowId)
          return
        }
        onEditingRowChange(rowId === editingRowId ? "" : rowId)
        return
      }

      if (editingRowId) {
        onEditingRowChange("")
      }
    }

    document.addEventListener("pointerdown", handlePointerDown, true)
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown, true)
    }
  }, [editingEnabled, editingRowId, onEditingRowChange, onSelectRow])

  return (
    <div
      ref={rootRef}
      className={`flex h-full min-h-0 flex-1 flex-col overflow-hidden ${DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS}`}
    >
      <WorkspaceToolbar
        mode="subtitle"
        searchValue={searchValue}
        onSearchChange={onSearchChange}
        replaceValue={replaceValue}
        onReplaceValueChange={onReplaceValueChange}
        onApplyReplace={onApplyReplace}
        filterValue={filterValue}
        onFilterChange={onFilterChange}
        qaFilter={qaFilter}
        onQaFilterChange={onQaFilterChange}
        density={density}
        onDensityChange={onDensityChange}
        replaceDisabled={!editingEnabled}
      />
      <div className="min-h-0 flex-1 overflow-hidden">
        <SubtitleTablePane
          mode="subtitle"
          title={t("library.workspace.table.editorTitle")}
          chrome="plain"
          rows={rows}
          selectedRowId={selectedRowId}
          editingRowId={editingRowId}
          currentRowId={currentRowId}
          hoveredRowId={hoveredRowId}
          displayMode={displayMode}
          density={density}
          autoFollow={false}
          isLoading={isLoading}
          errorMessage={errorMessage}
          qaCheckSettings={qaCheckSettings}
          reviewPending={reviewPending}
          reviewApplying={reviewApplying}
          onSelectRow={onSelectRow}
          onEditingRowChange={onEditingRowChange}
          onHoverRow={onHoverRow}
          onEditSourceText={editingEnabled ? onEditSourceText : undefined}
          onReviewDecisionChange={onReviewDecisionChange}
        />
      </div>
      <div className={`flex shrink-0 items-center justify-between gap-3 px-3 py-2 ${DASHBOARD_WORKSPACE_META_BAR_CLASS}`}>
        <div className="flex items-center gap-3">
          <span className="font-medium text-foreground">{t("library.workspace.table.editorTitle")}</span>
          <span>{t("library.workspace.table.visibleCount").replace("{count}", String(rows.length))}</span>
          <span>{t("library.workspace.table.needReviewCount").replace("{count}", String(reviewCount))}</span>
        </div>
        <div className="flex min-w-0 items-center gap-3">
          <span>{t("library.workspace.table.current")} {currentRowId ? currentRowId.replace("cue-", "") : "-"}</span>
          <span>{t("library.workspace.table.editing")} {editingRowId ? editingRowId.replace("cue-", "") : "-"}</span>
          <span>{displayModeLabel}</span>
          {reviewPending ? (
            <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-800">
              {t("library.workspace.table.pendingReview")}
            </span>
          ) : null}
          {displayMode === "dual" ? (
            <>
              <span className="max-w-[220px] truncate">{t("library.workspace.table.primaryTrack")}: {primaryTrackLabel}</span>
              <span className="max-w-[220px] truncate">
                {t("library.workspace.table.secondaryTrack")}:{" "}
                {secondaryTrackLabel || t("library.workspace.table.notSelected")}
              </span>
            </>
          ) : (
            <span className="max-w-[220px] truncate">{t("library.workspace.table.track")}: {primaryTrackLabel}</span>
          )}
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-[11px] font-medium",
              draftStatusClassName,
            )}
            title={saveState === "error" && saveErrorMessage ? saveErrorMessage : undefined}
          >
            {t("library.workspace.table.draftStatus")} {draftStatusLabel}
          </span>
        </div>
      </div>
    </div>
  )
}
