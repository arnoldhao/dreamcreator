package dto

type SubtitleCue struct {
	Index int    `json:"index"`
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}

type SubtitleDocument struct {
	Format   string         `json:"format"`
	Cues     []SubtitleCue  `json:"cues"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type SubtitleParseRequest struct {
	FileID     string `json:"fileId,omitempty"`
	DocumentID string `json:"documentId,omitempty"`
	Path       string `json:"path,omitempty"`
	Content    string `json:"content,omitempty"`
	Format     string `json:"format,omitempty"`
}

type SubtitleParseResult struct {
	Format   string           `json:"format"`
	CueCount int              `json:"cueCount"`
	Document SubtitleDocument `json:"document"`
	Warnings []string         `json:"warnings,omitempty"`
}

type SubtitleConvertRequest struct {
	FileID       string `json:"fileId,omitempty"`
	DocumentID   string `json:"documentId,omitempty"`
	Path         string `json:"path,omitempty"`
	Content      string `json:"content,omitempty"`
	FromFormat   string `json:"fromFormat,omitempty"`
	TargetFormat string `json:"targetFormat"`
}

type SubtitleConvertResult struct {
	TargetFormat string   `json:"targetFormat"`
	Content      string   `json:"content"`
	Warnings     []string `json:"warnings,omitempty"`
}

type SubtitleExportConfig struct {
	SRT    *SubtitleSRTExportConfig    `json:"srt,omitempty"`
	VTT    *SubtitleVTTExportConfig    `json:"vtt,omitempty"`
	ASS    *SubtitleASSExportConfig    `json:"ass,omitempty"`
	ITT    *SubtitleITTExportConfig    `json:"itt,omitempty"`
	FCPXML *SubtitleFCPXMLExportConfig `json:"fcpxml,omitempty"`
}

type SubtitleSRTExportConfig struct {
	Encoding string `json:"encoding,omitempty"`
}

type SubtitleVTTExportConfig struct {
	Kind     string `json:"kind,omitempty"`
	Language string `json:"language,omitempty"`
}

type SubtitleASSExportConfig struct {
	PlayResX int    `json:"playResX,omitempty"`
	PlayResY int    `json:"playResY,omitempty"`
	Title    string `json:"title,omitempty"`
}

type SubtitleITTExportConfig struct {
	FrameRate           int    `json:"frameRate,omitempty"`
	FrameRateMultiplier string `json:"frameRateMultiplier,omitempty"`
	Language            string `json:"language,omitempty"`
}

type SubtitleFCPXMLExportConfig struct {
	FrameDuration        string `json:"frameDuration,omitempty"`
	Width                int    `json:"width,omitempty"`
	Height               int    `json:"height,omitempty"`
	ColorSpace           string `json:"colorSpace,omitempty"`
	Version              string `json:"version,omitempty"`
	LibraryName          string `json:"libraryName,omitempty"`
	EventName            string `json:"eventName,omitempty"`
	ProjectName          string `json:"projectName,omitempty"`
	DefaultLane          int    `json:"defaultLane,omitempty"`
	StartTimecodeSeconds int64  `json:"startTimecodeSeconds,omitempty"`
}

type SubtitleExportRequest struct {
	ExportPath           string                `json:"exportPath"`
	FileID               string                `json:"fileId,omitempty"`
	DocumentID           string                `json:"documentId,omitempty"`
	Path                 string                `json:"path,omitempty"`
	Content              string                `json:"content,omitempty"`
	Format               string                `json:"format,omitempty"`
	TargetFormat         string                `json:"targetFormat,omitempty"`
	StyleDocumentContent string                `json:"styleDocumentContent,omitempty"`
	ExportConfig         *SubtitleExportConfig `json:"exportConfig,omitempty"`
	Document             *SubtitleDocument     `json:"document,omitempty"`
}

type SubtitleExportResult struct {
	ExportPath string `json:"exportPath"`
	Format     string `json:"format"`
	Bytes      int    `json:"bytes"`
}

type SubtitleValidateRequest struct {
	FileID     string            `json:"fileId,omitempty"`
	DocumentID string            `json:"documentId,omitempty"`
	Path       string            `json:"path,omitempty"`
	Content    string            `json:"content,omitempty"`
	Format     string            `json:"format,omitempty"`
	Document   *SubtitleDocument `json:"document,omitempty"`
}

type SubtitleValidateIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	CueIndex int    `json:"cueIndex,omitempty"`
}

type SubtitleValidateResult struct {
	Valid      bool                    `json:"valid"`
	IssueCount int                     `json:"issueCount"`
	Issues     []SubtitleValidateIssue `json:"issues,omitempty"`
}

type SubtitleReviewSuggestionDTO struct {
	CueIndex      int      `json:"cueIndex"`
	OriginalText  string   `json:"originalText"`
	SuggestedText string   `json:"suggestedText"`
	Categories    []string `json:"categories,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	SourceCode    string   `json:"sourceCode,omitempty"`
	Severity      string   `json:"severity,omitempty"`
}

type SubtitleReviewCueDecisionDTO struct {
	CueIndex int    `json:"cueIndex"`
	Action   string `json:"action"`
}

type SubtitleReviewSessionDetailDTO struct {
	SessionID           string                        `json:"sessionId"`
	LibraryID           string                        `json:"libraryId"`
	FileID              string                        `json:"fileId"`
	Kind                string                        `json:"kind"`
	Status              string                        `json:"status"`
	SourceRevisionID    string                        `json:"sourceRevisionId"`
	CandidateRevisionID string                        `json:"candidateRevisionId"`
	AppliedRevisionID   string                        `json:"appliedRevisionId,omitempty"`
	ChangedCueCount     int                           `json:"changedCueCount"`
	SourceDocument      SubtitleDocument              `json:"sourceDocument"`
	CandidateDocument   SubtitleDocument              `json:"candidateDocument"`
	Suggestions         []SubtitleReviewSuggestionDTO `json:"suggestions,omitempty"`
}

type ApplySubtitleReviewSessionResult struct {
	SessionID         string `json:"sessionId"`
	FileID            string `json:"fileId"`
	AppliedRevisionID string `json:"appliedRevisionId"`
	ChangedCueCount   int    `json:"changedCueCount"`
}

type DiscardSubtitleReviewSessionResult struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type SubtitleFixTyposRequest struct {
	FileID     string            `json:"fileId,omitempty"`
	DocumentID string            `json:"documentId,omitempty"`
	Path       string            `json:"path,omitempty"`
	Content    string            `json:"content,omitempty"`
	Format     string            `json:"format,omitempty"`
	Document   *SubtitleDocument `json:"document,omitempty"`
}

type SubtitleFixTyposResult struct {
	Format      string           `json:"format"`
	Content     string           `json:"content"`
	ChangeCount int              `json:"changeCount"`
	Document    SubtitleDocument `json:"document"`
}

type SubtitleSaveRequest struct {
	FileID       string            `json:"fileId,omitempty"`
	DocumentID   string            `json:"documentId,omitempty"`
	Path         string            `json:"path,omitempty"`
	Format       string            `json:"format,omitempty"`
	TargetFormat string            `json:"targetFormat,omitempty"`
	Content      string            `json:"content,omitempty"`
	Document     *SubtitleDocument `json:"document,omitempty"`
}

type SubtitleSaveResult struct {
	Path   string `json:"path,omitempty"`
	Format string `json:"format"`
	Bytes  int    `json:"bytes"`
}

type SubtitleTranslateRequest struct {
	FileID                string   `json:"fileId,omitempty"`
	DocumentID            string   `json:"documentId,omitempty"`
	Path                  string   `json:"path,omitempty"`
	LibraryID             string   `json:"libraryId,omitempty"`
	RootFileID            string   `json:"rootFileId,omitempty"`
	AssistantID           string   `json:"assistantId,omitempty"`
	TargetLanguage        string   `json:"targetLanguage"`
	OutputFormat          string   `json:"outputFormat,omitempty"`
	Mode                  string   `json:"mode,omitempty"`
	Source                string   `json:"source,omitempty"`
	GlossaryProfileIDs    []string `json:"glossaryProfileIds,omitempty"`
	ReferenceTrackFileIDs []string `json:"referenceTrackFileIds,omitempty"`
	PromptProfileIDs      []string `json:"promptProfileIds,omitempty"`
	InlinePrompt          string   `json:"inlinePrompt,omitempty"`
	SessionKey            string   `json:"sessionKey,omitempty"`
	RunID                 string   `json:"runId,omitempty"`
}

type SubtitleProofreadRequest struct {
	FileID             string   `json:"fileId,omitempty"`
	DocumentID         string   `json:"documentId,omitempty"`
	Path               string   `json:"path,omitempty"`
	LibraryID          string   `json:"libraryId,omitempty"`
	RootFileID         string   `json:"rootFileId,omitempty"`
	AssistantID        string   `json:"assistantId,omitempty"`
	Language           string   `json:"language,omitempty"`
	OutputFormat       string   `json:"outputFormat,omitempty"`
	Source             string   `json:"source,omitempty"`
	Spelling           bool     `json:"spelling,omitempty"`
	Punctuation        bool     `json:"punctuation,omitempty"`
	Terminology        bool     `json:"terminology,omitempty"`
	GlossaryProfileIDs []string `json:"glossaryProfileIds,omitempty"`
	PromptProfileIDs   []string `json:"promptProfileIds,omitempty"`
	InlinePrompt       string   `json:"inlinePrompt,omitempty"`
	SessionKey         string   `json:"sessionKey,omitempty"`
	RunID              string   `json:"runId,omitempty"`
}

type SubtitleQAReviewRequest struct {
	FileID              string `json:"fileId,omitempty"`
	DocumentID          string `json:"documentId,omitempty"`
	Path                string `json:"path,omitempty"`
	LibraryID           string `json:"libraryId,omitempty"`
	OutputFormat        string `json:"outputFormat,omitempty"`
	Source              string `json:"source,omitempty"`
	NormalizeWhitespace bool   `json:"normalizeWhitespace,omitempty"`
	SessionKey          string `json:"sessionKey,omitempty"`
	RunID               string `json:"runId,omitempty"`
}

type RestoreSubtitleOriginalRequest struct {
	FileID     string `json:"fileId,omitempty"`
	DocumentID string `json:"documentId,omitempty"`
	Path       string `json:"path,omitempty"`
}

type RestoreSubtitleOriginalResult struct {
	FileID string `json:"fileId,omitempty"`
	Format string `json:"format"`
	Bytes  int    `json:"bytes"`
}
