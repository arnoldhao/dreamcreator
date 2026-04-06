import * as React from "react"
import { MediaPlayer, MediaProvider } from "@vidstack/react"
import type { AudioSrc, PlayerSrc, VideoSrc } from "@vidstack/react"
import { CaptionsRenderer, parseText } from "media-captions"
import { Pause, Play } from "lucide-react"

import "@vidstack/react/player/styles/base.css"
import "media-captions/styles/captions.css"
import "media-captions/styles/regions.css"

import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"

import { clampMs, formatCueTime } from "./utils"

type VideoPreviewPaneProps = {
  mediaUrl: string
  mediaType?: string
  durationMs: number
  playheadMs: number
  isPlaying: boolean
  previewVttContent: string
  previewStylesheet?: string
  captionsClassName?: string
  onRenderedVideoSizeChange?: (size: { width: number; height: number }) => void
  onPlayheadChange: (value: number) => void
  onPlayingChange: (value: boolean) => void
}

const PREVIEW_SHELL_CLASS =
  "flex h-full min-h-0 flex-col overflow-hidden rounded-[18px] border border-border/70 bg-[#0b1118] text-white shadow-[0_20px_60px_-36px_rgba(15,23,42,0.65)]"
const PREVIEW_CONTROL_BUTTON_CLASS = "border-white/12 bg-white/5 text-white hover:bg-white/10"

type MediaPlayerElement = React.ElementRef<typeof MediaPlayer>

export function VideoPreviewPane({
  mediaUrl,
  mediaType,
  durationMs,
  playheadMs,
  isPlaying,
  previewVttContent,
  previewStylesheet,
  captionsClassName,
  onRenderedVideoSizeChange,
  onPlayheadChange,
  onPlayingChange,
}: VideoPreviewPaneProps) {
  const { t } = useI18n()
  const [playerElement, setPlayerElement] = React.useState<MediaPlayerElement | null>(null)
  const previewViewportRef = React.useRef<HTMLDivElement | null>(null)
  const overlayRef = React.useRef<HTMLDivElement | null>(null)
  const rendererRef = React.useRef<CaptionsRenderer | null>(null)
  const animationFrameRef = React.useRef<number>()
  const [viewportSize, setViewportSize] = React.useState({ width: 0, height: 0 })
  const [mediaNaturalSize, setMediaNaturalSize] = React.useState({ width: 0, height: 0 })
  const handlePlayerRef = React.useCallback((node: MediaPlayerElement | null) => {
    setPlayerElement(node)
  }, [])
  const playerSource = React.useMemo<PlayerSrc | undefined>(() => {
    if (!mediaUrl) {
      return undefined
    }
    const sourceType = mediaType ?? ""
    if (sourceType.startsWith("audio/")) {
      return {
        src: mediaUrl,
        type: sourceType as AudioSrc["type"],
      }
    }
    if (sourceType.startsWith("video/")) {
      return {
        src: mediaUrl,
        type: sourceType as VideoSrc["type"],
      }
    }
    return mediaUrl
  }, [mediaType, mediaUrl])
  const handleMediaLoadedMetadata = React.useCallback((event: React.SyntheticEvent<HTMLMediaElement>) => {
    const target = event.currentTarget
    if (!(target instanceof HTMLVideoElement)) {
      return
    }
    setMediaNaturalSize({
      width: target.videoWidth || 0,
      height: target.videoHeight || 0,
    })
  }, [])

  React.useEffect(() => {
    const viewport = previewViewportRef.current
    if (!viewport) {
      return
    }
    const updateSize = () => {
      setViewportSize({
        width: viewport.clientWidth,
        height: viewport.clientHeight,
      })
    }
    updateSize()
    if (typeof ResizeObserver == "undefined") {
      window.addEventListener("resize", updateSize)
      return () => window.removeEventListener("resize", updateSize)
    }
    const observer = new ResizeObserver(() => updateSize())
    observer.observe(viewport)
    return () => observer.disconnect()
  }, [])

  React.useEffect(() => {
    setMediaNaturalSize({ width: 0, height: 0 })
  }, [mediaUrl])

  const fittedViewportSize = React.useMemo(() => {
    const { width: viewportWidth, height: viewportHeight } = viewportSize
    const { width: mediaWidth, height: mediaHeight } = mediaNaturalSize
    if (viewportWidth <= 0 || viewportHeight <= 0 || mediaWidth <= 0 || mediaHeight <= 0) {
      return {
        width: Math.max(0, Math.floor(viewportWidth)),
        height: Math.max(0, Math.floor(viewportHeight)),
      }
    }
    const scale = Math.min(viewportWidth / mediaWidth, viewportHeight / mediaHeight)
    return {
      width: Math.max(1, Math.floor(mediaWidth * scale)),
      height: Math.max(1, Math.floor(mediaHeight * scale)),
    }
  }, [mediaNaturalSize, viewportSize])
  const fittedViewportStyle = React.useMemo<React.CSSProperties>(
    () =>
      fittedViewportSize.width > 0 && fittedViewportSize.height > 0
        ? {
            width: `${fittedViewportSize.width}px`,
            height: `${fittedViewportSize.height}px`,
          }
        : { width: "100%", height: "100%" },
    [fittedViewportSize],
  )

  React.useEffect(() => {
    onRenderedVideoSizeChange?.(fittedViewportSize)
  }, [fittedViewportSize, onRenderedVideoSizeChange])

  React.useEffect(() => {
    const overlay = overlayRef.current
    if (!overlay) {
      return
    }
    const renderer = new CaptionsRenderer(overlay, { dir: "ltr" })
    rendererRef.current = renderer
    return () => {
      rendererRef.current = null
      renderer.destroy()
    }
  }, [])

  React.useEffect(() => {
    const renderer = rendererRef.current
    if (!renderer) {
      return
    }
    let disposed = false
    const content = previewVttContent.trim()
    if (!content) {
      renderer.reset()
      return
    }
    void parseText(content, { type: "vtt" })
      .then((track) => {
        if (disposed || rendererRef.current !== renderer) {
          return
        }
        renderer.changeTrack(track)
        renderer.currentTime = playheadMs / 1000
        renderer.update(true)
      })
      .catch(() => {
        if (!disposed) {
          renderer.reset()
        }
      })
    return () => {
      disposed = true
    }
  }, [previewVttContent])

  React.useEffect(() => {
    const renderer = rendererRef.current
    if (!renderer) {
      return
    }
    renderer.currentTime = playheadMs / 1000
    renderer.update(true)
  }, [playheadMs])

  React.useEffect(() => {
    const player = playerElement
    if (!player || !mediaUrl) {
      return
    }
    const currentTime = Number(player.currentTime || 0) * 1000
    if (Math.abs(currentTime - playheadMs) > 180) {
      player.currentTime = playheadMs / 1000
    }
  }, [mediaUrl, playheadMs, playerElement])

  React.useEffect(() => {
    const player = playerElement
    if (!player || !mediaUrl) {
      return
    }
    if (isPlaying) {
      void player.play().catch(() => {
        onPlayingChange(false)
      })
      return
    }
    void player.pause().catch(() => undefined)
  }, [isPlaying, mediaUrl, onPlayingChange, playerElement])

  React.useEffect(() => {
    if (!mediaUrl || !isPlaying) {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current)
        animationFrameRef.current = undefined
      }
      return
    }

    const tick = () => {
      const player = playerElement
      if (!player) {
        return
      }
      const next = clampMs(Number(player.currentTime || 0) * 1000, durationMs)
      onPlayheadChange(next)
      if (next >= durationMs) {
        onPlayingChange(false)
        return
      }
      animationFrameRef.current = requestAnimationFrame(tick)
    }

    animationFrameRef.current = requestAnimationFrame(tick)
    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current)
        animationFrameRef.current = undefined
      }
    }
  }, [durationMs, isPlaying, mediaUrl, onPlayheadChange, onPlayingChange, playerElement])

  const handleTogglePlay = () => {
    onPlayingChange(!isPlaying)
  }

  return (
    <div className={PREVIEW_SHELL_CLASS}>
      {previewStylesheet ? <style>{previewStylesheet}</style> : null}
      <div className="relative min-h-0 flex-1 bg-black p-3">
        {mediaUrl ? (
          <div ref={previewViewportRef} className="relative h-full w-full overflow-hidden bg-black">
            <div
              className="absolute left-1/2 top-1/2 overflow-hidden -translate-x-1/2 -translate-y-1/2"
              style={fittedViewportStyle}
            >
              <div className="relative h-full w-full overflow-hidden">
                <MediaPlayer
                  ref={handlePlayerRef}
                  src={playerSource}
                  controls={false}
                  playsInline
                  preload="metadata"
                  style={{ aspectRatio: "auto" }}
                  className="h-full w-full bg-black"
                >
                  <MediaProvider
                    className="h-full w-full overflow-hidden bg-black"
                    mediaProps={{
                      className: "h-full w-full object-contain object-center",
                      onLoadedMetadata: handleMediaLoadedMetadata,
                    }}
                  />
                </MediaPlayer>
                <div
                  ref={overlayRef}
                  className={`pointer-events-none absolute inset-0 z-10 ${captionsClassName ?? ""}`.trim()}
                  style={{ "--overlay-padding": "0" } as React.CSSProperties}
                />
              </div>
            </div>
          </div>
        ) : (
          <div className="flex h-full items-center justify-center bg-[radial-gradient(circle_at_top,rgba(56,189,248,0.08),transparent_42%),linear-gradient(180deg,rgba(15,23,42,0.92),rgba(2,6,23,0.98))] px-8 text-center">
            <div className="space-y-3">
              <div className="text-lg font-semibold text-white/90">
                {t("library.workspace.preview.placeholderTitle")}
              </div>
              <div className="max-w-md text-sm text-white/55">
                {t("library.workspace.preview.placeholderDescription")}
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="border-t border-white/10 bg-black/75 px-4 py-3 backdrop-blur">
        <div className="flex items-center gap-3">
          <Button variant="outline" size="compactIcon" className={PREVIEW_CONTROL_BUTTON_CLASS} onClick={handleTogglePlay}>
            {isPlaying ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
          </Button>
          <div className="flex min-w-0 flex-1 items-center">
            <span className="mr-2 shrink-0 font-mono text-xs tabular-nums text-white/75">{formatCueTime(playheadMs)}</span>
            <input
              type="range"
              min={0}
              max={durationMs || 1}
              value={Math.min(playheadMs, durationMs || 1)}
              onChange={(event) => onPlayheadChange(Number(event.target.value))}
              className="h-1.5 w-full cursor-pointer accent-sky-400"
            />
            <span className="ml-2 shrink-0 font-mono text-xs tabular-nums text-white/75">{formatCueTime(durationMs)}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
