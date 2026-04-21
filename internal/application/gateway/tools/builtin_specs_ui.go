package tools

func specBrowser() toolSpec {
	waitConditionProperties := map[string]any{
		"timeMs":    map[string]any{"type": "number"},
		"text":      map[string]any{"type": "string"},
		"textGone":  map[string]any{"type": "string"},
		"selector":  map[string]any{"type": "string"},
		"url":       map[string]any{"type": "string"},
		"fn":        map[string]any{"type": "string"},
		"timeoutMs": map[string]any{"type": "number"},
	}
	waitConditionSchema := map[string]any{
		"type":                 "object",
		"properties":           waitConditionProperties,
		"additionalProperties": false,
	}
	return toolSpec{
		ID:            "browser",
		Name:          "browser",
		Description:   "Control a local CDP browser (`open`/`navigate`/`snapshot`/`act`/`wait`/`scroll`/`upload`/`dialog`/`reset`) using a browser-use style loop. For `open`, `navigate`, or `snapshot`, pass `url` or `targetUrl` when needed; these actions return `stateAvailable`, `itemCount`, and the current page `state`/`items` whenever capture succeeds, so review that result before deciding the next action. After the page changes, call `snapshot` to refresh refs, then continue with `act` using `ref` on the same `targetId`. Do not use raw CSS `selector` for normal interactions; use `ref` from the latest snapshot. Matching connector cookies are injected automatically before navigation.",
		PromptSnippet: "Interactive CDP browser. Loop: `open`/`navigate` -> `snapshot` -> `act` with the latest `ref`; after page changes, snapshot again. Prefer `ref` over `selector`.",
		Category:      "ui",
		RiskLevel:     "high",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{
						"open",
						"navigate",
						"snapshot",
						"act",
						"wait",
						"scroll",
						"upload",
						"dialog",
						"reset",
					},
				},
				"target": map[string]any{
					"type": "string",
					"enum": []string{"sandbox", "host", "node"},
				},
				"node":      map[string]any{"type": "string"},
				"targetUrl": map[string]any{"type": "string"},
				"targetId":  map[string]any{"type": "string"},
				"newTab":    map[string]any{"type": "boolean"},
				"restart":   map[string]any{"type": "boolean"},
				"limit":     map[string]any{"type": "integer"},
				"timeoutMs": map[string]any{"type": "integer"},
				"selector":  map[string]any{"type": "string"},
				"fullPage":  map[string]any{"type": "boolean"},
				"ref":       map[string]any{"type": "string"},
				"x":         map[string]any{"type": "integer"},
				"y":         map[string]any{"type": "integer"},
				"amount":    map[string]any{"type": "integer"},
				"direction": map[string]any{
					"type": "string",
					"enum": []string{"up", "down", "left", "right"},
				},
				"text":       map[string]any{"type": "string"},
				"textGone":   map[string]any{"type": "string"},
				"fn":         map[string]any{"type": "string"},
				"timeMs":     map[string]any{"type": "number"},
				"url":        map[string]any{"type": "string"},
				"paths":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"accept":     map[string]any{"type": "boolean"},
				"promptText": map[string]any{"type": "string"},
				"waitFor":    waitConditionSchema,
				"request": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"kind": map[string]any{
							"type": "string",
							"enum": []string{
								"click",
								"type",
								"press",
								"hover",
								"select",
								"fill",
								"resize",
								"wait",
								"evaluate",
								"close",
							},
						},
						"targetId":   map[string]any{"type": "string"},
						"ref":        map[string]any{"type": "string"},
						"text":       map[string]any{"type": "string"},
						"key":        map[string]any{"type": "string"},
						"value":      map[string]any{"type": "string"},
						"width":      map[string]any{"type": "number"},
						"height":     map[string]any{"type": "number"},
						"timeMs":     map[string]any{"type": "number"},
						"textGone":   map[string]any{"type": "string"},
						"selector":   map[string]any{"type": "string"},
						"url":        map[string]any{"type": "string"},
						"fn":         map[string]any{"type": "string"},
						"expression": map[string]any{"type": "string"},
						"timeoutMs":  map[string]any{"type": "number"},
						"waitFor":    waitConditionSchema,
					},
					"required": []string{"kind"},
				},
			},
			"required": []string{"action"},
			"allOf": []any{
				map[string]any{
					"anyOf": []any{
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{"const": "open"},
							},
							"required": []string{"action"},
							"anyOf": []any{
								map[string]any{"required": []string{"targetUrl"}},
								map[string]any{"required": []string{"url"}},
							},
						},
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{
									"enum": []string{
										"snapshot",
										"navigate",
										"wait",
										"scroll",
										"upload",
										"dialog",
										"act",
										"reset",
									},
								},
							},
							"required": []string{"action"},
						},
					},
				},
				map[string]any{
					"anyOf": []any{
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{"const": "navigate"},
							},
							"required": []string{"action"},
							"anyOf": []any{
								map[string]any{"required": []string{"targetUrl"}},
								map[string]any{"required": []string{"url"}},
							},
						},
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{
									"enum": []string{
										"open",
										"snapshot",
										"wait",
										"scroll",
										"upload",
										"dialog",
										"act",
										"reset",
									},
								},
							},
							"required": []string{"action"},
						},
					},
				},
				map[string]any{
					"anyOf": []any{
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{"const": "act"},
							},
							"required": []string{"action", "request"},
						},
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{
									"enum": []string{
										"open",
										"snapshot",
										"navigate",
										"wait",
										"scroll",
										"upload",
										"dialog",
										"reset",
									},
								},
							},
							"required": []string{"action"},
						},
					},
				},
				map[string]any{
					"anyOf": []any{
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{"const": "wait"},
							},
							"allOf": []any{
								map[string]any{
									"anyOf": []any{
										map[string]any{"required": []string{"timeMs"}},
										map[string]any{"required": []string{"text"}},
										map[string]any{"required": []string{"textGone"}},
										map[string]any{"required": []string{"selector"}},
										map[string]any{"required": []string{"url"}},
										map[string]any{"required": []string{"fn"}},
									},
								},
							},
						},
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{
									"enum": []string{
										"open",
										"snapshot",
										"navigate",
										"scroll",
										"upload",
										"dialog",
										"act",
										"reset",
									},
								},
							},
							"required": []string{"action"},
						},
					},
				},
				map[string]any{
					"anyOf": []any{
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{"const": "upload"},
							},
							"required": []string{"action", "ref", "paths"},
						},
						map[string]any{
							"properties": map[string]any{
								"action": map[string]any{
									"enum": []string{
										"open",
										"snapshot",
										"navigate",
										"wait",
										"scroll",
										"dialog",
										"act",
										"reset",
									},
								},
							},
							"required": []string{"action"},
						},
					},
				},
			},
		}),
		RequiresSandbox:  true,
		RequiresApproval: true,
		Enabled:          true,
	}
}

func specCanvas() toolSpec {
	return toolSpec{
		ID:          "canvas",
		Name:        "canvas",
		Description: "Control node canvases (present/hide/navigate/eval/snapshot/a2ui). Temporarily unavailable until remote node runtime support is implemented.",
		Category:    "ui",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{
						"present",
						"hide",
						"navigate",
						"eval",
						"snapshot",
						"a2ui_push",
						"a2ui_reset",
					},
				},
				"gatewayUrl":   map[string]any{"type": "string"},
				"gatewayToken": map[string]any{"type": "string"},
				"timeoutMs":    map[string]any{"type": "number"},
				"node":         map[string]any{"type": "string"},
				"target":       map[string]any{"type": "string"},
				"x":            map[string]any{"type": "number"},
				"y":            map[string]any{"type": "number"},
				"width":        map[string]any{"type": "number"},
				"height":       map[string]any{"type": "number"},
				"url":          map[string]any{"type": "string"},
				"javaScript":   map[string]any{"type": "string"},
				"outputFormat": map[string]any{
					"type": "string",
					"enum": []string{"png", "jpg", "jpeg"},
				},
				"maxWidth": map[string]any{"type": "number"},
				"quality":  map[string]any{"type": "number"},
				"delayMs":  map[string]any{"type": "number"},
				"jsonl":    map[string]any{"type": "string"},
				"jsonlPath": map[string]any{
					"type": "string",
				},
			},
			"required": []string{"action"},
		}),
		Enabled: false,
	}
}

func specImage() toolSpec {
	return toolSpec{
		ID:          "image",
		Name:        "image",
		Description: "Analyze one or more images with the configured image model.",
		Category:    "media",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string"},
				"image":  map[string]any{"type": "string"},
				"images": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"model":      map[string]any{"type": "string"},
				"maxBytesMb": map[string]any{"type": "number"},
				"maxImages":  map[string]any{"type": "number"},
			},
		}),
		Enabled: true,
	}
}
