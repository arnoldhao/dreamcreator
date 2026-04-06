import { formatTemplate } from "./i18n"

type TranslateFn = (key: string) => string

const TRANSLATED_CHUNK_PATTERN = /^translated chunk\s+(\d+)\s+of\s+(\d+)$/i
const PROOFREAD_CHUNK_PATTERN = /^proofread chunk\s+(\d+)\s+of\s+(\d+)$/i
const I18N_PROGRESS_PREFIX = "i18n:"

export function translateLibraryProgressLabel(label: string, t: TranslateFn): string {
  const raw = label?.trim()
  if (!raw) {
    return "-"
  }

  const translatedSpec = translateProgressSpec(raw, t)
  if (translatedSpec) {
    return translatedSpec
  }

  switch (normalizeProgressText(raw)) {
    case "preparing":
      return t("library.progress.preparing")
    case "starting":
      return t("library.progress.starting")
    case "fetching metadata":
      return t("library.progress.fetchingMetadata")
    case "translating":
      return t("library.progress.translating")
    case "proofreading":
      return t("library.progress.proofreading")
    case "qa reviewing":
    case "qa review":
      return t("library.progress.qaReviewing")
    case "transcoding":
      return t("library.progress.transcoding")
    case "downloading":
      return t("library.progress.downloading")
    case "downloading video":
      return t("library.progress.downloadingVideo")
    case "downloading audio":
      return t("library.progress.downloadingAudio")
    case "downloading subtitles":
      return t("library.progress.downloadingSubtitles")
    case "downloading thumbnail":
      return t("library.progress.downloadingThumbnail")
    case "muxing":
      return t("library.progress.muxing")
    case "cleaning up":
      return t("library.progress.cleaningUp")
    case "post-processing":
    case "post processing":
    case "postprocessing":
      return t("library.progress.postProcessing")
    case "queued":
      return t("library.status.queued")
    case "in progress":
      return t("library.status.running")
    case "done":
    case "completed":
      return t("library.status.succeeded")
    case "failed":
      return t("library.status.failed")
    case "canceled":
    case "cancelled":
      return t("library.status.canceled")
    default:
      return raw
  }
}

export function translateLibraryProgressDetail(detail: string, t: TranslateFn): string {
  const raw = detail?.trim()
  if (!raw) {
    return ""
  }

  const translatedSpec = translateProgressSpec(raw, t)
  if (translatedSpec) {
    return translatedSpec
  }
  const normalized = normalizeProgressText(raw)

  const translatedLabel = translateLibraryProgressLabel(raw, t)
  if (translatedLabel !== raw) {
    return translatedLabel
  }

  switch (normalized) {
    case "preparing download":
      return t("library.progressDetail.preparingDownload")
    case "preparing subtitle translation":
      return t("library.progressDetail.preparingSubtitleTranslation")
    case "preparing subtitle proofread":
      return t("library.progressDetail.preparingSubtitleProofread")
    case "preparing subtitle qa review":
      return t("library.progressDetail.preparingSubtitleQaReview")
    case "preparing ffmpeg transcode":
      return t("library.progressDetail.preparingFfmpegTranscode")
    case "ffmpeg is rendering the output":
      return t("library.progressDetail.ffmpegRenderingOutput")
    case "ffmpeg transcode queued":
      return t("library.progressDetail.ffmpegTranscodeQueued")
    case "ffmpeg transcode completed":
      return t("library.progressDetail.ffmpegTranscodeCompleted")
    case "subtitle translation queued":
      return t("library.progressDetail.subtitleTranslationQueued")
    case "subtitle translation completed":
      return t("library.progressDetail.subtitleTranslationCompleted")
    case "subtitle proofread queued":
      return t("library.progressDetail.subtitleProofreadQueued")
    case "subtitle proofread completed":
      return t("library.progressDetail.subtitleProofreadCompleted")
    case "subtitle qa review queued":
      return t("library.progressDetail.subtitleQaReviewQueued")
    case "subtitle qa review completed":
      return t("library.progressDetail.subtitleQaReviewCompleted")
    case "resume requested":
      return t("library.progressDetail.resumeRequested")
    case "canceled by user":
      return t("library.progressDetail.canceledByUser")
    case "download canceled":
      return t("library.progressDetail.downloadCanceled")
    case "transcode canceled":
      return t("library.progressDetail.transcodeCanceled")
    case "subtitle translation canceled":
      return t("library.progressDetail.subtitleTranslationCanceled")
    case "subtitle proofread canceled":
      return t("library.progressDetail.subtitleProofreadCanceled")
    case "subtitle qa review canceled":
      return t("library.progressDetail.subtitleQaReviewCanceled")
    case "operation canceled":
      return t("library.progressDetail.operationCanceled")
    case "download failed":
      return t("library.progressDetail.downloadFailed")
    case "transcode failed":
      return t("library.progressDetail.transcodeFailed")
    case "subtitle translation failed":
      return t("library.progressDetail.subtitleTranslationFailed")
    case "subtitle proofread failed":
      return t("library.progressDetail.subtitleProofreadFailed")
    case "subtitle qa review failed":
      return t("library.progressDetail.subtitleQaReviewFailed")
    case "operation failed":
      return t("library.progressDetail.operationFailed")
    default:
      break
  }

  if (
    containsProgressPhrase(
      normalized,
      "downloading webpage",
      "downloading api json",
      "downloading m3u8 information",
      "downloading m3u8 info",
      "downloading m3u8",
    )
  ) {
    return t("library.progress.fetchingMetadata")
  }
  if (containsProgressPhrase(normalized, "downloading thumbnail")) {
    return t("library.progress.downloadingThumbnail")
  }
  if (containsProgressPhrase(normalized, "downloading video")) {
    return t("library.progress.downloadingVideo")
  }
  if (containsProgressPhrase(normalized, "downloading audio")) {
    return t("library.progress.downloadingAudio")
  }
  if (
    containsProgressPhrase(
      normalized,
      "downloading subtitles",
      "downloading subtitle",
      "writing video subtitles",
      "writing subtitles",
    )
  ) {
    return t("library.progress.downloadingSubtitles")
  }
  if (containsProgressPhrase(normalized, "merging formats")) {
    return t("library.progress.muxing")
  }
  if (containsProgressPhrase(normalized, "deleting original")) {
    return t("library.progress.cleaningUp")
  }
  if (
    containsProgressPhrase(
      normalized,
      "post-process",
      "postprocessing",
      "extracting",
      "converting",
      "fixup",
      "remuxing",
    )
  ) {
    return t("library.progress.postProcessing")
  }

  const translatedChunkMatch = raw.match(TRANSLATED_CHUNK_PATTERN)
  if (translatedChunkMatch) {
    return formatTemplate(t("library.progressDetail.translatedChunk"), {
      current: translatedChunkMatch[1],
      total: translatedChunkMatch[2],
    })
  }

  const proofreadChunkMatch = raw.match(PROOFREAD_CHUNK_PATTERN)
  if (proofreadChunkMatch) {
    return formatTemplate(t("library.progressDetail.proofreadChunk"), {
      current: proofreadChunkMatch[1],
      total: proofreadChunkMatch[2],
    })
  }

  return raw
}

function normalizeProgressText(value: string) {
  return value.trim().toLowerCase().replace(/\s+/g, " ")
}

function containsProgressPhrase(value: string, ...phrases: string[]) {
  return phrases.some((phrase) => value.includes(phrase))
}

function translateProgressSpec(raw: string, t: TranslateFn) {
  const spec = parseProgressSpec(raw)
  if (!spec) {
    return ""
  }
  return formatTemplate(t(spec.key), spec.params)
}

function parseProgressSpec(raw: string) {
  if (!raw.startsWith(I18N_PROGRESS_PREFIX)) {
    return null
  }
  const payload = raw.slice(I18N_PROGRESS_PREFIX.length).trim()
  if (!payload) {
    return null
  }
  const [keyPart, query = ""] = payload.split("?", 2)
  const key = keyPart.trim()
  if (!key) {
    return null
  }
  const params: Record<string, string> = {}
  const searchParams = new URLSearchParams(query)
  for (const [name, value] of searchParams.entries()) {
    params[name] = value
  }
  return { key, params }
}
