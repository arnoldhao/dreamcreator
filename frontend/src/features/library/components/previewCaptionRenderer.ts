import { renderVTTCueString } from "media-captions"
import type { VTTCue } from "media-captions"
import type { CSSProperties } from "react"

import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"

export type PreviewSize = {
  width: number
  height: number
}

export type PreviewScale = {
  uniform: number
  x: number
  y: number
}

export type PreviewCueKey = "mono" | "primary" | "secondary"

export type RenderedPreviewCue = {
  key: PreviewCueKey
  html: string
  style: AssStyleSpecDTO
}

export const DEFAULT_PREVIEW_SIZE: PreviewSize = { width: 1920, height: 1080 }

const PREVIEW_CUE_ORDER: PreviewCueKey[] = ["mono", "primary", "secondary"]

export function normalizePreviewFontMappings(fontMappings: LibrarySubtitleStyleFontDTO[]) {
  return fontMappings.map((item) => ({
    id: item.id,
    family: item.family,
    source: item.source,
    systemFamily: item.systemFamily,
    enabled: item.enabled !== false,
  }))
}

export function normalizeBaseResolution(width: number, height: number) {
  return {
    width: Math.max(1, width || DEFAULT_PREVIEW_SIZE.width),
    height: Math.max(1, height || DEFAULT_PREVIEW_SIZE.height),
  }
}

export function resolvePreviewScale(baseResolution: PreviewSize, previewSize: PreviewSize): PreviewScale {
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

export function buildRenderedPreviewCues({
  cues,
  kind,
  mono,
  bilingual,
  currentTimeSeconds,
  latestOnlyPerKey = false,
}: {
  cues: VTTCue[]
  kind: "mono" | "bilingual"
  mono: LibraryMonoStyleDTO | null
  bilingual: LibraryBilingualStyleDTO | null
  currentTimeSeconds?: number
  latestOnlyPerKey?: boolean
}) {
  const cueStyles = resolvePreviewCueStyles(kind, mono, bilingual)
  if (!cueStyles) {
    return []
  }

  const filteredCues =
    typeof currentTimeSeconds === "number"
      ? cues.filter((cue) => cue.startTime <= currentTimeSeconds && currentTimeSeconds < cue.endTime)
      : cues

  if (latestOnlyPerKey) {
    const latestByKey = new Map<PreviewCueKey, RenderedPreviewCue>()
    for (const cue of filteredCues) {
      const key = resolveCueKey(cue)
      if (!key) {
        continue
      }
      const style = cueStyles[key]
      if (!style) {
        continue
      }
      latestByKey.set(key, {
        key,
        html: renderVTTCueString(cue, currentTimeSeconds),
        style,
      })
    }
    return PREVIEW_CUE_ORDER.flatMap((key) => {
      const cue = latestByKey.get(key)
      return cue ? [cue] : []
    })
  }

  const rendered: RenderedPreviewCue[] = []
  for (const cue of filteredCues) {
    const key = resolveCueKey(cue)
    if (!key) {
      continue
    }
    const style = cueStyles[key]
    if (!style) {
      continue
    }
    rendered.push({
      key,
      html: renderVTTCueString(cue, currentTimeSeconds),
      style,
    })
  }
  return rendered
}

export function buildCueDisplayStyle(style: AssStyleSpecDTO, scale: PreviewScale): CSSProperties {
  const leftMargin = Math.max(0, (style.marginL || 0) * scale.x)
  const rightMargin = Math.max(0, (style.marginR || 0) * scale.x)
  const verticalMargin = Math.max(0, (style.marginV || 0) * scale.y)
  const verticalAnchor = resolvePreviewVerticalAnchor(style.alignment)

  const displayStyle: CSSProperties = {
    left: formatPreviewLength(leftMargin),
    right: formatPreviewLength(rightMargin),
    minWidth: 0,
    overflow: "visible",
    textAlign: resolvePreviewTextAlign(style.alignment),
  }

  if (verticalAnchor === "top") {
    displayStyle.top = formatPreviewLength(verticalMargin)
  } else if (verticalAnchor === "middle") {
    displayStyle.top = "50%"
    displayStyle.transform = "translateY(-50%)"
  } else {
    displayStyle.bottom = formatPreviewLength(verticalMargin)
  }

  return displayStyle
}

export function buildCueTextStyle(
  style: AssStyleSpecDTO,
  fontMappings: LibrarySubtitleStyleFontDTO[],
  scale: PreviewScale,
): CSSProperties {
  const fontFamily = formatPreviewFontFamily(resolvePreviewFontFamily(style.fontname, fontMappings))
  const fontSize = Math.max(1, (style.fontsize || 0) * scale.uniform)
  const borderStyle = style.borderStyle || 1
  const paddingX = borderStyle === 3 ? Math.max(4, fontSize * 0.28) : 0
  const paddingY = borderStyle === 3 ? Math.max(2, fontSize * 0.16) : 0

  return {
    display: "inline-block",
    maxWidth: "100%",
    whiteSpace: "pre-wrap",
    overflowWrap: "anywhere",
    unicodeBidi: "plaintext",
    fontFamily,
    fontSize: formatPreviewLength(fontSize),
    lineHeight: formatPreviewLength(fontSize * 1.2),
    fontWeight: style.bold ? 700 : 400,
    fontStyle: style.italic ? "italic" : "normal",
    textAlign: "inherit",
    textDecoration: resolveTextDecoration(style),
    letterSpacing: formatPreviewLength((style.spacing || 0) * scale.x),
    color: formatAssColor(style.primaryColour, "rgba(255, 255, 255, 1)"),
    backgroundColor:
      borderStyle === 3 ? formatAssColor(style.backColour, "rgba(0, 0, 0, 0)") : "transparent",
    padding: `${formatPreviewLength(paddingY)} ${formatPreviewLength(paddingX)}`,
    borderRadius: 2,
    textShadow: buildPreviewTextShadow(style, scale.uniform),
    transform: resolvePreviewTransform(style),
    transformOrigin: resolvePreviewTransformOrigin(style.alignment),
  }
}

function resolvePreviewCueStyles(
  kind: "mono" | "bilingual",
  mono: LibraryMonoStyleDTO | null,
  bilingual: LibraryBilingualStyleDTO | null,
) {
  if (kind === "bilingual" && bilingual) {
    const pair = resolveBilingualPreviewCueStylePair(bilingual)
    return {
      primary: pair.primary,
      secondary: pair.secondary,
    } satisfies Partial<Record<PreviewCueKey, AssStyleSpecDTO>>
  }

  if (!mono) {
    return null
  }

  return {
    mono: mono.style,
  } satisfies Partial<Record<PreviewCueKey, AssStyleSpecDTO>>
}

function resolveCueKey(cue: VTTCue): PreviewCueKey | null {
  if (cue.text.includes("<c.mono>")) {
    return "mono"
  }
  if (cue.text.includes("<c.primary>")) {
    return "primary"
  }
  if (cue.text.includes("<c.secondary>")) {
    return "secondary"
  }
  return null
}

function resolveBilingualPreviewCueStylePair(bilingual: LibraryBilingualStyleDTO) {
  const anchor = Number.isFinite(bilingual.layout.blockAnchor)
    ? Math.min(9, Math.max(1, Math.round(bilingual.layout.blockAnchor)))
    : 2
  const gapValue = bilingual.layout.gap
  const gap = Number.isFinite(gapValue) && gapValue >= 0 ? gapValue : 24
  const primary = { ...bilingual.primary.style }
  const secondary = { ...bilingual.secondary.style }
  const primaryOffset = Math.round(gap + (secondary.fontsize || 0))
  const secondaryOffset = Math.round(gap + (primary.fontsize || 0))

  if (anchor >= 4 && anchor <= 6) {
    const topAnchor = anchor + 3
    primary.alignment = topAnchor
    secondary.alignment = topAnchor

    const playResY = bilingual.basePlayResY > 0 ? bilingual.basePlayResY : DEFAULT_PREVIEW_SIZE.height
    const blockHeight = (primary.fontsize || 0) + (secondary.fontsize || 0) + gap
    const baseTop = Math.max(0, Math.round(playResY / 2 - blockHeight / 2))
    primary.marginV = baseTop
    secondary.marginV = baseTop + secondaryOffset
    return { primary, secondary }
  }

  primary.alignment = anchor
  secondary.alignment = anchor
  if (anchor >= 7 && anchor <= 9) {
    secondary.marginV = (secondary.marginV || 0) + secondaryOffset
  } else {
    primary.marginV = (primary.marginV || 0) + primaryOffset
  }
  return { primary, secondary }
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

function resolvePreviewTextAlign(alignment: number): "left" | "center" | "right" {
  if (alignment === 1 || alignment === 4 || alignment === 7) {
    return "left"
  }
  if (alignment === 3 || alignment === 6 || alignment === 9) {
    return "right"
  }
  return "center"
}

function resolvePreviewVerticalAnchor(alignment: number): "top" | "middle" | "bottom" {
  if (alignment === 7 || alignment === 8 || alignment === 9) {
    return "top"
  }
  if (alignment === 4 || alignment === 5 || alignment === 6) {
    return "middle"
  }
  return "bottom"
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
