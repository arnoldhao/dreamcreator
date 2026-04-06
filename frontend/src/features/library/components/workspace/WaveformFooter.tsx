import * as React from "react"
import WavesurferPlayer from "@wavesurfer/react"
import type WaveSurfer from "wavesurfer.js"

import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"
import { DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS } from "@/shared/ui/dashboard"

import type { WorkspaceResolvedSubtitleRow } from "./types"
import { clampMs, formatCueTime } from "./utils"

type WaveformFooterProps = {
  mediaUrl: string
  disabledReason?: string
  durationMs: number
  playheadMs: number
  rows: WorkspaceResolvedSubtitleRow[]
  selectedRowId: string
  currentRowId: string
  hoveredRowId: string
  onSeek: (value: number) => void
  onSelectRow: (rowId: string) => void
  onHoverRow: (rowId: string) => void
}

const PLACEHOLDER_BARS = Array.from({ length: 90 }, (_, index) => 24 + (Math.sin(index / 3) + 1) * 16 + (index % 7) * 2)
const WAVEFORM_STAGE_SHELL_CLASS =
  "relative overflow-hidden rounded-[16px] border border-border/70 bg-[#101722] px-3 py-3"
const WAVEFORM_STAGE_CANVAS_CLASS =
  "relative h-[108px] rounded-[12px] bg-[linear-gradient(180deg,rgba(148,163,184,0.06),rgba(15,23,42,0.22))]"
const WAVEFORM_STAGE_OVERLAY_CLASS = "pointer-events-none absolute inset-0 rounded-[12px] border border-white/6"

export function WaveformFooter({
  mediaUrl,
  disabledReason,
  durationMs,
  playheadMs,
  rows,
  selectedRowId,
  currentRowId,
  hoveredRowId,
  onSeek,
  onSelectRow,
  onHoverRow,
}: WaveformFooterProps) {
  const { t } = useI18n()
  const [wavesurfer, setWavesurfer] = React.useState<WaveSurfer | null>(null)
  const [waveformFailed, setWaveformFailed] = React.useState(false)
  const syncInteractionRef = React.useRef(false)

  React.useEffect(() => {
    setWavesurfer(null)
    setWaveformFailed(false)
    syncInteractionRef.current = false
  }, [disabledReason, mediaUrl])

  React.useEffect(() => {
    if (!wavesurfer) {
      return
    }
    const currentTime = wavesurfer.getCurrentTime()
    const nextTime = playheadMs / 1000
    if (Math.abs(currentTime - nextTime) > 0.08) {
      syncInteractionRef.current = true
      wavesurfer.setTime(nextTime)
      queueMicrotask(() => {
        syncInteractionRef.current = false
      })
    }
  }, [playheadMs, wavesurfer])

  const shouldUsePlaceholder = !mediaUrl || waveformFailed || Boolean(disabledReason)

  return (
    <div className={`flex min-h-[148px] shrink-0 flex-col px-4 py-3 shadow-[0_18px_45px_-34px_rgba(15,23,42,0.42)] ${DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS}`}>
      <div className="mb-3 shrink-0 flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
        <div className="flex items-center gap-3">
          <span className="font-medium text-foreground">{t("library.workspace.waveform.title")}</span>
          <span>{t("library.workspace.waveform.cueCount").replace("{count}", String(rows.length))}</span>
          <span>{formatCueTime(playheadMs)} / {formatCueTime(durationMs)}</span>
        </div>
        <div className="flex items-center gap-3">
          <span>{t("library.workspace.waveform.segmentMarkers")}</span>
          {disabledReason ? (
            <span className="rounded-full border border-amber-300/60 bg-amber-50/85 px-2 py-1 text-[11px] font-medium text-amber-900 dark:border-amber-700/60 dark:bg-amber-900/40 dark:text-amber-100">
              {t("library.workspace.waveform.disabled")}
            </span>
          ) : null}
          <span>
            {currentRowId
              ? t("library.workspace.waveform.liveCue").replace("{count}", currentRowId.replace("cue-", ""))
              : t("library.workspace.waveform.idle")}
          </span>
        </div>
      </div>

      <div className={WAVEFORM_STAGE_SHELL_CLASS}>
        <div className={WAVEFORM_STAGE_CANVAS_CLASS}>
          {shouldUsePlaceholder ? (
            <div className="absolute inset-0 flex items-end gap-[2px] px-1 pb-3">
              {PLACEHOLDER_BARS.map((height, index) => (
                <div
                  key={index}
                  className="flex-1 rounded-full bg-slate-400/25"
                  style={{ height }}
                />
              ))}
            </div>
          ) : (
            <div className="absolute inset-x-0 bottom-0 top-2 opacity-85">
              <WavesurferPlayer
                height={80}
                waveColor="#556274"
                progressColor="#94b8ff"
                cursorColor="transparent"
                barWidth={2}
                barGap={1}
                normalize
                interact
                autoCenter={false}
                hideScrollbar
                url={mediaUrl}
                onReady={(instance) => {
                  setWavesurfer((current) => (current == instance ? current : instance))
                  setWaveformFailed(false)
                }}
                onInteraction={(_, newTime) => {
                  if (syncInteractionRef.current) {
                    return
                  }
                  onSeek(Math.round(clampMs(newTime * 1000, durationMs)))
                }}
                onError={() => {
                  setWaveformFailed(true)
                }}
              />
            </div>
          )}

          {disabledReason ? (
            <div className="pointer-events-none absolute inset-x-8 top-1/2 -translate-y-1/2 rounded-lg border border-amber-300/55 bg-background/92 px-4 py-3 text-center text-xs leading-5 text-foreground shadow-[0_12px_32px_-20px_rgba(15,23,42,0.55)] backdrop-blur-sm dark:border-amber-700/50 dark:bg-slate-950/72 dark:text-slate-100">
              {disabledReason}
            </div>
          ) : null}

          <div className={WAVEFORM_STAGE_OVERLAY_CLASS} />

          <div className="absolute inset-x-0 top-0 h-full">
            {rows.map((row) => {
              const left = durationMs > 0 ? (row.startMs / durationMs) * 100 : 0
              const width = durationMs > 0 ? Math.max(1.6, ((row.endMs - row.startMs) / durationMs) * 100) : 1.6
              const selected = row.id === selectedRowId
              const current = row.id === currentRowId
              const hovered = row.id === hoveredRowId
              return (
                <button
                  key={row.id}
                  type="button"
                  className={cn(
                    "absolute bottom-3 top-3 rounded-md border transition-colors",
                    "border-sky-200/14 bg-sky-200/[0.08] hover:bg-sky-200/[0.12]",
                    row.qaIssues.length > 0 && "border-amber-300/18 bg-amber-300/[0.12] hover:bg-amber-300/[0.16]",
                    current && "border-sky-300/40 bg-sky-300/[0.2]",
                    selected && "border-white/40 bg-white/[0.18]",
                    hovered && "bg-white/[0.16]",
                  )}
                  style={{ left: `${left}%`, width: `${width}%` }}
                  onClick={() => {
                    onSelectRow(row.id)
                    onSeek(row.startMs)
                  }}
                  onMouseEnter={() => onHoverRow(row.id)}
                  onMouseLeave={() => onHoverRow("")}
                  aria-label={t("library.workspace.preview.cueLabel").replace("{count}", String(row.index))}
                />
              )
            })}
          </div>

          <div
            className="pointer-events-none absolute bottom-2 top-2 w-px bg-sky-300 shadow-[0_0_0_1px_rgba(125,211,252,0.4),0_0_22px_rgba(56,189,248,0.28)]"
            style={{ left: `${durationMs > 0 ? (playheadMs / durationMs) * 100 : 0}%` }}
          />

          {[0.25, 0.5, 0.75].map((marker) => (
            <div
              key={marker}
              className="pointer-events-none absolute bottom-2 top-2 w-px border-l border-dashed border-white/10"
              style={{ left: `${marker * 100}%` }}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
