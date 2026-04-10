import type { CSSProperties } from "react"

import type {
  AssStyleSpecDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"
import {
  resolveAssStyleFontItalic,
  resolveAssStyleFontWeight,
} from "@/shared/fonts/fontCatalog"

import {
  formatPreviewFontFamily,
  resolvePreviewFontFamily,
} from "../components/previewCaptionRenderer"

const SUBTITLE_STYLE_PREVIEW_BASE_ASS_SIZE = 48
const SUBTITLE_STYLE_PREVIEW_BASE_PX = 15
const SUBTITLE_STYLE_PREVIEW_MIN_PX = 11
const SUBTITLE_STYLE_PREVIEW_MAX_PX = 22
const SUBTITLE_STYLE_PREVIEW_LINE_HEIGHT = 1.16

export function buildSubtitleStyleNamePreviewStyle(
  style: AssStyleSpecDTO | null | undefined,
  fontMappings: LibrarySubtitleStyleFontDTO[],
): CSSProperties {
  if (!style) {
    return {}
  }

  const decoration = [style.underline ? "underline" : "", style.strikeOut ? "line-through" : ""]
    .filter(Boolean)
    .join(" ")
  const previewFontSize = resolveSubtitleStylePreviewSize(style.fontsize)

  return {
    fontFamily: formatPreviewFontFamily(resolvePreviewFontFamily(style.fontname, fontMappings)),
    fontSize: previewFontSize,
    lineHeight: `${(previewFontSize * SUBTITLE_STYLE_PREVIEW_LINE_HEIGHT).toFixed(2)}px`,
    fontWeight: resolveAssStyleFontWeight(style),
    fontStyle: resolveAssStyleFontItalic(style) ? "italic" : "normal",
    textDecoration: decoration || "none",
  }
}

function resolveSubtitleStylePreviewSize(fontSize: number) {
  const normalizedFontSize = Number.isFinite(fontSize)
    ? Math.max(1, fontSize)
    : SUBTITLE_STYLE_PREVIEW_BASE_ASS_SIZE
  const scaledFontSize =
    (normalizedFontSize / SUBTITLE_STYLE_PREVIEW_BASE_ASS_SIZE) * SUBTITLE_STYLE_PREVIEW_BASE_PX

  return Math.min(
    SUBTITLE_STYLE_PREVIEW_MAX_PX,
    Math.max(SUBTITLE_STYLE_PREVIEW_MIN_PX, Number(scaledFontSize.toFixed(2))),
  )
}
