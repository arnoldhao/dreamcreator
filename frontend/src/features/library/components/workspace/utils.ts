import type {
  LibraryDTO,
  LibraryFileDTO,
  SubtitleCue,
  SubtitleDocument,
  SubtitleReviewSuggestionDTO,
  WorkspaceSubtitleTrackDTO,
  WorkspaceVideoTrackDTO,
} from "@/shared/contracts/library"
import {
  isWorkspaceQaCheckEnabled,
  resolveWorkspaceQaIssueCheckId,
  type WorkspaceQaCheckId,
  type WorkspaceQaCheckSettings,
} from "../../model/workspaceQa"

import type {
  WorkspaceCueIssue,
  WorkspaceCueIssueSeverity,
  WorkspaceDisplayMode,
  WorkspaceFilter,
  WorkspaceGuidelineOption,
  WorkspaceGuidelineProfileId,
  WorkspaceQaSummary,
  WorkspaceQaFilter,
  WorkspaceReviewDecision,
  WorkspaceReviewSuggestion,
  WorkspaceResolvedSubtitleRow,
  WorkspaceSubtitleRow,
  WorkspaceSubtitleTrackOption,
  WorkspaceSubtitleMetrics,
  WorkspaceVideoOption,
} from "./types"
import { t } from "@/shared/i18n"

function formatLabel(template: string, count: string | number) {
  return template.replace("{count}", String(count))
}

const MOCK_SOURCE_LINES = [
  "We keep the opening beat a touch earlier so the subtitle lands with the cut.",
  "Use the preview pane to confirm the line still sits inside the title safe area.",
  "This cue intentionally runs a little longer than usual so QA has something to flag.",
  "Small timing trims are easier to review when the waveform and table stay in sync.",
  "A bilingual layout helps compare pacing before we commit the final export.",
  "The current line marker should follow playback without stealing table selection.",
  "Keep one sentence per beat whenever the speaker pauses naturally.",
  "If a translation feels dense, shorten the text before stretching the duration.",
  "The footer reserves room for segment markers and later waveform annotations.",
  "Workspace actions remain in the header so mode changes do not disrupt the flow.",
  "Selected cues need a stronger outline than the surrounding playback highlight.",
  "This editor view is intentionally denser and tuned for long subtitle sessions.",
]

const MOCK_TRANSLATIONS = new Map<string, string>([
  [
    MOCK_SOURCE_LINES[0],
    "Opening timing is shifted slightly earlier so the subtitle arrives with the cut.",
  ],
  [
    MOCK_SOURCE_LINES[1],
    "Check the preview pane to make sure the subtitle stays inside the title safe area.",
  ],
  [
    MOCK_SOURCE_LINES[2],
    "This cue is a little longer on purpose so the QA column has a visible warning.",
  ],
  [
    MOCK_SOURCE_LINES[3],
    "Tiny timing trims are easier to judge when the waveform and table stay linked.",
  ],
  [
    MOCK_SOURCE_LINES[4],
    "Bilingual preview makes rhythm comparison easier before the export is locked.",
  ],
])

const GUIDELINE_PRESETS: Record<
  WorkspaceGuidelineProfileId,
  {
    label: string
    hint: string
    cpsSoft: number
    cpsHard: number
    cplSoft: number
    cplHard: number
    maxLines: number
  }
> = {
  netflix: {
    label: "Netflix",
    hint: "Balanced streaming defaults",
    cpsSoft: 17,
    cpsHard: 20,
    cplSoft: 40,
    cplHard: 42,
    maxLines: 2,
  },
  bbc: {
    label: "BBC",
    hint: "Broadcast readability",
    cpsSoft: 16,
    cpsHard: 18,
    cplSoft: 37,
    cplHard: 40,
    maxLines: 2,
  },
  ade: {
    label: "ADE",
    hint: "Conservative accessibility profile",
    cpsSoft: 15,
    cpsHard: 17,
    cplSoft: 35,
    cplHard: 38,
    maxLines: 2,
  },
}

function pad(value: number, size = 2) {
  return String(value).padStart(size, "0")
}

export function normalizeFileKind(value: string) {
  return value.trim().toLowerCase()
}

export function isWorkspaceVideoFile(file: LibraryFileDTO) {
  const kind = normalizeFileKind(file.kind)
  return kind === "video" || kind === "audio" || kind === "transcode"
}

export function resolveActiveFile<T extends { id: string }>(files: T[], fileId: string) {
  return files.find((file) => file.id === fileId) ?? files[0] ?? null
}

export function resolveLibraryFilePrimaryLabel(file: LibraryFileDTO | null | undefined) {
  return file?.displayLabel?.trim() || file?.name || ""
}

export function resolveFileFormat(file: LibraryFileDTO | null | undefined) {
  const explicit = file?.media?.format?.trim() ?? ""
  if (explicit) {
    return explicit.toUpperCase()
  }
  const name = file?.storage.localPath || file?.name || ""
  const index = name.lastIndexOf(".")
  if (index === -1 || index === name.length - 1) {
    return ""
  }
  return name.slice(index + 1).toUpperCase()
}

const AUDIO_MIME_TYPES: Record<string, string> = {
  aac: "audio/aac",
  flac: "audio/flac",
  mp3: "audio/mpeg",
  oga: "audio/ogg",
  ogg: "audio/ogg",
  opus: "audio/ogg",
  webm: "audio/webm",
}

const VIDEO_MIME_TYPES: Record<string, string> = {
  avi: "video/avi",
  m4v: "video/mp4",
  mp4: "video/mp4",
  mpeg: "video/mpeg",
  mpg: "video/mpeg",
  ogg: "video/ogg",
  ogv: "video/ogg",
  webm: "video/webm",
}

const WORKSPACE_WAVEFORM_MAX_MEDIA_SIZE_BYTES = 1024 * 1024 * 1024
const WORKSPACE_WAVEFORM_MAX_MEDIA_DURATION_MS = 2 * 60 * 60 * 1000

export function resolveLibraryMediaMimeType(file: LibraryFileDTO | null | undefined) {
  const kind = normalizeFileKind(file?.kind ?? "")
  const format = resolveFileFormat(file).trim().toLowerCase()
  if (!format) {
    return ""
  }
  if (kind === "audio" && AUDIO_MIME_TYPES[format]) {
    return AUDIO_MIME_TYPES[format]
  }
  if (VIDEO_MIME_TYPES[format]) {
    return VIDEO_MIME_TYPES[format]
  }
  return AUDIO_MIME_TYPES[format] ?? ""
}

export function resolveWorkspaceWaveformGuardKind(file: LibraryFileDTO | null | undefined) {
  const sizeBytes = file?.media?.sizeBytes ?? 0
  if (sizeBytes >= WORKSPACE_WAVEFORM_MAX_MEDIA_SIZE_BYTES) {
    return "size" as const
  }
  const durationMs = file?.media?.durationMs ?? 0
  if (durationMs >= WORKSPACE_WAVEFORM_MAX_MEDIA_DURATION_MS) {
    return "duration" as const
  }
  return "" as const
}

export function resolveSubtitleTrackOptions(files: LibraryFileDTO[]): WorkspaceSubtitleTrackOption[] {
  return files.map((file, index) => {
    const language =
      file.media?.language?.trim() ||
      formatLabel(t("library.workspace.header.trackNumber"), index + 1)
    const label =
      resolveLibraryFilePrimaryLabel(file) ||
      `${language.toUpperCase()} · ${t("library.type.subtitle")}`
    return {
      value: file.id,
      label,
      language,
    }
  })
}

export function resolveVideoVersionOptions(files: LibraryFileDTO[]): WorkspaceVideoOption[] {
  return files.map((file, index) => {
    const label =
      resolveLibraryFilePrimaryLabel(file) ||
      formatLabel(t("library.workspace.header.videoNumber"), index + 1)
    const format = resolveFileFormat(file) || file.kind.toUpperCase()
    return {
      value: file.id,
      label,
      hint: format,
    }
  })
}

function resolveWorkspaceSubtitleRoleLabel(track: WorkspaceSubtitleTrackDTO) {
  switch (track.role) {
    case "translation":
      return t("library.workspace.track.translation")
    case "reference":
      return t("library.workspace.track.reference")
    default:
      return ""
  }
}

function resolveWorkspaceTrackHint(displayHint: string | undefined, fallbackHint: string) {
  const explicit = displayHint?.trim()
  if (explicit) {
    return explicit
  }
  return fallbackHint.trim()
}

export function resolveWorkspaceProjectSubtitleTrackOptions(
  tracks: WorkspaceSubtitleTrackDTO[],
): WorkspaceSubtitleTrackOption[] {
  return tracks.map((track, index) => {
    const language =
      track.file.media?.language?.trim() ||
      formatLabel(t("library.workspace.header.trackNumber"), index + 1)
    return {
      value: track.trackId,
      label:
        track.display.label?.trim() ||
        `${language.toUpperCase()} · ${resolveWorkspaceSubtitleRoleLabel(track) || t("library.type.subtitle")}`,
      language,
      hint: resolveWorkspaceTrackHint(track.display.hint, track.file.name || language),
    }
  })
}

export function resolveWorkspaceProjectVideoOptions(
  tracks: WorkspaceVideoTrackDTO[],
): WorkspaceVideoOption[] {
  return tracks.map((track, index) => ({
    value: track.trackId,
    label:
      track.display.label?.trim() ||
      formatLabel(t("library.workspace.header.videoNumber"), index + 1),
    hint: resolveWorkspaceTrackHint(track.display.hint, resolveFileFormat(track.file) || track.file.kind.toUpperCase()),
  }))
}

export function resolveWorkspaceGuidelineOptions(): WorkspaceGuidelineOption[] {
  return Object.entries(GUIDELINE_PRESETS).map(([value, preset]) => ({
    value: value as WorkspaceGuidelineProfileId,
    label: t(`library.workspace.guideline.${value}.label`),
    hint: t(`library.workspace.guideline.${value}.hint`),
  }))
}

export function resolveWorkspaceGuidelineLabel(guidelineProfileId: WorkspaceGuidelineProfileId) {
  return t(`library.workspace.guideline.${guidelineProfileId}.label`)
}

export function buildAssetPreviewUrl(baseUrl: string, path: string) {
  if (!baseUrl || !path) {
    return ""
  }
  const trimmed = baseUrl.replace(/\/+$/, "")
  const previewName = path.replace(/\\/g, "/").split("/").pop()?.trim() || "asset"
  return `${trimmed}/api/library/asset/${encodeURIComponent(previewName)}?path=${encodeURIComponent(path)}`
}

export function formatMediaDuration(durationMs?: number) {
  if (!durationMs || durationMs <= 0) {
    return "-"
  }
  const totalSeconds = Math.max(0, Math.floor(durationMs / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (hours > 0) {
    return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`
  }
  return `${pad(minutes)}:${pad(seconds)}`
}

export function formatCueTime(milliseconds: number) {
  const safeMs = Math.max(0, Math.floor(milliseconds))
  const hours = Math.floor(safeMs / 3_600_000)
  const minutes = Math.floor((safeMs % 3_600_000) / 60_000)
  const seconds = Math.floor((safeMs % 60_000) / 1000)
  const millis = safeMs % 1000
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}.${pad(millis, 3)}`
}

export function formatCueDuration(durationMs: number) {
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return "0.0s"
  }
  const seconds = durationMs / 1000
  return `${seconds.toFixed(seconds >= 10 ? 1 : 2)}s`
}

export function parseCueTime(value?: string) {
  if (!value) {
    return 0
  }
  const normalized = value.trim().replace(",", ".")
  const smpteParts = normalized.split(":")
  if (smpteParts.length === 4) {
    const [hoursRaw, minutesRaw, secondsRaw, framesRaw] = smpteParts
    const hours = Number.parseInt(hoursRaw, 10)
    const minutes = Number.parseInt(minutesRaw, 10)
    const seconds = Number.parseInt(secondsRaw, 10)
    const frames = Number.parseInt(framesRaw.split(".")[0] ?? "", 10)
    if (
      !Number.isNaN(hours) &&
      !Number.isNaN(minutes) &&
      !Number.isNaN(seconds) &&
      !Number.isNaN(frames)
    ) {
      return (
        hours * 3_600_000 +
        minutes * 60_000 +
        seconds * 1000 +
        Math.round((frames / 30) * 1000)
      )
    }
  }
  const [timePart, fractionPart = "0"] = normalized.split(".")
  const pieces = timePart.split(":").map((part) => Number.parseInt(part, 10))
  if (pieces.some((part) => Number.isNaN(part))) {
    return 0
  }
  let hours = 0
  let minutes = 0
  let seconds = 0
  if (pieces.length === 3) {
    ;[hours, minutes, seconds] = pieces
  } else if (pieces.length === 2) {
    ;[minutes, seconds] = pieces
  } else if (pieces.length === 1) {
    ;[seconds] = pieces
  }
  const millis = Number.parseInt(fractionPart.padEnd(3, "0").slice(0, 3), 10)
  return hours * 3_600_000 + minutes * 60_000 + seconds * 1000 + (Number.isNaN(millis) ? 0 : millis)
}

export function clampMs(value: number, durationMs: number) {
  if (!Number.isFinite(value)) {
    return 0
  }
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return Math.max(0, value)
  }
  return Math.max(0, Math.min(value, durationMs))
}

export function buildMockSubtitleDocument(totalDurationMs: number, cueCount = 12): SubtitleDocument {
  const duration = Math.max(totalDurationMs, 90_000)
  const span = duration / (cueCount + 2)
  const cues: SubtitleCue[] = Array.from({ length: cueCount }).map((_, index) => {
    const startMs = Math.round((index + 1) * span)
    const endMs = Math.min(duration, startMs + Math.round(span * (index % 3 === 0 ? 0.9 : 0.72)))
    return {
      index: index + 1,
      start: formatCueTime(startMs),
      end: formatCueTime(endMs),
      text: MOCK_SOURCE_LINES[index % MOCK_SOURCE_LINES.length],
    }
  })
  return {
    format: "srt",
    cues,
  }
}

export function buildSubtitleRows(document: SubtitleDocument) {
  return document.cues.map<WorkspaceSubtitleRow>((cue, index) => {
    const startMs = parseCueTime(cue.start)
    const endMs = Math.max(startMs + 100, parseCueTime(cue.end))
    return {
      id: `cue-${cue.index || index + 1}`,
      index: cue.index > 0 ? cue.index : index + 1,
      start: cue.start,
      end: cue.end,
      startMs,
      endMs,
      durationMs: Math.max(100, endMs - startMs),
      sourceText: cue.text,
    }
  })
}

export function buildSubtitleDocument(rows: WorkspaceSubtitleRow[], format: string, baseDocument?: SubtitleDocument | null) {
  const baseCues = baseDocument?.cues ?? []
  return {
    ...(baseDocument ?? {}),
    format,
    cues: rows.map((row, index) => ({
      ...(baseCues[index] ?? {}),
      index: row.index,
      start: row.start,
      end: row.end,
      text: row.sourceText,
    })),
  } satisfies SubtitleDocument
}

export function buildMockTranslation(sourceText: string, language: string) {
  const known = MOCK_TRANSLATIONS.get(sourceText)
  if (known) {
    return known
  }
  const normalizedLanguage = language.trim().toUpperCase() || "TR"
  return `[${normalizedLanguage}] ${sourceText}`
}

function roundMetric(value: number) {
  return Math.round(value * 10) / 10
}

function measureTextLength(value: string) {
  return Array.from(value).length
}

function buildSubtitleMetrics(row: WorkspaceSubtitleRow): WorkspaceSubtitleMetrics {
  const lines = row.sourceText
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
  const characters = lines.reduce((sum, line) => sum + measureTextLength(line), 0)
  const cpl = lines.reduce((max, line) => Math.max(max, measureTextLength(line)), 0)
  const durationSeconds = row.durationMs > 0 ? row.durationMs / 1000 : 0

  return {
    cps: durationSeconds > 0 ? roundMetric(characters / durationSeconds) : 0,
    cpl,
    characters,
    lineCount: lines.length,
  }
}

function buildHeuristicQaIssues(
  row: WorkspaceSubtitleRow,
  previousRow: WorkspaceSubtitleRow | null,
  metrics: WorkspaceSubtitleMetrics,
  guidelineProfile: WorkspaceGuidelineProfileId,
  qaCheckSettings: WorkspaceQaCheckSettings,
) {
  const issues: WorkspaceCueIssue[] = []
  const guideline = GUIDELINE_PRESETS[guidelineProfile]
  const guidelineLabel = t(`library.workspace.guideline.${guidelineProfile}.label`)

  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "text") && !row.sourceText.trim()) {
    const reason = t("library.workspace.qaIssues.emptyText")
    issues.push({
      code: "empty-text",
      label: reason,
      severity: "error",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("empty-text"),
      reason,
    })
  }
  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "timing") && row.durationMs < 1_000) {
    const reason = t("library.workspace.qaIssues.shortDuration")
    issues.push({
      code: "fast-cue",
      label: reason,
      severity: "warning",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("fast-cue"),
      reason,
    })
  }
  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "layout") && metrics.lineCount > guideline.maxLines) {
    const reason = t("library.workspace.qaIssues.tooManyLines")
    issues.push({
      code: "too-many-lines",
      label: reason,
      severity: "warning",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("too-many-lines"),
      reason,
    })
  }
  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "cps") && metrics.cps > guideline.cpsHard) {
    const reason = t("library.workspace.qaIssues.cpsError")
      .replace("{value}", metrics.cps.toFixed(1))
      .replace("{guideline}", guidelineLabel)
    issues.push({
      code: "cps-error",
      label: reason,
      severity: "error",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("cps-error"),
      reason,
    })
  } else if (isWorkspaceQaCheckEnabled(qaCheckSettings, "cps") && metrics.cps > guideline.cpsSoft) {
    const reason = t("library.workspace.qaIssues.cpsWarning")
      .replace("{value}", metrics.cps.toFixed(1))
      .replace("{guideline}", guidelineLabel)
    issues.push({
      code: "cps-warning",
      label: reason,
      severity: "warning",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("cps-warning"),
      reason,
    })
  }
  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "cpl") && metrics.cpl > guideline.cplHard) {
    const reason = t("library.workspace.qaIssues.cplError")
      .replace("{value}", String(metrics.cpl))
      .replace("{guideline}", guidelineLabel)
    issues.push({
      code: "cpl-error",
      label: reason,
      severity: "error",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("cpl-error"),
      reason,
    })
  } else if (isWorkspaceQaCheckEnabled(qaCheckSettings, "cpl") && metrics.cpl > guideline.cplSoft) {
    const reason = t("library.workspace.qaIssues.cplWarning")
      .replace("{value}", String(metrics.cpl))
      .replace("{guideline}", guidelineLabel)
    issues.push({
      code: "cpl-warning",
      label: reason,
      severity: "warning",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("cpl-warning"),
      reason,
    })
  }
  if (isWorkspaceQaCheckEnabled(qaCheckSettings, "timing") && previousRow && row.startMs - previousRow.endMs < 120) {
    const reason = t("library.workspace.qaIssues.tightGap")
    issues.push({
      code: "tight-gap",
      label: reason,
      severity: "info",
      sourceLabel: resolveQaIssueSourceLabel(),
      detailLabel: resolveQaIssueDetailLabel("tight-gap"),
      reason,
    })
  }

  return issues
}

function dedupeIssues(issues: WorkspaceCueIssue[]) {
  const seen = new Set<string>()
  return issues.filter((issue) => {
    const key = `${issue.code}-${issue.label}-${issue.severity}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

export function resolveWorkspaceRows(
  rows: WorkspaceSubtitleRow[],
  baseRows: WorkspaceSubtitleRow[],
  guidelineProfile: WorkspaceGuidelineProfileId,
  qaCheckSettings: WorkspaceQaCheckSettings,
  comparisonRows: WorkspaceSubtitleRow[] = [],
  reviewSuggestions: SubtitleReviewSuggestionDTO[] = [],
  reviewKind: "proofread" | "qa" | string = "",
  reviewDecisions: Record<number, WorkspaceReviewDecision> = {},
) {
  const baseTextById = new Map(baseRows.map((row) => [row.id, row.sourceText]))
  const comparisonByIndex = new Map(comparisonRows.map((row) => [row.index, row]))
  const reviewSuggestionMap = new Map(
    reviewSuggestions.map((item) => [item.cueIndex, buildWorkspaceReviewSuggestion(item, reviewKind)]),
  )

  return rows.map<WorkspaceResolvedSubtitleRow>((row, index) => {
    const previousRow = rows[index - 1] ?? null
    const metrics = buildSubtitleMetrics(row)
    const reviewSuggestion = reviewSuggestionMap.get(row.index)
    const qaIssues = dedupeIssues([
      ...buildHeuristicQaIssues(row, previousRow, metrics, guidelineProfile, qaCheckSettings),
      ...(reviewSuggestion ? [buildReviewIssue(reviewSuggestion)] : []),
    ])
    const edited = (baseTextById.get(row.id) ?? row.sourceText) !== row.sourceText
    const comparisonRow = comparisonByIndex.get(row.index) ?? comparisonRows[index] ?? null
    return {
      ...row,
      durationLabel: formatCueDuration(row.durationMs),
      translationText: comparisonRow?.sourceText ?? "",
      qaIssues,
      edited,
      metrics,
      reviewSuggestion,
      reviewDecision: reviewSuggestion ? reviewDecisions[row.index] ?? "undecided" : undefined,
      status: reviewSuggestion ? "review" : edited ? "edited" : qaIssues.length > 0 ? "review" : "ready",
    }
  })
}

export function resolveWorkspaceQaSummary(rows: WorkspaceResolvedSubtitleRow[]): WorkspaceQaSummary {
  return rows.reduce<WorkspaceQaSummary>(
    (summary, row) => {
      const matchedChecks = new Set<WorkspaceQaCheckId>()
      const qaIssues = row.qaIssues.filter((issue) => {
        const checkId = resolveWorkspaceQaIssueCheckId(issue.code)
        if (checkId) {
          matchedChecks.add(checkId)
          return true
        }
        return false
      })
      if (qaIssues.length > 0) {
        summary.flaggedCueCount += 1
      }
      if (qaIssues.some((issue) => issue.severity === "error")) {
        summary.errorCount += 1
      }
      if (qaIssues.some((issue) => issue.severity === "warning")) {
        summary.warningCount += 1
      }
      for (const checkId of matchedChecks) {
        summary.issueCounts[checkId] += 1
      }
      return summary
    },
    {
      flaggedCueCount: 0,
      errorCount: 0,
      warningCount: 0,
      issueCounts: { text: 0, timing: 0, layout: 0, cps: 0, cpl: 0 },
    },
  )
}

export function filterWorkspaceRows(
  rows: WorkspaceResolvedSubtitleRow[],
  searchValue: string,
  filterValue: WorkspaceFilter,
  qaFilter: WorkspaceQaFilter,
  displayMode: WorkspaceDisplayMode,
  currentRowId: string,
) {
  const normalizedSearch = searchValue.trim().toLowerCase()
  const currentRow = rows.find((row) => row.id === currentRowId) ?? null

  return rows.filter((row) => {
    if (normalizedSearch) {
      const haystack =
        displayMode === "mono"
          ? row.sourceText
          : `${row.sourceText}\n${row.translationText}`
      if (!haystack.toLowerCase().includes(normalizedSearch)) {
        return false
      }
    }

    if (filterValue === "needs-review" && row.qaIssues.length === 0) {
      return false
    }
    if (filterValue === "edited" && !row.edited) {
      return false
    }
    if (filterValue === "current-window" && currentRow && Math.abs(row.index - currentRow.index) > 2) {
      return false
    }

    if (qaFilter === "issues" && row.qaIssues.length === 0) {
      return false
    }
    if (qaFilter === "warnings" && !row.qaIssues.some((issue) => issue.severity === "warning")) {
      return false
    }
    if (qaFilter === "errors" && !row.qaIssues.some((issue) => issue.severity === "error")) {
      return false
    }
    if (qaFilter === "clean" && row.qaIssues.length > 0) {
      return false
    }

    return true
  })
}

export function resolveCurrentRow(rows: WorkspaceResolvedSubtitleRow[], playheadMs: number) {
  if (rows.length === 0) {
    return null
  }
  let low = 0
  let high = rows.length
  while (low < high) {
    const mid = Math.floor((low + high) / 2)
    if (rows[mid].startMs <= playheadMs) {
      low = mid + 1
      continue
    }
    high = mid
  }
  const nextRow = rows[low] ?? null
  const previousRow = rows[low - 1] ?? null
  if (previousRow && playheadMs < previousRow.endMs) {
    return previousRow
  }
  return nextRow ?? rows[rows.length - 1]
}

export function resolveWorkspaceDurationMs(videoFile: LibraryFileDTO | null, rows: WorkspaceSubtitleRow[]) {
  const mediaDuration = videoFile?.media?.durationMs ?? 0
  const subtitleDuration = rows[rows.length - 1]?.endMs ?? 0
  return Math.max(mediaDuration, subtitleDuration, 90_000)
}

export function formatFrameRate(frameRate?: number) {
  if (!frameRate || frameRate <= 0) {
    return "-"
  }
  const value = frameRate.toFixed(frameRate >= 10 ? 2 : 3).replace(/\.0+$/, "").replace(/(\.\d*[1-9])0+$/, "$1")
  return `${value} fps`
}

export function formatResolution(file: LibraryFileDTO | null) {
  if (!file?.media?.width || !file.media.height) {
    return "-"
  }
  return `${file.media.width} x ${file.media.height}`
}

export function resolveResourceName(library: LibraryDTO | undefined, videoFile: LibraryFileDTO | null, subtitleFile: LibraryFileDTO | null) {
  return (
    resolveLibraryFilePrimaryLabel(videoFile) ||
    resolveLibraryFilePrimaryLabel(subtitleFile) ||
    library?.name ||
    "Workspace"
  )
}

export function resolveResourceTypeLabel(videoFile: LibraryFileDTO | null, subtitleFile: LibraryFileDTO | null) {
  const file = videoFile ?? subtitleFile
  const format = resolveFileFormat(file)
  if (format) {
    return format
  }
  const kind = file?.kind?.trim() || t("library.workspace.page.resourceType")
  return kind.toUpperCase()
}

export function resolveVisibleText(row: WorkspaceResolvedSubtitleRow, displayMode: WorkspaceDisplayMode) {
  if (displayMode === "mono") {
    return row.sourceText
  }
  if (!row.translationText.trim()) {
    return row.sourceText
  }
  return `${row.sourceText}\n${row.translationText}`
}

function buildWorkspaceReviewSuggestion(
  item: SubtitleReviewSuggestionDTO,
  reviewKind: "proofread" | "qa" | string,
): WorkspaceReviewSuggestion {
  const normalizedKind = item.sourceCode?.trim() ? reviewKind : reviewKind
  return {
    cueIndex: item.cueIndex,
    kind: normalizedKind,
    sourceLabel: resolveReviewSourceLabel(reviewKind, item),
    detailLabel: resolveReviewDetailLabel(reviewKind, item),
    reason: item.reason?.trim() || resolveReviewDefaultReason(reviewKind),
    originalText: item.originalText,
    suggestedText: item.suggestedText,
    severity: normalizeReviewSeverity(item.severity),
  }
}

function buildReviewIssue(item: WorkspaceReviewSuggestion): WorkspaceCueIssue {
  return {
    code: `review-${item.kind}-${item.cueIndex}`,
    label: item.reason,
    severity: item.severity,
    sourceLabel: item.sourceLabel,
    detailLabel: item.detailLabel,
    reason: item.reason,
  }
}

function normalizeReviewSeverity(value?: string): WorkspaceCueIssueSeverity {
  switch ((value ?? "").trim().toLowerCase()) {
    case "error":
      return "error"
    case "info":
      return "info"
    default:
      return "warning"
  }
}

function resolveReviewSourceLabel(
  reviewKind: "proofread" | "qa" | string,
  item: SubtitleReviewSuggestionDTO,
) {
  return reviewKind === "qa"
    ? t("library.workspace.review.sourceQa")
    : t("library.workspace.review.sourceProofread")
}

function resolveReviewDefaultReason(reviewKind: "proofread" | "qa" | string) {
  return reviewKind === "qa"
    ? t("library.workspace.review.defaultQaReason")
    : t("library.workspace.review.defaultProofreadReason")
}

function resolveReviewDetailLabel(
  reviewKind: "proofread" | "qa" | string,
  item: SubtitleReviewSuggestionDTO,
) {
  const primaryCategory = item.categories?.find((value) => value.trim()) || item.sourceCode || ""
  const normalized = primaryCategory.trim().toLowerCase()
  if (!normalized) {
    return reviewKind === "qa"
      ? t("library.workspace.review.detailQa")
      : t("library.workspace.review.detailProofread")
  }
  switch (normalized) {
    case "spelling":
      return t("library.workspace.review.categorySpelling")
    case "punctuation":
      return t("library.workspace.review.categoryPunctuation")
    case "terminology":
      return t("library.workspace.review.categoryTerminology")
    case "fluency":
      return t("library.workspace.review.categoryFluency")
    case "spacing":
    case "whitespace":
      return t("library.workspace.review.categorySpacing")
    case "cps":
    case "cps-error":
    case "cps-warning":
      return t("library.workspace.review.categoryCps")
    case "cpl":
    case "cpl-error":
    case "cpl-warning":
      return t("library.workspace.review.categoryCpl")
    case "timing":
    case "fast-cue":
    case "tight-gap":
    case "invalid_timing":
    case "non_positive_duration":
    case "overlap":
      return t("library.workspace.review.categoryTiming")
    case "layout":
    case "too-many-lines":
      return t("library.workspace.review.categoryLayout")
    case "text":
    case "empty-text":
    case "empty_text":
    case "empty_content":
      return t("library.workspace.review.categoryText")
    default:
      return normalized.toUpperCase()
  }
}

function resolveQaIssueSourceLabel() {
  return t("library.workspace.review.sourceQa")
}

function resolveQaIssueDetailLabel(code: string) {
  const normalized = code.trim().toLowerCase()
  switch (normalized) {
    case "cps-error":
    case "cps-warning":
      return t("library.workspace.review.categoryCps")
    case "cpl-error":
    case "cpl-warning":
      return t("library.workspace.review.categoryCpl")
    case "fast-cue":
    case "tight-gap":
    case "invalid_timing":
    case "non_positive_duration":
    case "overlap":
      return t("library.workspace.review.categoryTiming")
    case "too-many-lines":
      return t("library.workspace.review.categoryLayout")
    case "empty-text":
    case "empty_text":
    case "empty_content":
      return t("library.workspace.review.categoryText")
    default:
      return t("library.workspace.review.detailQa")
  }
}
