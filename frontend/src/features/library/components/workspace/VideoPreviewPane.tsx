import * as React from "react"
import { MediaPlayer, MediaProvider } from "@vidstack/react"
import type { AudioSrc, PlayerSrc, VideoSrc } from "@vidstack/react"
import { Events, Window } from "@wailsio/runtime"
import { parseText } from "media-captions"
import type { VTTCue } from "media-captions"
import { Maximize, Maximize2, Minimize, Minimize2, Pause, Play, Volume2, VolumeX } from "lucide-react"

import "@vidstack/react/player/styles/base.css"

import { cn } from "@/lib/utils"
import type {
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"
import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"

import {
  buildCueDisplayStyle,
  buildCueTextStyle,
  buildRenderedPreviewCues,
  normalizeBaseResolution,
  resolvePreviewScale,
} from "../previewCaptionRenderer"
import { clampMs, formatCueTime } from "./utils"

type VideoPreviewPaneProps = {
  mediaUrl: string
  mediaType?: string
  durationMs: number
  playheadMs: number
  isPlaying: boolean
  previewVttContent: string
  displayMode: "single" | "dual"
  monoStyle: LibraryMonoStyleDTO | null
  lingualStyle: LibraryBilingualStyleDTO | null
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  onRenderedVideoSizeChange?: (size: { width: number; height: number }) => void
  onPlayheadChange: (value: number) => void
  onPlayingChange: (value: boolean) => void
}

const PREVIEW_SHELL_CLASS =
  "flex h-full min-h-0 flex-col overflow-hidden rounded-[18px] border border-border/70 bg-[#0b1118] text-white shadow-[0_20px_60px_-36px_rgba(15,23,42,0.65)]"
const PREVIEW_CONTROL_BUTTON_CLASS =
  "rounded-full text-white/90 hover:bg-white/15 hover:text-white focus-visible:ring-white/50 focus-visible:ring-offset-0"
const PREVIEW_CONTROL_RANGE_CLASS = "h-1 cursor-pointer accent-primary"
const PREVIEW_VOLUME_RANGE_CLASS = cn(
  PREVIEW_CONTROL_RANGE_CLASS,
  "ml-0 w-0 min-w-0 opacity-0 transition-[margin,width,opacity] duration-150 ease-out",
  "pointer-events-none group-hover/volume:ml-2 group-hover/volume:w-20 group-hover/volume:opacity-100 group-hover/volume:pointer-events-auto",
  "group-focus-within/volume:ml-2 group-focus-within/volume:w-20 group-focus-within/volume:opacity-100 group-focus-within/volume:pointer-events-auto",
  "sm:group-hover/volume:w-24 sm:group-focus-within/volume:w-24",
)

function clampVolume(value: number) {
  if (!Number.isFinite(value)) {
    return 1
  }
  return Math.min(1, Math.max(0, value))
}

type MediaPlayerElement = React.ElementRef<typeof MediaPlayer>
type PreviewFullscreenMode = "dom" | "wails"

export function VideoPreviewPane({
  mediaUrl,
  mediaType,
  durationMs,
  playheadMs,
  isPlaying,
  previewVttContent,
  displayMode,
  monoStyle,
  lingualStyle,
  fontMappings = [],
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
  const [viewportSize, setViewportSize] = React.useState({ width: 0, height: 0 })
  const [mediaNaturalSize, setMediaNaturalSize] = React.useState({ width: 0, height: 0 })
  const [parsedCues, setParsedCues] = React.useState<VTTCue[]>([])
  const [volume, setVolume] = React.useState(1)
  const [muted, setMuted] = React.useState(false)
  const [windowedFullscreen, setWindowedFullscreen] = React.useState(false)
  const [screenFullscreen, setScreenFullscreen] = React.useState(false)
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

  const effectiveMonoStyle = React.useMemo(() => monoStyle ?? buildWorkspacePreviewDefaultMonoStyle(), [monoStyle])

  const effectiveLingualStyle = React.useMemo(
    () => lingualStyle ?? buildWorkspacePreviewFallbackLingualStyle(effectiveMonoStyle),
    [effectiveMonoStyle, lingualStyle],
  )

  const previewBaseResolution = React.useMemo(() => {
    if (displayMode === "dual") {
      return normalizeBaseResolution(effectiveLingualStyle.basePlayResX, effectiveLingualStyle.basePlayResY)
    }
    return normalizeBaseResolution(effectiveMonoStyle.basePlayResX, effectiveMonoStyle.basePlayResY)
  }, [displayMode, effectiveLingualStyle.basePlayResX, effectiveLingualStyle.basePlayResY, effectiveMonoStyle.basePlayResX, effectiveMonoStyle.basePlayResY])

  const previewScale = React.useMemo(
    () => resolvePreviewScale(previewBaseResolution, fittedViewportSize),
    [fittedViewportSize, previewBaseResolution],
  )

  React.useEffect(() => {
    let disposed = false
    const content = previewVttContent.trim()
    if (!content) {
      setParsedCues([])
      return
    }
    void parseText(content, { type: "vtt" })
      .then((track) => {
        if (disposed) {
          return
        }
        setParsedCues(track.cues)
      })
      .catch(() => {
        if (disposed) {
          return
        }
        setParsedCues([])
      })
    return () => {
      disposed = true
    }
  }, [previewVttContent])

  const renderedCues = React.useMemo(
    () =>
      buildRenderedPreviewCues({
        cues: parsedCues,
        kind: displayMode === "dual" ? "bilingual" : "mono",
        mono: effectiveMonoStyle,
        bilingual: effectiveLingualStyle,
        currentTimeSeconds: playheadMs / 1000,
        latestOnlyPerKey: true,
      }),
    [displayMode, effectiveLingualStyle, effectiveMonoStyle, parsedCues, playheadMs],
  )

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
                <div className="pointer-events-none absolute inset-0 z-10 overflow-hidden">
                  {renderedCues.map((cue) => (
                    <div
                      key={`${cue.key}:${cue.html}`}
                      className="absolute overflow-visible"
                      style={buildCueDisplayStyle(cue.style, previewScale)}
                    >
                      <div
                        style={buildCueTextStyle(cue.style, fontMappings, previewScale)}
                        dangerouslySetInnerHTML={{ __html: cue.html }}
                      />
                    </div>
                  ))}
                </div>
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

      <div className="shrink-0 border-t border-white/5 bg-[#0f0f0f] px-4 py-2.5">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="compactIcon"
            className={PREVIEW_CONTROL_BUTTON_CLASS}
            onClick={handleTogglePlay}
            aria-label={isPlaying ? t("library.workspace.preview.pause") : t("library.workspace.preview.play")}
            title={isPlaying ? t("library.workspace.preview.pause") : t("library.workspace.preview.play")}
          >
            {isPlaying ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
          </Button>
          <div className="flex min-w-[12rem] flex-1 items-center">
            <span className="mr-2 shrink-0 font-mono text-xs tabular-nums text-white/75">{formatCueTime(playheadMs)}</span>
            <input
              type="range"
              min={0}
              max={durationMs || 1}
              value={Math.min(playheadMs, durationMs || 1)}
              onChange={(event) => onPlayheadChange(Number(event.target.value))}
              aria-label={t("library.workspace.preview.seek")}
              className={`${PREVIEW_CONTROL_RANGE_CLASS} w-full`}
            />
            <span className="ml-2 shrink-0 font-mono text-xs tabular-nums text-white/75">{formatCueTime(durationMs)}</span>
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
              {muted || volume <= 0 ? <VolumeX className="h-3.5 w-3.5" /> : <Volume2 className="h-3.5 w-3.5" />}
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
              {windowedFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
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
              {screenFullscreen ? <Minimize className="h-3.5 w-3.5" /> : <Maximize className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function buildWorkspacePreviewDefaultMonoStyle(): LibraryMonoStyleDTO {
  return {
    id: "workspace-mono-default",
    name: "Workspace Mono",
    basePlayResX: 1920,
    basePlayResY: 1080,
    baseAspectRatio: "16:9",
    style: {
      fontname: "Arial",
      fontsize: 48,
      primaryColour: "&H00FFFFFF",
      secondaryColour: "&H00FFFFFF",
      outlineColour: "&H00111111",
      backColour: "&HFF111111",
      bold: false,
      italic: false,
      underline: false,
      strikeOut: false,
      scaleX: 100,
      scaleY: 100,
      spacing: 0,
      angle: 0,
      borderStyle: 1,
      outline: 2,
      shadow: 0,
      alignment: 2,
      marginL: 72,
      marginR: 72,
      marginV: 56,
      encoding: 1,
    },
  }
}

function buildWorkspacePreviewFallbackLingualStyle(mono: LibraryMonoStyleDTO): LibraryBilingualStyleDTO {
  const primarySnapshot = {
    sourceMonoStyleID: mono.id,
    sourceMonoStyleName: mono.name,
    name: mono.name || "Primary",
    basePlayResX: mono.basePlayResX,
    basePlayResY: mono.basePlayResY,
    baseAspectRatio: mono.baseAspectRatio,
    style: { ...mono.style },
  }
  const secondarySnapshot = { ...primarySnapshot, style: { ...primarySnapshot.style } }

  return {
    id: "workspace-lingual-fallback",
    name: "Workspace Lingual",
    basePlayResX: mono.basePlayResX,
    basePlayResY: mono.basePlayResY,
    baseAspectRatio: mono.baseAspectRatio,
    primary: primarySnapshot,
    secondary: secondarySnapshot,
    layout: {
      gap: 24,
      blockAnchor: 2,
    },
  }
}
