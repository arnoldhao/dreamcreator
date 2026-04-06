import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"

type PreviewSize = {
  width: number
  height: number
}

type PreviewScale = {
  uniform: number
  x: number
  y: number
}

export function buildWorkspacePreviewStylesheet({
  rootClassName,
  displayMode,
  monoStyle,
  lingualStyle,
  fontMappings,
  previewSize,
}: {
  rootClassName: string
  displayMode: "single" | "dual"
  monoStyle: LibraryMonoStyleDTO | null
  lingualStyle: LibraryBilingualStyleDTO | null
  fontMappings: LibrarySubtitleStyleFontDTO[]
  previewSize: PreviewSize
}) {
  const selectors: string[] = [
    `.${rootClassName}[data-part='captions'] { --overlay-padding: 0; }`,
    `.${rootClassName} > [data-part='cue-display'] { background: transparent !important; }`,
    `.${rootClassName} [data-part='cue'] { background: transparent !important; color: inherit !important; padding: 0 !important; border: 0 !important; border-radius: 0 !important; box-shadow: none !important; outline: none !important; text-shadow: none !important; }`,
  ]

  if (displayMode === "dual" && lingualStyle) {
    const baseResolution = normalizeBaseResolution(lingualStyle.basePlayResX, lingualStyle.basePlayResY)
    const scale = resolvePreviewScale(baseResolution, previewSize)
    selectors.push(buildCueClassRule(rootClassName, "primary", lingualStyle.primary.style, fontMappings, scale))
    selectors.push(buildCueClassRule(rootClassName, "secondary", lingualStyle.secondary.style, fontMappings, scale))
    return selectors.join("\n")
  }

  if (monoStyle) {
    const baseResolution = normalizeBaseResolution(monoStyle.basePlayResX, monoStyle.basePlayResY)
    const scale = resolvePreviewScale(baseResolution, previewSize)
    selectors.push(buildCueClassRule(rootClassName, "mono", monoStyle.style, fontMappings, scale))
  }

  return selectors.join("\n")
}

function buildCueClassRule(
  rootClassName: string,
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
  const textAlign = resolvePreviewTextAlign(style.alignment)

  return [
    `.${rootClassName} .${className} {`,
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
    `  text-align: ${textAlign};`,
    `  transform: ${transform};`,
    `  transform-origin: ${transformOrigin};`,
    "}",
  ].join("\n")
}

function normalizeBaseResolution(width: number, height: number) {
  return {
    width: Math.max(1, width || 1920),
    height: Math.max(1, height || 1080),
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

function resolvePreviewTextAlign(alignment: number) {
  if (alignment === 1 || alignment === 4 || alignment === 7) {
    return "left"
  }
  if (alignment === 3 || alignment === 6 || alignment === 9) {
    return "right"
  }
  return "center"
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
