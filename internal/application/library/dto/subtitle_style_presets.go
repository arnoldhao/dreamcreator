package dto

import "encoding/json"

type AssStyleSpecDTO struct {
	Fontname        string  `json:"fontname"`
	Fontsize        float64 `json:"fontsize"`
	PrimaryColour   string  `json:"primaryColour"`
	SecondaryColour string  `json:"secondaryColour"`
	OutlineColour   string  `json:"outlineColour"`
	BackColour      string  `json:"backColour"`
	Bold            bool    `json:"bold"`
	Italic          bool    `json:"italic"`
	Underline       bool    `json:"underline"`
	StrikeOut       bool    `json:"strikeOut"`
	ScaleX          float64 `json:"scaleX"`
	ScaleY          float64 `json:"scaleY"`
	Spacing         float64 `json:"spacing"`
	Angle           float64 `json:"angle"`
	BorderStyle     int     `json:"borderStyle"`
	Outline         float64 `json:"outline"`
	Shadow          float64 `json:"shadow"`
	Alignment       int     `json:"alignment"`
	MarginL         int     `json:"marginL"`
	MarginR         int     `json:"marginR"`
	MarginV         int     `json:"marginV"`
	Encoding        int     `json:"encoding"`
}

type LibraryMonoStyleDTO struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	BuiltIn            bool            `json:"builtIn,omitempty"`
	BasePlayResX       int             `json:"basePlayResX"`
	BasePlayResY       int             `json:"basePlayResY"`
	BaseAspectRatio    string          `json:"baseAspectRatio"`
	SourceAssStyleName string          `json:"sourceAssStyleName,omitempty"`
	Style              AssStyleSpecDTO `json:"style"`
}

type LibraryMonoStyleSnapshotDTO struct {
	SourceMonoStyleID   string          `json:"sourceMonoStyleID,omitempty"`
	SourceMonoStyleName string          `json:"sourceMonoStyleName,omitempty"`
	Name                string          `json:"name"`
	BasePlayResX        int             `json:"basePlayResX"`
	BasePlayResY        int             `json:"basePlayResY"`
	BaseAspectRatio     string          `json:"baseAspectRatio"`
	Style               AssStyleSpecDTO `json:"style"`
}

type LibraryBilingualLayoutDTO struct {
	Gap         float64 `json:"gap"`
	BlockAnchor int     `json:"blockAnchor"`
}

type LibraryBilingualStyleDTO struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	BuiltIn         bool                        `json:"builtIn,omitempty"`
	BasePlayResX    int                         `json:"basePlayResX"`
	BasePlayResY    int                         `json:"basePlayResY"`
	BaseAspectRatio string                      `json:"baseAspectRatio"`
	Primary         LibraryMonoStyleSnapshotDTO `json:"primary"`
	Secondary       LibraryMonoStyleSnapshotDTO `json:"secondary"`
	Layout          LibraryBilingualLayoutDTO   `json:"layout"`
}

type DCSSPFileDTO struct {
	Format        string          `json:"format"`
	SchemaVersion int             `json:"schemaVersion"`
	Type          string          `json:"type"`
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Author        string          `json:"author,omitempty"`
	Homepage      string          `json:"homepage,omitempty"`
	Description   string          `json:"description,omitempty"`
	License       string          `json:"license,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	CreatedAt     string          `json:"createdAt,omitempty"`
	UpdatedAt     string          `json:"updatedAt,omitempty"`
	AppVersion    string          `json:"appVersion,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

type DCSSPMonoPayloadDTO struct {
	BasePlayResX    int             `json:"basePlayResX"`
	BasePlayResY    int             `json:"basePlayResY"`
	BaseAspectRatio string          `json:"baseAspectRatio"`
	Style           AssStyleSpecDTO `json:"style"`
}

type DCSSPBilingualPayloadDTO struct {
	BasePlayResX    int                         `json:"basePlayResX"`
	BasePlayResY    int                         `json:"basePlayResY"`
	BaseAspectRatio string                      `json:"baseAspectRatio"`
	Primary         LibraryMonoStyleSnapshotDTO `json:"primary"`
	Secondary       LibraryMonoStyleSnapshotDTO `json:"secondary"`
	Layout          LibraryBilingualLayoutDTO   `json:"layout"`
}

type GenerateSubtitleStylePreviewASSRequest struct {
	Type         string                        `json:"type"`
	Mono         *LibraryMonoStyleDTO          `json:"mono,omitempty"`
	Bilingual    *LibraryBilingualStyleDTO     `json:"bilingual,omitempty"`
	FontMappings []LibrarySubtitleStyleFontDTO `json:"fontMappings,omitempty"`
}

type GenerateSubtitleStylePreviewASSResult struct {
	ASSContent string `json:"assContent"`
}

type GenerateSubtitleStylePreviewVTTRequest struct {
	Type          string                        `json:"type"`
	Mono          *LibraryMonoStyleDTO          `json:"mono,omitempty"`
	Bilingual     *LibraryBilingualStyleDTO     `json:"bilingual,omitempty"`
	FontMappings  []LibrarySubtitleStyleFontDTO `json:"fontMappings,omitempty"`
	PrimaryText   string                        `json:"primaryText,omitempty"`
	SecondaryText string                        `json:"secondaryText,omitempty"`
	PreviewWidth  int                           `json:"previewWidth,omitempty"`
	PreviewHeight int                           `json:"previewHeight,omitempty"`
}

type GenerateSubtitleStylePreviewVTTResult struct {
	VTTContent string `json:"vttContent"`
}

type WorkspacePreviewCueDTO struct {
	StartMS       int64  `json:"startMs"`
	EndMS         int64  `json:"endMs"`
	PrimaryText   string `json:"primaryText"`
	SecondaryText string `json:"secondaryText,omitempty"`
}

type GenerateWorkspacePreviewVTTRequest struct {
	DisplayMode   string                        `json:"displayMode,omitempty"`
	Mono          *LibraryMonoStyleDTO          `json:"mono,omitempty"`
	Lingual       *LibraryBilingualStyleDTO     `json:"lingual,omitempty"`
	Rows          []WorkspacePreviewCueDTO      `json:"rows,omitempty"`
	FontMappings  []LibrarySubtitleStyleFontDTO `json:"fontMappings,omitempty"`
	PreviewWidth  int                           `json:"previewWidth,omitempty"`
	PreviewHeight int                           `json:"previewHeight,omitempty"`
}

type GenerateWorkspacePreviewVTTResult struct {
	VTTContent string `json:"vttContent"`
}

type ParseSubtitleStyleImportRequest struct {
	Content  string `json:"content"`
	Format   string `json:"format,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type ParseSubtitleStyleImportResult struct {
	ImportFormat       string                    `json:"importFormat"`
	DCSSP              *DCSSPFileDTO             `json:"dcssp,omitempty"`
	MonoStyles         []LibraryMonoStyleDTO     `json:"monoStyles,omitempty"`
	BilingualStyle     *LibraryBilingualStyleDTO `json:"bilingualStyle,omitempty"`
	DetectedRatio      string                    `json:"detectedRatio,omitempty"`
	NormalizedPlayResX int                       `json:"normalizedPlayResX,omitempty"`
	NormalizedPlayResY int                       `json:"normalizedPlayResY,omitempty"`
	Warnings           []string                  `json:"warnings,omitempty"`
}

type ExportSubtitleStylePresetRequest struct {
	DirectoryPath string                    `json:"directoryPath"`
	Type          string                    `json:"type"`
	Mono          *LibraryMonoStyleDTO      `json:"mono,omitempty"`
	Bilingual     *LibraryBilingualStyleDTO `json:"bilingual,omitempty"`
}

type ExportSubtitleStylePresetResult struct {
	ExportPath string `json:"exportPath"`
	FileName   string `json:"fileName"`
}
