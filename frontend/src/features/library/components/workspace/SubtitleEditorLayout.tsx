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
import { DASHBOARD_DIALOG_SOFT_SURFACE_CLASS } from "@/shared/ui/dashboard-dialog"
import { cn } from "@/lib/utils"
import { WorkspaceDensityToggle } from "./WorkspaceDensityToggle"
import {
  WorkspaceMetaFlag,
  WorkspaceMetaItem,
} from "./WorkspaceMetaBar"
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
  showStyleSidebar?: boolean
  styleSidebarContent?: React.ReactNode
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
  showStyleSidebar = false,
  styleSidebarContent,
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
  const [editorShellWidth, setEditorShellWidth] = React.useState(0)
  const reviewCount = rows.filter((row) => row.qaIssues.length > 0).length
  const densityToggleMinWidth = displayMode === "bilingual" ? 1180 : 960
  const showDensityToggle = !showStyleSidebar && editorShellWidth >= densityToggleMinWidth
  const displayModeLabel =
    displayMode === "bilingual"
      ? t("library.workspace.table.modeBilingual")
      : t("library.workspace.table.modeMono")
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

  React.useEffect(() => {
    const root = rootRef.current
    if (!root) {
      return
    }

    const updateWidth = () => {
      setEditorShellWidth(root.clientWidth)
    }

    updateWidth()
    window.addEventListener("resize", updateWidth)
    if (typeof ResizeObserver === "undefined") {
      return () => window.removeEventListener("resize", updateWidth)
    }

    const observer = new ResizeObserver(() => updateWidth())
    observer.observe(root)
    return () => {
      observer.disconnect()
      window.removeEventListener("resize", updateWidth)
    }
  }, [showStyleSidebar])

  return (
    <div className={cn("grid h-full min-h-0 flex-1 gap-3 overflow-hidden", showStyleSidebar && "grid-cols-[minmax(0,1fr)_320px]")}>
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
        <div className={`grid shrink-0 gap-2 px-3 py-2 ${DASHBOARD_WORKSPACE_META_BAR_CLASS} md:grid-cols-[auto_minmax(0,1fr)] md:items-center`}>
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <WorkspaceMetaItem
              value={t("library.workspace.table.visibleCount").replace("{count}", String(rows.length))}
            />
            <WorkspaceMetaItem
              value={t("library.workspace.table.needReviewCount").replace("{count}", String(reviewCount))}
            />
          </div>
          <div className="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">
            <WorkspaceMetaItem
              value={`${t("library.workspace.table.current")} ${currentRowId ? currentRowId.replace("cue-", "") : "-"}`}
            />
            <WorkspaceMetaItem
              value={`${t("library.workspace.table.editing")} ${editingRowId ? editingRowId.replace("cue-", "") : "-"}`}
            />
            <WorkspaceMetaItem value={displayModeLabel} />
            {reviewPending ? (
              <WorkspaceMetaFlag className="border-amber-500/30 bg-amber-500/10 text-amber-800">
                {t("library.workspace.table.pendingReview")}
              </WorkspaceMetaFlag>
            ) : null}
            {displayMode === "bilingual" ? (
              <>
                <WorkspaceMetaItem
                  value={primaryTrackLabel}
                  label={t("library.workspace.table.primaryTrack")}
                  className="max-w-[240px]"
                  title={primaryTrackLabel}
                />
                <WorkspaceMetaItem
                  value={secondaryTrackLabel || t("library.workspace.table.notSelected")}
                  label={t("library.workspace.table.secondaryTrack")}
                  className="max-w-[240px]"
                  title={secondaryTrackLabel || t("library.workspace.table.notSelected")}
                />
              </>
            ) : (
              <WorkspaceMetaItem
                value={primaryTrackLabel}
                label={t("library.workspace.table.track")}
                className="max-w-[240px]"
                title={primaryTrackLabel}
              />
            )}
            <WorkspaceMetaFlag
              className={cn(
                "border px-2 py-0",
                draftStatusClassName,
              )}
              title={saveState === "error" && saveErrorMessage ? saveErrorMessage : undefined}
            >
              {t("library.workspace.table.draftStatus")} {draftStatusLabel}
            </WorkspaceMetaFlag>
            {showDensityToggle ? (
              <WorkspaceDensityToggle
                density={density}
                onDensityChange={onDensityChange}
                className="shrink-0"
              />
            ) : null}
          </div>
        </div>
      </div>

      {showStyleSidebar ? (
        <aside className={`min-h-0 overflow-hidden ${DASHBOARD_DIALOG_SOFT_SURFACE_CLASS}`}>
          {styleSidebarContent}
        </aside>
      ) : null}
    </div>
  )
}
