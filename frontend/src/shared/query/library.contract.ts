import { z } from "zod"

import type {
  ApplySubtitleReviewSessionResult,
  CheckYtdlpOperationFailureResponse,
  DiscardSubtitleReviewSessionResult,
  FileEventRecordDTO,
  GenerateSubtitleStylePreviewASSResult,
  GenerateWorkspacePreviewVTTResult,
  GetYtdlpOperationLogResponse,
  LibraryDTO,
  LibraryFileDTO,
  LibraryHistoryRecordDTO,
  LibraryModuleConfigDTO,
  LibraryOperationDTO,
  OperationListItemDTO,
  ParseYtdlpDownloadResponse,
  PrepareYtdlpDownloadResponse,
  ResolveDomainIconResponse,
  RestoreSubtitleOriginalResult,
  SubtitleConvertResult,
  SubtitleExportResult,
  SubtitleFixTyposResult,
  SubtitleParseResult,
  SubtitleReviewSessionDetailDTO,
  SubtitleSaveResult,
  SubtitleValidateResult,
  TranscodePreset,
  WorkspaceProjectDTO,
  WorkspaceStateRecordDTO,
} from "@/shared/contracts/library"

function formatIssuePath(path: Array<string | number | symbol>): string {
  if (path.length === 0) {
    return "root"
  }
  return path
    .map((segment) => (typeof segment === "number" ? `[${segment}]` : String(segment)))
    .join(".")
}

function formatSchemaError(error: z.ZodError): string {
  return Array.from(
    new Set(error.issues.map((issue) => `${formatIssuePath(issue.path)}: ${issue.message}`)),
  ).join("; ")
}

function parseContract<T>(schema: z.ZodTypeAny, input: unknown, name: string): T {
  const result = schema.safeParse(input)
  if (!result.success) {
    throw new Error(`Invalid ${name} payload: ${formatSchemaError(result.error)}`)
  }
  return result.data as T
}

const stringArraySchema = z.array(z.string())
const unknownRecordSchema = z.record(z.string(), z.unknown())

const libraryCreateMetaSchema = z
  .object({
    source: z.string(),
    triggerOperationId: z.string().optional(),
    importBatchId: z.string().optional(),
    actor: z.string().optional(),
  })
  .passthrough()

const libraryWorkspaceConfigSchema = z
  .object({
    fastReadLatestState: z.boolean(),
  })
  .passthrough()

const libraryTranslateLanguageSchema = z
  .object({
    code: z.string(),
    label: z.string(),
    aliases: stringArraySchema.optional(),
  })
  .passthrough()

const libraryTranslateLanguagesConfigSchema = z
  .object({
    builtin: z.array(libraryTranslateLanguageSchema),
    custom: z.array(libraryTranslateLanguageSchema),
  })
  .passthrough()

const libraryGlossaryTermSchema = z
  .object({
    source: z.string(),
    target: z.string(),
    note: z.string().optional(),
  })
  .passthrough()

const libraryGlossaryProfileSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    category: z.string().optional(),
    description: z.string().optional(),
    sourceLanguage: z.string().optional(),
    targetLanguage: z.string().optional(),
    terms: z.array(libraryGlossaryTermSchema).optional(),
  })
  .passthrough()

const libraryPromptProfileSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    category: z.string().optional(),
    description: z.string().optional(),
    prompt: z.string(),
  })
  .passthrough()

const libraryLanguageAssetsConfigSchema = z
  .object({
    glossaryProfiles: z.array(libraryGlossaryProfileSchema),
    promptProfiles: z.array(libraryPromptProfileSchema),
  })
  .passthrough()

const libraryTaskRuntimeSettingsSchema = z
  .object({
    structuredOutputMode: z.string(),
    thinkingMode: z.string(),
    maxTokensFloor: z.number(),
    maxTokensCeiling: z.number(),
    retryTokenStep: z.number(),
  })
  .passthrough()

const libraryTaskRuntimeConfigSchema = z
  .object({
    translate: libraryTaskRuntimeSettingsSchema,
    proofread: libraryTaskRuntimeSettingsSchema,
  })
  .passthrough()

const assStyleSpecSchema = z
  .object({
    fontname: z.string(),
    fontFace: z.string().optional(),
    fontWeight: z.number().optional(),
    fontPostScriptName: z.string().optional(),
    fontsize: z.number(),
    primaryColour: z.string(),
    secondaryColour: z.string(),
    outlineColour: z.string(),
    backColour: z.string(),
    bold: z.boolean(),
    italic: z.boolean(),
    underline: z.boolean(),
    strikeOut: z.boolean(),
    scaleX: z.number(),
    scaleY: z.number(),
    spacing: z.number(),
    angle: z.number(),
    borderStyle: z.number(),
    outline: z.number(),
    shadow: z.number(),
    alignment: z.number(),
    marginL: z.number(),
    marginR: z.number(),
    marginV: z.number(),
    encoding: z.number(),
  })
  .passthrough()

const libraryMonoStyleSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    builtIn: z.boolean().optional(),
    basePlayResX: z.number(),
    basePlayResY: z.number(),
    baseAspectRatio: z.string(),
    sourceAssStyleName: z.string().optional(),
    style: assStyleSpecSchema,
  })
  .passthrough()

const libraryMonoStyleSnapshotSchema = z
  .object({
    sourceMonoStyleID: z.string().optional(),
    sourceMonoStyleName: z.string().optional(),
    name: z.string(),
    basePlayResX: z.number(),
    basePlayResY: z.number(),
    baseAspectRatio: z.string(),
    style: assStyleSpecSchema,
  })
  .passthrough()

const libraryBilingualStyleSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    builtIn: z.boolean().optional(),
    basePlayResX: z.number(),
    basePlayResY: z.number(),
    baseAspectRatio: z.string(),
    primary: libraryMonoStyleSnapshotSchema,
    secondary: libraryMonoStyleSnapshotSchema,
    layout: z
      .object({
        gap: z.number(),
        blockAnchor: z.number(),
      })
      .passthrough(),
  })
  .passthrough()

const subtitleExportConfigSchema = z
  .object({
    srt: z
      .object({
        encoding: z.string().optional(),
      })
      .passthrough()
      .optional(),
    vtt: z
      .object({
        kind: z.string().optional(),
        language: z.string().optional(),
      })
      .passthrough()
      .optional(),
    ass: z
      .object({
        playResX: z.number().optional(),
        playResY: z.number().optional(),
        title: z.string().optional(),
      })
      .passthrough()
      .optional(),
    itt: z
      .object({
        frameRate: z.number().optional(),
        frameRateMultiplier: z.string().optional(),
        language: z.string().optional(),
      })
      .passthrough()
      .optional(),
    fcpxml: z
      .object({
        frameDuration: z.string().optional(),
        width: z.number().optional(),
        height: z.number().optional(),
        colorSpace: z.string().optional(),
        version: z.string().optional(),
        libraryName: z.string().optional(),
        eventName: z.string().optional(),
        projectName: z.string().optional(),
        defaultLane: z.number().optional(),
        startTimecodeSeconds: z.number().optional(),
      })
      .passthrough()
      .optional(),
  })
  .passthrough()

const librarySubtitleStyleFontSchema = z
  .object({
    id: z.string(),
    family: z.string(),
    source: z.string().optional(),
    systemFamily: z.string().optional(),
    enabled: z.boolean().optional(),
  })
  .passthrough()

const librarySubtitleExportPresetSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    format: z.string().optional(),
    targetFormat: z.string().optional(),
    mediaStrategy: z.string().optional(),
    config: subtitleExportConfigSchema.optional(),
  })
  .passthrough()

const libraryRemoteFontManifestVariantSchema = z
  .object({
    name: z.string().optional(),
    weight: z.number().optional(),
    style: z.string().optional(),
    subsets: stringArraySchema.optional(),
    files: z.record(z.string(), z.string()).optional(),
  })
  .passthrough()

const libraryRemoteFontManifestFontSchema = z
  .object({
    name: z.string().optional(),
    family: z.string().optional(),
    license: z.string().optional(),
    licenseUrl: z.string().optional(),
    designer: z.string().optional(),
    foundry: z.string().optional(),
    version: z.string().optional(),
    description: z.string().optional(),
    categories: stringArraySchema.optional(),
    tags: stringArraySchema.optional(),
    popularity: z.number().optional(),
    lastModified: z.string().optional(),
    metadataUrl: z.string().optional(),
    sourceUrl: z.string().optional(),
    variants: z.array(libraryRemoteFontManifestVariantSchema).optional(),
    unicodeRanges: stringArraySchema.optional(),
    languages: stringArraySchema.optional(),
    sampleText: z.string().optional(),
  })
  .passthrough()

const libraryRemoteFontManifestSchema = z
  .object({
    sourceInfo: z
      .object({
        name: z.string().optional(),
        description: z.string().optional(),
        url: z.string().optional(),
        apiEndpoint: z.string().optional(),
        version: z.string().optional(),
        lastUpdated: z.string().optional(),
        totalFonts: z.number().optional(),
      })
      .passthrough()
      .optional(),
    fonts: z.record(z.string(), libraryRemoteFontManifestFontSchema).optional(),
  })
  .passthrough()

const librarySubtitleStyleSourceSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    kind: z.string().optional(),
    provider: z.string().optional(),
    url: z.string().optional(),
    prefix: z.string().optional(),
    filename: z.string().optional(),
    priority: z.number().optional(),
    builtIn: z.boolean().optional(),
    owner: z.string().optional(),
    repo: z.string().optional(),
    ref: z.string().optional(),
    manifestPath: z.string().optional(),
    remoteFontManifest: libraryRemoteFontManifestSchema.optional(),
    enabled: z.boolean().optional(),
    fontCount: z.number().optional(),
    syncStatus: z.string().optional(),
    lastSyncedAt: z.string().optional(),
    lastError: z.string().optional(),
  })
  .passthrough()

const librarySubtitleStyleConfigSchema = z
  .object({
    monoStyles: z.array(libraryMonoStyleSchema).optional(),
    bilingualStyles: z.array(libraryBilingualStyleSchema).optional(),
    sources: z.array(librarySubtitleStyleSourceSchema),
    fonts: z.array(librarySubtitleStyleFontSchema).optional(),
    subtitleExportPresets: z.array(librarySubtitleExportPresetSchema).optional(),
    defaults: z
      .object({
        monoStyleId: z.string().optional(),
        bilingualStyleId: z.string().optional(),
        subtitleExportPresetId: z.string().optional(),
      })
      .passthrough(),
  })
  .passthrough()

const libraryModuleConfigSchema = z
  .object({
    workspace: libraryWorkspaceConfigSchema,
    translateLanguages: libraryTranslateLanguagesConfigSchema,
    languageAssets: libraryLanguageAssetsConfigSchema,
    subtitleStyles: librarySubtitleStyleConfigSchema,
    taskRuntime: libraryTaskRuntimeConfigSchema,
  })
  .passthrough()

const libraryImportOriginSchema = z
  .object({
    batchId: z.string(),
    importPath: z.string(),
    importedAt: z.string(),
    keepSourceFile: z.boolean(),
  })
  .passthrough()

const libraryFileStorageSchema = z
  .object({
    mode: z.string(),
    localPath: z.string().optional(),
    documentId: z.string().optional(),
  })
  .passthrough()

const libraryFileOriginSchema = z
  .object({
    kind: z.string(),
    operationId: z.string().optional(),
    import: libraryImportOriginSchema.optional(),
  })
  .passthrough()

const libraryFileLineageSchema = z
  .object({
    rootFileId: z.string().optional(),
  })
  .passthrough()

const libraryMediaInfoSchema = z
  .object({
    format: z.string().optional(),
    codec: z.string().optional(),
    videoCodec: z.string().optional(),
    audioCodec: z.string().optional(),
    durationMs: z.number().optional(),
    width: z.number().optional(),
    height: z.number().optional(),
    frameRate: z.number().optional(),
    bitrateKbps: z.number().optional(),
    channels: z.number().optional(),
    sizeBytes: z.number().optional(),
    language: z.string().optional(),
    cueCount: z.number().optional(),
  })
  .passthrough()

const libraryFileStateSchema = z
  .object({
    status: z.string(),
    deleted: z.boolean(),
    archived: z.boolean(),
    lastError: z.string().optional(),
    lastChecked: z.string().optional(),
  })
  .passthrough()

const libraryFileSchema = z
  .object({
    id: z.string(),
    libraryId: z.string(),
    kind: z.string(),
    name: z.string(),
    displayLabel: z.string().optional(),
    storage: libraryFileStorageSchema,
    origin: libraryFileOriginSchema,
    lineage: libraryFileLineageSchema,
    latestOperationId: z.string().optional(),
    media: libraryMediaInfoSchema.optional(),
    state: libraryFileStateSchema,
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .passthrough()

const operationCorrelationSchema = z
  .object({
    requestId: z.string().optional(),
    runId: z.string().optional(),
    parentOperationId: z.string().optional(),
  })
  .passthrough()

const operationMetaSchema = z
  .object({
    platform: z.string().optional(),
    uploader: z.string().optional(),
    publishTime: z.string().optional(),
  })
  .passthrough()

const operationRequestPreviewSchema = z
  .object({
    url: z.string().optional(),
    caller: z.string().optional(),
    extractor: z.string().optional(),
    author: z.string().optional(),
    thumbnailUrl: z.string().optional(),
  })
  .passthrough()

const operationProgressSchema = z
  .object({
    stage: z.string().optional(),
    percent: z.number().optional(),
    current: z.number().optional(),
    total: z.number().optional(),
    speed: z.string().optional(),
    message: z.string().optional(),
    updatedAt: z.string().optional(),
  })
  .passthrough()

const operationOutputFileSchema = z
  .object({
    fileId: z.string(),
    kind: z.string(),
    format: z.string().optional(),
    sizeBytes: z.number().optional(),
    isPrimary: z.boolean().optional(),
    deleted: z.boolean().optional(),
  })
  .passthrough()

const operationMetricsSchema = z
  .object({
    fileCount: z.number(),
    totalSizeBytes: z.number().optional(),
    durationMs: z.number().optional(),
  })
  .passthrough()

const libraryOperationSchema = z
  .object({
    id: z.string(),
    libraryId: z.string(),
    kind: z.string(),
    status: z.string(),
    displayName: z.string(),
    correlation: operationCorrelationSchema,
    inputJson: z.string(),
    outputJson: z.string(),
    sourceDomain: z.string().optional(),
    sourceIcon: z.string().optional(),
    meta: operationMetaSchema,
    request: operationRequestPreviewSchema.optional(),
    progress: operationProgressSchema.optional(),
    outputFiles: z.array(operationOutputFileSchema).optional(),
    metrics: operationMetricsSchema,
    errorCode: z.string().optional(),
    errorMessage: z.string().optional(),
    createdAt: z.string(),
    startedAt: z.string().optional(),
    finishedAt: z.string().optional(),
  })
  .passthrough()

const operationListItemSchema = z
  .object({
    operationId: z.string(),
    libraryId: z.string(),
    libraryName: z.string().optional(),
    name: z.string(),
    kind: z.string(),
    status: z.string(),
    domain: z.string().optional(),
    sourceIcon: z.string().optional(),
    platform: z.string().optional(),
    uploader: z.string().optional(),
    publishTime: z.string().optional(),
    progress: operationProgressSchema.optional(),
    outputFiles: z.array(operationOutputFileSchema).optional(),
    metrics: operationMetricsSchema,
    startedAt: z.string().optional(),
    finishedAt: z.string().optional(),
    createdAt: z.string(),
  })
  .passthrough()

const libraryHistoryRecordSchema = z
  .object({
    recordId: z.string(),
    libraryId: z.string(),
    category: z.string(),
    action: z.string(),
    displayName: z.string(),
    status: z.string(),
    source: z
      .object({
        kind: z.string(),
        caller: z.string().optional(),
        runId: z.string().optional(),
        actor: z.string().optional(),
      })
      .passthrough(),
    refs: z
      .object({
        operationId: z.string().optional(),
        importBatchId: z.string().optional(),
        fileIds: stringArraySchema.optional(),
        fileEventIds: stringArraySchema.optional(),
      })
      .passthrough(),
    files: z.array(operationOutputFileSchema).optional(),
    metrics: operationMetricsSchema,
    importMeta: z
      .object({
        importPath: z.string().optional(),
        keepSourceFile: z.boolean(),
        importedAt: z.string(),
      })
      .passthrough()
      .optional(),
    operationMeta: z
      .object({
        kind: z.string(),
        errorCode: z.string().optional(),
        errorMessage: z.string().optional(),
      })
      .passthrough()
      .optional(),
    occurredAt: z.string(),
    createdAt: z.string(),
  })
  .passthrough()

const workspaceStateRecordSchema = z
  .object({
    id: z.string(),
    libraryId: z.string(),
    stateVersion: z.number(),
    stateJson: z.string(),
    operationId: z.string().optional(),
    createdAt: z.string(),
  })
  .passthrough()

const fileEventRecordSchema = z
  .object({
    id: z.string(),
    libraryId: z.string(),
    fileId: z.string(),
    operationId: z.string().optional(),
    eventType: z.string(),
    detail: z
      .object({
        cause: z
          .object({
            category: z.string(),
            operationId: z.string().optional(),
            importBatchId: z.string().optional(),
            actor: z.string().optional(),
          })
          .passthrough(),
        before: z
          .object({
            fileId: z.string(),
            kind: z.string(),
            name: z.string(),
            localPath: z.string().optional(),
            documentId: z.string().optional(),
          })
          .passthrough()
          .optional(),
        after: z
          .object({
            fileId: z.string(),
            kind: z.string(),
            name: z.string(),
            localPath: z.string().optional(),
            documentId: z.string().optional(),
          })
          .passthrough()
          .optional(),
        changes: z
          .array(
            z
              .object({
                field: z.string(),
                before: z.string().optional(),
                after: z.string().optional(),
              })
              .passthrough(),
          )
          .optional(),
        import: libraryImportOriginSchema.optional(),
      })
      .passthrough(),
    createdAt: z.string(),
  })
  .passthrough()

const libraryRecordsSchema = z
  .object({
    history: z.array(libraryHistoryRecordSchema),
    workspaceStateHead: workspaceStateRecordSchema.optional(),
    workspaceStates: z.array(workspaceStateRecordSchema),
    fileEvents: z.array(fileEventRecordSchema),
  })
  .passthrough()

const librarySchema = z
  .object({
    version: z.string(),
    id: z.string(),
    name: z.string(),
    createdAt: z.string(),
    updatedAt: z.string(),
    createdBy: libraryCreateMetaSchema,
    files: z.array(libraryFileSchema),
    records: libraryRecordsSchema,
  })
  .passthrough()

const workspaceTrackDisplaySchema = z
  .object({
    label: z.string(),
    hint: z.string().optional(),
    badges: stringArraySchema.optional(),
  })
  .passthrough()

const workspaceTaskSummarySchema = z
  .object({
    operationId: z.string(),
    kind: z.string(),
    status: z.string(),
    displayName: z.string(),
    stage: z.string().optional(),
    current: z.number().optional(),
    total: z.number().optional(),
    updatedAt: z.string().optional(),
  })
  .passthrough()

const workspaceTrackTasksSchema = z
  .object({
    translate: z.array(workspaceTaskSummarySchema).optional(),
    proofread: workspaceTaskSummarySchema.optional(),
    qa: workspaceTaskSummarySchema.optional(),
  })
  .passthrough()

const workspaceTrackPendingReviewSchema = z
  .object({
    sessionId: z.string(),
    kind: z.string(),
    status: z.string(),
    sourceRevisionId: z.string(),
    candidateRevisionId: z.string(),
    changedCueCount: z.number(),
    blockedActions: stringArraySchema.optional(),
  })
  .passthrough()

const workspaceProjectSchema = z
  .object({
    version: z.string(),
    libraryId: z.string(),
    title: z.string(),
    updatedAt: z.string(),
    viewStateHead: workspaceStateRecordSchema.optional(),
    videoTracks: z.array(
      z
        .object({
          trackId: z.string(),
          file: libraryFileSchema,
          display: workspaceTrackDisplaySchema,
        })
        .passthrough(),
    ),
    subtitleTracks: z.array(
      z
        .object({
          trackId: z.string(),
          role: z.string(),
          file: libraryFileSchema,
          display: workspaceTrackDisplaySchema,
          runningTasks: workspaceTrackTasksSchema,
          pendingReview: workspaceTrackPendingReviewSchema.optional(),
        })
        .passthrough(),
    ),
    subtitleMonoStyle: libraryMonoStyleSchema.optional(),
    subtitleLingualStyle: libraryBilingualStyleSchema.optional(),
  })
  .passthrough()

const subtitleCueSchema = z
  .object({
    index: z.number(),
    start: z.string(),
    end: z.string(),
    text: z.string(),
  })
  .passthrough()

const subtitleDocumentSchema = z
  .object({
    format: z.string(),
    cues: z.array(subtitleCueSchema),
    metadata: unknownRecordSchema.optional(),
  })
  .passthrough()

const subtitleReviewSessionDetailSchema = z
  .object({
    sessionId: z.string(),
    libraryId: z.string(),
    fileId: z.string(),
    kind: z.string(),
    status: z.string(),
    sourceRevisionId: z.string(),
    candidateRevisionId: z.string(),
    appliedRevisionId: z.string().optional(),
    changedCueCount: z.number(),
    sourceDocument: subtitleDocumentSchema,
    candidateDocument: subtitleDocumentSchema,
    suggestions: z
      .array(
        z
          .object({
            cueIndex: z.number(),
            originalText: z.string(),
            suggestedText: z.string(),
            categories: stringArraySchema.optional(),
            reason: z.string().optional(),
            sourceCode: z.string().optional(),
            severity: z.string().optional(),
          })
          .passthrough(),
      )
      .optional(),
  })
  .passthrough()

const generateWorkspacePreviewVttResultSchema = z
  .object({
    vttContent: z.string(),
  })
  .passthrough()

const generateSubtitleStylePreviewAssResultSchema = z
  .object({
    assContent: z.string(),
  })
  .passthrough()

const applySubtitleReviewSessionResultSchema = z
  .object({
    sessionId: z.string(),
    fileId: z.string(),
    appliedRevisionId: z.string(),
    changedCueCount: z.number(),
  })
  .passthrough()

const discardSubtitleReviewSessionResultSchema = z
  .object({
    sessionId: z.string(),
    status: z.string(),
  })
  .passthrough()

const checkYtdlpOperationFailureResponseSchema = z
  .object({
    items: z.array(
      z
        .object({
          id: z.string(),
          label: z.string(),
          status: z.string(),
          message: z.string().optional(),
          action: z.string().optional(),
        })
        .passthrough(),
    ),
    canRetry: z.boolean(),
  })
  .passthrough()

const getYtdlpOperationLogResponseSchema = z
  .object({
    operationId: z.string(),
    path: z.string().optional(),
    content: z.string().optional(),
    truncated: z.boolean().optional(),
  })
  .passthrough()

const prepareYtdlpDownloadResponseSchema = z
  .object({
    url: z.string(),
    domain: z.string(),
    icon: z.string().optional(),
    connectorId: z.string().optional(),
    connectorAvailable: z.boolean(),
    reachable: z.boolean().optional(),
  })
  .passthrough()

const resolveDomainIconResponseSchema = z
  .object({
    domain: z.string().optional(),
    icon: z.string().optional(),
  })
  .passthrough()

const parseYtdlpDownloadResponseSchema = z
  .object({
    title: z.string().optional(),
    domain: z.string().optional(),
    extractor: z.string().optional(),
    author: z.string().optional(),
    thumbnailUrl: z.string().optional(),
    formats: z.array(
      z
        .object({
          id: z.string(),
          label: z.string(),
          hasVideo: z.boolean(),
          hasAudio: z.boolean(),
          ext: z.string().optional(),
          height: z.number().optional(),
          vcodec: z.string().optional(),
          acodec: z.string().optional(),
          filesize: z.number().optional(),
        })
        .passthrough(),
    ),
    subtitles: z.array(
      z
        .object({
          id: z.string(),
          language: z.string(),
          name: z.string().optional(),
          isAuto: z.boolean().optional(),
          ext: z.string().optional(),
        })
        .passthrough(),
    ),
  })
  .passthrough()

const transcodePresetSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    outputType: z.enum(["video", "audio"]),
    container: z.string(),
    videoCodec: z.string().optional(),
    audioCodec: z.string().optional(),
    qualityMode: z.enum(["crf", "bitrate"]).optional(),
    crf: z.number().optional(),
    bitrateKbps: z.number().optional(),
    audioBitrateKbps: z.number().optional(),
    scale: z.enum(["original", "2160p", "1080p", "720p", "480p", "custom"]).optional(),
    width: z.number().optional(),
    height: z.number().optional(),
    ffmpegPreset: z.enum(["ultrafast", "fast", "medium", "slow"]).optional(),
    allowUpscale: z.boolean().optional(),
    requiresVideo: z.boolean().optional(),
    requiresAudio: z.boolean().optional(),
    isBuiltin: z.boolean().optional(),
    createdAt: z.string().optional(),
    updatedAt: z.string().optional(),
  })
  .passthrough()

const subtitleParseResultSchema = z
  .object({
    format: z.string(),
    cueCount: z.number(),
    document: subtitleDocumentSchema,
    warnings: stringArraySchema.optional(),
  })
  .passthrough()

const subtitleConvertResultSchema = z
  .object({
    targetFormat: z.string(),
    content: z.string(),
    warnings: stringArraySchema.optional(),
  })
  .passthrough()

const subtitleExportResultSchema = z
  .object({
    exportPath: z.string(),
    format: z.string(),
    bytes: z.number(),
  })
  .passthrough()

const subtitleValidateResultSchema = z
  .object({
    valid: z.boolean(),
    issueCount: z.number(),
    issues: z
      .array(
        z
          .object({
            severity: z.string(),
            code: z.string(),
            message: z.string(),
            cueIndex: z.number().optional(),
          })
          .passthrough(),
      )
      .optional(),
  })
  .passthrough()

const subtitleFixTyposResultSchema = z
  .object({
    format: z.string(),
    content: z.string(),
    changeCount: z.number(),
    document: subtitleDocumentSchema,
  })
  .passthrough()

const subtitleSaveResultSchema = z
  .object({
    path: z.string().optional(),
    format: z.string(),
    bytes: z.number(),
  })
  .passthrough()

const restoreSubtitleOriginalResultSchema = z
  .object({
    fileId: z.string().optional(),
    format: z.string(),
    bytes: z.number(),
  })
  .passthrough()

export function parseLibraryListPayload(input: unknown): LibraryDTO[] {
  return parseContract<LibraryDTO[]>(z.array(librarySchema), input, "library list")
}

export function parseLibraryPayload(input: unknown): LibraryDTO {
  return parseContract<LibraryDTO>(librarySchema, input, "library")
}

export function parseLibraryModuleConfigPayload(input: unknown): LibraryModuleConfigDTO {
  return parseContract<LibraryModuleConfigDTO>(libraryModuleConfigSchema, input, "library module config")
}

export function parseOperationListPayload(input: unknown): OperationListItemDTO[] {
  return parseContract<OperationListItemDTO[]>(z.array(operationListItemSchema), input, "operation list")
}

export function parseLibraryOperationPayload(input: unknown): LibraryOperationDTO {
  return parseContract<LibraryOperationDTO>(libraryOperationSchema, input, "library operation")
}

export function parseLibraryHistoryPayload(input: unknown): LibraryHistoryRecordDTO[] {
  return parseContract<LibraryHistoryRecordDTO[]>(
    z.array(libraryHistoryRecordSchema),
    input,
    "library history",
  )
}

export function parseFileEventPayload(input: unknown): FileEventRecordDTO[] {
  return parseContract<FileEventRecordDTO[]>(z.array(fileEventRecordSchema), input, "file events")
}

export function parseWorkspaceStatePayload(input: unknown): WorkspaceStateRecordDTO {
  return parseContract<WorkspaceStateRecordDTO>(workspaceStateRecordSchema, input, "workspace state")
}

export function parseWorkspaceProjectPayload(input: unknown): WorkspaceProjectDTO {
  return parseContract<WorkspaceProjectDTO>(workspaceProjectSchema, input, "workspace project")
}

export function parseGenerateWorkspacePreviewPayload(input: unknown): GenerateWorkspacePreviewVTTResult {
  return parseContract<GenerateWorkspacePreviewVTTResult>(
    generateWorkspacePreviewVttResultSchema,
    input,
    "workspace preview",
  )
}

export function parseGenerateSubtitleStylePreviewPayload(
  input: unknown,
): GenerateSubtitleStylePreviewASSResult {
  return parseContract<GenerateSubtitleStylePreviewASSResult>(
    generateSubtitleStylePreviewAssResultSchema,
    input,
    "subtitle style preview",
  )
}

export function parseSubtitleReviewSessionPayload(input: unknown): SubtitleReviewSessionDetailDTO {
  return parseContract<SubtitleReviewSessionDetailDTO>(
    subtitleReviewSessionDetailSchema,
    input,
    "subtitle review session",
  )
}

export function parseApplySubtitleReviewSessionPayload(input: unknown): ApplySubtitleReviewSessionResult {
  return parseContract<ApplySubtitleReviewSessionResult>(
    applySubtitleReviewSessionResultSchema,
    input,
    "apply subtitle review session",
  )
}

export function parseDiscardSubtitleReviewSessionPayload(
  input: unknown,
): DiscardSubtitleReviewSessionResult {
  return parseContract<DiscardSubtitleReviewSessionResult>(
    discardSubtitleReviewSessionResultSchema,
    input,
    "discard subtitle review session",
  )
}

export function parsePrepareYtdlpDownloadPayload(input: unknown): PrepareYtdlpDownloadResponse {
  return parseContract<PrepareYtdlpDownloadResponse>(
    prepareYtdlpDownloadResponseSchema,
    input,
    "prepare ytdlp download",
  )
}

export function parseParseYtdlpDownloadPayload(input: unknown): ParseYtdlpDownloadResponse {
  return parseContract<ParseYtdlpDownloadResponse>(
    parseYtdlpDownloadResponseSchema,
    input,
    "parse ytdlp download",
  )
}

export function parseResolveDomainIconPayload(input: unknown): ResolveDomainIconResponse {
  return parseContract<ResolveDomainIconResponse>(
    resolveDomainIconResponseSchema,
    input,
    "resolve domain icon",
  )
}

export function parseCheckYtdlpOperationFailurePayload(
  input: unknown,
): CheckYtdlpOperationFailureResponse {
  return parseContract<CheckYtdlpOperationFailureResponse>(
    checkYtdlpOperationFailureResponseSchema,
    input,
    "check ytdlp operation failure",
  )
}

export function parseGetYtdlpOperationLogPayload(input: unknown): GetYtdlpOperationLogResponse {
  return parseContract<GetYtdlpOperationLogResponse>(
    getYtdlpOperationLogResponseSchema,
    input,
    "ytdlp operation log",
  )
}

export function parseLibraryFilePayload(input: unknown): LibraryFileDTO {
  return parseContract<LibraryFileDTO>(libraryFileSchema, input, "library file")
}

export function parseTranscodePresetListPayload(input: unknown): TranscodePreset[] {
  return parseContract<TranscodePreset[]>(z.array(transcodePresetSchema), input, "transcode preset list")
}

export function parseTranscodePresetPayload(input: unknown): TranscodePreset {
  return parseContract<TranscodePreset>(transcodePresetSchema, input, "transcode preset")
}

export function parseSubtitleParsePayload(input: unknown): SubtitleParseResult {
  return parseContract<SubtitleParseResult>(subtitleParseResultSchema, input, "subtitle parse")
}

export function parseSubtitleConvertPayload(input: unknown): SubtitleConvertResult {
  return parseContract<SubtitleConvertResult>(subtitleConvertResultSchema, input, "subtitle convert")
}

export function parseSubtitleExportPayload(input: unknown): SubtitleExportResult {
  return parseContract<SubtitleExportResult>(subtitleExportResultSchema, input, "subtitle export")
}

export function parseSubtitleValidatePayload(input: unknown): SubtitleValidateResult {
  return parseContract<SubtitleValidateResult>(subtitleValidateResultSchema, input, "subtitle validate")
}

export function parseSubtitleFixTyposPayload(input: unknown): SubtitleFixTyposResult {
  return parseContract<SubtitleFixTyposResult>(subtitleFixTyposResultSchema, input, "subtitle fix typos")
}

export function parseSubtitleSavePayload(input: unknown): SubtitleSaveResult {
  return parseContract<SubtitleSaveResult>(subtitleSaveResultSchema, input, "subtitle save")
}

export function parseRestoreSubtitleOriginalPayload(input: unknown): RestoreSubtitleOriginalResult {
  return parseContract<RestoreSubtitleOriginalResult>(
    restoreSubtitleOriginalResultSchema,
    input,
    "restore subtitle original",
  )
}
