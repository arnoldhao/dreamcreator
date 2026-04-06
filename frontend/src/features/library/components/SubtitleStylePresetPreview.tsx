import * as React from "react"
import { CaptionsRenderer, parseText } from "media-captions"

import "media-captions/styles/captions.css"
import "media-captions/styles/regions.css"

import type { GenerateSubtitleStylePreviewVTTRequest as GenerateSubtitleStylePreviewVTTBindingRequest } from "../../../../bindings/dreamcreator/internal/application/library/dto"
import { GenerateSubtitleStylePreviewVTT } from "../../../../bindings/dreamcreator/internal/presentation/wails/libraryhandler"
import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"

type SubtitleStylePresetPreviewProps = {
  kind: "mono" | "bilingual"
  mono?: LibraryMonoStyleDTO | null
  bilingual?: LibraryBilingualStyleDTO | null
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  onPreviewSizeChange?: (size: PreviewSize) => void
}

type PreviewSize = {
  width: number
  height: number
}

type PreviewScale = {
  uniform: number
  x: number
  y: number
}

const PREVIEW_REQUEST_DEBOUNCE_MS = 80
const DEFAULT_PREVIEW_SIZE: PreviewSize = { width: 1920, height: 1080 }
const PREVIEW_CAPTIONS_CLASS = "dc-subtitle-style-preview__captions"
const PREVIEW_SURFACE_CLASS = "dc-subtitle-style-preview__surface"
const PREVIEW_TIME_SECONDS = 1

export function SubtitleStylePresetPreview({
  kind,
  mono,
  bilingual,
  fontMappings = [],
  onPreviewSizeChange,
}: SubtitleStylePresetPreviewProps) {
  const frameRef = React.useRef<HTMLDivElement | null>(null)
  const overlayRef = React.useRef<HTMLDivElement | null>(null)
  const rendererRef = React.useRef<CaptionsRenderer | null>(null)
  const requestVersionRef = React.useRef(0)
  const [previewSize, setPreviewSize] = React.useState<PreviewSize>(DEFAULT_PREVIEW_SIZE)
  const [trackContent, setTrackContent] = React.useState("")

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
        previewSize,
      }),
    [bilingual, fontMappings, kind, mono, previewSize],
  )
  const previewRequestKey = React.useMemo(() => JSON.stringify(previewRequest), [previewRequest])

  const previewStylesheet = React.useMemo(
    () =>
      buildPreviewStylesheet({
        kind,
        mono: mono ?? null,
        bilingual: bilingual ?? null,
        fontMappings,
        baseResolution,
        previewSize,
      }),
    [baseResolution, bilingual, fontMappings, kind, mono, previewSize],
  )

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
    const overlay = overlayRef.current
    if (!overlay) {
      return
    }

    const renderer = new CaptionsRenderer(overlay, { dir: "ltr" })
    rendererRef.current = renderer
    renderer.currentTime = PREVIEW_TIME_SECONDS

    return () => {
      rendererRef.current = null
      renderer.destroy()
    }
  }, [])

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
        .catch((error) => {
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
    const renderer = rendererRef.current
    if (!renderer) {
      return
    }

    let disposed = false

    if (!trackContent.trim()) {
      renderer.reset()
      return
    }

    void parseText(trackContent, { type: "vtt" })
      .then((result) => {
        if (disposed || rendererRef.current !== renderer) {
          return
        }
        renderer.changeTrack(result)
        renderer.currentTime = PREVIEW_TIME_SECONDS
        renderer.update(true)
      })
      .catch(() => {
        if (disposed) {
          return
        }
        renderer.reset()
      })

    return () => {
      disposed = true
    }
  }, [trackContent])

  return (
    <div className="overflow-hidden rounded-xl border border-border/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(241,245,249,0.98))] shadow-[0_24px_72px_-42px_rgba(15,23,42,0.28)]">
      {previewStylesheet ? <style>{previewStylesheet}</style> : null}

      <div
        ref={frameRef}
        className={`relative w-full overflow-hidden ${PREVIEW_SURFACE_CLASS}`}
        style={{ aspectRatio: `${baseResolution.width} / ${baseResolution.height}` }}
      >
        <PreviewBackdrop />
        <div ref={overlayRef} className={`absolute inset-0 ${PREVIEW_CAPTIONS_CLASS}`} />
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
      <div className="absolute inset-0 opacity-[0.16]" style={{ backgroundImage: "linear-gradient(rgba(148,163,184,0.45) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.45) 1px, transparent 1px)", backgroundSize: "24px 24px" }} />
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
      previewWidth: previewSize.width,
      previewHeight: previewSize.height,
    }
  }

  return {
    type: "mono",
    mono: mono ?? undefined,
    fontMappings: normalizePreviewFontMappings(fontMappings),
    previewWidth: previewSize.width,
    previewHeight: previewSize.height,
  }
}

function buildPreviewStylesheet({
  kind,
  mono,
  bilingual,
  fontMappings,
  baseResolution,
  previewSize,
}: {
  kind: "mono" | "bilingual"
  mono: LibraryMonoStyleDTO | null
  bilingual: LibraryBilingualStyleDTO | null
  fontMappings: LibrarySubtitleStyleFontDTO[]
  baseResolution: PreviewSize
  previewSize: PreviewSize
}) {
  const scale = resolvePreviewScale(baseResolution, previewSize)
  const selectors: string[] = [
    `.${PREVIEW_CAPTIONS_CLASS}[data-part='captions'] { --overlay-padding: 0; }`,
    `.${PREVIEW_CAPTIONS_CLASS} > [data-part='cue-display'] { background: transparent !important; }`,
    `.${PREVIEW_CAPTIONS_CLASS} [data-part='cue'] { background: transparent !important; color: inherit !important; padding: 0 !important; border: 0 !important; border-radius: 0 !important; box-shadow: none !important; outline: none !important; text-shadow: none !important; }`,
  ]

  if (kind === "bilingual" && bilingual) {
    selectors.push(buildCueClassRule("primary", bilingual.primary.style, fontMappings, scale))
    selectors.push(buildCueClassRule("secondary", bilingual.secondary.style, fontMappings, scale))
  } else if (mono) {
    selectors.push(buildCueClassRule("mono", mono.style, fontMappings, scale))
  }

  return selectors.join("\n")
}

function buildCueClassRule(
  className: "mono" | "primary" | "secondary",
  style: AssStyleSpecDTO,
  fontMappings: LibrarySubtitleStyleFontDTO[],
  scale: PreviewScale,
) {
  const fontFamily = formatPreviewFontFamily(resolvePreviewFontFamily(style.fontname, fontMappings))
  const fontSize = Math.max(1, (style.fontsize || 0) * scale.uniform)
  const borderStyle = style.borderStyle || 1
  const paddingX = borderStyle === 3 ? Math.max(4, fontSize * 0.28) : 0
  const paddingY = borderStyle === 3 ? Math.max(2, fontSize * 0.16) : 0
  const textDecoration = resolveTextDecoration(style)
  const textShadow = buildPreviewTextShadow(style, scale.uniform)
  const transform = resolvePreviewTransform(style)
  const transformOrigin = resolvePreviewTransformOrigin(style.alignment)

  return [
    `.${PREVIEW_CAPTIONS_CLASS} .${className} {`,
    "  display: inline-block;",
    "  white-space: pre-wrap;",
    `  font-family: ${fontFamily};`,
    `  font-size: ${formatPreviewLength(fontSize)};`,
    `  line-height: ${formatPreviewLength(fontSize * 1.2)};`,
    `  font-weight: ${style.bold ? "700" : "400"};`,
    `  font-style: ${style.italic ? "italic" : "normal"};`,
    `  text-decoration: ${textDecoration};`,
    `  letter-spacing: ${formatPreviewLength((style.spacing || 0) * scale.x)};`,
    `  color: ${formatAssColor(style.primaryColour, "rgba(255, 255, 255, 1)")};`,
    `  background-color: ${borderStyle === 3 ? formatAssColor(style.backColour, "rgba(0, 0, 0, 0)") : "transparent"};`,
    `  padding: ${formatPreviewLength(paddingY)} ${formatPreviewLength(paddingX)};`,
    "  border-radius: 2px;",
    `  text-shadow: ${textShadow};`,
    `  transform: ${transform};`,
    `  transform-origin: ${transformOrigin};`,
    "}",
  ].join("\n")
}

function normalizePreviewFontMappings(fontMappings: LibrarySubtitleStyleFontDTO[]) {
  return fontMappings.map((item) => ({
    id: item.id,
    family: item.family,
    source: item.source,
    systemFamily: item.systemFamily,
    enabled: item.enabled !== false,
  }))
}

function normalizeBaseResolution(width: number, height: number) {
  return {
    width: Math.max(1, width || DEFAULT_PREVIEW_SIZE.width),
    height: Math.max(1, height || DEFAULT_PREVIEW_SIZE.height),
  }
}

function resolvePreviewScale(baseResolution: PreviewSize, previewSize: PreviewSize): PreviewScale {
  if (baseResolution.width <= 0 || baseResolution.height <= 0 || previewSize.width <= 0 || previewSize.height <= 0) {
    return { uniform: 1, x: 1, y: 1 }
  }

  const x = previewSize.width / Math.max(1, baseResolution.width)
  const y = previewSize.height / Math.max(1, baseResolution.height)
  return {
    uniform: Math.max(0.01, Math.min(x, y)),
    x: Math.max(0.01, x),
    y: Math.max(0.01, y),
  }
}

function resolvePreviewFontFamily(fontName: string, mappings: LibrarySubtitleStyleFontDTO[]) {
  const normalized = normalizePreviewFontFamilyKey(fontName)
  for (const item of mappings) {
    if (item.enabled === false) {
      continue
    }
    if (normalizePreviewFontFamilyKey(item.family) !== normalized) {
      continue
    }
    const systemFamily = item.systemFamily?.trim()
    if (systemFamily) {
      return systemFamily
    }
  }
  return fontName.trim() || "sans-serif"
}

function normalizePreviewFontFamilyKey(value: string) {
  return value.trim().toLowerCase().replace(/_/gu, "").replace(/-/gu, "")
}

function formatPreviewFontFamily(value: string) {
  const safe = value.trim()
  if (!safe) {
    return "sans-serif"
  }
  return `"${safe.replace(/"/gu, String.raw`\"`)}", sans-serif`
}

function resolveTextDecoration(style: AssStyleSpecDTO) {
  const values: string[] = []
  if (style.underline) {
    values.push("underline")
  }
  if (style.strikeOut) {
    values.push("line-through")
  }
  return values.length > 0 ? values.join(" ") : "none"
}

function resolvePreviewTransform(style: AssStyleSpecDTO) {
  const scaleX = (style.scaleX || 100) / 100
  const scaleY = (style.scaleY || 100) / 100
  const angle = style.angle || 0
  const parts = [`scale(${formatPreviewNumber(scaleX)}, ${formatPreviewNumber(scaleY)})`]
  if (angle !== 0) {
    parts.push(`rotate(${formatPreviewNumber(angle)}deg)`)
  }
  return parts.join(" ")
}

function resolvePreviewTransformOrigin(alignment: number) {
  const horizontal =
    alignment === 1 || alignment === 4 || alignment === 7
      ? "left"
      : alignment === 3 || alignment === 6 || alignment === 9
        ? "right"
        : "center"
  const vertical =
    alignment === 7 || alignment === 8 || alignment === 9
      ? "top"
      : alignment === 4 || alignment === 5 || alignment === 6
        ? "center"
        : "bottom"
  return `${horizontal} ${vertical}`
}

function formatAssColor(value: string, fallback: string) {
  const color = parseAssColor(value)
  if (!color) {
    return fallback
  }
  return `rgba(${color.red}, ${color.green}, ${color.blue}, ${formatPreviewNumber(color.alpha / 255)})`
}

function parseAssColor(value: string) {
  const normalized = value.trim().replace(/&$/u, "").replace(/^&h/iu, "")
  if (!normalized || (normalized.length !== 6 && normalized.length !== 8)) {
    return null
  }
  const parsed = Number.parseInt(normalized, 16)
  if (!Number.isFinite(parsed)) {
    return null
  }
  if (normalized.length === 6) {
    return {
      red: parsed & 0xff,
      green: (parsed >> 8) & 0xff,
      blue: (parsed >> 16) & 0xff,
      alpha: 0xff,
    }
  }
  const assAlpha = (parsed >> 24) & 0xff
  return {
    red: parsed & 0xff,
    green: (parsed >> 8) & 0xff,
    blue: (parsed >> 16) & 0xff,
    alpha: 0xff - assAlpha,
  }
}

function buildPreviewTextShadow(style: AssStyleSpecDTO, scale: number) {
  const outline = Math.max(0, (style.outline || 0) * scale)
  const shadow = Math.max(0, (style.shadow || 0) * scale)
  const outlineColor = formatAssColor(style.outlineColour, "rgba(0, 0, 0, 0.85)")
  const shadowColor = formatAssColor(style.backColour, "rgba(0, 0, 0, 0.45)")
  const layers: string[] = []

  if (outline > 0) {
    const offsets: Array<[number, number]> = [
      [-outline, 0],
      [outline, 0],
      [0, -outline],
      [0, outline],
      [-outline, -outline],
      [outline, -outline],
      [-outline, outline],
      [outline, outline],
    ]
    for (const [x, y] of offsets) {
      layers.push(`${formatPreviewNumber(x)}px ${formatPreviewNumber(y)}px 0 ${outlineColor}`)
    }
  }

  if (shadow > 0) {
    const blur = Math.max(1, shadow * 1.6)
    layers.push(
      `${formatPreviewNumber(shadow)}px ${formatPreviewNumber(shadow)}px ${formatPreviewNumber(blur)}px ${shadowColor}`,
    )
  }

  return layers.length > 0 ? layers.join(", ") : "none"
}

function formatPreviewLength(value: number) {
  return `${formatPreviewNumber(value)}px`
}

function formatPreviewNumber(value: number) {
  if (!Number.isFinite(value)) {
    return "0"
  }
  return value.toFixed(2).replace(/\.?0+$/u, "")
}
