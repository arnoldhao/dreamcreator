import type {
  LibraryBilingualStyleDTO,
  LibraryModuleConfigDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleExportPresetDTO,
  LibrarySubtitleStyleDocumentAnalysisDTO,
  LibrarySubtitleStyleDocumentDTO,
  LibrarySubtitleStyleSourceDTO,
  SubtitleExportConfig,
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

export type FCPXMLFrameDurationPreset = {
  value: string
  label: string
  fps: number
}

export type ITTFrameRateMultiplierPreset = {
  value: string
  label: string
}

export type ITTFrameTiming = {
  frameRate: number
  frameRateMultiplier: string
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

export type VideoExportSoftSubtitleRoute = {
  format: "srt" | "vtt" | "ass"
  label: string
  codec: string
}

type BuildAssSubtitleContentOptions = {
  rows: SubtitleStyleCueLike[]
  displayMode: "single" | "dual"
  document: LibrarySubtitleStyleDocumentDTO | null
  title?: string
}

type ExtractedAssDocumentLayout = {
  lines: string[]
  eventFormat: string
  styleNames: string[]
}

const DEFAULT_ASS_DOCUMENT_CONTENT = [
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

export const DEFAULT_FCPXML_FRAME_DURATION = "1/30s"
export const DEFAULT_ITT_FRAME_RATE = 30
export const DEFAULT_ITT_FRAME_RATE_MULTIPLIER = "1 1"
export const DEFAULT_SUBTITLE_EXPORT_ASS_TITLE = "DreamCreator Export"
export const DEFAULT_SUBTITLE_EXPORT_PROJECT_NAME = "DreamCreator Project"
export const DEFAULT_SUBTITLE_EXPORT_LIBRARY_NAME = "DreamCreator Project Library"
export const DEFAULT_SUBTITLE_EXPORT_EVENT_NAME = "DreamCreator Project Event"
export const DEFAULT_SUBTITLE_EXPORT_LIBRARY_SUFFIX = " Library"
export const DEFAULT_SUBTITLE_EXPORT_EVENT_SUFFIX = " Event"

const DEFAULT_BUILTIN_SUBTITLE_STYLE_FONT_SOURCES: LibrarySubtitleStyleSourceDTO[] = [
  {
    id: "fontget-google-fonts",
    name: "Google Fonts",
    kind: "font",
    provider: "fontget",
    url: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json",
    prefix: "google",
    filename: "google-fonts.json",
    priority: 1,
    builtIn: true,
    enabled: true,
    syncStatus: "idle",
    lastSyncedAt: "",
    lastError: "",
  },
  {
    id: "fontget-nerd-fonts",
    name: "Nerd Fonts",
    kind: "font",
    provider: "fontget",
    url: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json",
    prefix: "nerd",
    filename: "nerd-fonts.json",
    priority: 2,
    builtIn: true,
    enabled: true,
    syncStatus: "idle",
    lastSyncedAt: "",
    lastError: "",
  },
  {
    id: "fontget-font-squirrel",
    name: "Font Squirrel",
    kind: "font",
    provider: "fontget",
    url: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json",
    prefix: "squirrel",
    filename: "font-squirrel.json",
    priority: 3,
    builtIn: true,
    enabled: true,
    syncStatus: "idle",
    lastSyncedAt: "",
    lastError: "",
  },
]

export const FCPXML_FRAME_DURATION_PRESETS: FCPXMLFrameDurationPreset[] = [
  { value: "1001/24000s", label: "23.976", fps: 24000 / 1001 },
  { value: "1/24s", label: "24", fps: 24 },
  { value: "1/25s", label: "25", fps: 25 },
  { value: "1001/30000s", label: "29.97", fps: 30000 / 1001 },
  { value: "1/30s", label: "30", fps: 30 },
  { value: "1/50s", label: "50", fps: 50 },
  { value: "1001/60000s", label: "59.94", fps: 60000 / 1001 },
  { value: "1/60s", label: "60", fps: 60 },
]

export const ITT_FRAME_RATE_MULTIPLIER_PRESETS: ITTFrameRateMultiplierPreset[] = [
  { value: "1 1", label: "1/1 (Exact)" },
  { value: "1000 1001", label: "1000/1001 (NTSC)" },
]

function pad(value: number, size = 2) {
  return String(value).padStart(size, "0")
}

function createSubtitleDocumentId(prefix: string) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`
}

function normalizeAssDocumentContent(content: string | undefined) {
  const normalized = (content ?? "").replace(/\r\n?/g, "\n").trim()
  return normalized ? `${normalized}\n` : ""
}

export function resolveSubtitleStyleDocuments(
  config?: LibraryModuleConfigDTO,
): LibrarySubtitleStyleDocumentDTO[] {
  void config
  return []
}

export function resolveSubtitleStyleSources(config?: LibraryModuleConfigDTO) {
  return sortSubtitleStyleSources(config?.subtitleStyles?.sources ?? [])
}

export function resolveSubtitleStyleDefaults(config?: LibraryModuleConfigDTO) {
  return {
    monoStyleId: config?.subtitleStyles?.defaults?.monoStyleId ?? "",
    bilingualStyleId: config?.subtitleStyles?.defaults?.bilingualStyleId ?? "",
    subtitleExportPresetId: config?.subtitleStyles?.defaults?.subtitleExportPresetId ?? "",
  }
}

export function resolveDefaultMonoStyle(config?: LibraryModuleConfigDTO): LibraryMonoStyleDTO | null {
  const styles = config?.subtitleStyles?.monoStyles ?? []
  if (styles.length === 0) {
    return null
  }
  const defaultID = config?.subtitleStyles?.defaults?.monoStyleId?.trim()
  return styles.find((style) => style.id === defaultID) ?? styles[0] ?? null
}

export function resolveDefaultBilingualStyle(config?: LibraryModuleConfigDTO): LibraryBilingualStyleDTO | null {
  const styles = config?.subtitleStyles?.bilingualStyles ?? []
  if (styles.length === 0) {
    return null
  }
  const defaultID = config?.subtitleStyles?.defaults?.bilingualStyleId?.trim()
  return styles.find((style) => style.id === defaultID) ?? styles[0] ?? null
}

export function resolveSubtitleExportPresets(config?: LibraryModuleConfigDTO) {
  return config?.subtitleStyles?.subtitleExportPresets ?? []
}

export function buildSubtitleStyleDocumentSummary(document: LibrarySubtitleStyleDocumentDTO) {
  const source =
    document.source === "builtin"
      ? t("library.config.subtitleStyles.sourceBuiltin")
      : document.source === "remote"
        ? t("library.config.subtitleStyles.sourceRemote")
        : t("library.config.subtitleStyles.sourceLibrary")
  const summary = resolveAssDocumentSummary(document)
  const resolution = summary.playResX && summary.playResY ? `${summary.playResX}x${summary.playResY}` : summary.detectedFormat.toUpperCase()
  return `${source} · ${resolution} · ${summary.styleCount} styles`
}

export function duplicateSubtitleStyleDocument(document: LibrarySubtitleStyleDocumentDTO): LibrarySubtitleStyleDocumentDTO {
  return {
    ...document,
    id: createSubtitleDocumentId("ass"),
    name: `${document.name} ${t("library.config.subtitleStyles.copySuffix")}`,
    source: "library",
    sourceRef: "",
    version: "1",
    enabled: true,
  }
}

export function duplicateSubtitleExportPreset(profile: LibrarySubtitleExportPresetDTO): LibrarySubtitleExportPresetDTO {
  return {
    ...profile,
    id: createSubtitleDocumentId("subtitle-export-preset"),
    name: `${profile.name || t("library.config.subtitleStyles.subtitleExportPresetFallback")} ${t("library.config.subtitleStyles.copySuffix")}`,
  }
}

export function createEmptySubtitleStyleDocument(): LibrarySubtitleStyleDocumentDTO {
  return {
    id: createSubtitleDocumentId("ass"),
    name: t("library.config.subtitleStyles.customDocument"),
    description: "",
    source: "library",
    sourceRef: "",
    version: "1",
    enabled: true,
    format: "ass",
    content: DEFAULT_ASS_DOCUMENT_CONTENT,
  }
}

export function createEmptySubtitleExportPreset(format = "ass"): LibrarySubtitleExportPresetDTO {
  const normalizedFormat = normalizeSubtitleExportFormat(format)
  return {
    id: createSubtitleDocumentId("subtitle-export-preset"),
    name: t("library.config.subtitleStyles.subtitleExportPresetDefaultName"),
    format: normalizedFormat,
    targetFormat: normalizedFormat,
    mediaStrategy: "auto",
    config: createDefaultSubtitleExportConfig(),
  }
}

export function createEmptySubtitleStyleSource(): LibrarySubtitleStyleSourceDTO {
  return {
    id: createSubtitleDocumentId("style-source"),
    name: t("library.config.subtitleStyles.githubSource"),
    kind: "style",
    provider: "github",
    url: "",
    prefix: "",
    filename: "manifest.json",
    priority: 100,
    builtIn: false,
    owner: "",
    repo: "",
    ref: "main",
    manifestPath: "manifest.json",
    enabled: false,
    syncStatus: "idle",
    lastSyncedAt: "",
    lastError: "",
  }
}

export function ensureBuiltInSubtitleStyleFontSources(sources: LibrarySubtitleStyleSourceDTO[]) {
  const seen = new Set(sources.map((source) => source.id.trim()))
  const nextSources = [...sources]
  for (const source of DEFAULT_BUILTIN_SUBTITLE_STYLE_FONT_SOURCES) {
    if (seen.has(source.id)) {
      continue
    }
    nextSources.push({ ...source })
  }
  return sortSubtitleStyleSources(nextSources)
}

export function buildDefaultSubtitleExportProjectName(libraryName: string) {
  const normalized = libraryName.trim()
  return normalized ? `DreamCreator - ${normalized}` : DEFAULT_SUBTITLE_EXPORT_PROJECT_NAME
}

export function buildDefaultSubtitleExportAssTitle(libraryName: string) {
  return libraryName.trim() ? buildDefaultSubtitleExportProjectName(libraryName) : DEFAULT_SUBTITLE_EXPORT_ASS_TITLE
}

export function buildDefaultSubtitleExportLibraryName(libraryName: string) {
  return libraryName.trim()
    ? `${buildDefaultSubtitleExportProjectName(libraryName)}${DEFAULT_SUBTITLE_EXPORT_LIBRARY_SUFFIX}`
    : DEFAULT_SUBTITLE_EXPORT_LIBRARY_NAME
}

export function resolveVideoExportSoftSubtitleRoute(container: string): VideoExportSoftSubtitleRoute {
  switch (container.trim().toLowerCase()) {
    case "mp4":
    case "mov":
      return { format: "srt", label: "SRT", codec: "mov_text" }
    case "mkv":
      return { format: "ass", label: "ASS", codec: "ass" }
    case "webm":
      return { format: "vtt", label: "WebVTT", codec: "webvtt" }
    default:
      return { format: "srt", label: "SRT", codec: "subrip" }
  }
}

export function buildDefaultSubtitleExportEventName(libraryName: string) {
  return libraryName.trim()
    ? `${buildDefaultSubtitleExportProjectName(libraryName)}${DEFAULT_SUBTITLE_EXPORT_EVENT_SUFFIX}`
    : DEFAULT_SUBTITLE_EXPORT_EVENT_NAME
}

export function normalizeSubtitleExportFormat(value: string) {
  const normalized = value.trim().toLowerCase()
  switch (normalized) {
    case "srt":
    case "vtt":
    case "webvtt":
    case "ass":
    case "ssa":
    case "itt":
    case "fcpxml":
      return normalized === "webvtt" ? "vtt" : normalized
    case "ttml":
    case "xml":
    case "dfxp":
      return "itt"
    default:
      return "srt"
  }
}

export function normalizeSubtitleExportMediaStrategy(value: string) {
  return value.trim().toLowerCase() === "fixed" ? "fixed" : "auto"
}

export function normalizeFCPXMLFrameDuration(value: string | undefined) {
  const normalized = (value ?? "").trim()
  if (!normalized) {
    return DEFAULT_FCPXML_FRAME_DURATION
  }
  const preset = FCPXML_FRAME_DURATION_PRESETS.find((item) => item.value === normalized)
  if (preset) {
    return preset.value
  }
  if (/^\d+\/\d+s$/.test(normalized) || /^\d+s$/.test(normalized)) {
    return normalized
  }
  return DEFAULT_FCPXML_FRAME_DURATION
}

export function resolveFCPXMLFrameDurationFromFrameRate(frameRate: number | undefined) {
  if (typeof frameRate !== "number" || !Number.isFinite(frameRate) || frameRate <= 0) {
    return DEFAULT_FCPXML_FRAME_DURATION
  }
  let best = FCPXML_FRAME_DURATION_PRESETS[0]
  let bestDistance = Math.abs(frameRate - best.fps)
  for (const preset of FCPXML_FRAME_DURATION_PRESETS.slice(1)) {
    const distance = Math.abs(frameRate - preset.fps)
    if (distance < bestDistance) {
      best = preset
      bestDistance = distance
    }
  }
  return best.value
}

export function resolveFCPXMLFrameDurationLabel(value: string | undefined) {
  const normalized = normalizeFCPXMLFrameDuration(value)
  const preset = FCPXML_FRAME_DURATION_PRESETS.find((item) => item.value === normalized)
  if (preset) {
    return `${preset.label} (${preset.value})`
  }
  return normalized
}

export function normalizeITTFrameRate(value: number | undefined) {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return DEFAULT_ITT_FRAME_RATE
  }
  return Math.max(1, Math.round(value))
}

export function normalizeITTFrameRateMultiplier(value: string | undefined) {
  const trimmed = (value ?? "").trim()
  if (!trimmed) {
    return DEFAULT_ITT_FRAME_RATE_MULTIPLIER
  }
  const normalized = trimmed.replace(/\//g, " ")
  const parts = normalized.split(/\s+/).filter(Boolean)
  if (parts.length !== 2) {
    return DEFAULT_ITT_FRAME_RATE_MULTIPLIER
  }
  const numerator = Number.parseInt(parts[0] ?? "", 10)
  const denominator = Number.parseInt(parts[1] ?? "", 10)
  if (!Number.isFinite(numerator) || !Number.isFinite(denominator) || numerator <= 0 || denominator <= 0) {
    return DEFAULT_ITT_FRAME_RATE_MULTIPLIER
  }
  return `${numerator} ${denominator}`
}

function parseITTFrameRateMultiplier(value: string | undefined) {
  const normalized = normalizeITTFrameRateMultiplier(value)
  const [numeratorRaw, denominatorRaw] = normalized.split(" ")
  const numerator = Number.parseInt(numeratorRaw ?? "1", 10)
  const denominator = Number.parseInt(denominatorRaw ?? "1", 10)
  if (!Number.isFinite(numerator) || !Number.isFinite(denominator) || numerator <= 0 || denominator <= 0) {
    return { numerator: 1, denominator: 1 }
  }
  return { numerator, denominator }
}

export function resolveITTEffectiveFrameRate(frameRate: number | undefined, frameRateMultiplier: string | undefined) {
  const base = normalizeITTFrameRate(frameRate)
  const multiplier = parseITTFrameRateMultiplier(frameRateMultiplier)
  return (base * multiplier.numerator) / multiplier.denominator
}

export function resolveITTFrameTimingFromFrameRate(frameRate: number | undefined): ITTFrameTiming {
  if (typeof frameRate !== "number" || !Number.isFinite(frameRate) || frameRate <= 0) {
    return { frameRate: DEFAULT_ITT_FRAME_RATE, frameRateMultiplier: DEFAULT_ITT_FRAME_RATE_MULTIPLIER }
  }
  const candidates: ITTFrameTiming[] = [
    { frameRate: 24, frameRateMultiplier: "1 1" },
    { frameRate: 24, frameRateMultiplier: "1000 1001" },
    { frameRate: 25, frameRateMultiplier: "1 1" },
    { frameRate: 30, frameRateMultiplier: "1 1" },
    { frameRate: 30, frameRateMultiplier: "1000 1001" },
    { frameRate: 50, frameRateMultiplier: "1 1" },
    { frameRate: 60, frameRateMultiplier: "1 1" },
    { frameRate: 60, frameRateMultiplier: "1000 1001" },
  ]
  let best = candidates[0]
  let bestDistance = Math.abs(frameRate - resolveITTEffectiveFrameRate(best.frameRate, best.frameRateMultiplier))
  for (const candidate of candidates.slice(1)) {
    const effective = resolveITTEffectiveFrameRate(candidate.frameRate, candidate.frameRateMultiplier)
    const distance = Math.abs(frameRate - effective)
    if (distance < bestDistance) {
      best = candidate
      bestDistance = distance
    }
  }
  return best
}

export function resolveITTFrameRateLabel(frameRate: number | undefined, frameRateMultiplier: string | undefined) {
  const base = normalizeITTFrameRate(frameRate)
  const normalizedMultiplier = normalizeITTFrameRateMultiplier(frameRateMultiplier)
  const { numerator, denominator } = parseITTFrameRateMultiplier(normalizedMultiplier)
  const effective = resolveITTEffectiveFrameRate(base, normalizedMultiplier)
  const rounded = Number.isInteger(effective) ? String(effective) : effective.toFixed(3)
  if (normalizedMultiplier === "1 1") {
    return `${base} fps`
  }
  return `${base} * ${numerator}/${denominator} (${rounded} fps)`
}

function createDefaultSubtitleExportConfig(): SubtitleExportConfig {
  const ittTiming = resolveITTFrameTimingFromFrameRate(DEFAULT_ITT_FRAME_RATE)
  return {
    srt: {
      encoding: "utf-8",
    },
    vtt: {
      kind: "subtitles",
      language: "en-US",
    },
    ass: {
      playResX: 1920,
      playResY: 1080,
      title: "DreamCreator Export",
    },
    itt: {
      frameRate: ittTiming.frameRate,
      frameRateMultiplier: ittTiming.frameRateMultiplier,
      language: "en-US",
    },
    fcpxml: {
      frameDuration: DEFAULT_FCPXML_FRAME_DURATION,
      width: 1920,
      height: 1080,
      colorSpace: "1-1-1 (Rec. 709)",
      version: "1.11",
      projectName: "DreamCreator Project",
      libraryName: "DreamCreator Project_Library",
      eventName: "DreamCreator Project_Event",
      defaultLane: 1,
      startTimecodeSeconds: 3600,
    },
  }
}

export function sortSubtitleStyleSources(sources: LibrarySubtitleStyleSourceDTO[]) {
  return [...sources].sort((left, right) => {
    const leftBuiltIn = left.builtIn === true
    const rightBuiltIn = right.builtIn === true
    if (leftBuiltIn !== rightBuiltIn) {
      return leftBuiltIn ? -1 : 1
    }
    const leftPriority = typeof left.priority === "number" && left.priority > 0 ? left.priority : Number.MAX_SAFE_INTEGER
    const rightPriority = typeof right.priority === "number" && right.priority > 0 ? right.priority : Number.MAX_SAFE_INTEGER
    if (leftPriority !== rightPriority) {
      return leftPriority - rightPriority
    }
    return (left.name ?? "").localeCompare(right.name ?? "")
  })
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
    case "dual":
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

    if (options.displayMode === "single") {
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
