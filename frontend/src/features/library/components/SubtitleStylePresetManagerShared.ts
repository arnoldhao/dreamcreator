import {
  resolveAssStyleFontFace,
  resolveAssStyleFontItalic,
  resolveAssStyleFontWeight,
} from "@/shared/fonts/fontCatalog"
import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
  ParseSubtitleStyleImportResult,
} from "@/shared/contracts/library"

import {
  SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS,
  cloneAssStyleSpec,
  cloneBilingualStyle,
  cloneMonoStyle,
  createMonoSnapshotFromStyle,
  type SubtitleStylePresetKind,
  type SubtitleStylePresetSelection,
} from "../utils/subtitleStylePresets"

export type LeftPaneView = "preview" | "all" | "create"
export type CreatePaneMode = "form" | "import"

export type ImportDraftState =
  | {
      kind: "ass"
      fileName: string
      detectedRatio?: string
      normalizedPlayResX?: number
      normalizedPlayResY?: number
      styles: LibraryMonoStyleDTO[]
    }
  | {
      kind: "dcssp"
      fileName: string
      result: ParseSubtitleStyleImportResult
    }
  | null

export type CreateStyleDraft = {
  kind: SubtitleStylePresetKind
  name: string
  aspectRatio: string
  monoTemplateID: string
  bilingualPrimaryTemplateID: string
  bilingualSecondaryTemplateID: string
}

export function signatureOf(value: unknown) {
  return JSON.stringify(value)
}

export function resolveSelectionItem(
  selection: SubtitleStylePresetSelection,
  monoStyles: LibraryMonoStyleDTO[],
  bilingualStyles: LibraryBilingualStyleDTO[],
) {
  if (!selection) {
    return null
  }
  if (selection.kind === "mono") {
    return monoStyles.find((item) => item.id === selection.id) ?? null
  }
  return bilingualStyles.find((item) => item.id === selection.id) ?? null
}

export function isDefaultMonoStyle(
  defaults: { monoStyleId?: string },
  style: LibraryMonoStyleDTO,
) {
  return (defaults.monoStyleId ?? "").trim() === style.id
}

export function isDefaultBilingualStyle(
  defaults: { bilingualStyleId?: string },
  style: LibraryBilingualStyleDTO,
) {
  return (defaults.bilingualStyleId ?? "").trim() === style.id
}

export function createInitialCreateStyleDraft(monoStyles: LibraryMonoStyleDTO[]): CreateStyleDraft {
  const firstMono = monoStyles[0]?.id ?? ""
  const secondMono = monoStyles[1]?.id ?? firstMono
  return {
    kind: "mono",
    name: "",
    aspectRatio: "16:9",
    monoTemplateID: firstMono,
    bilingualPrimaryTemplateID: firstMono,
    bilingualSecondaryTemplateID: secondMono,
  }
}

export function normalizeCreateStyleDraft(current: CreateStyleDraft, monoStyles: LibraryMonoStyleDTO[]): CreateStyleDraft {
  const next = { ...current }
  const monoIDs = new Set(monoStyles.map((item) => item.id))
  const firstMono = monoStyles[0]?.id ?? ""
  const secondMono = monoStyles[1]?.id ?? firstMono

  if (!monoIDs.has(next.monoTemplateID)) {
    next.monoTemplateID = firstMono
  }
  if (!monoIDs.has(next.bilingualPrimaryTemplateID)) {
    next.bilingualPrimaryTemplateID = firstMono
  }
  if (!monoIDs.has(next.bilingualSecondaryTemplateID)) {
    next.bilingualSecondaryTemplateID = secondMono
  }

  if (!SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS.some((option) => option.value === next.aspectRatio)) {
    next.aspectRatio = "16:9"
  }

  return next
}

export function resolveStyleForRendering(
  kind: SubtitleStylePresetKind,
  item: LibraryMonoStyleDTO | LibraryBilingualStyleDTO,
): AssStyleSpecDTO {
  if (kind === "mono") {
    return (item as LibraryMonoStyleDTO).style
  }
  return (item as LibraryBilingualStyleDTO).primary.style
}

export function normalizeAssStyleForEditor(style: AssStyleSpecDTO): AssStyleSpecDTO {
  const fontWeight = resolveAssStyleFontWeight(style)
  return {
    ...style,
    fontname: style.fontname?.trim() || "Arial",
    fontFace: resolveAssStyleFontFace(style),
    fontWeight,
    fontPostScriptName: style.fontPostScriptName?.trim() || "",
    bold: fontWeight >= 700,
    italic: resolveAssStyleFontItalic(style),
    fontsize: Math.max(1, Math.round(style.fontsize || 0)),
    marginL: Math.round(style.marginL || 0),
    marginR: Math.round(style.marginR || 0),
    marginV: Math.round(style.marginV || 0),
    encoding: Math.round(style.encoding || 0),
  }
}

export function normalizeMonoStyleForEditor(style: LibraryMonoStyleDTO): LibraryMonoStyleDTO {
  return {
    ...style,
    style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.style)),
  }
}

export function normalizeBilingualStyleForEditor(style: LibraryBilingualStyleDTO): LibraryBilingualStyleDTO {
  return {
    ...style,
    primary: {
      ...style.primary,
      style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.primary.style)),
    },
    secondary: {
      ...style.secondary,
      style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.secondary.style)),
    },
    layout: {
      ...style.layout,
      gap: Number.isFinite(style.layout.gap) ? style.layout.gap : 0,
      blockAnchor: Math.round(style.layout.blockAnchor || 2),
    },
  }
}

export function normalizeSnapshotBase(
  snapshot: ReturnType<typeof createMonoSnapshotFromStyle>,
  aspectRatio: string,
  basePlayResX: number,
  basePlayResY: number,
) {
  return {
    ...snapshot,
    baseAspectRatio: aspectRatio,
    basePlayResX,
    basePlayResY,
  }
}

export function normalizeImportResult(result: ParseSubtitleStyleImportResult): ParseSubtitleStyleImportResult {
  const next: ParseSubtitleStyleImportResult = {
    ...result,
  }

  if (result.monoStyles) {
    next.monoStyles = result.monoStyles.map((style) => normalizeMonoStyleForEditor(cloneMonoStyle(style)))
  }

  if (result.bilingualStyle) {
    next.bilingualStyle = normalizeBilingualStyleForEditor(cloneBilingualStyle(result.bilingualStyle))
  }

  return next
}

export function formatStyleFlags(style: AssStyleSpecDTO, t: (key: string) => string) {
  const flags: string[] = []
  if (style.bold) {
    flags.push("B")
  }
  if (style.italic) {
    flags.push("I")
  }
  if (style.underline) {
    flags.push("U")
  }
  if (style.strikeOut) {
    flags.push("S")
  }
  return flags.length > 0 ? flags.join(" / ") : t("library.config.subtitleStyles.styleFlagsNormal")
}

export function resolveAlignmentOptions(t: (key: string) => string) {
  return [
    { value: 1, label: t("library.config.subtitleStyles.alignmentBottomLeft") },
    { value: 2, label: t("library.config.subtitleStyles.alignmentBottomCenter") },
    { value: 3, label: t("library.config.subtitleStyles.alignmentBottomRight") },
    { value: 4, label: t("library.config.subtitleStyles.alignmentMiddleLeft") },
    { value: 5, label: t("library.config.subtitleStyles.alignmentMiddleCenter") },
    { value: 6, label: t("library.config.subtitleStyles.alignmentMiddleRight") },
    { value: 7, label: t("library.config.subtitleStyles.alignmentTopLeft") },
    { value: 8, label: t("library.config.subtitleStyles.alignmentTopCenter") },
    { value: 9, label: t("library.config.subtitleStyles.alignmentTopRight") },
  ]
}

export function resolveBorderStyleOptions(t: (key: string) => string) {
  return [
    { value: 1, label: t("library.config.subtitleStyles.borderStyleOutline") },
    { value: 3, label: t("library.config.subtitleStyles.borderStyleBox") },
  ]
}

export function parseAssColor(value: string) {
  const normalized = value.trim().replace(/^&?H/i, "").replace(/[^0-9a-f]/gi, "").toUpperCase()
  if (normalized.length !== 6 && normalized.length !== 8) {
    return null
  }

  const hex = normalized.length === 6 ? `00${normalized}` : normalized
  const alpha = hex.slice(0, 2)
  const blue = hex.slice(2, 4)
  const green = hex.slice(4, 6)
  const red = hex.slice(6, 8)

  return {
    alpha,
    rgb: `#${red}${green}${blue}`.toLowerCase(),
  }
}

export function formatAssColorWithRgb(rgb: string, currentValue: string) {
  const parsed = parseAssColor(currentValue)
  const alpha = parsed?.alpha ?? "00"
  const normalized = rgb.trim().replace(/^#/, "").replace(/[^0-9a-f]/gi, "").toUpperCase()
  if (normalized.length !== 6) {
    return currentValue
  }

  const red = normalized.slice(0, 2)
  const green = normalized.slice(2, 4)
  const blue = normalized.slice(4, 6)
  return `&H${alpha}${blue}${green}${red}`
}
