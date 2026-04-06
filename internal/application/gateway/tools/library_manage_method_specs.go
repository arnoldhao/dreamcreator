package tools

import (
	"reflect"

	librarydto "dreamcreator/internal/application/library/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
)

type libraryManageMethodDefinition struct {
	name          string
	inputSchema   map[string]any
	inputExample  map[string]any
	outputSchema  map[string]any
	outputExample map[string]any
}

type libraryManageFollowUp struct {
	Tool        string `json:"tool"`
	Action      string `json:"action"`
	OperationID string `json:"operationId"`
	Guidance    string `json:"guidance,omitempty"`
}

type libraryManageAsyncAcceptedResult struct {
	OperationID string                         `json:"operationId"`
	LibraryID   string                         `json:"libraryId,omitempty"`
	Kind        string                         `json:"kind"`
	Status      string                         `json:"status"`
	Message     string                         `json:"message"`
	FollowUp    libraryManageFollowUp          `json:"followUp"`
	Operation   librarydto.LibraryOperationDTO `json:"operation"`
}

var libraryManageMethodDefinitions = []libraryManageMethodDefinition{
	{
		name:         "download.prepare",
		inputSchema:  libraryManageDownloadPrepareParamsSchema(),
		inputExample: map[string]any{"action": "download.prepare", "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(librarydto.PrepareYTDLPDownloadResponse{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"download.prepare",
			false,
			buildToolTypeEmptyValue(reflect.TypeOf(librarydto.PrepareYTDLPDownloadResponse{})),
		),
	},
	{
		name:         "download.parse",
		inputSchema:  libraryManageDownloadParseParamsSchema(),
		inputExample: map[string]any{"action": "download.parse", "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(librarydto.ParseYTDLPDownloadResponse{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"download.parse",
			false,
			buildToolTypeEmptyValue(reflect.TypeOf(librarydto.ParseYTDLPDownloadResponse{})),
		),
	},
	{
		name:         "download.create",
		inputSchema:  libraryManageDownloadCreateParamsSchema(),
		inputExample: map[string]any{"action": "download.create", "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "title": "Episode", "subtitleAuto": true},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"download.create",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "download.retry",
		inputSchema:  libraryManageOperationIDParamsSchema("Retry a failed download operation by id."),
		inputExample: map[string]any{"action": "download.retry", "operationId": "op_failed_download"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"download.retry",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "transcode.create",
		inputSchema:  libraryManageTranscodeCreateParamsSchema(),
		inputExample: map[string]any{"action": "transcode.create", "fileId": "file_video_123", "presetId": "h264_1080p"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"transcode.create",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "subtitle.translate.create",
		inputSchema:  libraryManageSubtitleTranslateParamsSchema(),
		inputExample: map[string]any{"action": "subtitle.translate.create", "fileId": "file_subtitle_123", "targetLanguage": "zh-CN"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"subtitle.translate.create",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "subtitle.proofread.create",
		inputSchema:  libraryManageSubtitleProofreadParamsSchema(),
		inputExample: map[string]any{"action": "subtitle.proofread.create", "fileId": "file_subtitle_123", "spelling": true, "punctuation": true},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"subtitle.proofread.create",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "subtitle.qa_review.create",
		inputSchema:  libraryManageSubtitleQAReviewParamsSchema(),
		inputExample: map[string]any{"action": "subtitle.qa_review.create", "fileId": "file_subtitle_123", "normalizeWhitespace": true},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"subtitle.qa_review.create",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
	{
		name:         "operation.cancel",
		inputSchema:  libraryManageOperationIDParamsSchema("Cancel a running or queued operation when that kind supports cancellation."),
		inputExample: map[string]any{"action": "operation.cancel", "operationId": "op_running_123"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(librarydto.LibraryOperationDTO{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"operation.cancel",
			false,
			buildToolTypeEmptyValue(reflect.TypeOf(librarydto.LibraryOperationDTO{})),
		),
	},
	{
		name:         "operation.resume",
		inputSchema:  libraryManageOperationIDParamsSchema("Resume a canceled or failed subtitle operation when checkpoint resume is supported."),
		inputExample: map[string]any{"action": "operation.resume", "operationId": "op_canceled_subtitle_123"},
		outputSchema: buildLibraryManageSuccessOutputSchema(buildToolTypeSchema(reflect.TypeOf(libraryManageAsyncAcceptedResult{}))),
		outputExample: buildLibraryManageSuccessOutputExample(
			"operation.resume",
			true,
			buildToolTypeEmptyValue(reflect.TypeOf(libraryManageAsyncAcceptedResult{})),
		),
	},
}

func libraryManageMethodSpecs() []tooldto.ToolMethodSpec {
	specs := make([]tooldto.ToolMethodSpec, 0, len(libraryManageMethodDefinitions))
	for _, definition := range libraryManageMethodDefinitions {
		specs = append(specs, tooldto.ToolMethodSpec{
			Name:          definition.name,
			InputSchema:   buildLibraryManageMethodInputSchema(definition.name, definition.inputSchema),
			OutputSchema:  definition.outputSchema,
			InputExample:  definition.inputExample,
			OutputExample: definition.outputExample,
		})
	}
	return specs
}

func libraryManageToolSchema() string {
	actions := make([]string, 0, len(libraryManageMethodDefinitions))
	properties := map[string]any{
		"action": map[string]any{
			"type": "string",
		},
		"type": map[string]any{
			"type": "string",
		},
		"params": map[string]any{
			"type": "object",
		},
	}
	for _, definition := range libraryManageMethodDefinitions {
		actions = append(actions, definition.name)
		paramProperties, _ := definition.inputSchema["properties"].(map[string]any)
		for key, value := range paramProperties {
			properties[key] = value
		}
	}
	return schemaJSON(map[string]any{
		"type":       "object",
		"properties": mergeLibraryActionEnum(properties, actions),
		"required":   []string{"action"},
	})
}

func buildLibraryManageMethodInputSchema(action string, paramsSchema map[string]any) map[string]any {
	paramsStyle := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":  "string",
				"const": action,
			},
			"params": cloneLibraryManageSchema(paramsSchema),
		},
		"required": []string{"action"},
	}
	flatStyle := buildLibraryManageFlatInputSchema(action, paramsSchema)
	if flatStyle == nil {
		return paramsStyle
	}
	return map[string]any{"oneOf": []any{paramsStyle, flatStyle}}
}

func buildLibraryManageFlatInputSchema(action string, paramsSchema map[string]any) map[string]any {
	if paramsSchema == nil {
		return nil
	}
	typ, _ := paramsSchema["type"].(string)
	if typ != "" && typ != "object" {
		return nil
	}

	flat := cloneLibraryManageSchema(paramsSchema)
	properties, _ := paramsSchema["properties"].(map[string]any)
	flatProperties := make(map[string]any, len(properties)+1)
	flatProperties["action"] = map[string]any{
		"type":  "string",
		"const": action,
	}
	for key, value := range properties {
		flatProperties[key] = value
	}
	flat["type"] = "object"
	flat["properties"] = flatProperties
	required := []string{"action"}
	required = append(required, readSchemaRequired(paramsSchema["required"])...)
	flat["required"] = required
	return flat
}

func cloneLibraryManageSchema(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func buildLibraryManageSuccessOutputSchema(resultSchema any) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ok": map[string]any{
				"type":  "boolean",
				"const": true,
			},
			"action": map[string]any{
				"type": "string",
			},
			"async": map[string]any{
				"type": "boolean",
			},
			"result": resultSchema,
		},
		"required": []string{"ok", "action", "async", "result"},
	}
}

func buildLibraryManageSuccessOutputExample(action string, async bool, result any) map[string]any {
	return map[string]any{
		"ok":     true,
		"action": action,
		"async":  async,
		"result": result,
	}
}

func libraryManageDownloadPrepareParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Target page URL to inspect before download.",
			},
		},
		"required": []string{"url"},
	}
}

func libraryManageDownloadParseParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Target page URL to parse for formats and subtitles.",
			},
			"connectorId": map[string]any{"type": "string"},
			"useConnector": map[string]any{
				"type":        "boolean",
				"description": "Read cookies from the selected connector during metadata parsing.",
			},
		},
		"required": []string{"url"},
	}
}

func libraryManageDownloadCreateParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":            map[string]any{"type": "string"},
			"libraryId":      map[string]any{"type": "string"},
			"title":          map[string]any{"type": "string"},
			"extractor":      map[string]any{"type": "string"},
			"author":         map[string]any{"type": "string"},
			"thumbnailUrl":   map[string]any{"type": "string"},
			"writeThumbnail": map[string]any{"type": "boolean"},
			"connectorId":    map[string]any{"type": "string"},
			"useConnector":   map[string]any{"type": "boolean"},
			"quality":        map[string]any{"type": "string"},
			"formatId":       map[string]any{"type": "string"},
			"audioFormatId":  map[string]any{"type": "string"},
			"subtitleLangs":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"subtitleAuto":   map[string]any{"type": "boolean"},
			"subtitleAll":    map[string]any{"type": "boolean"},
			"subtitleFormat": map[string]any{"type": "string"},
			"transcodePresetId": map[string]any{
				"type":        "string",
				"description": "Optional preset to enqueue an automatic follow-up transcode after download.",
			},
			"deleteSourceFileAfterTranscode": map[string]any{"type": "boolean"},
			"source":                         map[string]any{"type": "string"},
			"caller":                         map[string]any{"type": "string"},
			"sessionKey":                     map[string]any{"type": "string"},
			"runId":                          map[string]any{"type": "string"},
		},
		"required": []string{"url"},
	}
}

func libraryManageOperationIDParamsSchema(operationDescription string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operationId": map[string]any{
				"type":        "string",
				"description": operationDescription,
			},
			"source":     map[string]any{"type": "string"},
			"caller":     map[string]any{"type": "string"},
			"sessionKey": map[string]any{"type": "string"},
			"runId":      map[string]any{"type": "string"},
		},
		"required": []string{"operationId"},
	}
}

func libraryManageTranscodeCreateParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fileId":                  map[string]any{"type": "string"},
			"inputPath":               map[string]any{"type": "string"},
			"libraryId":               map[string]any{"type": "string"},
			"rootFileId":              map[string]any{"type": "string"},
			"presetId":                map[string]any{"type": "string"},
			"format":                  map[string]any{"type": "string"},
			"title":                   map[string]any{"type": "string"},
			"source":                  map[string]any{"type": "string"},
			"sessionKey":              map[string]any{"type": "string"},
			"runId":                   map[string]any{"type": "string"},
			"videoCodec":              map[string]any{"type": "string"},
			"qualityMode":             map[string]any{"type": "string"},
			"crf":                     map[string]any{"type": "integer"},
			"bitrateKbps":             map[string]any{"type": "integer"},
			"preset":                  map[string]any{"type": "string"},
			"audioCodec":              map[string]any{"type": "string"},
			"audioBitrateKbps":        map[string]any{"type": "integer"},
			"scale":                   map[string]any{"type": "string"},
			"width":                   map[string]any{"type": "integer"},
			"height":                  map[string]any{"type": "integer"},
			"subtitleHandling":        map[string]any{"type": "string"},
			"subtitleFileId":          map[string]any{"type": "string"},
			"secondarySubtitleFileId": map[string]any{"type": "string"},
			"displayMode":             map[string]any{"type": "string"},
			"subtitleDocumentId":      map[string]any{"type": "string"},
			"generatedSubtitleFormat": map[string]any{"type": "string"},
			"generatedSubtitleName":   map[string]any{"type": "string"},
			"generatedSubtitleContent": map[string]any{
				"type":        "string",
				"description": "Inline subtitle payload for embed or burn-in flows.",
			},
			"deleteSourceFileAfterTranscode": map[string]any{"type": "boolean"},
		},
		"anyOf": []any{
			map[string]any{"required": []string{"fileId"}},
			map[string]any{"required": []string{"inputPath"}},
		},
	}
}

func libraryManageSubtitleTranslateParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fileId":                map[string]any{"type": "string"},
			"documentId":            map[string]any{"type": "string"},
			"libraryId":             map[string]any{"type": "string"},
			"rootFileId":            map[string]any{"type": "string"},
			"assistantId":           map[string]any{"type": "string"},
			"targetLanguage":        map[string]any{"type": "string"},
			"outputFormat":          map[string]any{"type": "string"},
			"mode":                  map[string]any{"type": "string"},
			"source":                map[string]any{"type": "string"},
			"glossaryProfileIds":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"referenceTrackFileIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"promptProfileIds":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"inlinePrompt":          map[string]any{"type": "string"},
			"sessionKey":            map[string]any{"type": "string"},
			"runId":                 map[string]any{"type": "string"},
		},
		"required": []string{"targetLanguage"},
		"anyOf": []any{
			map[string]any{"required": []string{"fileId"}},
			map[string]any{"required": []string{"documentId"}},
		},
	}
}

func libraryManageSubtitleProofreadParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fileId":             map[string]any{"type": "string"},
			"documentId":         map[string]any{"type": "string"},
			"libraryId":          map[string]any{"type": "string"},
			"rootFileId":         map[string]any{"type": "string"},
			"assistantId":        map[string]any{"type": "string"},
			"language":           map[string]any{"type": "string"},
			"outputFormat":       map[string]any{"type": "string"},
			"source":             map[string]any{"type": "string"},
			"spelling":           map[string]any{"type": "boolean"},
			"punctuation":        map[string]any{"type": "boolean"},
			"terminology":        map[string]any{"type": "boolean"},
			"glossaryProfileIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"promptProfileIds":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"inlinePrompt":       map[string]any{"type": "string"},
			"sessionKey":         map[string]any{"type": "string"},
			"runId":              map[string]any{"type": "string"},
		},
		"anyOf": []any{
			map[string]any{"required": []string{"fileId"}},
			map[string]any{"required": []string{"documentId"}},
		},
	}
}

func libraryManageSubtitleQAReviewParamsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fileId":              map[string]any{"type": "string"},
			"documentId":          map[string]any{"type": "string"},
			"libraryId":           map[string]any{"type": "string"},
			"outputFormat":        map[string]any{"type": "string"},
			"source":              map[string]any{"type": "string"},
			"normalizeWhitespace": map[string]any{"type": "boolean"},
			"sessionKey":          map[string]any{"type": "string"},
			"runId":               map[string]any{"type": "string"},
		},
		"anyOf": []any{
			map[string]any{"required": []string{"fileId"}},
			map[string]any{"required": []string{"documentId"}},
		},
	}
}
