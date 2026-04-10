import type {
  LibrarySubtitleStyleDocumentAnalysisDTO,
  LibrarySubtitleStyleDocumentDTO,
} from "@/shared/contracts/library"
import { t } from "@/shared/i18n"

export type SubtitleStyleCueLike = {
  start: string
  end: string
  startMs: number
  endMs: number
  sourceText: string
  translationText?: string
}

export type AssDocumentSummary = {
  detectedFormat: string
  scriptType: string
  playResX: number
  playResY: number
  styleCount: number
  styleNames: string[]
  dialogueCount: number
  commentCount: number
  fonts: string[]
  featureFlags: string[]
  validationIssues: string[]
}

type BuildAssSubtitleContentOptions = {
  rows: SubtitleStyleCueLike[]
  displayMode: "mono" | "bilingual"
  document: LibrarySubtitleStyleDocumentDTO | null
  title?: string
}

type ExtractedAssDocumentLayout = {
  lines: string[]
  eventFormat: string
  styleNames: string[]
}

export const DEFAULT_ASS_DOCUMENT_CONTENT = [
  "[Script Info]",
  "Title: Custom subtitle style",
  "ScriptType: v4.00+",
  "WrapStyle: 0",
  "ScaledBorderAndShadow: yes",
  "PlayResX: 1920",
  "PlayResY: 1080",
  "",
  "[V4+ Styles]",
  "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
  "Style: Default,Arial,48,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1",
  "",
  "[Events]",
  "Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
  "",
].join("\n")

function pad(value: number, size = 2) {
  return String(value).padStart(size, "0")
}

function normalizeAssDocumentContent(content: string | undefined) {
  const normalized = (content ?? "").replace(/\r\n?/g, "\n").trim()
  return normalized ? `${normalized}\n` : ""
}

export function resolveAssDocumentSummary(document: LibrarySubtitleStyleDocumentDTO): AssDocumentSummary {
  if (document.analysis) {
    return normalizeAssDocumentAnalysis(document.analysis)
  }
  return parseAssDocumentSummary(document.content)
}

export function parseAssDocumentSummary(content: string | undefined): AssDocumentSummary {
  const safeContent = normalizeAssDocumentContent(content)
  const lines = safeContent.split("\n")
  let detectedFormat = "ass"
  let scriptType = ""
  let playResX = 0
  let playResY = 0
  let inStyles = false
  let styleCount = 0
  let dialogueCount = 0
  let commentCount = 0
  const styleNames = new Set<string>()
  const fonts = new Set<string>()

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) {
      continue
    }
    if (line.startsWith("[")) {
      const normalizedSection = line.toLowerCase()
      if (normalizedSection === "[v4 styles]") {
        detectedFormat = "ssa"
      }
      inStyles = normalizedSection === "[v4+ styles]" || normalizedSection === "[v4 styles]"
      continue
    }
    if (/^ScriptType\s*:/i.test(line)) {
      scriptType = line.split(":").slice(1).join(":").trim()
      if (scriptType.toLowerCase() === "v4.00") {
        detectedFormat = "ssa"
      }
      continue
    }
    if (/^PlayResX\s*:/i.test(line)) {
      playResX = Number.parseInt(line.split(":").slice(1).join(":").trim(), 10) || 0
      continue
    }
    if (/^PlayResY\s*:/i.test(line)) {
      playResY = Number.parseInt(line.split(":").slice(1).join(":").trim(), 10) || 0
      continue
    }
    if (inStyles && /^Style\s*:/i.test(line)) {
      styleCount += 1
      const segments = line.split(",")
      const styleName = segments[0]?.split(":").slice(1).join(":").trim()
      if (styleName) {
        styleNames.add(styleName)
      }
      if (segments.length >= 2) {
        const fontName = segments[1]?.trim()
        if (fontName) {
          fonts.add(fontName)
        }
      }
      continue
    }
    if (/^Dialogue\s*:/i.test(line)) {
      dialogueCount += 1
      continue
    }
    if (/^Comment\s*:/i.test(line)) {
      commentCount += 1
    }
    for (const match of line.matchAll(/\\fn([^\\}]+)/g)) {
      const fontName = match[1]?.trim()
      if (fontName) {
        fonts.add(fontName)
      }
    }
  }

  return {
    detectedFormat,
    scriptType,
    playResX,
    playResY,
    styleCount,
    styleNames: [...styleNames],
    dialogueCount,
    commentCount,
    fonts: [...fonts],
    featureFlags: [],
    validationIssues: [],
  }
}

function normalizeAssDocumentAnalysis(analysis: LibrarySubtitleStyleDocumentAnalysisDTO): AssDocumentSummary {
  return {
    detectedFormat: analysis.detectedFormat?.trim().toLowerCase() === "ssa" ? "ssa" : "ass",
    scriptType: analysis.scriptType?.trim() ?? "",
    playResX: analysis.playResX ?? 0,
    playResY: analysis.playResY ?? 0,
    styleCount: analysis.styleCount ?? analysis.styleNames?.length ?? 0,
    styleNames: analysis.styleNames ?? [],
    dialogueCount: analysis.dialogueCount ?? 0,
    commentCount: analysis.commentCount ?? 0,
    fonts: analysis.fonts ?? [],
    featureFlags: analysis.featureFlags ?? [],
    validationIssues: analysis.validationIssues ?? [],
  }
}

export function formatSubtitleStyleDocumentFeatureFlag(flag: string) {
  switch (flag.trim().toLowerCase()) {
    case "override-tags":
      return t("library.config.subtitleStyles.featureOverrideTags")
    case "font-override":
      return t("library.config.subtitleStyles.featureFontOverride")
    case "positioning":
      return t("library.config.subtitleStyles.featurePositioning")
    case "transform":
      return t("library.config.subtitleStyles.featureTransform")
    case "karaoke":
      return t("library.config.subtitleStyles.featureKaraoke")
    case "vector-drawing":
      return t("library.config.subtitleStyles.featureVectorDrawing")
    case "clipping":
      return t("library.config.subtitleStyles.featureClipping")
    case "fade":
      return t("library.config.subtitleStyles.featureFade")
    default:
      return flag
        .split("-")
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(" ")
  }
}

function formatAssTime(milliseconds: number) {
  const safeMs = Math.max(0, Math.floor(milliseconds))
  const hours = Math.floor(safeMs / 3_600_000)
  const minutes = Math.floor((safeMs % 3_600_000) / 60_000)
  const seconds = Math.floor((safeMs % 60_000) / 1_000)
  const centiseconds = Math.floor((safeMs % 1_000) / 10)
  return `${hours}:${pad(minutes)}:${pad(seconds)}.${pad(centiseconds)}`
}

function escapeAssText(value: string) {
  return value
    .replace(/\\/g, "\\\\")
    .replace(/\{/g, "\\{")
    .replace(/\}/g, "\\}")
    .split(/\r?\n/)
    .map((line) => line.trimEnd())
    .join("\\N")
}

function extractStyleName(line: string) {
  if (!/^Style\s*:/i.test(line.trim())) {
    return ""
  }
  const payload = line.split(":").slice(1).join(":")
  return payload.split(",")[0]?.trim() ?? ""
}

function normalizeScriptInfoHeaders(lines: string[]) {
  const normalizedLines: string[] = []
  const aggregatedScriptInfoLines: string[] = []
  let firstScriptInfoInsertIndex = -1
  let index = 0

  while (index < lines.length) {
    const line = lines[index] ?? ""
    if (line.trim().toLowerCase() !== "[script info]") {
      normalizedLines.push(line)
      index += 1
      continue
    }

    if (firstScriptInfoInsertIndex < 0) {
      firstScriptInfoInsertIndex = normalizedLines.length
      normalizedLines.push("[Script Info]")
    }
    index += 1

    while (index < lines.length) {
      const candidate = lines[index] ?? ""
      const trimmedCandidate = candidate.trim()
      if (trimmedCandidate.startsWith("[") && trimmedCandidate.endsWith("]")) {
        break
      }
      aggregatedScriptInfoLines.push(candidate)
      index += 1
    }
  }

  if (firstScriptInfoInsertIndex < 0) {
    return normalizedLines
  }

  const deduplicatedScriptInfoLines = normalizeDuplicateScriptInfoLines(aggregatedScriptInfoLines)
  normalizedLines.splice(firstScriptInfoInsertIndex + 1, 0, ...deduplicatedScriptInfoLines)

  return normalizedLines
}

function normalizeDuplicateScriptInfoLines(scriptInfoLines: string[]) {
  const normalizedLines: string[] = []
  const lastHeaderLineIndex = new Map<string, number>()

  for (let scriptInfoIndex = 0; scriptInfoIndex < scriptInfoLines.length; scriptInfoIndex += 1) {
    const scriptInfoLine = scriptInfoLines[scriptInfoIndex] ?? ""
    const trimmedScriptInfoLine = scriptInfoLine.trim()
    if (!trimmedScriptInfoLine || trimmedScriptInfoLine.startsWith(";")) {
      continue
    }
    const delimiterIndex = trimmedScriptInfoLine.indexOf(":")
    if (delimiterIndex <= 0) {
      continue
    }
    const headerName = trimmedScriptInfoLine.slice(0, delimiterIndex).trim().toLowerCase()
    if (!headerName) {
      continue
    }
    lastHeaderLineIndex.set(headerName, scriptInfoIndex)
  }

  for (let scriptInfoIndex = 0; scriptInfoIndex < scriptInfoLines.length; scriptInfoIndex += 1) {
    const scriptInfoLine = scriptInfoLines[scriptInfoIndex] ?? ""
    const trimmedScriptInfoLine = scriptInfoLine.trim()
    if (!trimmedScriptInfoLine || trimmedScriptInfoLine.startsWith(";")) {
      normalizedLines.push(scriptInfoLine)
      continue
    }
    const delimiterIndex = trimmedScriptInfoLine.indexOf(":")
    if (delimiterIndex <= 0) {
      normalizedLines.push(scriptInfoLine)
      continue
    }
    const headerName = trimmedScriptInfoLine.slice(0, delimiterIndex).trim().toLowerCase()
    if (lastHeaderLineIndex.get(headerName) !== scriptInfoIndex) {
      continue
    }
    normalizedLines.push(scriptInfoLine)
  }

  return normalizedLines
}

function extractAssDocumentLayout(document: LibrarySubtitleStyleDocumentDTO | null, titleOverride?: string): ExtractedAssDocumentLayout {
  const safeContent = normalizeAssDocumentContent(document?.content) || DEFAULT_ASS_DOCUMENT_CONTENT
  const lines = safeContent.split("\n")
  const preservedSections: string[] = []
  const styleNames: string[] = []
  let eventFormat = "Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text"
  let currentSection = ""
  let hasScriptInfo = false
  let hasStyles = false
  let injectedTitle = false

  for (const rawLine of lines) {
    const trimmed = rawLine.trim()
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      currentSection = trimmed.toLowerCase()
      if (currentSection !== "[events]") {
        if (preservedSections.length > 0 && preservedSections[preservedSections.length - 1] !== "") {
          preservedSections.push("")
        }
        preservedSections.push(trimmed)
        if (currentSection === "[script info]") {
          hasScriptInfo = true
        }
        if (currentSection === "[v4+ styles]" || currentSection === "[v4 styles]") {
          hasStyles = true
        }
      }
      continue
    }

    if (currentSection === "[events]") {
      if (/^Format\s*:/i.test(trimmed)) {
        eventFormat = trimmed.split(":").slice(1).join(":").trim() || eventFormat
      }
      continue
    }

    if (currentSection === "[script info]" && titleOverride && /^Title\s*:/i.test(trimmed)) {
      if (!injectedTitle) {
        preservedSections.push(`Title: ${titleOverride}`)
        injectedTitle = true
      }
      continue
    }

    if ((currentSection === "[v4+ styles]" || currentSection === "[v4 styles]") && /^Style\s*:/i.test(trimmed)) {
      const styleName = extractStyleName(trimmed)
      if (styleName) {
        styleNames.push(styleName)
      }
    }

    preservedSections.push(rawLine)
  }

  if (!hasScriptInfo || !hasStyles) {
    return extractAssDocumentLayout(
      {
        id: "fallback-ass",
        name: "Fallback ASS",
        format: "ass",
        content: DEFAULT_ASS_DOCUMENT_CONTENT,
      },
      titleOverride,
    )
  }

  if (titleOverride && hasScriptInfo && !injectedTitle) {
    const scriptInfoHeaderIndex = preservedSections.findIndex((line) => line.trim().toLowerCase() === "[script info]")
    if (scriptInfoHeaderIndex >= 0) {
      preservedSections.splice(scriptInfoHeaderIndex + 1, 0, `Title: ${titleOverride}`)
    }
  }

  return {
    lines: normalizeScriptInfoHeaders(preservedSections),
    eventFormat,
    styleNames,
  }
}

function pickStyleName(styleNames: string[], preferred: string[], fallbackIndex: number) {
  const lowered = styleNames.map((value) => value.trim().toLowerCase())
  for (const candidate of preferred) {
    const index = lowered.indexOf(candidate.trim().toLowerCase())
    if (index >= 0) {
      return styleNames[index]
    }
  }
  if (styleNames[fallbackIndex]) {
    return styleNames[fallbackIndex]
  }
  return styleNames[0] ?? "Default"
}

function buildAssDialogue(styleName: string, startMs: number, endMs: number, text: string) {
  return `Dialogue: ${[
    0,
    formatAssTime(startMs),
    formatAssTime(Math.max(startMs + 100, endMs)),
    styleName,
    "",
    0,
    0,
    0,
    "",
    escapeAssText(text),
  ].join(",")}`
}

function formatTextSubtitleTimestamp(milliseconds: number, separator: "," | ".") {
  const totalMs = Math.max(0, Math.floor(milliseconds))
  const hours = Math.floor(totalMs / 3_600_000)
  const minutes = Math.floor((totalMs % 3_600_000) / 60_000)
  const seconds = Math.floor((totalMs % 60_000) / 1000)
  const millis = totalMs % 1000
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}${separator}${pad(millis, 3)}`
}

function resolveTextSubtitleCueText(
  row: SubtitleStyleCueLike,
  displayMode: BuildAssSubtitleContentOptions["displayMode"],
) {
  const sourceText = row.sourceText.trim()
  const translationText = row.translationText?.trim() ?? ""
  switch (displayMode) {
    case "bilingual":
      return [sourceText, translationText].filter(Boolean).join("\n")
    default:
      return sourceText
  }
}

function sanitizeTextSubtitleCueText(value: string) {
  return value
    .replace(/\r\n?/g, "\n")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .join("\n")
}

export function buildTextSubtitleContent(options: {
  rows: SubtitleStyleCueLike[]
  displayMode: BuildAssSubtitleContentOptions["displayMode"]
  format: "srt" | "vtt"
}) {
  const cueLines = options.rows
    .map((row, index) => {
      const text = sanitizeTextSubtitleCueText(resolveTextSubtitleCueText(row, options.displayMode))
      if (!text) {
        return ""
      }
      const start = formatTextSubtitleTimestamp(row.startMs, options.format === "vtt" ? "." : ",")
      const end = formatTextSubtitleTimestamp(Math.max(row.startMs + 100, row.endMs), options.format === "vtt" ? "." : ",")
      if (options.format === "vtt") {
        return `${start} --> ${end}\n${text}`
      }
      return `${index + 1}\n${start} --> ${end}\n${text}`
    })
    .filter(Boolean)

  if (options.format === "vtt") {
    return cueLines.length > 0 ? `WEBVTT\n\n${cueLines.join("\n\n")}\n` : "WEBVTT\n"
  }
  return cueLines.length > 0 ? `${cueLines.join("\n\n")}\n` : ""
}

export function buildAssSubtitleContent(options: BuildAssSubtitleContentOptions) {
  const documentLayout = extractAssDocumentLayout(options.document, options.title?.trim() || "")
  const primaryStyle = pickStyleName(documentLayout.styleNames, ["Primary", "Default"], 0)
  const secondaryStyle = pickStyleName(documentLayout.styleNames, ["Secondary"], 1)
  const useSeparateSecondaryStyle = secondaryStyle !== primaryStyle && documentLayout.styleNames.length > 1
  const dialogueLines: string[] = []

  for (const row of options.rows) {
    const sourceText = row.sourceText.trim()
    const translationText = row.translationText?.trim() ?? ""

    if (options.displayMode === "mono") {
      if (sourceText) {
        dialogueLines.push(buildAssDialogue(primaryStyle, row.startMs, row.endMs, sourceText))
      }
      continue
    }

    if (!useSeparateSecondaryStyle) {
      const inlineText = [sourceText, translationText].filter(Boolean).join(" / ")
      if (inlineText) {
        dialogueLines.push(buildAssDialogue(primaryStyle, row.startMs, row.endMs, inlineText))
      }
      continue
    }

    if (sourceText) {
      dialogueLines.push(buildAssDialogue(primaryStyle, row.startMs, row.endMs, sourceText))
    }
    if (translationText) {
      dialogueLines.push(buildAssDialogue(secondaryStyle, row.startMs, row.endMs, translationText))
    }
  }

  return [
    ...documentLayout.lines,
    "",
    "[Events]",
    `Format: ${documentLayout.eventFormat}`,
    ...dialogueLines,
    "",
  ].join("\n")
}
