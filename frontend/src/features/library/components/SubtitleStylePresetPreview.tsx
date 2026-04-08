import * as React from "react"
import { parseText } from "media-captions"

import type { GenerateSubtitleStylePreviewVTTRequest as GenerateSubtitleStylePreviewVTTBindingRequest } from "../../../../bindings/dreamcreator/internal/application/library/dto"
import { GenerateSubtitleStylePreviewVTT } from "../../../../bindings/dreamcreator/internal/presentation/wails/libraryhandler"
import type {
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"

import {
  DEFAULT_PREVIEW_SIZE,
  buildCueDisplayStyle,
  buildCueTextStyle,
  buildRenderedPreviewCues,
  normalizeBaseResolution,
  normalizePreviewFontMappings,
  resolvePreviewScale,
  type PreviewSize,
  type RenderedPreviewCue,
} from "./previewCaptionRenderer"

type SubtitleStylePresetPreviewProps = {
  kind: "mono" | "bilingual"
  mono?: LibraryMonoStyleDTO | null
  bilingual?: LibraryBilingualStyleDTO | null
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  onPreviewSizeChange?: (size: PreviewSize) => void
}

const PREVIEW_REQUEST_DEBOUNCE_MS = 80
const PREVIEW_PRIMARY_TEXT = "DreamCreator 追创作 0214"
const PREVIEW_SECONDARY_TEXT = "DreamCreator 追创作 0214"

export function SubtitleStylePresetPreview({
  kind,
  mono,
  bilingual,
  fontMappings = [],
  onPreviewSizeChange,
}: SubtitleStylePresetPreviewProps) {
  const frameRef = React.useRef<HTMLDivElement | null>(null)
  const requestVersionRef = React.useRef(0)
  const [previewSize, setPreviewSize] = React.useState<PreviewSize>(DEFAULT_PREVIEW_SIZE)
  const [trackContent, setTrackContent] = React.useState("")
  const [renderedCues, setRenderedCues] = React.useState<RenderedPreviewCue[]>([])

  const baseResolution = React.useMemo(() => {
    if (kind === "bilingual" && bilingual) {
      return normalizeBaseResolution(bilingual.basePlayResX, bilingual.basePlayResY)
    }
    return normalizeBaseResolution(mono?.basePlayResX ?? 0, mono?.basePlayResY ?? 0)
  }, [bilingual, kind, mono?.basePlayResX, mono?.basePlayResY])

  const previewScale = React.useMemo(
    () => resolvePreviewScale(baseResolution, previewSize),
    [baseResolution, previewSize],
  )

  const previewRequest = React.useMemo(
    () =>
      buildPreviewRequest({
        kind,
        mono: mono ?? null,
        bilingual: bilingual ?? null,
        fontMappings,
        previewSize,
      }),
    [bilingual, fontMappings, kind, mono, previewSize],
  )
  const previewRequestKey = React.useMemo(() => JSON.stringify(previewRequest), [previewRequest])

  React.useEffect(() => {
    const element = frameRef.current
    if (!element) {
      return
    }

    const updateSize = () => {
      const nextWidth = Math.max(1, Math.round(element.clientWidth))
      const nextHeight = Math.max(1, Math.round(element.clientHeight))
      setPreviewSize((current) =>
        current.width === nextWidth && current.height === nextHeight
          ? current
          : { width: nextWidth, height: nextHeight },
      )
    }

    updateSize()
    const observer = new ResizeObserver(updateSize)
    observer.observe(element)
    return () => observer.disconnect()
  }, [baseResolution.height, baseResolution.width])

  React.useEffect(() => {
    onPreviewSizeChange?.(previewSize)
  }, [onPreviewSizeChange, previewSize])

  React.useEffect(() => {
    let cancelled = false
    const requestVersion = requestVersionRef.current + 1
    requestVersionRef.current = requestVersion

    const timer = window.setTimeout(() => {
      void GenerateSubtitleStylePreviewVTT(previewRequest)
        .then((value) => {
          const nextVersion = requestVersionRef.current
          if (cancelled || requestVersion !== nextVersion) {
            return
          }
          const vttContent = value.vttContent?.trim()
          setTrackContent(vttContent ? `${vttContent}\n` : "")
        })
        .catch(() => {
          const nextVersion = requestVersionRef.current
          if (cancelled || requestVersion !== nextVersion) {
            return
          }
          setTrackContent("")
        })
    }, PREVIEW_REQUEST_DEBOUNCE_MS)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [previewRequestKey])

  React.useEffect(() => {
    let disposed = false

    if (!trackContent.trim()) {
      setRenderedCues([])
      return
    }

    void parseText(trackContent, { type: "vtt" })
      .then((result) => {
        if (disposed) {
          return
        }
        setRenderedCues(
          buildRenderedPreviewCues({
            cues: result.cues,
            kind,
            mono: mono ?? null,
            bilingual: bilingual ?? null,
          }),
        )
      })
      .catch(() => {
        if (disposed) {
          return
        }
        setRenderedCues([])
      })

    return () => {
      disposed = true
    }
  }, [bilingual, kind, mono, trackContent])

  return (
    <div className="overflow-hidden rounded-xl border border-border/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(241,245,249,0.98))] shadow-[0_24px_72px_-42px_rgba(15,23,42,0.28)]">
      <div
        ref={frameRef}
        className="relative w-full overflow-hidden"
        style={{ aspectRatio: `${baseResolution.width} / ${baseResolution.height}` }}
      >
        <PreviewBackdrop />

        <div className="absolute inset-0 overflow-hidden">
          {renderedCues.map((cue) => (
            <div
              key={cue.key}
              className="pointer-events-none absolute overflow-visible"
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
  )
}

function PreviewBackdrop() {
  return (
    <div className="absolute inset-0 overflow-hidden bg-[linear-gradient(180deg,#f8fafc_0%,#e2e8f0_100%)]">
      <div className="absolute inset-0 bg-[linear-gradient(135deg,rgba(14,165,233,0.12),transparent_42%,rgba(249,115,22,0.08))]" />
      <div className="absolute inset-x-0 top-0 h-[52%] bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.88),rgba(255,255,255,0)_72%)]" />
      <div className="absolute left-[8%] top-[12%] h-[24%] w-[34%] rounded-full bg-[radial-gradient(circle,rgba(56,189,248,0.16),rgba(56,189,248,0)_72%)] blur-3xl" />
      <div className="absolute right-[6%] top-[18%] h-[22%] w-[26%] rounded-full bg-[radial-gradient(circle,rgba(251,191,36,0.14),rgba(251,191,36,0)_72%)] blur-3xl" />
      <div
        className="absolute inset-0 opacity-[0.16]"
        style={{
          backgroundImage:
            "linear-gradient(rgba(148,163,184,0.45) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.45) 1px, transparent 1px)",
          backgroundSize: "24px 24px",
        }}
      />
      <div className="absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-[linear-gradient(180deg,transparent,rgba(14,165,233,0.7)_12%,rgba(14,165,233,0.7)_88%,transparent)] opacity-80" />
      <div className="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-[linear-gradient(90deg,transparent,rgba(14,165,233,0.7)_12%,rgba(14,165,233,0.7)_88%,transparent)] opacity-80" />
      <div className="absolute left-1/2 top-3 -translate-x-1/2 rounded-full border border-sky-400/35 bg-white/78 px-2 py-0.5 text-[10px] font-medium tracking-[0.12em] text-sky-700 shadow-sm">
        50%
      </div>
      <div className="absolute left-3 top-1/2 -translate-y-1/2 rounded-full border border-sky-400/35 bg-white/78 px-2 py-0.5 text-[10px] font-medium tracking-[0.12em] text-sky-700 shadow-sm">
        50%
      </div>
      <div className="absolute inset-x-[8%] bottom-[14%] h-px bg-[linear-gradient(90deg,transparent,rgba(148,163,184,0.72),transparent)]" />
    </div>
  )
}

function buildPreviewRequest({
  kind,
  mono,
  bilingual,
  fontMappings,
  previewSize,
}: {
  kind: "mono" | "bilingual"
  mono: LibraryMonoStyleDTO | null
  bilingual: LibraryBilingualStyleDTO | null
  fontMappings: LibrarySubtitleStyleFontDTO[]
  previewSize: PreviewSize
}): GenerateSubtitleStylePreviewVTTBindingRequest {
  if (kind === "bilingual" && bilingual) {
    return {
      type: "bilingual",
      bilingual,
      fontMappings: normalizePreviewFontMappings(fontMappings),
      primaryText: PREVIEW_PRIMARY_TEXT,
      secondaryText: PREVIEW_SECONDARY_TEXT,
      previewWidth: previewSize.width,
      previewHeight: previewSize.height,
    }
  }

  return {
    type: "mono",
    mono: mono ?? undefined,
    fontMappings: normalizePreviewFontMappings(fontMappings),
    primaryText: PREVIEW_PRIMARY_TEXT,
    previewWidth: previewSize.width,
    previewHeight: previewSize.height,
  }
}
