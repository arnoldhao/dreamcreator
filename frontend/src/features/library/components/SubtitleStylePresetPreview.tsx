import * as React from "react"
import { Crosshair } from "lucide-react"

import type { GenerateSubtitleStylePreviewASSRequest as GenerateSubtitleStylePreviewASSBindingRequest } from "../../../../bindings/dreamcreator/internal/application/library/dto"
import { GenerateSubtitleStylePreviewASS } from "../../../../bindings/dreamcreator/internal/presentation/wails/libraryhandler"
import { cn } from "@/lib/utils"
import type {
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"
import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"

import {
  PreviewGuideLegend,
  RenderedFrameReferenceGuides,
} from "./PreviewReferenceGuides"
import {
  DEFAULT_PREVIEW_SIZE,
  normalizeBaseResolution,
  normalizePreviewFontMappings,
  type PreviewSize,
} from "./previewCaptionRenderer"
import { useJassubPreview } from "./useJassubPreview"

type SubtitleStylePresetPreviewProps = {
  kind: "mono" | "bilingual"
  mono?: LibraryMonoStyleDTO | null
  bilingual?: LibraryBilingualStyleDTO | null
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  onPreviewSizeChange?: (size: PreviewSize) => void
}

const PREVIEW_REQUEST_DEBOUNCE_MS = 80
const STATIC_PREVIEW_TIME_SECONDS = 1

export function SubtitleStylePresetPreview({
  kind,
  mono,
  bilingual,
  fontMappings = [],
  onPreviewSizeChange,
}: SubtitleStylePresetPreviewProps) {
  const { t } = useI18n()
  const frameRef = React.useRef<HTMLDivElement | null>(null)
  const requestVersionRef = React.useRef(0)
  const [previewSize, setPreviewSize] = React.useState<PreviewSize>(DEFAULT_PREVIEW_SIZE)
  const [canvasElement, setCanvasElement] = React.useState<HTMLCanvasElement | null>(null)
  const [trackContent, setTrackContent] = React.useState("")
  const [referencedFontFamilies, setReferencedFontFamilies] = React.useState<string[]>([])
  const [showReferenceGuides, setShowReferenceGuides] = React.useState(false)
  const previewPrimaryText = t("library.config.subtitleStyles.previewPrimaryText")
  const previewSecondaryText = t("library.config.subtitleStyles.previewSecondaryText")

  const baseResolution = React.useMemo(() => {
    if (kind === "bilingual" && bilingual) {
      return normalizeBaseResolution(bilingual.basePlayResX, bilingual.basePlayResY)
    }
    return normalizeBaseResolution(mono?.basePlayResX ?? 0, mono?.basePlayResY ?? 0)
  }, [bilingual, kind, mono?.basePlayResX, mono?.basePlayResY])

  const previewRequest = React.useMemo(
    () =>
      buildPreviewRequest({
        kind,
        mono: mono ?? null,
        bilingual: bilingual ?? null,
        fontMappings,
        previewPrimaryText,
        previewSecondaryText,
      }),
    [bilingual, fontMappings, kind, mono, previewPrimaryText, previewSecondaryText],
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
      void GenerateSubtitleStylePreviewASS(previewRequest)
        .then((value) => {
          const nextVersion = requestVersionRef.current
          if (cancelled || requestVersion !== nextVersion) {
            return
          }
          const nextContent = value.assContent?.trim()
          setTrackContent(nextContent ? `${nextContent}\n` : "")
          setReferencedFontFamilies(value.referencedFontFamilies ?? [])
        })
        .catch(() => {
          const nextVersion = requestVersionRef.current
          if (cancelled || requestVersion !== nextVersion) {
            return
          }
          setTrackContent("")
          setReferencedFontFamilies([])
        })
    }, PREVIEW_REQUEST_DEBOUNCE_MS)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [previewRequestKey, previewRequest])

  useJassubPreview({
    debugLabel: "subtitle-style",
    assContent: trackContent,
    referencedFontFamilies,
    canvas: canvasElement,
    currentTimeSeconds: STATIC_PREVIEW_TIME_SECONDS,
    enabled: trackContent.trim().length > 0,
  })

  const referenceGuidesLabel = showReferenceGuides
    ? t("library.workspace.preview.hideReferenceGuides")
    : t("library.workspace.preview.showReferenceGuides")

  return (
    <div className="relative overflow-hidden rounded-none border border-slate-800/80 bg-[linear-gradient(180deg,rgba(10,16,24,0.98),rgba(3,6,10,0.99))] shadow-[0_28px_90px_-46px_rgba(2,6,23,0.78)]">
      <div className="absolute right-2 top-2 z-30">
        <Button
          type="button"
          variant="ghost"
          size="compactIcon"
          className={cn(
            "rounded-full bg-slate-950/78 text-white/90 hover:bg-slate-950/88 hover:text-white focus-visible:ring-white/50 focus-visible:ring-offset-0",
            showReferenceGuides && "bg-slate-950/92 text-white",
          )}
          onClick={() => setShowReferenceGuides((value) => !value)}
          aria-label={referenceGuidesLabel}
          title={referenceGuidesLabel}
        >
          <Crosshair className="h-3 w-3" />
        </Button>
      </div>
      <div
        ref={frameRef}
        className="relative w-full overflow-hidden"
        style={{ aspectRatio: `${baseResolution.width} / ${baseResolution.height}` }}
      >
        <PreviewBackdrop />
        {showReferenceGuides ? (
          <>
            <RenderedFrameReferenceGuides />
            <PreviewGuideLegend renderedSize={previewSize} />
          </>
        ) : null}

        <canvas
          ref={setCanvasElement}
          className="pointer-events-none absolute inset-0 z-10 h-full w-full"
        />
      </div>
    </div>
  )
}

function PreviewBackdrop() {
  return (
    <div className="absolute inset-0 overflow-hidden bg-[linear-gradient(180deg,#111923_0%,#070b11_52%,#03060b_100%)]">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(103,232,249,0.16),rgba(103,232,249,0)_42%),radial-gradient(circle_at_82%_18%,rgba(251,191,36,0.12),rgba(251,191,36,0)_24%),radial-gradient(circle_at_50%_100%,rgba(15,23,42,0.72),rgba(15,23,42,0)_52%)]" />
      <div className="absolute inset-[4%] border border-white/8 shadow-[inset_0_0_0_1px_rgba(103,232,249,0.08)]" />
      <div className="absolute inset-x-0 top-0 h-[34%] bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.1),rgba(255,255,255,0)_72%)]" />
      <div className="absolute left-[10%] top-[10%] h-[26%] w-[30%] rounded-full bg-[radial-gradient(circle,rgba(56,189,248,0.14),rgba(56,189,248,0)_72%)] blur-3xl" />
      <div className="absolute right-[8%] top-[16%] h-[18%] w-[22%] rounded-full bg-[radial-gradient(circle,rgba(251,191,36,0.12),rgba(251,191,36,0)_72%)] blur-3xl" />
      <div className="absolute inset-x-[16%] bottom-[8%] h-[24%] rounded-[50%] bg-[radial-gradient(circle,rgba(56,189,248,0.08),rgba(56,189,248,0)_72%)] blur-3xl" />
      <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(255,255,255,0.03),rgba(255,255,255,0)_18%,rgba(255,255,255,0)_82%,rgba(0,0,0,0.24))]" />
      <div
        className="absolute inset-0 opacity-[0.12]"
        style={{
          backgroundImage:
            "repeating-linear-gradient(180deg, rgba(255,255,255,0.05) 0 1px, transparent 1px 4px), linear-gradient(rgba(148,163,184,0.22) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.18) 1px, transparent 1px)",
          backgroundSize: "100% 100%, 32px 32px, 32px 32px",
        }}
      />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(0,0,0,0)_40%,rgba(0,0,0,0.32)_100%)]" />
    </div>
  )
}

function buildPreviewRequest({
  kind,
  mono,
  bilingual,
  fontMappings,
  previewPrimaryText,
  previewSecondaryText,
}: {
  kind: "mono" | "bilingual"
  mono: LibraryMonoStyleDTO | null
  bilingual: LibraryBilingualStyleDTO | null
  fontMappings: LibrarySubtitleStyleFontDTO[]
  previewPrimaryText: string
  previewSecondaryText: string
}): GenerateSubtitleStylePreviewASSBindingRequest {
  if (kind === "bilingual" && bilingual) {
    return {
      type: "bilingual",
      bilingual,
      fontMappings: normalizePreviewFontMappings(fontMappings),
      primaryText: previewPrimaryText,
      secondaryText: previewSecondaryText,
    }
  }

  return {
    type: "mono",
    mono: mono ?? undefined,
    fontMappings: normalizePreviewFontMappings(fontMappings),
    primaryText: previewPrimaryText,
  }
}
