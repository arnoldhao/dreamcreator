import { Search, Sparkles } from "lucide-react"
import * as React from "react"
import type { ReactNode } from "react"

import { useI18n } from "@/shared/i18n"
import { Input } from "@/shared/ui/input"
import { Select } from "@/shared/ui/select"
import type {
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"
import type {
  WorkspaceDensity,
  WorkspaceDisplayMode,
  WorkspaceQaFilter,
  WorkspaceResolvedSubtitleRow,
} from "./types"
import { WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS } from "./controlStyles"
import { SubtitleTablePane } from "./SubtitleTablePane"
import { VideoPreviewPane } from "./VideoPreviewPane"
import { WaveformFooter } from "./WaveformFooter"
import { WorkspaceDensityToggle } from "./WorkspaceDensityToggle"
import { WorkspaceMetaItem } from "./WorkspaceMetaBar"
import {
  DASHBOARD_WORKSPACE_META_BAR_CLASS,
  DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS,
} from "@/shared/ui/dashboard"
import { DASHBOARD_DIALOG_SOFT_SURFACE_CLASS } from "@/shared/ui/dashboard-dialog"
import { cn } from "@/lib/utils"

type VideoEditorLayoutProps = {
  mediaUrl: string
  mediaType?: string
  waveformDisabledReason?: string
  rows: WorkspaceResolvedSubtitleRow[]
  selectedRowId: string
  currentRowId: string
  hoveredRowId: string
  displayMode: WorkspaceDisplayMode
  density: WorkspaceDensity
  searchValue: string
  qaFilter: WorkspaceQaFilter
  autoFollow: boolean
  playheadMs: number
  durationMs: number
  isPlaying: boolean
  previewVttContent: string
  previewMonoStyle?: LibraryMonoStyleDTO | null
  previewLingualStyle?: LibraryBilingualStyleDTO | null
  previewFontMappings?: LibrarySubtitleStyleFontDTO[]
  onPreviewRenderSizeChange?: (size: { width: number; height: number }) => void
  showStyleSidebar: boolean
  styleSidebarContent?: ReactNode
  isLoading: boolean
  errorMessage?: string
  onDensityChange: (value: WorkspaceDensity) => void
  onSearchChange: (value: string) => void
  onQaFilterChange: (value: WorkspaceQaFilter) => void
  onSelectRow: (rowId: string) => void
  onHoverRow: (rowId: string) => void
  onSeek: (value: number) => void
  onPlayingChange: (value: boolean) => void
}

export function VideoEditorLayout({
  mediaUrl,
  mediaType,
  waveformDisabledReason,
  rows,
  selectedRowId,
  currentRowId,
  hoveredRowId,
  displayMode,
  density,
  searchValue,
  qaFilter,
  autoFollow,
  playheadMs,
  durationMs,
  isPlaying,
  previewVttContent,
  previewMonoStyle,
  previewLingualStyle,
  previewFontMappings = [],
  onPreviewRenderSizeChange,
  showStyleSidebar,
  styleSidebarContent,
  isLoading,
  errorMessage,
  onDensityChange,
  onSearchChange,
  onQaFilterChange,
  onSelectRow,
  onHoverRow,
  onSeek,
  onPlayingChange,
}: VideoEditorLayoutProps) {
  const { t } = useI18n()
  const timelineCompressed = showStyleSidebar
  const reviewCount = rows.filter((row) => row.qaIssues.length > 0).length
  const timelineHeaderRef = React.useRef<HTMLDivElement | null>(null)
  const [timelineHeaderWidth, setTimelineHeaderWidth] = React.useState(0)
  const showDensityToggle = !timelineCompressed && timelineHeaderWidth >= 560
  const showQaFilter = timelineHeaderWidth >= 470

  React.useEffect(() => {
    if (timelineCompressed) {
      return
    }
    const headerElement = timelineHeaderRef.current
    if (!headerElement) {
      return
    }
    const updateWidth = () => {
      setTimelineHeaderWidth(headerElement.clientWidth)
    }
    updateWidth()
    window.addEventListener("resize", updateWidth)
    if (typeof ResizeObserver == "undefined") {
      return () => window.removeEventListener("resize", updateWidth)
    }
    const observer = new ResizeObserver(() => updateWidth())
    observer.observe(headerElement)
    return () => {
      observer.disconnect()
      window.removeEventListener("resize", updateWidth)
    }
  }, [timelineCompressed])

  const qaFilterOptions: Array<{ value: WorkspaceQaFilter; label: string }> = [
    { value: "all", label: t("library.workspace.toolbar.qaAll") },
    { value: "issues", label: t("library.workspace.toolbar.qaIssues") },
    { value: "warnings", label: t("library.workspace.toolbar.qaWarnings") },
    { value: "errors", label: t("library.workspace.toolbar.qaErrors") },
    { value: "clean", label: t("library.workspace.toolbar.qaClean") },
  ]

  return (
    <div className={cn("grid h-full min-h-0 flex-1 gap-3 overflow-hidden", showStyleSidebar && "grid-cols-[minmax(0,1fr)_320px]")}>
      <div className="grid min-h-0 grid-rows-[minmax(0,1fr)_auto] gap-3 overflow-hidden">
        <div className="grid min-h-0 gap-3 overflow-hidden xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
          <VideoPreviewPane
            mediaUrl={mediaUrl}
            mediaType={mediaType}
            durationMs={durationMs}
            playheadMs={playheadMs}
            isPlaying={isPlaying}
            previewVttContent={previewVttContent}
            displayMode={displayMode}
            monoStyle={previewMonoStyle ?? null}
            lingualStyle={previewLingualStyle ?? null}
            fontMappings={previewFontMappings}
            onRenderedVideoSizeChange={onPreviewRenderSizeChange}
            onPlayheadChange={onSeek}
            onPlayingChange={onPlayingChange}
          />

          <div className={`flex min-h-0 min-w-0 flex-col overflow-hidden ${DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS}`}>
            {!timelineCompressed ? (
              <div
                ref={timelineHeaderRef}
                className="flex shrink-0 items-center justify-between gap-2 overflow-hidden border-b border-border/70 bg-muted/[0.14] px-3 py-2"
              >
                <div className="flex min-w-0 flex-1 items-center gap-2">
                  <div
                    className={cn(
                      "flex h-8 min-w-0 items-center gap-2 rounded-md border border-border/70 bg-background/80 px-2",
                      WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS,
                    )}
                  >
                    <Search className="h-3.5 w-3.5 shrink-0 text-muted-foreground/80" />
                    <Input
                      value={searchValue}
                      onChange={(event) => onSearchChange(event.target.value)}
                      placeholder={t("library.workspace.toolbar.searchPlaceholder")}
                      className="h-6 border-0 bg-transparent px-0 text-xs md:text-xs shadow-none focus-visible:ring-0"
                    />
                  </div>
                  {showQaFilter ? (
                    <div
                      className={cn(
                        "flex h-8 min-w-0 items-center gap-2 rounded-md border border-border/70 bg-background/80 px-2",
                        WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS,
                      )}
                    >
                      <Sparkles className="h-3.5 w-3.5 shrink-0 text-muted-foreground/80" />
                      <Select
                        value={qaFilter}
                        onChange={(event) => onQaFilterChange(event.target.value as WorkspaceQaFilter)}
                        className="h-6 w-full border-0 bg-transparent px-0 shadow-none"
                      >
                        {qaFilterOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </Select>
                    </div>
                  ) : null}
                </div>
              </div>
            ) : null}
            <div className="min-h-0 flex-1 overflow-hidden">
              <SubtitleTablePane
                mode="video"
                title={t("library.workspace.table.timelineTitle")}
                chrome="plain"
                compressed={timelineCompressed}
                rows={rows}
                selectedRowId={selectedRowId}
                currentRowId={currentRowId}
                hoveredRowId={hoveredRowId}
                displayMode={displayMode}
                density={density}
                autoFollow={autoFollow}
                isLoading={isLoading}
                errorMessage={errorMessage}
                onSelectRow={onSelectRow}
                onHoverRow={onHoverRow}
              />
            </div>
            <div className={`grid shrink-0 gap-2 px-3 py-2 ${DASHBOARD_WORKSPACE_META_BAR_CLASS} md:grid-cols-[auto_minmax(0,1fr)] md:items-center`}>
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <WorkspaceMetaItem
                  value={t("library.workspace.table.visibleCount").replace("{count}", String(rows.length))}
                />
                {timelineCompressed ? null : (
                  <WorkspaceMetaItem
                    value={t("library.workspace.table.needReviewCount").replace("{count}", String(reviewCount))}
                  />
                )}
              </div>
              <div className="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">
                <WorkspaceMetaItem
                  value={`${t("library.workspace.table.current")} ${currentRowId ? currentRowId.replace("cue-", "") : "-"}`}
                />
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
        </div>

        <WaveformFooter
          mediaUrl={mediaUrl}
          disabledReason={waveformDisabledReason}
          durationMs={durationMs}
          playheadMs={playheadMs}
          rows={rows}
          selectedRowId={selectedRowId}
          currentRowId={currentRowId}
          hoveredRowId={hoveredRowId}
          onSeek={onSeek}
          onSelectRow={onSelectRow}
          onHoverRow={onHoverRow}
        />
      </div>

      {showStyleSidebar ? (
        <aside className={`min-h-0 overflow-hidden ${DASHBOARD_DIALOG_SOFT_SURFACE_CLASS}`}>
          {styleSidebarContent}
        </aside>
      ) : null}
    </div>
  )
}
