import type {
  FFmpegSpeedPreset,
  TranscodePresetOutputType,
  TranscodeQualityMode,
  TranscodeScaleMode,
} from "../store/transcodePresets"

export const LIBRARY_SCHEMA_VERSION = "current"

export interface LibraryCreateMetaDTO {
  source: string
  triggerOperationId?: string
  importBatchId?: string
  actor?: string
}

export interface LibraryWorkspaceConfigDTO {
  fastReadLatestState: boolean
}

export interface LibraryTranslateLanguageDTO {
  code: string
  label: string
  aliases?: string[]
}

export interface LibraryTranslateLanguagesConfigDTO {
  builtin: LibraryTranslateLanguageDTO[]
  custom: LibraryTranslateLanguageDTO[]
}

export interface LibraryGlossaryTermDTO {
  source: string
  target: string
  note?: string
}

export interface LibraryGlossaryProfileDTO {
  id: string
  name: string
  category?: string
  description?: string
  sourceLanguage?: string
  targetLanguage?: string
  terms?: LibraryGlossaryTermDTO[]
}

export interface LibraryPromptProfileDTO {
  id: string
  name: string
  category?: string
  description?: string
  prompt: string
}

export interface LibraryLanguageAssetsConfigDTO {
  glossaryProfiles: LibraryGlossaryProfileDTO[]
  promptProfiles: LibraryPromptProfileDTO[]
}

export interface LibraryTaskRuntimeSettingsDTO {
  structuredOutputMode: string
  thinkingMode: string
  maxTokensFloor: number
  maxTokensCeiling: number
  retryTokenStep: number
}

export interface LibraryTaskRuntimeConfigDTO {
  translate: LibraryTaskRuntimeSettingsDTO
  proofread: LibraryTaskRuntimeSettingsDTO
}

export interface LibrarySubtitleStyleDefaultsDTO {
  monoStyleId?: string
  bilingualStyleId?: string
  subtitleExportPresetId?: string
}

export interface LibrarySubtitleStyleDocumentAnalysisDTO {
  detectedFormat?: string
  scriptType?: string
  playResX?: number
  playResY?: number
  styleCount?: number
  dialogueCount?: number
  commentCount?: number
  styleNames?: string[]
  fonts?: string[]
  featureFlags?: string[]
  validationIssues?: string[]
}

export interface LibrarySubtitleStyleDocumentDTO {
  id: string
  name: string
  description?: string
  source?: string
  sourceRef?: string
  version?: string
  enabled?: boolean
  format?: string
  content?: string
  analysis?: LibrarySubtitleStyleDocumentAnalysisDTO
}

export interface AssStyleSpecDTO {
  fontname: string
  fontFace?: string
  fontWeight?: number
  fontPostScriptName?: string
  fontsize: number
  primaryColour: string
  secondaryColour: string
  outlineColour: string
  backColour: string
  bold: boolean
  italic: boolean
  underline: boolean
  strikeOut: boolean
  scaleX: number
  scaleY: number
  spacing: number
  angle: number
  borderStyle: number
  outline: number
  shadow: number
  alignment: number
  marginL: number
  marginR: number
  marginV: number
  encoding: number
}

export interface LibraryMonoStyleDTO {
  id: string
  name: string
  builtIn?: boolean
  basePlayResX: number
  basePlayResY: number
  baseAspectRatio: string
  sourceAssStyleName?: string
  style: AssStyleSpecDTO
}

export interface LibraryMonoStyleSnapshotDTO {
  sourceMonoStyleID?: string
  sourceMonoStyleName?: string
  name: string
  basePlayResX: number
  basePlayResY: number
  baseAspectRatio: string
  style: AssStyleSpecDTO
}

export interface LibraryBilingualLayoutDTO {
  gap: number
  blockAnchor: number
}

export interface LibraryBilingualStyleDTO {
  id: string
  name: string
  builtIn?: boolean
  basePlayResX: number
  basePlayResY: number
  baseAspectRatio: string
  primary: LibraryMonoStyleSnapshotDTO
  secondary: LibraryMonoStyleSnapshotDTO
  layout: LibraryBilingualLayoutDTO
}

export interface DCSSPFileDTO {
  format: string
  schemaVersion: number
  type: string
  id: string
  name: string
  author?: string
  homepage?: string
  description?: string
  license?: string
  tags?: string[]
  createdAt?: string
  updatedAt?: string
  appVersion?: string
  payload?: unknown
}

export interface LibraryRemoteFontManifestInfoDTO {
  name?: string
  description?: string
  url?: string
  apiEndpoint?: string
  version?: string
  lastUpdated?: string
  totalFonts?: number
}

export interface LibraryRemoteFontManifestVariantDTO {
  name?: string
  weight?: number
  style?: string
  subsets?: string[]
  files?: Record<string, string>
}

export interface LibraryRemoteFontManifestFontDTO {
  name?: string
  family?: string
  license?: string
  licenseUrl?: string
  designer?: string
  foundry?: string
  version?: string
  description?: string
  categories?: string[]
  tags?: string[]
  popularity?: number
  lastModified?: string
  metadataUrl?: string
  sourceUrl?: string
  variants?: LibraryRemoteFontManifestVariantDTO[]
  unicodeRanges?: string[]
  languages?: string[]
  sampleText?: string
}

export interface LibraryRemoteFontManifestDTO {
  sourceInfo?: LibraryRemoteFontManifestInfoDTO
  fonts?: Record<string, LibraryRemoteFontManifestFontDTO>
}

export interface LibrarySubtitleStyleSourceDTO {
  id: string
  name: string
  kind?: string
  provider?: string
  url?: string
  prefix?: string
  filename?: string
  priority?: number
  builtIn?: boolean
  owner?: string
  repo?: string
  ref?: string
  manifestPath?: string
  remoteFontManifest?: LibraryRemoteFontManifestDTO
  enabled?: boolean
  fontCount?: number
  syncStatus?: string
  lastSyncedAt?: string
  lastError?: string
}

export interface LibrarySubtitleStyleFontDTO {
  id: string
  family: string
  source?: string
  systemFamily?: string
  enabled?: boolean
}

export interface LibrarySubtitleExportPresetDTO {
  id: string
  name: string
  format?: string
  targetFormat?: string
  mediaStrategy?: string
  config?: SubtitleExportConfig
}

export interface LibrarySubtitleStyleConfigDTO {
  monoStyles?: LibraryMonoStyleDTO[]
  bilingualStyles?: LibraryBilingualStyleDTO[]
  sources: LibrarySubtitleStyleSourceDTO[]
  fonts?: LibrarySubtitleStyleFontDTO[]
  subtitleExportPresets?: LibrarySubtitleExportPresetDTO[]
  defaults: LibrarySubtitleStyleDefaultsDTO
}

export interface GenerateSubtitleStylePreviewASSRequest {
  type: "mono" | "bilingual" | string
  mono?: LibraryMonoStyleDTO
  bilingual?: LibraryBilingualStyleDTO
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  primaryText?: string
  secondaryText?: string
}

export interface GenerateSubtitleStylePreviewASSResult {
  assContent: string
  referencedFontFamilies?: string[]
}

export interface GenerateSubtitleStylePreviewVTTRequest {
  type: "mono" | "bilingual" | string
  mono?: LibraryMonoStyleDTO
  bilingual?: LibraryBilingualStyleDTO
  fontMappings?: LibrarySubtitleStyleFontDTO[]
  primaryText?: string
  secondaryText?: string
  previewWidth?: number
  previewHeight?: number
}

export interface GenerateSubtitleStylePreviewVTTResult {
  vttContent: string
}

export interface ParseSubtitleStyleImportRequest {
  content: string
  format?: string
  filename?: string
}

export interface ParseSubtitleStyleImportResult {
  importFormat: string
  dcssp?: DCSSPFileDTO
  monoStyles?: LibraryMonoStyleDTO[]
  bilingualStyle?: LibraryBilingualStyleDTO
  detectedRatio?: string
  normalizedPlayResX?: number
  normalizedPlayResY?: number
  warnings?: string[]
}

export interface ExportSubtitleStylePresetRequest {
  directoryPath: string
  type: "mono" | "bilingual" | string
  mono?: LibraryMonoStyleDTO
  bilingual?: LibraryBilingualStyleDTO
}

export interface ExportSubtitleStylePresetResult {
  exportPath: string
  fileName: string
}

export interface LibraryModuleConfigDTO {
  workspace: LibraryWorkspaceConfigDTO
  translateLanguages: LibraryTranslateLanguagesConfigDTO
  languageAssets: LibraryLanguageAssetsConfigDTO
  subtitleStyles: LibrarySubtitleStyleConfigDTO
  taskRuntime: LibraryTaskRuntimeConfigDTO
}

export interface LibraryFileStorageDTO {
  mode: "local_path" | "db_document" | "hybrid" | string
  localPath?: string
  documentId?: string
}

export interface LibraryImportOriginDTO {
  batchId: string
  importPath: string
  importedAt: string
  keepSourceFile: boolean
}

export interface LibraryFileOriginDTO {
  kind: string
  operationId?: string
  import?: LibraryImportOriginDTO
}

export interface LibraryFileLineageDTO {
  rootFileId?: string
}

export interface LibraryMediaInfoDTO {
  format?: string
  codec?: string
  videoCodec?: string
  audioCodec?: string
  durationMs?: number
  width?: number
  height?: number
  frameRate?: number
  bitrateKbps?: number
  channels?: number
  sizeBytes?: number
  language?: string
  cueCount?: number
}

export interface LibraryFileStateDTO {
  status: string
  deleted: boolean
  archived: boolean
  lastError?: string
  lastChecked?: string
}

export interface LibraryFileDTO {
  id: string
  libraryId: string
  kind: string
  name: string
  displayLabel?: string
  storage: LibraryFileStorageDTO
  origin: LibraryFileOriginDTO
  lineage: LibraryFileLineageDTO
  latestOperationId?: string
  media?: LibraryMediaInfoDTO
  state: LibraryFileStateDTO
  createdAt: string
  updatedAt: string
}

export interface OperationCorrelationDTO {
  requestId?: string
  runId?: string
  parentOperationId?: string
}

export interface OperationMetaDTO {
  platform?: string
  uploader?: string
  publishTime?: string
}

export interface OperationRequestPreviewDTO {
  url?: string
  caller?: string
  extractor?: string
  author?: string
  thumbnailUrl?: string
}

export interface OperationProgressDTO {
  stage?: string
  percent?: number
  current?: number
  total?: number
  speed?: string
  message?: string
  updatedAt?: string
}

export interface OperationOutputFileDTO {
  fileId: string
  kind: string
  format?: string
  sizeBytes?: number
  isPrimary?: boolean
  deleted?: boolean
}

export interface OperationMetricsDTO {
  fileCount: number
  totalSizeBytes?: number
  durationMs?: number
}

export interface LibraryOperationDTO {
  id: string
  libraryId: string
  kind: string
  status: string
  displayName: string
  correlation: OperationCorrelationDTO
  inputJson: string
  outputJson: string
  sourceDomain?: string
  sourceIcon?: string
  meta: OperationMetaDTO
  request?: OperationRequestPreviewDTO
  progress?: OperationProgressDTO
  outputFiles?: OperationOutputFileDTO[]
  metrics: OperationMetricsDTO
  errorCode?: string
  errorMessage?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
}

export interface OperationListItemDTO {
  operationId: string
  libraryId: string
  libraryName?: string
  name: string
  kind: string
  status: string
  domain?: string
  sourceIcon?: string
  platform?: string
  uploader?: string
  publishTime?: string
  progress?: OperationProgressDTO
  outputFiles?: OperationOutputFileDTO[]
  metrics: OperationMetricsDTO
  startedAt?: string
  finishedAt?: string
  createdAt: string
}

export interface LibraryHistoryRecordSourceDTO {
  kind: string
  caller?: string
  runId?: string
  actor?: string
}

export interface LibraryHistoryRecordRefsDTO {
  operationId?: string
  importBatchId?: string
  fileIds?: string[]
  fileEventIds?: string[]
}

export interface LibraryImportRecordMetaDTO {
  importPath?: string
  keepSourceFile: boolean
  importedAt: string
}

export interface LibraryOperationRecordMetaDTO {
  kind: string
  errorCode?: string
  errorMessage?: string
}

export interface LibraryHistoryRecordDTO {
  recordId: string
  libraryId: string
  category: "operation" | "import" | string
  action: string
  displayName: string
  status: string
  source: LibraryHistoryRecordSourceDTO
  refs: LibraryHistoryRecordRefsDTO
  files?: OperationOutputFileDTO[]
  metrics: OperationMetricsDTO
  importMeta?: LibraryImportRecordMetaDTO
  operationMeta?: LibraryOperationRecordMetaDTO
  occurredAt: string
  createdAt: string
}

export interface WorkspaceStateRecordDTO {
  id: string
  libraryId: string
  stateVersion: number
  stateJson: string
  operationId?: string
  createdAt: string
}

export interface WorkspaceTrackDisplayDTO {
  label: string
  hint?: string
  badges?: string[]
}

export interface WorkspaceTaskSummaryDTO {
  operationId: string
  kind: string
  status: string
  displayName: string
  stage?: string
  current?: number
  total?: number
  updatedAt?: string
}

export interface WorkspaceTrackTasksDTO {
  translate?: WorkspaceTaskSummaryDTO[]
  proofread?: WorkspaceTaskSummaryDTO
  qa?: WorkspaceTaskSummaryDTO
}

export interface WorkspaceTrackPendingReviewDTO {
  sessionId: string
  kind: "proofread" | "qa" | string
  status: string
  sourceRevisionId: string
  candidateRevisionId: string
  changedCueCount: number
  blockedActions?: string[]
}

export interface WorkspaceVideoTrackDTO {
  trackId: string
  file: LibraryFileDTO
  display: WorkspaceTrackDisplayDTO
}

export interface WorkspaceSubtitleTrackDTO {
  trackId: string
  role: "source" | "translation" | "reference" | string
  file: LibraryFileDTO
  display: WorkspaceTrackDisplayDTO
  runningTasks: WorkspaceTrackTasksDTO
  pendingReview?: WorkspaceTrackPendingReviewDTO
}

export interface WorkspaceProjectDTO {
  version: string
  libraryId: string
  title: string
  updatedAt: string
  viewStateHead?: WorkspaceStateRecordDTO
  videoTracks: WorkspaceVideoTrackDTO[]
  subtitleTracks: WorkspaceSubtitleTrackDTO[]
  subtitleMonoStyle?: LibraryMonoStyleDTO
  subtitleLingualStyle?: LibraryBilingualStyleDTO
}

export interface GenerateWorkspacePreviewASSRequest {
  libraryId: string
  displayMode?: "mono" | "bilingual" | "single" | "dual" | string
  primarySubtitleTrackId: string
  secondarySubtitleTrackId?: string
}

export interface GenerateWorkspacePreviewASSResult {
  assContent: string
  referencedFontFamilies?: string[]
}

export interface FileEventCauseDTO {
  category: string
  operationId?: string
  importBatchId?: string
  actor?: string
}

export interface FileEventFileSnapshotDTO {
  fileId: string
  kind: string
  name: string
  localPath?: string
  documentId?: string
}

export interface FileFieldChangeDTO {
  field: string
  before?: string
  after?: string
}

export interface FileEventDetailDTO {
  cause: FileEventCauseDTO
  before?: FileEventFileSnapshotDTO
  after?: FileEventFileSnapshotDTO
  changes?: FileFieldChangeDTO[]
  import?: LibraryImportOriginDTO
}

export interface FileEventRecordDTO {
  id: string
  libraryId: string
  fileId: string
  operationId?: string
  eventType: string
  detail: FileEventDetailDTO
  createdAt: string
}

export interface LibraryRecordsDTO {
  history: LibraryHistoryRecordDTO[]
  workspaceStateHead?: WorkspaceStateRecordDTO
  workspaceStates: WorkspaceStateRecordDTO[]
  fileEvents: FileEventRecordDTO[]
}

export interface LibraryDTO {
  version: typeof LIBRARY_SCHEMA_VERSION | string
  id: string
  name: string
  createdAt: string
  updatedAt: string
  createdBy: LibraryCreateMetaDTO
  files: LibraryFileDTO[]
  records: LibraryRecordsDTO
}

export interface GetLibraryRequest {
  libraryId: string
}

export interface RenameLibraryRequest {
  libraryId: string
  name: string
}

export interface DeleteLibraryRequest {
  libraryId: string
}

export interface UpdateLibraryModuleConfigRequest {
  config: LibraryModuleConfigDTO
}

export interface ListOperationsRequest {
  libraryId?: string
  status?: string[]
  kinds?: string[]
  query?: string
  limit?: number
  offset?: number
}

export interface GetOperationRequest {
  operationId: string
}

export interface CancelOperationRequest {
  operationId: string
}

export interface ResumeOperationRequest {
  operationId: string
}

export interface DeleteOperationRequest {
  operationId: string
  cascadeFiles?: boolean
}

export interface DeleteOperationsRequest {
  operationIds: string[]
  cascadeFiles?: boolean
}

export interface DeleteFileRequest {
  fileId: string
  deleteFiles?: boolean
}

export interface DeleteFilesRequest {
  fileIds: string[]
  deleteFiles?: boolean
}

export interface ListLibraryHistoryRequest {
  libraryId: string
  categories?: string[]
  actions?: string[]
  limit?: number
  offset?: number
}

export interface ListFileEventsRequest {
  libraryId: string
  limit?: number
  offset?: number
}

export interface SaveWorkspaceStateRequest {
  libraryId: string
  stateJson: string
  operationId?: string
}

export interface GetWorkspaceStateRequest {
  libraryId: string
}

export interface GetWorkspaceProjectRequest {
  libraryId: string
}

export interface GetSubtitleReviewSessionRequest {
  sessionId: string
}

export interface SubtitleReviewCueDecisionDTO {
  cueIndex: number
  action: string
}

export interface ApplySubtitleReviewSessionRequest {
  sessionId: string
  decisions?: SubtitleReviewCueDecisionDTO[]
}

export interface ApplySubtitleReviewSessionResult {
  sessionId: string
  fileId: string
  appliedRevisionId: string
  changedCueCount: number
}

export interface DiscardSubtitleReviewSessionRequest {
  sessionId: string
}

export interface DiscardSubtitleReviewSessionResult {
  sessionId: string
  status: string
}

export interface OpenFileLocationRequest {
  fileId: string
}

export interface OpenPathRequest {
  path: string
}

export interface CreateYtdlpJobRequest {
  url: string
  libraryId?: string
  title?: string
  extractor?: string
  author?: string
  thumbnailUrl?: string
  writeThumbnail?: boolean
  cookiesPath?: string
  source?: string
  caller?: string
  sessionKey?: string
  runId?: string
  retryOf?: string
  retryCount?: number
  mode?: string
  logPolicy?: string
  quality?: string
  formatId?: string
  audioFormatId?: string
  subtitleLangs?: string[]
  subtitleAuto?: boolean
  subtitleAll?: boolean
  subtitleFormat?: string
  transcodePresetId?: string
  deleteSourceFileAfterTranscode?: boolean
  connectorId?: string
  useConnector?: boolean
}

export interface CheckYtdlpOperationFailureRequest {
  operationId: string
}

export interface CheckYtdlpOperationFailureItem {
  id: string
  label: string
  status: string
  message?: string
  action?: string
}

export interface CheckYtdlpOperationFailureResponse {
  items: CheckYtdlpOperationFailureItem[]
  canRetry: boolean
}

export interface RetryYtdlpOperationRequest {
  operationId: string
  source?: string
  caller?: string
  runId?: string
}

export interface GetYtdlpOperationLogRequest {
  operationId: string
  maxBytes?: number
  tailLines?: number
}

export interface GetYtdlpOperationLogResponse {
  operationId: string
  path?: string
  content?: string
  truncated?: boolean
}

export interface PrepareYtdlpDownloadRequest {
  url: string
}

export interface PrepareYtdlpDownloadResponse {
  url: string
  domain: string
  icon?: string
  connectorId?: string
  connectorAvailable: boolean
  reachable?: boolean
}

export interface ResolveDomainIconRequest {
  domain?: string
  url?: string
}

export interface ResolveDomainIconResponse {
  domain?: string
  icon?: string
}

export interface ParseYtdlpDownloadRequest {
  url: string
  connectorId?: string
  useConnector?: boolean
}

export interface YtdlpFormatOption {
  id: string
  label: string
  hasVideo: boolean
  hasAudio: boolean
  ext?: string
  height?: number
  vcodec?: string
  acodec?: string
  filesize?: number
}

export interface YtdlpSubtitleOption {
  id: string
  language: string
  name?: string
  isAuto?: boolean
  ext?: string
}

export interface ParseYtdlpDownloadResponse {
  title?: string
  domain?: string
  extractor?: string
  author?: string
  thumbnailUrl?: string
  formats: YtdlpFormatOption[]
  subtitles: YtdlpSubtitleOption[]
}

export interface CreateSubtitleImportRequest {
  path: string
  libraryId?: string
  title?: string
  source?: string
  sessionKey?: string
  runId?: string
}

export interface CreateVideoImportRequest {
  path: string
  libraryId?: string
  title?: string
  source?: string
  sessionKey?: string
  runId?: string
}

export interface CreateTranscodeJobRequest {
  fileId?: string
  inputPath?: string
  libraryId?: string
  rootFileId?: string
  presetId?: string
  format?: string
  title?: string
  source?: string
  sessionKey?: string
  runId?: string
  videoCodec?: string
  qualityMode?: string
  crf?: number
  bitrateKbps?: number
  preset?: string
  audioCodec?: string
  audioBitrateKbps?: number
  scale?: string
  width?: number
  height?: number
  subtitleHandling?: string
  subtitleFileId?: string
  secondarySubtitleFileId?: string
  displayMode?: string
  subtitleDocumentId?: string
  generatedSubtitleFormat?: string
  generatedSubtitleName?: string
  generatedSubtitleStyleDocumentContent?: string
  generatedSubtitleDocument?: SubtitleDocument
  generatedSubtitleContent?: string
  deleteSourceFileAfterTranscode?: boolean
}

export interface ListTranscodePresetsForDownloadRequest {
  mediaType: string
}

export interface TranscodePreset {
  id: string
  name: string
  outputType: TranscodePresetOutputType
  container: string
  videoCodec?: string
  audioCodec?: string
  qualityMode?: TranscodeQualityMode
  crf?: number
  bitrateKbps?: number
  audioBitrateKbps?: number
  scale?: TranscodeScaleMode
  width?: number
  height?: number
  ffmpegPreset?: FFmpegSpeedPreset
  allowUpscale?: boolean
  requiresVideo?: boolean
  requiresAudio?: boolean
  isBuiltin?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface DeleteTranscodePresetRequest {
  id: string
}

export interface SubtitleCue {
  index: number
  start: string
  end: string
  text: string
}

export interface SubtitleDocument {
  format: string
  cues: SubtitleCue[]
  metadata?: Record<string, unknown>
}

export interface SubtitleParseRequest {
  fileId?: string
  documentId?: string
  path?: string
  content?: string
  format?: string
}

export interface SubtitleParseResult {
  format: string
  cueCount: number
  document: SubtitleDocument
  warnings?: string[]
}

export interface SubtitleConvertRequest {
  fileId?: string
  documentId?: string
  path?: string
  content?: string
  fromFormat?: string
  targetFormat: string
}

export interface SubtitleConvertResult {
  targetFormat: string
  content: string
  warnings?: string[]
}

export interface SubtitleExportConfig {
  srt?: SubtitleSRTExportConfig
  vtt?: SubtitleVTTExportConfig
  ass?: SubtitleASSExportConfig
  itt?: SubtitleITTExportConfig
  fcpxml?: SubtitleFCPXMLExportConfig
}

export interface SubtitleSRTExportConfig {
  encoding?: string
}

export interface SubtitleVTTExportConfig {
  kind?: string
  language?: string
}

export interface SubtitleASSExportConfig {
  playResX?: number
  playResY?: number
  title?: string
}

export interface SubtitleITTExportConfig {
  frameRate?: number
  frameRateMultiplier?: string
  language?: string
}

export interface SubtitleFCPXMLExportConfig {
  frameDuration?: string
  width?: number
  height?: number
  colorSpace?: string
  version?: string
  libraryName?: string
  eventName?: string
  projectName?: string
  defaultLane?: number
  startTimecodeSeconds?: number
}

export interface SubtitleExportRequest {
  exportPath: string
  fileId?: string
  documentId?: string
  path?: string
  content?: string
  format?: string
  targetFormat?: string
  styleDocumentContent?: string
  exportConfig?: SubtitleExportConfig
  document?: SubtitleDocument
}

export interface SubtitleExportResult {
  exportPath: string
  format: string
  bytes: number
}

export interface SubtitleValidateRequest {
  fileId?: string
  documentId?: string
  path?: string
  content?: string
  format?: string
  document?: SubtitleDocument
}

export interface SubtitleValidateIssue {
  severity: string
  code: string
  message: string
  cueIndex?: number
}

export interface SubtitleValidateResult {
  valid: boolean
  issueCount: number
  issues?: SubtitleValidateIssue[]
}

export interface SubtitleReviewSuggestionDTO {
  cueIndex: number
  originalText: string
  suggestedText: string
  categories?: string[]
  reason?: string
  sourceCode?: string
  severity?: string
}

export interface SubtitleReviewSessionDetailDTO {
  sessionId: string
  libraryId: string
  fileId: string
  kind: "proofread" | "qa" | string
  status: string
  sourceRevisionId: string
  candidateRevisionId: string
  appliedRevisionId?: string
  changedCueCount: number
  sourceDocument: SubtitleDocument
  candidateDocument: SubtitleDocument
  suggestions?: SubtitleReviewSuggestionDTO[]
}

export interface SubtitleFixTyposRequest {
  fileId?: string
  documentId?: string
  path?: string
  content?: string
  format?: string
  document?: SubtitleDocument
}

export interface SubtitleFixTyposResult {
  format: string
  content: string
  changeCount: number
  document: SubtitleDocument
}

export interface SubtitleSaveRequest {
  fileId?: string
  documentId?: string
  path?: string
  format?: string
  targetFormat?: string
  content?: string
  document?: SubtitleDocument
}

export interface SubtitleSaveResult {
  path?: string
  format: string
  bytes: number
}

export interface SubtitleTranslateRequest {
  fileId?: string
  documentId?: string
  path?: string
  libraryId?: string
  rootFileId?: string
  assistantId?: string
  targetLanguage: string
  outputFormat?: string
  mode?: string
  source?: string
  glossaryProfileIds?: string[]
  referenceTrackFileIds?: string[]
  promptProfileIds?: string[]
  inlinePrompt?: string
  sessionKey?: string
  runId?: string
}

export interface SubtitleProofreadRequest {
  fileId?: string
  documentId?: string
  path?: string
  libraryId?: string
  rootFileId?: string
  assistantId?: string
  language?: string
  outputFormat?: string
  source?: string
  spelling?: boolean
  punctuation?: boolean
  terminology?: boolean
  glossaryProfileIds?: string[]
  promptProfileIds?: string[]
  inlinePrompt?: string
  sessionKey?: string
  runId?: string
}

export interface SubtitleQAReviewRequest {
  fileId?: string
  documentId?: string
  path?: string
  libraryId?: string
  outputFormat?: string
  source?: string
  normalizeWhitespace?: boolean
  sessionKey?: string
  runId?: string
}

export interface RestoreSubtitleOriginalRequest {
  fileId?: string
  documentId?: string
  path?: string
}

export interface RestoreSubtitleOriginalResult {
  fileId?: string
  format: string
  bytes: number
}
