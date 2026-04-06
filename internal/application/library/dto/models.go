package dto

const LibrarySchemaVersion = "current"
const WorkspaceProjectSchemaVersion = "current"

type LibraryDTO struct {
	Version   string               `json:"version"`
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	CreatedAt string               `json:"createdAt"`
	UpdatedAt string               `json:"updatedAt"`
	CreatedBy LibraryCreateMetaDTO `json:"createdBy"`
	Files     []LibraryFileDTO     `json:"files"`
	Records   LibraryRecordsDTO    `json:"records"`
}

type LibraryCreateMetaDTO struct {
	Source             string `json:"source"`
	TriggerOperationID string `json:"triggerOperationId,omitempty"`
	ImportBatchID      string `json:"importBatchId,omitempty"`
	Actor              string `json:"actor,omitempty"`
}

type LibraryFileDTO struct {
	ID                string                `json:"id"`
	LibraryID         string                `json:"libraryId"`
	Kind              string                `json:"kind"`
	Name              string                `json:"name"`
	DisplayLabel      string                `json:"displayLabel,omitempty"`
	Storage           LibraryFileStorageDTO `json:"storage"`
	Origin            LibraryFileOriginDTO  `json:"origin"`
	Lineage           LibraryFileLineageDTO `json:"lineage"`
	LatestOperationID string                `json:"latestOperationId,omitempty"`
	Media             *LibraryMediaInfoDTO  `json:"media,omitempty"`
	State             LibraryFileStateDTO   `json:"state"`
	CreatedAt         string                `json:"createdAt"`
	UpdatedAt         string                `json:"updatedAt"`
}

type LibraryFileStorageDTO struct {
	Mode       string `json:"mode"`
	LocalPath  string `json:"localPath,omitempty"`
	DocumentID string `json:"documentId,omitempty"`
}

type LibraryFileOriginDTO struct {
	Kind        string                  `json:"kind"`
	OperationID string                  `json:"operationId,omitempty"`
	Import      *LibraryImportOriginDTO `json:"import,omitempty"`
}

type LibraryImportOriginDTO struct {
	BatchID        string `json:"batchId"`
	ImportPath     string `json:"importPath"`
	ImportedAt     string `json:"importedAt"`
	KeepSourceFile bool   `json:"keepSourceFile"`
}

type LibraryFileLineageDTO struct {
	RootFileID string `json:"rootFileId,omitempty"`
}

type LibraryMediaInfoDTO struct {
	Format      string   `json:"format,omitempty"`
	Codec       string   `json:"codec,omitempty"`
	VideoCodec  string   `json:"videoCodec,omitempty"`
	AudioCodec  string   `json:"audioCodec,omitempty"`
	DurationMs  *int64   `json:"durationMs,omitempty"`
	Width       *int     `json:"width,omitempty"`
	Height      *int     `json:"height,omitempty"`
	FrameRate   *float64 `json:"frameRate,omitempty"`
	BitrateKbps *int     `json:"bitrateKbps,omitempty"`
	Channels    *int     `json:"channels,omitempty"`
	SizeBytes   *int64   `json:"sizeBytes,omitempty"`
	Language    string   `json:"language,omitempty"`
	CueCount    *int     `json:"cueCount,omitempty"`
}

type LibraryFileStateDTO struct {
	Status      string `json:"status"`
	Deleted     bool   `json:"deleted"`
	Archived    bool   `json:"archived"`
	LastError   string `json:"lastError,omitempty"`
	LastChecked string `json:"lastChecked,omitempty"`
}

type LibraryOperationDTO struct {
	ID           string                      `json:"id"`
	LibraryID    string                      `json:"libraryId"`
	Kind         string                      `json:"kind"`
	Status       string                      `json:"status"`
	DisplayName  string                      `json:"displayName"`
	Correlation  OperationCorrelationDTO     `json:"correlation"`
	InputJSON    string                      `json:"inputJson"`
	OutputJSON   string                      `json:"outputJson"`
	SourceDomain string                      `json:"sourceDomain,omitempty"`
	SourceIcon   string                      `json:"sourceIcon,omitempty"`
	Meta         OperationMetaDTO            `json:"meta"`
	Request      *OperationRequestPreviewDTO `json:"request,omitempty"`
	Progress     *OperationProgressDTO       `json:"progress,omitempty"`
	OutputFiles  []OperationOutputFileDTO    `json:"outputFiles,omitempty"`
	Metrics      OperationMetricsDTO         `json:"metrics"`
	ErrorCode    string                      `json:"errorCode,omitempty"`
	ErrorMessage string                      `json:"errorMessage,omitempty"`
	CreatedAt    string                      `json:"createdAt"`
	StartedAt    string                      `json:"startedAt,omitempty"`
	FinishedAt   string                      `json:"finishedAt,omitempty"`
}

type OperationMetaDTO struct {
	Platform    string `json:"platform,omitempty"`
	Uploader    string `json:"uploader,omitempty"`
	PublishTime string `json:"publishTime,omitempty"`
}

type OperationRequestPreviewDTO struct {
	URL          string `json:"url,omitempty"`
	Caller       string `json:"caller,omitempty"`
	Extractor    string `json:"extractor,omitempty"`
	Author       string `json:"author,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

type OperationCorrelationDTO struct {
	RequestID         string `json:"requestId,omitempty"`
	RunID             string `json:"runId,omitempty"`
	ParentOperationID string `json:"parentOperationId,omitempty"`
}

type OperationProgressDTO struct {
	Stage     string `json:"stage,omitempty"`
	Percent   *int   `json:"percent,omitempty"`
	Current   *int64 `json:"current,omitempty"`
	Total     *int64 `json:"total,omitempty"`
	Speed     string `json:"speed,omitempty"`
	Message   string `json:"message,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type OperationOutputFileDTO struct {
	FileID    string `json:"fileId"`
	Kind      string `json:"kind"`
	Format    string `json:"format,omitempty"`
	SizeBytes *int64 `json:"sizeBytes,omitempty"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

type OperationMetricsDTO struct {
	FileCount      int    `json:"fileCount"`
	TotalSizeBytes *int64 `json:"totalSizeBytes,omitempty"`
	DurationMs     *int64 `json:"durationMs,omitempty"`
}

type OperationListItemDTO struct {
	OperationID string                   `json:"operationId"`
	LibraryID   string                   `json:"libraryId"`
	LibraryName string                   `json:"libraryName,omitempty"`
	Name        string                   `json:"name"`
	Kind        string                   `json:"kind"`
	Status      string                   `json:"status"`
	Domain      string                   `json:"domain,omitempty"`
	SourceIcon  string                   `json:"sourceIcon,omitempty"`
	Platform    string                   `json:"platform,omitempty"`
	Uploader    string                   `json:"uploader,omitempty"`
	PublishTime string                   `json:"publishTime,omitempty"`
	Progress    *OperationProgressDTO    `json:"progress,omitempty"`
	OutputFiles []OperationOutputFileDTO `json:"outputFiles,omitempty"`
	Metrics     OperationMetricsDTO      `json:"metrics"`
	StartedAt   string                   `json:"startedAt,omitempty"`
	FinishedAt  string                   `json:"finishedAt,omitempty"`
	CreatedAt   string                   `json:"createdAt"`
}

type LibraryRecordsDTO struct {
	History            []LibraryHistoryRecordDTO `json:"history"`
	WorkspaceStateHead *WorkspaceStateRecordDTO  `json:"workspaceStateHead,omitempty"`
	WorkspaceStates    []WorkspaceStateRecordDTO `json:"workspaceStates"`
	FileEvents         []FileEventRecordDTO      `json:"fileEvents"`
}

type WorkspaceStateRecordDTO struct {
	ID           string `json:"id"`
	LibraryID    string `json:"libraryId"`
	StateVersion int    `json:"stateVersion"`
	StateJSON    string `json:"stateJson"`
	OperationID  string `json:"operationId,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type WorkspaceProjectDTO struct {
	Version              string                      `json:"version"`
	LibraryID            string                      `json:"libraryId"`
	Title                string                      `json:"title"`
	UpdatedAt            string                      `json:"updatedAt"`
	ViewStateHead        *WorkspaceStateRecordDTO    `json:"viewStateHead,omitempty"`
	VideoTracks          []WorkspaceVideoTrackDTO    `json:"videoTracks"`
	SubtitleTracks       []WorkspaceSubtitleTrackDTO `json:"subtitleTracks"`
	SubtitleMonoStyle    *LibraryMonoStyleDTO        `json:"subtitleMonoStyle,omitempty"`
	SubtitleLingualStyle *LibraryBilingualStyleDTO   `json:"subtitleLingualStyle,omitempty"`
}

type WorkspaceTrackDisplayDTO struct {
	Label  string   `json:"label"`
	Hint   string   `json:"hint,omitempty"`
	Badges []string `json:"badges,omitempty"`
}

type WorkspaceTaskSummaryDTO struct {
	OperationID string `json:"operationId"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	DisplayName string `json:"displayName"`
	Stage       string `json:"stage,omitempty"`
	Current     int64  `json:"current,omitempty"`
	Total       int64  `json:"total,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type WorkspaceTrackTasksDTO struct {
	Translate []WorkspaceTaskSummaryDTO `json:"translate,omitempty"`
	Proofread *WorkspaceTaskSummaryDTO  `json:"proofread,omitempty"`
	QA        *WorkspaceTaskSummaryDTO  `json:"qa,omitempty"`
}

type WorkspaceTrackPendingReviewDTO struct {
	SessionID           string   `json:"sessionId"`
	Kind                string   `json:"kind"`
	Status              string   `json:"status"`
	SourceRevisionID    string   `json:"sourceRevisionId"`
	CandidateRevisionID string   `json:"candidateRevisionId"`
	ChangedCueCount     int      `json:"changedCueCount"`
	BlockedActions      []string `json:"blockedActions,omitempty"`
}

type WorkspaceVideoTrackDTO struct {
	TrackID string                   `json:"trackId"`
	File    LibraryFileDTO           `json:"file"`
	Display WorkspaceTrackDisplayDTO `json:"display"`
}

type WorkspaceSubtitleTrackDTO struct {
	TrackID       string                          `json:"trackId"`
	Role          string                          `json:"role"`
	File          LibraryFileDTO                  `json:"file"`
	Display       WorkspaceTrackDisplayDTO        `json:"display"`
	RunningTasks  WorkspaceTrackTasksDTO          `json:"runningTasks"`
	PendingReview *WorkspaceTrackPendingReviewDTO `json:"pendingReview,omitempty"`
}

type LibraryHistoryRecordDTO struct {
	RecordID      string                         `json:"recordId"`
	LibraryID     string                         `json:"libraryId"`
	Category      string                         `json:"category"`
	Action        string                         `json:"action"`
	DisplayName   string                         `json:"displayName"`
	Status        string                         `json:"status"`
	Source        LibraryHistoryRecordSourceDTO  `json:"source"`
	Refs          LibraryHistoryRecordRefsDTO    `json:"refs"`
	Files         []OperationOutputFileDTO       `json:"files,omitempty"`
	Metrics       OperationMetricsDTO            `json:"metrics"`
	ImportMeta    *LibraryImportRecordMetaDTO    `json:"importMeta,omitempty"`
	OperationMeta *LibraryOperationRecordMetaDTO `json:"operationMeta,omitempty"`
	OccurredAt    string                         `json:"occurredAt"`
	CreatedAt     string                         `json:"createdAt"`
}

type LibraryHistoryRecordSourceDTO struct {
	Kind   string `json:"kind"`
	Caller string `json:"caller,omitempty"`
	RunID  string `json:"runId,omitempty"`
	Actor  string `json:"actor,omitempty"`
}

type LibraryHistoryRecordRefsDTO struct {
	OperationID   string   `json:"operationId,omitempty"`
	ImportBatchID string   `json:"importBatchId,omitempty"`
	FileIDs       []string `json:"fileIds,omitempty"`
	FileEventIDs  []string `json:"fileEventIds,omitempty"`
}

type LibraryImportRecordMetaDTO struct {
	ImportPath     string `json:"importPath,omitempty"`
	KeepSourceFile bool   `json:"keepSourceFile"`
	ImportedAt     string `json:"importedAt"`
}

type LibraryOperationRecordMetaDTO struct {
	Kind         string `json:"kind"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type FileEventRecordDTO struct {
	ID          string             `json:"id"`
	LibraryID   string             `json:"libraryId"`
	FileID      string             `json:"fileId"`
	EventType   string             `json:"eventType"`
	OperationID string             `json:"operationId,omitempty"`
	Detail      FileEventDetailDTO `json:"detail"`
	CreatedAt   string             `json:"createdAt"`
}

type FileEventDetailDTO struct {
	Cause   FileEventCauseDTO         `json:"cause"`
	Before  *FileEventFileSnapshotDTO `json:"before,omitempty"`
	After   *FileEventFileSnapshotDTO `json:"after,omitempty"`
	Changes []FileFieldChangeDTO      `json:"changes,omitempty"`
	Import  *LibraryImportOriginDTO   `json:"import,omitempty"`
}

type FileEventCauseDTO struct {
	Category      string `json:"category"`
	OperationID   string `json:"operationId,omitempty"`
	ImportBatchID string `json:"importBatchId,omitempty"`
	Actor         string `json:"actor,omitempty"`
}

type FileEventFileSnapshotDTO struct {
	FileID     string `json:"fileId"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	LocalPath  string `json:"localPath,omitempty"`
	DocumentID string `json:"documentId,omitempty"`
}

type FileFieldChangeDTO struct {
	Field  string `json:"field"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type LibraryModuleConfigDTO struct {
	Workspace          LibraryWorkspaceConfigDTO          `json:"workspace"`
	TranslateLanguages LibraryTranslateLanguagesConfigDTO `json:"translateLanguages"`
	LanguageAssets     LibraryLanguageAssetsConfigDTO     `json:"languageAssets"`
	SubtitleStyles     LibrarySubtitleStyleConfigDTO      `json:"subtitleStyles"`
	TaskRuntime        LibraryTaskRuntimeConfigDTO        `json:"taskRuntime"`
}

type LibraryWorkspaceConfigDTO struct {
	FastReadLatestState bool `json:"fastReadLatestState"`
}

type LibraryTranslateLanguageDTO struct {
	Code    string   `json:"code"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases,omitempty"`
}

type LibraryTranslateLanguagesConfigDTO struct {
	Builtin []LibraryTranslateLanguageDTO `json:"builtin"`
	Custom  []LibraryTranslateLanguageDTO `json:"custom"`
}

type LibraryGlossaryTermDTO struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Note   string `json:"note,omitempty"`
}

type LibraryGlossaryProfileDTO struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	Category       string                   `json:"category,omitempty"`
	Description    string                   `json:"description,omitempty"`
	SourceLanguage string                   `json:"sourceLanguage,omitempty"`
	TargetLanguage string                   `json:"targetLanguage,omitempty"`
	Terms          []LibraryGlossaryTermDTO `json:"terms,omitempty"`
}

type LibraryPromptProfileDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Prompt      string `json:"prompt"`
}

type LibraryLanguageAssetsConfigDTO struct {
	GlossaryProfiles []LibraryGlossaryProfileDTO `json:"glossaryProfiles"`
	PromptProfiles   []LibraryPromptProfileDTO   `json:"promptProfiles"`
}

type LibraryTaskRuntimeSettingsDTO struct {
	StructuredOutputMode string `json:"structuredOutputMode"`
	ThinkingMode         string `json:"thinkingMode"`
	MaxTokensFloor       int    `json:"maxTokensFloor"`
	MaxTokensCeiling     int    `json:"maxTokensCeiling"`
	RetryTokenStep       int    `json:"retryTokenStep"`
}

type LibraryTaskRuntimeConfigDTO struct {
	Translate LibraryTaskRuntimeSettingsDTO `json:"translate"`
	Proofread LibraryTaskRuntimeSettingsDTO `json:"proofread"`
}

type LibrarySubtitleStyleDefaultsDTO struct {
	MonoStyleID            string `json:"monoStyleId,omitempty"`
	BilingualStyleID       string `json:"bilingualStyleId,omitempty"`
	SubtitleExportPresetID string `json:"subtitleExportPresetId,omitempty"`
}

type LibrarySubtitleStyleDocumentAnalysisDTO struct {
	DetectedFormat   string   `json:"detectedFormat,omitempty"`
	ScriptType       string   `json:"scriptType,omitempty"`
	PlayResX         int      `json:"playResX,omitempty"`
	PlayResY         int      `json:"playResY,omitempty"`
	StyleCount       int      `json:"styleCount,omitempty"`
	DialogueCount    int      `json:"dialogueCount,omitempty"`
	CommentCount     int      `json:"commentCount,omitempty"`
	StyleNames       []string `json:"styleNames,omitempty"`
	Fonts            []string `json:"fonts,omitempty"`
	FeatureFlags     []string `json:"featureFlags,omitempty"`
	ValidationIssues []string `json:"validationIssues,omitempty"`
}

type LibrarySubtitleStyleDocumentDTO struct {
	ID          string                                  `json:"id"`
	Name        string                                  `json:"name"`
	Description string                                  `json:"description,omitempty"`
	Source      string                                  `json:"source,omitempty"`
	SourceRef   string                                  `json:"sourceRef,omitempty"`
	Version     string                                  `json:"version,omitempty"`
	Enabled     bool                                    `json:"enabled"`
	Format      string                                  `json:"format,omitempty"`
	Content     string                                  `json:"content,omitempty"`
	Analysis    LibrarySubtitleStyleDocumentAnalysisDTO `json:"analysis,omitempty"`
}

type LibraryRemoteFontManifestInfoDTO struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
	Version     string `json:"version,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`
	TotalFonts  int    `json:"totalFonts,omitempty"`
}

type LibraryRemoteFontManifestVariantDTO struct {
	Name    string            `json:"name,omitempty"`
	Weight  int               `json:"weight,omitempty"`
	Style   string            `json:"style,omitempty"`
	Subsets []string          `json:"subsets,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

type LibraryRemoteFontManifestFontDTO struct {
	Name          string                                `json:"name,omitempty"`
	Family        string                                `json:"family,omitempty"`
	License       string                                `json:"license,omitempty"`
	LicenseURL    string                                `json:"licenseUrl,omitempty"`
	Designer      string                                `json:"designer,omitempty"`
	Foundry       string                                `json:"foundry,omitempty"`
	Version       string                                `json:"version,omitempty"`
	Description   string                                `json:"description,omitempty"`
	Categories    []string                              `json:"categories,omitempty"`
	Tags          []string                              `json:"tags,omitempty"`
	Popularity    int                                   `json:"popularity,omitempty"`
	LastModified  string                                `json:"lastModified,omitempty"`
	MetadataURL   string                                `json:"metadataUrl,omitempty"`
	SourceURL     string                                `json:"sourceUrl,omitempty"`
	Variants      []LibraryRemoteFontManifestVariantDTO `json:"variants,omitempty"`
	UnicodeRanges []string                              `json:"unicodeRanges,omitempty"`
	Languages     []string                              `json:"languages,omitempty"`
	SampleText    string                                `json:"sampleText,omitempty"`
}

type LibraryRemoteFontManifestDTO struct {
	SourceInfo LibraryRemoteFontManifestInfoDTO            `json:"sourceInfo,omitempty"`
	Fonts      map[string]LibraryRemoteFontManifestFontDTO `json:"fonts,omitempty"`
}

type LibrarySubtitleStyleSourceDTO struct {
	ID                 string                       `json:"id"`
	Name               string                       `json:"name"`
	Kind               string                       `json:"kind,omitempty"`
	Provider           string                       `json:"provider,omitempty"`
	URL                string                       `json:"url,omitempty"`
	Prefix             string                       `json:"prefix,omitempty"`
	Filename           string                       `json:"filename,omitempty"`
	Priority           int                          `json:"priority,omitempty"`
	BuiltIn            bool                         `json:"builtIn,omitempty"`
	Owner              string                       `json:"owner,omitempty"`
	Repo               string                       `json:"repo,omitempty"`
	Ref                string                       `json:"ref,omitempty"`
	ManifestPath       string                       `json:"manifestPath,omitempty"`
	RemoteFontManifest LibraryRemoteFontManifestDTO `json:"remoteFontManifest,omitempty"`
	Enabled            bool                         `json:"enabled"`
	FontCount          int                          `json:"fontCount,omitempty"`
	SyncStatus         string                       `json:"syncStatus,omitempty"`
	LastSyncedAt       string                       `json:"lastSyncedAt,omitempty"`
	LastError          string                       `json:"lastError,omitempty"`
}

type LibrarySubtitleStyleFontDTO struct {
	ID           string `json:"id"`
	Family       string `json:"family"`
	Source       string `json:"source,omitempty"`
	SystemFamily string `json:"systemFamily,omitempty"`
	Enabled      bool   `json:"enabled"`
}

type LibrarySubtitleExportPresetDTO struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Format        string               `json:"format,omitempty"`
	TargetFormat  string               `json:"targetFormat,omitempty"`
	MediaStrategy string               `json:"mediaStrategy,omitempty"`
	Config        SubtitleExportConfig `json:"config,omitempty"`
}

type LibrarySubtitleStyleConfigDTO struct {
	MonoStyles            []LibraryMonoStyleDTO            `json:"monoStyles,omitempty"`
	BilingualStyles       []LibraryBilingualStyleDTO       `json:"bilingualStyles,omitempty"`
	Sources               []LibrarySubtitleStyleSourceDTO  `json:"sources"`
	Fonts                 []LibrarySubtitleStyleFontDTO    `json:"fonts,omitempty"`
	SubtitleExportPresets []LibrarySubtitleExportPresetDTO `json:"subtitleExportPresets,omitempty"`
	Defaults              LibrarySubtitleStyleDefaultsDTO  `json:"defaults"`
}

type BrowseSubtitleStyleRemoteSourceRequest struct {
	Source LibrarySubtitleStyleSourceDTO `json:"source"`
}

type SubtitleStyleRemoteManifestItemDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	FilePath    string   `json:"filePath,omitempty"`
	Fonts       []string `json:"fonts,omitempty"`
}

type ImportSubtitleStyleRemoteSourceItemRequest struct {
	Source LibrarySubtitleStyleSourceDTO `json:"source"`
	ItemID string                        `json:"itemId"`
}

type GetLibraryRequest struct {
	LibraryID string `json:"libraryId"`
}

type RenameLibraryRequest struct {
	LibraryID string `json:"libraryId"`
	Name      string `json:"name"`
}

type DeleteLibraryRequest struct {
	LibraryID string `json:"libraryId"`
}

type UpdateLibraryModuleConfigRequest struct {
	Config LibraryModuleConfigDTO `json:"config"`
}

type ListOperationsRequest struct {
	LibraryID string   `json:"libraryId,omitempty"`
	Status    []string `json:"status,omitempty"`
	Kinds     []string `json:"kinds,omitempty"`
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
}

type GetOperationRequest struct {
	OperationID string `json:"operationId"`
}

type CancelOperationRequest struct {
	OperationID string `json:"operationId"`
}

type ResumeOperationRequest struct {
	OperationID string `json:"operationId"`
}

type DeleteOperationRequest struct {
	OperationID  string `json:"operationId"`
	CascadeFiles bool   `json:"cascadeFiles,omitempty"`
}

type DeleteOperationsRequest struct {
	OperationIDs []string `json:"operationIds"`
	CascadeFiles bool     `json:"cascadeFiles,omitempty"`
}

type DeleteFileRequest struct {
	FileID      string `json:"fileId"`
	DeleteFiles bool   `json:"deleteFiles,omitempty"`
}

type DeleteFilesRequest struct {
	FileIDs     []string `json:"fileIds"`
	DeleteFiles bool     `json:"deleteFiles,omitempty"`
}

type ListLibraryHistoryRequest struct {
	LibraryID  string   `json:"libraryId"`
	Categories []string `json:"categories,omitempty"`
	Actions    []string `json:"actions,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

type ListFileEventsRequest struct {
	LibraryID string `json:"libraryId"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type SaveWorkspaceStateRequest struct {
	LibraryID   string `json:"libraryId"`
	StateJSON   string `json:"stateJson"`
	OperationID string `json:"operationId,omitempty"`
}

type GetWorkspaceStateRequest struct {
	LibraryID string `json:"libraryId"`
}

type GetWorkspaceProjectRequest struct {
	LibraryID string `json:"libraryId"`
}

type GetSubtitleReviewSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type ApplySubtitleReviewSessionRequest struct {
	SessionID string                         `json:"sessionId"`
	Decisions []SubtitleReviewCueDecisionDTO `json:"decisions,omitempty"`
}

type DiscardSubtitleReviewSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type OpenFileLocationRequest struct {
	FileID string `json:"fileId"`
}
type OpenPathRequest struct {
	Path string `json:"path"`
}

type CreateYTDLPJobRequest struct {
	URL                            string   `json:"url"`
	LibraryID                      string   `json:"libraryId,omitempty"`
	Title                          string   `json:"title"`
	Extractor                      string   `json:"extractor,omitempty"`
	Author                         string   `json:"author,omitempty"`
	ThumbnailURL                   string   `json:"thumbnailUrl,omitempty"`
	WriteThumbnail                 bool     `json:"writeThumbnail,omitempty"`
	CookiesPath                    string   `json:"cookiesPath,omitempty"`
	Source                         string   `json:"source,omitempty"`
	Caller                         string   `json:"caller,omitempty"`
	SessionKey                     string   `json:"sessionKey,omitempty"`
	RunID                          string   `json:"runId,omitempty"`
	RetryOf                        string   `json:"retryOf,omitempty"`
	RetryCount                     int      `json:"retryCount,omitempty"`
	Mode                           string   `json:"mode,omitempty"`
	LogPolicy                      string   `json:"logPolicy,omitempty"`
	Quality                        string   `json:"quality,omitempty"`
	FormatID                       string   `json:"formatId,omitempty"`
	AudioFormatID                  string   `json:"audioFormatId,omitempty"`
	SubtitleLangs                  []string `json:"subtitleLangs,omitempty"`
	SubtitleAuto                   bool     `json:"subtitleAuto,omitempty"`
	SubtitleAll                    bool     `json:"subtitleAll,omitempty"`
	SubtitleFormat                 string   `json:"subtitleFormat,omitempty"`
	TranscodePresetID              string   `json:"transcodePresetId,omitempty"`
	DeleteSourceFileAfterTranscode bool     `json:"deleteSourceFileAfterTranscode,omitempty"`
	ConnectorID                    string   `json:"connectorId,omitempty"`
	UseConnector                   bool     `json:"useConnector,omitempty"`
}

type CheckYTDLPOperationFailureRequest struct {
	OperationID string `json:"operationId"`
}

type CheckYTDLPOperationFailureItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Action  string `json:"action,omitempty"`
}

type CheckYTDLPOperationFailureResponse struct {
	Items    []CheckYTDLPOperationFailureItem `json:"items"`
	CanRetry bool                             `json:"canRetry"`
}

type RetryYTDLPOperationRequest struct {
	OperationID string `json:"operationId"`
	Source      string `json:"source,omitempty"`
	Caller      string `json:"caller,omitempty"`
	RunID       string `json:"runId,omitempty"`
}

type GetYTDLPOperationLogRequest struct {
	OperationID string `json:"operationId"`
	MaxBytes    int    `json:"maxBytes,omitempty"`
	TailLines   int    `json:"tailLines,omitempty"`
}

type GetYTDLPOperationLogResponse struct {
	OperationID string `json:"operationId"`
	Path        string `json:"path,omitempty"`
	Content     string `json:"content,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
}

type PrepareYTDLPDownloadRequest struct {
	URL string `json:"url"`
}
type PrepareYTDLPDownloadResponse struct {
	URL                string `json:"url"`
	Domain             string `json:"domain"`
	Icon               string `json:"icon,omitempty"`
	ConnectorID        string `json:"connectorId,omitempty"`
	ConnectorAvailable bool   `json:"connectorAvailable"`
	Reachable          bool   `json:"reachable,omitempty"`
}

type ResolveDomainIconRequest struct {
	Domain string `json:"domain"`
	URL    string `json:"url,omitempty"`
}
type ResolveDomainIconResponse struct {
	Domain string `json:"domain,omitempty"`
	Icon   string `json:"icon,omitempty"`
}

type ParseYTDLPDownloadRequest struct {
	URL          string `json:"url"`
	ConnectorID  string `json:"connectorId,omitempty"`
	UseConnector bool   `json:"useConnector,omitempty"`
}

type YTDLPFormatOption struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	HasVideo bool   `json:"hasVideo"`
	HasAudio bool   `json:"hasAudio"`
	Ext      string `json:"ext,omitempty"`
	Height   int    `json:"height,omitempty"`
	VCodec   string `json:"vcodec,omitempty"`
	ACodec   string `json:"acodec,omitempty"`
	Filesize int64  `json:"filesize,omitempty"`
}

type YTDLPSubtitleOption struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Name     string `json:"name,omitempty"`
	IsAuto   bool   `json:"isAuto,omitempty"`
	Ext      string `json:"ext,omitempty"`
}

type ParseYTDLPDownloadResponse struct {
	Title        string                `json:"title,omitempty"`
	Domain       string                `json:"domain,omitempty"`
	Extractor    string                `json:"extractor,omitempty"`
	Author       string                `json:"author,omitempty"`
	ThumbnailURL string                `json:"thumbnailUrl,omitempty"`
	Formats      []YTDLPFormatOption   `json:"formats"`
	Subtitles    []YTDLPSubtitleOption `json:"subtitles"`
}

type CreateSubtitleImportRequest struct {
	Path       string `json:"path"`
	LibraryID  string `json:"libraryId,omitempty"`
	Title      string `json:"title"`
	Source     string `json:"source,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
	RunID      string `json:"runId,omitempty"`
}

type CreateVideoImportRequest struct {
	Path       string `json:"path"`
	LibraryID  string `json:"libraryId,omitempty"`
	Title      string `json:"title"`
	Source     string `json:"source,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
	RunID      string `json:"runId,omitempty"`
}

type CreateTranscodeJobRequest struct {
	FileID                         string `json:"fileId,omitempty"`
	InputPath                      string `json:"inputPath,omitempty"`
	LibraryID                      string `json:"libraryId,omitempty"`
	RootFileID                     string `json:"rootFileId,omitempty"`
	PresetID                       string `json:"presetId,omitempty"`
	Format                         string `json:"format,omitempty"`
	Title                          string `json:"title"`
	Source                         string `json:"source,omitempty"`
	SessionKey                     string `json:"sessionKey,omitempty"`
	RunID                          string `json:"runId,omitempty"`
	VideoCodec                     string `json:"videoCodec,omitempty"`
	QualityMode                    string `json:"qualityMode,omitempty"`
	CRF                            int    `json:"crf,omitempty"`
	BitrateKbps                    int    `json:"bitrateKbps,omitempty"`
	Preset                         string `json:"preset,omitempty"`
	AudioCodec                     string `json:"audioCodec,omitempty"`
	AudioBitrateKbps               int    `json:"audioBitrateKbps,omitempty"`
	Scale                          string `json:"scale,omitempty"`
	Width                          int    `json:"width,omitempty"`
	Height                         int    `json:"height,omitempty"`
	SubtitleHandling               string `json:"subtitleHandling,omitempty"`
	SubtitleFileID                 string `json:"subtitleFileId,omitempty"`
	SecondarySubtitleFileID        string `json:"secondarySubtitleFileId,omitempty"`
	DisplayMode                    string `json:"displayMode,omitempty"`
	SubtitleDocumentID             string `json:"subtitleDocumentId,omitempty"`
	GeneratedSubtitleFormat        string `json:"generatedSubtitleFormat,omitempty"`
	GeneratedSubtitleName          string `json:"generatedSubtitleName,omitempty"`
	GeneratedSubtitleContent       string `json:"generatedSubtitleContent,omitempty"`
	DeleteSourceFileAfterTranscode bool   `json:"deleteSourceFileAfterTranscode,omitempty"`
}

type ListTranscodePresetsForDownloadRequest struct {
	MediaType string `json:"mediaType"`
}

type TranscodePreset struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	OutputType       string `json:"outputType"`
	Container        string `json:"container"`
	VideoCodec       string `json:"videoCodec,omitempty"`
	AudioCodec       string `json:"audioCodec,omitempty"`
	QualityMode      string `json:"qualityMode,omitempty"`
	CRF              int    `json:"crf,omitempty"`
	BitrateKbps      int    `json:"bitrateKbps,omitempty"`
	AudioBitrateKbps int    `json:"audioBitrateKbps,omitempty"`
	Scale            string `json:"scale,omitempty"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
	FFmpegPreset     string `json:"ffmpegPreset,omitempty"`
	AllowUpscale     bool   `json:"allowUpscale,omitempty"`
	RequiresVideo    bool   `json:"requiresVideo,omitempty"`
	RequiresAudio    bool   `json:"requiresAudio,omitempty"`
	IsBuiltin        bool   `json:"isBuiltin,omitempty"`
	CreatedAt        string `json:"createdAt,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

type DeleteTranscodePresetRequest struct {
	ID string `json:"id"`
}

type LibraryToolRequest struct {
	Action    string `json:"action,omitempty"`
	InputJSON string `json:"inputJson,omitempty"`
}

type LibraryToolResult struct {
	Status      string `json:"status,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	OutputJSON  string `json:"outputJson,omitempty"`
	Error       string `json:"error,omitempty"`
}
