import * as React from "react"
import { MediaPlayer, MediaProvider } from "@vidstack/react"
import type { AudioSrc, PlayerSrc, VideoSrc } from "@vidstack/react"
import { Events, Window } from "@wailsio/runtime"
import { Crosshair, Maximize, Maximize2, Minimize, Minimize2, Pause, Play, Volume2, VolumeX } from "lucide-react"

import "@vidstack/react/player/styles/base.css"

import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"

import {
  PreviewGuideLegend,
  PreviewViewportBoundary,
  RenderedFrameReferenceGuides,
} from "../PreviewReferenceGuides"
import { useJassubPreview } from "../useJassubPreview"
import { clampMs, formatCueTime } from "./utils"

type VideoPreviewPaneProps = {
  mediaUrl: string
  mediaType?: string
  durationMs: number
  playheadMs: number
  isPlaying: boolean
  previewASSContent: string
  previewFontFamilies?: string[]
  onRenderedVideoSizeChange?: (size: { width: number; height: number }) => void
  onPlayheadChange: (value: number) => void
  onPlayingChange: (value: boolean) => void
}

const PREVIEW_SHELL_CLASS =
  "flex h-full min-h-0 flex-col overflow-hidden rounded-[18px] border border-border/70 bg-[#0b1118] text-white shadow-[0_20px_60px_-36px_rgba(15,23,42,0.65)]"
const PREVIEW_CONTROL_BUTTON_CLASS =
  "rounded-full text-white/90 hover:bg-white/15 hover:text-white focus-visible:ring-white/50 focus-visible:ring-offset-0"
const PREVIEW_CONTROL_RANGE_CLASS = "h-0.5 cursor-pointer accent-primary"
const PLAYHEAD_PARENT_SYNC_INTERVAL_MS = 120
const PLAYHEAD_PARENT_DRIFT_MS = 180
const PREVIEW_VOLUME_RANGE_CLASS = cn(
  PREVIEW_CONTROL_RANGE_CLASS,
  "ml-0 w-0 min-w-0 opacity-0 transition-[margin,width,opacity] duration-150 ease-out",
  "pointer-events-none group-hover/volume:ml-1.5 group-hover/volume:w-16 group-hover/volume:opacity-100 group-hover/volume:pointer-events-auto",
  "group-focus-within/volume:ml-1.5 group-focus-within/volume:w-16 group-focus-within/volume:opacity-100 group-focus-within/volume:pointer-events-auto",
  "sm:group-hover/volume:w-20 sm:group-focus-within/volume:w-20",
)

function clampVolume(value: number) {
  if (!Number.isFinite(value)) {
    return 1
  }
  return Math.min(1, Math.max(0, value))
}

function normalizePlayhead(value: number, durationMs: number) {
  return Math.round(clampMs(value, durationMs))
}

type MediaPlayerElement = React.ElementRef<typeof MediaPlayer>
type PreviewFullscreenMode = "dom" | "wails"

export function VideoPreviewPane({
  mediaUrl,
  mediaType,
  durationMs,
  playheadMs,
  isPlaying,
  previewASSContent,
  previewFontFamilies = [],
  onRenderedVideoSizeChange,
  onPlayheadChange,
  onPlayingChange,
}: VideoPreviewPaneProps) {
  const { t } = useI18n()
  const [playerElement, setPlayerElement] = React.useState<MediaPlayerElement | null>(null)
  const shellRef = React.useRef<HTMLDivElement | null>(null)
  const previewViewportRef = React.useRef<HTMLDivElement | null>(null)
  const animationFrameRef = React.useRef<number>()
  const fullscreenModeRef = React.useRef<PreviewFullscreenMode | null>(null)
  const previousWindowedFullscreenRef = React.useRef(false)
  const lastNonZeroVolumeRef = React.useRef(1)
  const [videoElement, setVideoElement] = React.useState<HTMLVideoElement | null>(null)
  const [canvasElement, setCanvasElement] = React.useState<HTMLCanvasElement | null>(null)
  const [localPlayheadMs, setLocalPlayheadMs] = React.useState(() =>
    normalizePlayhead(playheadMs, durationMs),
  )
  const localPlayheadMsRef = React.useRef(localPlayheadMs)
  const lastCommittedPlayheadMsRef = React.useRef(localPlayheadMs)
  const pendingSeekTargetMsRef = React.useRef<number | null>(null)
  const [viewportSize, setViewportSize] = React.useState({ width: 0, height: 0 })
  const [mediaNaturalSize, setMediaNaturalSize] = React.useState({ width: 0, height: 0 })
  const [volume, setVolume] = React.useState(1)
  const [muted, setMuted] = React.useState(false)
  const [windowedFullscreen, setWindowedFullscreen] = React.useState(false)
  const [screenFullscreen, setScreenFullscreen] = React.useState(false)
  const [showReferenceGuides, setShowReferenceGuides] = React.useState(false)
  const previewCanvasKey = React.useMemo(
    () => `workspace-preview-canvas:${mediaUrl || "empty"}`,
    [mediaUrl],
  )
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
    setVideoElement(target)
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
    setVideoElement(null)
  }, [mediaUrl])

  React.useEffect(() => {
    const player = playerElement
    if (!player) {
      return
    }
    const nextVolume = clampVolume(Number(player.volume ?? 1))
    setVolume(nextVolume)
    if (nextVolume > 0) {
      lastNonZeroVolumeRef.current = nextVolume
    }
    setMuted(Boolean(player.muted))
  }, [playerElement])

  const updateLocalPlayhead = React.useCallback(
    (value: number) => {
      const next = normalizePlayhead(value, durationMs)
      localPlayheadMsRef.current = next
      setLocalPlayheadMs((current) => (current === next ? current : next))
      return next
    },
    [durationMs],
  )

  const syncPlayheadToParent = React.useCallback(
    (value: number, force = false) => {
      const next = normalizePlayhead(value, durationMs)
      if (!force && next === lastCommittedPlayheadMsRef.current) {
        return next
      }
      lastCommittedPlayheadMsRef.current = next
      onPlayheadChange(next)
      return next
    },
    [durationMs, onPlayheadChange],
  )

  const readCurrentPlaybackMs = React.useCallback(() => {
    const activeMedia = videoElement ?? playerElement
    if (!activeMedia) {
      return 0
    }
    return Number(activeMedia.currentTime || 0) * 1000
  }, [playerElement, videoElement])

  const writeCurrentPlaybackMs = React.useCallback(
    (value: number) => {
      const nextSeconds = normalizePlayhead(value, durationMs) / 1000
      const activeMedia = videoElement ?? playerElement
      if (activeMedia) {
        activeMedia.currentTime = nextSeconds
      }
      if (playerElement && playerElement !== activeMedia) {
        playerElement.currentTime = nextSeconds
      }
    },
    [durationMs, playerElement, videoElement],
  )

  React.useLayoutEffect(() => {
    const next = normalizePlayhead(playheadMs, durationMs)
    const shouldIgnoreOwnSync =
      next === lastCommittedPlayheadMsRef.current &&
      Math.abs(next - localPlayheadMsRef.current) <= PLAYHEAD_PARENT_DRIFT_MS
    if (shouldIgnoreOwnSync) {
      return
    }
    const currentPlaybackMs = readCurrentPlaybackMs()
    if (Math.abs(currentPlaybackMs - next) > PLAYHEAD_PARENT_DRIFT_MS) {
      pendingSeekTargetMsRef.current = next
    }
    updateLocalPlayhead(next)
    lastCommittedPlayheadMsRef.current = next
  }, [durationMs, mediaUrl, playheadMs, readCurrentPlaybackMs, updateLocalPlayhead])

  React.useEffect(() => {
    if (isPlaying || localPlayheadMsRef.current === lastCommittedPlayheadMsRef.current) {
      return
    }
    syncPlayheadToParent(localPlayheadMsRef.current, true)
  }, [isPlaying, syncPlayheadToParent])

  React.useEffect(() => {
    const player = playerElement
    if (!player) {
      return
    }
    player.volume = volume
    player.muted = muted
  }, [mediaUrl, muted, playerElement, volume])

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

  // The workspace subtitle canvas is intentionally video-backed only.
  // Audio-only media stays preview-less even though the player still supports it.
  useJassubPreview({
    debugLabel: "workspace-video",
    assContent: previewASSContent,
    referencedFontFamilies: previewFontFamilies,
    canvas: canvasElement,
    video: videoElement,
    enabled: Boolean(mediaUrl) && previewASSContent.trim().length > 0,
    requireVideo: true,
  })

  React.useLayoutEffect(() => {
    if (!mediaUrl) {
      return
    }
    const currentTime = readCurrentPlaybackMs()
    const pendingSeekTargetMs = pendingSeekTargetMsRef.current
    const shouldForceSeek =
      pendingSeekTargetMs !== null &&
      Math.abs(currentTime - pendingSeekTargetMs) > PLAYHEAD_PARENT_DRIFT_MS
    if (shouldForceSeek || Math.abs(currentTime - localPlayheadMs) > PLAYHEAD_PARENT_DRIFT_MS) {
      writeCurrentPlaybackMs(localPlayheadMs)
    }
  }, [localPlayheadMs, mediaUrl, readCurrentPlaybackMs, writeCurrentPlaybackMs])

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
      const currentPlaybackMs = readCurrentPlaybackMs()
      const pendingSeekTargetMs = pendingSeekTargetMsRef.current
      if (pendingSeekTargetMs !== null) {
        if (Math.abs(currentPlaybackMs - pendingSeekTargetMs) <= PLAYHEAD_PARENT_DRIFT_MS) {
          pendingSeekTargetMsRef.current = null
        } else {
          animationFrameRef.current = requestAnimationFrame(tick)
          return
        }
      }
      const next = updateLocalPlayhead(currentPlaybackMs)
      if (next >= durationMs) {
        syncPlayheadToParent(next, true)
        onPlayingChange(false)
        return
      }
      if (Math.abs(next - lastCommittedPlayheadMsRef.current) >= PLAYHEAD_PARENT_SYNC_INTERVAL_MS) {
        syncPlayheadToParent(next)
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
  }, [durationMs, isPlaying, mediaUrl, onPlayingChange, playerElement, readCurrentPlaybackMs, syncPlayheadToParent, updateLocalPlayhead])

  React.useEffect(() => {
    const restoreWindowedFullscreenAfterScreenExit = () => {
      setWindowedFullscreen(previousWindowedFullscreenRef.current)
      previousWindowedFullscreenRef.current = false
    }

    const handleDomFullscreenChange = () => {
      if (fullscreenModeRef.current !== "dom") {
        return
      }
      const isActive = document.fullscreenElement === shellRef.current
      setScreenFullscreen(isActive)
      if (!isActive) {
        fullscreenModeRef.current = null
        restoreWindowedFullscreenAfterScreenExit()
      }
    }

    document.addEventListener("fullscreenchange", handleDomFullscreenChange)
    return () => {
      document.removeEventListener("fullscreenchange", handleDomFullscreenChange)
    }
  }, [])

  React.useEffect(() => {
    const offWindowFullscreen = Events.On(Events.Types.Common.WindowFullscreen, () => {
      if (fullscreenModeRef.current === "wails") {
        setScreenFullscreen(true)
      }
    })
    const offWindowUnFullscreen = Events.On(Events.Types.Common.WindowUnFullscreen, () => {
      if (fullscreenModeRef.current === "wails") {
        fullscreenModeRef.current = null
        setScreenFullscreen(false)
        setWindowedFullscreen(previousWindowedFullscreenRef.current)
        previousWindowedFullscreenRef.current = false
      }
    })
    return () => {
      offWindowFullscreen()
      offWindowUnFullscreen()
    }
  }, [])

  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && windowedFullscreen && !screenFullscreen) {
        setWindowedFullscreen(false)
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [screenFullscreen, windowedFullscreen])

  const handleTogglePlay = () => {
    onPlayingChange(!isPlaying)
  }

  const handlePlayheadChange = (value: number) => {
    const next = updateLocalPlayhead(value)
    pendingSeekTargetMsRef.current = next
    syncPlayheadToParent(next, true)
  }

  const handleToggleMute = () => {
    if (muted || volume <= 0) {
      const restoredVolume = volume > 0 ? volume : lastNonZeroVolumeRef.current
      setVolume(restoredVolume)
      setMuted(false)
      return
    }
    lastNonZeroVolumeRef.current = volume
    setMuted(true)
  }

  const handleVolumeChange = (value: number) => {
    const nextVolume = clampVolume(value)
    setVolume(nextVolume)
    if (nextVolume > 0) {
      lastNonZeroVolumeRef.current = nextVolume
      setMuted(false)
      return
    }
    setMuted(true)
  }

  const restoreWindowedFullscreenAfterScreenExit = () => {
    setWindowedFullscreen(previousWindowedFullscreenRef.current)
    previousWindowedFullscreenRef.current = false
  }

  const exitScreenFullscreen = React.useCallback(async () => {
    const mode = fullscreenModeRef.current
    if (mode === "dom" && document.fullscreenElement) {
      await document.exitFullscreen()
      return
    }
    if (mode === "wails") {
      await Window.UnFullscreen()
      if (fullscreenModeRef.current === "wails") {
        fullscreenModeRef.current = null
        setScreenFullscreen(false)
        restoreWindowedFullscreenAfterScreenExit()
      }
      return
    }
    setScreenFullscreen(false)
    restoreWindowedFullscreenAfterScreenExit()
  }, [])

  const handleToggleWindowedFullscreen = () => {
    setWindowedFullscreen((value) => !value)
  }

  const handleToggleScreenFullscreen = () => {
    if (screenFullscreen) {
      void exitScreenFullscreen().catch(() => {
        setScreenFullscreen(false)
        restoreWindowedFullscreenAfterScreenExit()
      })
      return
    }

    previousWindowedFullscreenRef.current = windowedFullscreen
    setWindowedFullscreen(true)
    const shell = shellRef.current
    if (shell?.requestFullscreen) {
      void shell
        .requestFullscreen()
        .then(() => {
          fullscreenModeRef.current = "dom"
          setScreenFullscreen(true)
        })
        .catch(() => {
          fullscreenModeRef.current = "wails"
          void Window.Fullscreen()
            .then(() => setScreenFullscreen(true))
            .catch(() => {
              fullscreenModeRef.current = null
              setScreenFullscreen(false)
              restoreWindowedFullscreenAfterScreenExit()
            })
        })
      return
    }

    fullscreenModeRef.current = "wails"
    void Window.Fullscreen()
      .then(() => setScreenFullscreen(true))
      .catch(() => {
        fullscreenModeRef.current = null
        setScreenFullscreen(false)
        restoreWindowedFullscreenAfterScreenExit()
      })
  }

  const visibleVolume = muted ? 0 : volume
  const muteLabel = muted || volume <= 0 ? t("library.workspace.preview.unmute") : t("library.workspace.preview.mute")
  const windowedFullscreenLabel = windowedFullscreen
    ? t("library.workspace.preview.exitWindowedFullscreen")
    : t("library.workspace.preview.enterWindowedFullscreen")
  const screenFullscreenLabel = screenFullscreen
    ? t("library.workspace.preview.exitFullscreen")
    : t("library.workspace.preview.enterFullscreen")
  const referenceGuidesLabel = showReferenceGuides
    ? t("library.workspace.preview.hideReferenceGuides")
    : t("library.workspace.preview.showReferenceGuides")

  return (
    <div
      ref={shellRef}
      className={cn(
        PREVIEW_SHELL_CLASS,
        windowedFullscreen && "fixed inset-0 z-[200] rounded-none border-0 shadow-none",
        screenFullscreen && "rounded-none border-0 shadow-none",
      )}
    >
      <div className={cn("relative min-h-0 flex-1 bg-black p-3", (windowedFullscreen || screenFullscreen) && "p-0")}>
        {mediaUrl ? (
          <div ref={previewViewportRef} className="relative h-full w-full overflow-hidden bg-black">
            {showReferenceGuides ? (
              <>
                <PreviewViewportBoundary />
                <PreviewGuideLegend viewportSize={viewportSize} renderedSize={fittedViewportSize} />
              </>
            ) : null}
            <div
              className="absolute left-1/2 top-1/2 overflow-hidden -translate-x-1/2 -translate-y-1/2"
              style={fittedViewportStyle}
            >
              <div className="relative h-full w-full overflow-hidden">
                {showReferenceGuides ? <RenderedFrameReferenceGuides /> : null}
                <MediaPlayer
                  ref={handlePlayerRef}
                  src={playerSource}
                  controls={false}
                  crossOrigin="anonymous"
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
                <canvas
                  key={previewCanvasKey}
                  ref={setCanvasElement}
                  className="pointer-events-none absolute inset-0 z-10 h-full w-full"
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

      <div className="shrink-0 border-t border-white/5 bg-[#0f0f0f] px-3 py-1.5">
        <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <Button
            type="button"
            variant="ghost"
            size="compactIcon"
            className={PREVIEW_CONTROL_BUTTON_CLASS}
            onClick={handleTogglePlay}
            aria-label={isPlaying ? t("library.workspace.preview.pause") : t("library.workspace.preview.play")}
            title={isPlaying ? t("library.workspace.preview.pause") : t("library.workspace.preview.play")}
          >
            {isPlaying ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
          </Button>
          <div className="flex min-w-[10.5rem] flex-1 items-center">
            <span className="mr-1.5 shrink-0 font-mono text-2xs tabular-nums text-white/75">{formatCueTime(localPlayheadMs)}</span>
            <input
              type="range"
              min={0}
              max={durationMs || 1}
              value={Math.min(localPlayheadMs, durationMs || 1)}
              onChange={(event) => handlePlayheadChange(Number(event.target.value))}
              aria-label={t("library.workspace.preview.seek")}
              className={`${PREVIEW_CONTROL_RANGE_CLASS} w-full`}
            />
            <span className="ml-1.5 shrink-0 font-mono text-2xs tabular-nums text-white/75">{formatCueTime(durationMs)}</span>
          </div>
          <div className="group/volume flex shrink-0 items-center">
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className={PREVIEW_CONTROL_BUTTON_CLASS}
              onClick={handleToggleMute}
              aria-label={muteLabel}
              title={muteLabel}
            >
              {muted || volume <= 0 ? <VolumeX className="h-3 w-3" /> : <Volume2 className="h-3 w-3" />}
            </Button>
            <input
              type="range"
              min={0}
              max={1}
              step={0.01}
              value={visibleVolume}
              onChange={(event) => handleVolumeChange(Number(event.target.value))}
              aria-label={t("library.workspace.preview.volume")}
              title={t("library.workspace.preview.volume")}
              className={PREVIEW_VOLUME_RANGE_CLASS}
            />
          </div>
          <Button
            type="button"
            variant="ghost"
            size="compactIcon"
            className={cn(PREVIEW_CONTROL_BUTTON_CLASS, showReferenceGuides && "bg-white/15 text-white")}
            onClick={() => setShowReferenceGuides((value) => !value)}
            aria-label={referenceGuidesLabel}
            title={referenceGuidesLabel}
          >
            <Crosshair className="h-3 w-3" />
          </Button>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className={PREVIEW_CONTROL_BUTTON_CLASS}
              onClick={handleToggleWindowedFullscreen}
              aria-label={windowedFullscreenLabel}
              title={windowedFullscreenLabel}
            >
              {windowedFullscreen ? <Minimize2 className="h-3 w-3" /> : <Maximize2 className="h-3 w-3" />}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className={PREVIEW_CONTROL_BUTTON_CLASS}
              onClick={handleToggleScreenFullscreen}
              aria-label={screenFullscreenLabel}
              title={screenFullscreenLabel}
            >
              {screenFullscreen ? <Minimize className="h-3 w-3" /> : <Maximize className="h-3 w-3" />}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
