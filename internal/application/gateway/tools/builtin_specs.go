package tools

import (
	"context"
	"encoding/json"

	tooldto "dreamcreator/internal/application/tools/dto"
)

type toolSpec struct {
	ID               string
	Name             string
	Description      string
	Kind             string
	SchemaJSON       string
	Methods          []tooldto.ToolMethodSpec
	Category         string
	RiskLevel        string
	RequiresSandbox  bool
	RequiresApproval bool
	Enabled          bool
}

func (spec toolSpec) toDTO() tooldto.ToolSpec {
	kind := spec.Kind
	if kind == "" {
		kind = "local"
	}
	enabled := spec.Enabled
	if !spec.Enabled {
		enabled = false
	} else if spec.ID != "" || spec.Name != "" {
		enabled = true
	}
	return tooldto.ToolSpec{
		ID:               spec.ID,
		Name:             spec.Name,
		Description:      spec.Description,
		Kind:             kind,
		SchemaJSON:       spec.SchemaJSON,
		Methods:          spec.Methods,
		Category:         spec.Category,
		RiskLevel:        spec.RiskLevel,
		RequiresSandbox:  spec.RequiresSandbox,
		RequiresApproval: spec.RequiresApproval,
		Enabled:          enabled,
	}
}

func specRead() toolSpec {
	return toolSpec{
		ID:          "read",
		Name:        "read",
		Description: "Read a file from disk.",
		Category:    "fs",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":     map[string]any{"type": "string"},
				"rootPath": map[string]any{"type": "string"},
				"maxChars": map[string]any{"type": "integer"},
			},
			"required": []string{"path"},
		}),
		Enabled: true,
	}
}

func specWrite() toolSpec {
	return toolSpec{
		ID:          "write",
		Name:        "write",
		Description: "Write content to a file.",
		Category:    "fs",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":     map[string]any{"type": "string"},
				"rootPath": map[string]any{"type": "string"},
				"content":  map[string]any{"type": "string"},
				"append":   map[string]any{"type": "boolean"},
			},
			"required": []string{"path", "content"},
		}),
		Enabled: true,
	}
}

func specEdit() toolSpec {
	return toolSpec{
		ID:          "edit",
		Name:        "edit",
		Description: "Replace text inside a file.",
		Category:    "fs",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string"},
				"rootPath":   map[string]any{"type": "string"},
				"oldText":    map[string]any{"type": "string"},
				"newText":    map[string]any{"type": "string"},
				"replaceAll": map[string]any{"type": "boolean"},
			},
			"required": []string{"path", "oldText", "newText"},
		}),
		Enabled: true,
	}
}

func specApplyPatch() toolSpec {
	return toolSpec{
		ID:          "apply_patch",
		Name:        "apply_patch",
		Description: "Apply a unified diff patch via git apply.",
		Category:    "fs",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"patch":    map[string]any{"type": "string"},
				"rootPath": map[string]any{"type": "string"},
			},
			"required": []string{"patch"},
		}),
		Enabled: true,
	}
}

func specExec() toolSpec {
	return toolSpec{
		ID:          "exec",
		Name:        "exec",
		Description: "Execute a command on the host.",
		Category:    "runtime",
		RiskLevel:   "high",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":        map[string]any{"type": "string"},
				"cmd":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"cwd":            map[string]any{"type": "string"},
				"env":            map[string]any{"type": "object"},
				"timeoutSeconds": map[string]any{"type": "integer"},
			},
		}),
		Enabled: true,
	}
}

func specProcess() toolSpec {
	return toolSpec{
		ID:          "process",
		Name:        "process",
		Description: "Spawn a background process.",
		Category:    "runtime",
		RiskLevel:   "high",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
				"cmd":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"cwd":     map[string]any{"type": "string"},
				"env":     map[string]any{"type": "object"},
			},
		}),
		Enabled: true,
	}
}

func specWebFetch() toolSpec {
	return toolSpec{
		ID:          "web_fetch",
		Name:        "web_fetch",
		Description: "Fetch a web page through a local CDP browser, extract token-efficient main content, and return structured status fields (status, retryable, next_action, quality). Do not blind-retry the same call when status is not ok.",
		Category:    "web",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":            map[string]any{"type": "string"},
				"method":         map[string]any{"type": "string"},
				"maxChars":       map[string]any{"type": "integer"},
				"maxBodyBytes":   map[string]any{"type": "integer"},
				"timeoutSeconds": map[string]any{"type": "integer"},
			},
			"required": []string{"url"},
		}),
		Enabled: true,
	}
}

func specWebSearch() toolSpec {
	return toolSpec{
		ID:          "web_search",
		Name:        "web_search",
		Description: "Search the web via configured API providers. External tools mode is reserved and currently not configured.",
		Category:    "web",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":           map[string]any{"type": "string"},
				"count":           map[string]any{"type": "integer"},
				"maxResults":      map[string]any{"type": "integer"},
				"country":         map[string]any{"type": "string"},
				"search_lang":     map[string]any{"type": "string"},
				"ui_lang":         map[string]any{"type": "string"},
				"freshness":       map[string]any{"type": "string"},
				"engine":          map[string]any{"type": "string"},
				"headers":         map[string]any{"type": "object"},
				"timeoutSeconds":  map[string]any{"type": "integer"},
				"acceptMarkdown":  map[string]any{"type": "boolean"},
				"enableUserAgent": map[string]any{"type": "boolean"},
				"userAgent":       map[string]any{"type": "string"},
				"acceptLanguage":  map[string]any{"type": "string"},
			},
			"required": []string{"query"},
		}),
		Enabled: true,
	}
}

func specMessage(ctx context.Context, settings SettingsReader) toolSpec {
	spec := specMessageBase()
	profile := resolveMessageToolSchemaProfile(ctx, settings)
	spec.SchemaJSON = resolveMessageToolSchemaJSON(spec.SchemaJSON, profile)
	return spec
}

func resolveMessageToolSchemaJSON(baseSchema string, profile messageToolSchemaProfile) string {
	if !profile.loaded {
		return baseSchema
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(baseSchema), &schema); err != nil {
		return baseSchema
	}
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		return baseSchema
	}
	actions := profile.actions
	if len(actions) == 0 {
		actions = []string{"send"}
	}
	enumValues := make([]any, 0, len(actions))
	for _, action := range actions {
		enumValues = append(enumValues, action)
	}
	actionSchema, _ := properties["action"].(map[string]any)
	if actionSchema == nil {
		actionSchema = map[string]any{
			"type": "string",
		}
	}
	actionSchema["enum"] = enumValues
	properties["action"] = actionSchema
	if !profile.includeButtons {
		delete(properties, "buttons")
	}
	if !profile.includeCards {
		delete(properties, "card")
	}
	if !profile.includeComponents {
		delete(properties, "components")
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return baseSchema
	}
	return string(encoded)
}

func specMessageBase() toolSpec {
	return toolSpec{
		ID:          "message",
		Name:        "message",
		Description: "Send, delete, and manage messages via channel plugins.",
		Category:    "messaging",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{
						"send",
						"broadcast",
						"poll",
						"react",
						"reactions",
						"read",
						"edit",
						"unsend",
						"reply",
						"sendWithEffect",
						"renameGroup",
						"setGroupIcon",
						"addParticipant",
						"removeParticipant",
						"leaveGroup",
						"sendAttachment",
						"delete",
						"pin",
						"unpin",
						"list-pins",
						"permissions",
						"thread-create",
						"thread-list",
						"thread-reply",
						"search",
						"sticker",
						"sticker-search",
						"member-info",
						"role-info",
						"emoji-list",
						"emoji-upload",
						"sticker-upload",
						"role-add",
						"role-remove",
						"channel-info",
						"channel-list",
						"channel-create",
						"channel-edit",
						"channel-delete",
						"channel-move",
						"category-create",
						"category-edit",
						"category-delete",
						"topic-create",
						"voice-status",
						"event-list",
						"event-create",
						"timeout",
						"kick",
						"ban",
						"set-presence",
					},
				},
				"channel":   map[string]any{"type": "string"},
				"target":    map[string]any{"type": "string"},
				"targets":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"accountId": map[string]any{"type": "string"},
				"dryRun":    map[string]any{"type": "boolean"},
				"message":   map[string]any{"type": "string"},
				"effectId":  map[string]any{"type": "string"},
				"effect":    map[string]any{"type": "string"},
				"media":     map[string]any{"type": "string"},
				"filename":  map[string]any{"type": "string"},
				"buffer":    map[string]any{"type": "string"},
				"contentType": map[string]any{
					"type": "string",
				},
				"mimeType": map[string]any{
					"type": "string",
				},
				"caption":   map[string]any{"type": "string"},
				"path":      map[string]any{"type": "string"},
				"filePath":  map[string]any{"type": "string"},
				"replyTo":   map[string]any{"type": "string"},
				"threadId":  map[string]any{"type": "string"},
				"asVoice":   map[string]any{"type": "boolean"},
				"silent":    map[string]any{"type": "boolean"},
				"quoteText": map[string]any{"type": "string"},
				"bestEffort": map[string]any{
					"type": "boolean",
				},
				"gifPlayback": map[string]any{
					"type": "boolean",
				},
				"buttons": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text":          map[string]any{"type": "string"},
								"callback_data": map[string]any{"type": "string"},
								"style": map[string]any{
									"type": "string",
									"enum": []string{"danger", "success", "primary"},
								},
							},
							"required": []string{"text", "callback_data"},
						},
					},
					"description": "Telegram inline keyboard buttons (array of button rows)",
				},
				"card": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
					"description":          "Adaptive Card JSON object (when supported by the channel)",
				},
				"components": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text":     map[string]any{"type": "string"},
						"reusable": map[string]any{"type": "boolean"},
						"container": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"accentColor": map[string]any{"type": "string"},
								"spoiler":     map[string]any{"type": "boolean"},
							},
						},
						"blocks": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"type":  map[string]any{"type": "string"},
									"text":  map[string]any{"type": "string"},
									"texts": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
									"accessory": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"type": map[string]any{"type": "string"},
											"url":  map[string]any{"type": "string"},
											"button": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"label":    map[string]any{"type": "string"},
													"url":      map[string]any{"type": "string"},
													"disabled": map[string]any{"type": "boolean"},
												},
												"required": []string{"label"},
											},
										},
									},
									"spacing": map[string]any{
										"type": "string",
										"enum": []string{"small", "large"},
									},
									"divider": map[string]any{"type": "boolean"},
									"buttons": map[string]any{
										"type": "array",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"label": map[string]any{"type": "string"},
												"style": map[string]any{
													"type": "string",
													"enum": []string{"primary", "secondary", "success", "danger", "link"},
												},
												"url":      map[string]any{"type": "string"},
												"disabled": map[string]any{"type": "boolean"},
											},
											"required": []string{"label"},
										},
									},
									"select": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"type": map[string]any{
												"type": "string",
												"enum": []string{"string", "user", "role", "mentionable", "channel"},
											},
											"placeholder": map[string]any{"type": "string"},
											"minValues":   map[string]any{"type": "number"},
											"maxValues":   map[string]any{"type": "number"},
										},
									},
									"items": map[string]any{
										"type": "array",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"url":         map[string]any{"type": "string"},
												"description": map[string]any{"type": "string"},
												"spoiler":     map[string]any{"type": "boolean"},
											},
											"required": []string{"url"},
										},
									},
									"file":    map[string]any{"type": "string"},
									"spoiler": map[string]any{"type": "boolean"},
								},
								"required": []string{"type"},
							},
						},
						"modal": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"title":        map[string]any{"type": "string"},
								"triggerLabel": map[string]any{"type": "string"},
								"triggerStyle": map[string]any{
									"type": "string",
									"enum": []string{"primary", "secondary", "success", "danger", "link"},
								},
								"fields": map[string]any{
									"type": "array",
									"items": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"type":        map[string]any{"type": "string"},
											"name":        map[string]any{"type": "string"},
											"label":       map[string]any{"type": "string"},
											"description": map[string]any{"type": "string"},
											"placeholder": map[string]any{"type": "string"},
											"required":    map[string]any{"type": "boolean"},
											"minValues":   map[string]any{"type": "number"},
											"maxValues":   map[string]any{"type": "number"},
											"minLength":   map[string]any{"type": "number"},
											"maxLength":   map[string]any{"type": "number"},
											"style": map[string]any{
												"type": "string",
												"enum": []string{"short", "paragraph"},
											},
										},
										"required": []string{"type", "label"},
									},
								},
							},
							"required": []string{"title", "fields"},
						},
					},
					"description": "Discord components v2 payload.",
				},
				"messageId":  map[string]any{"type": "string"},
				"message_id": map[string]any{"type": "string"},
				"emoji":      map[string]any{"type": "string"},
				"remove":     map[string]any{"type": "boolean"},
				"targetAuthor": map[string]any{
					"type": "string",
				},
				"targetAuthorUuid": map[string]any{
					"type": "string",
				},
				"groupId": map[string]any{
					"type": "string",
				},
				"limit":           map[string]any{"type": "number"},
				"before":          map[string]any{"type": "string"},
				"after":           map[string]any{"type": "string"},
				"around":          map[string]any{"type": "string"},
				"fromMe":          map[string]any{"type": "boolean"},
				"includeArchived": map[string]any{"type": "boolean"},
				"pollQuestion":    map[string]any{"type": "string"},
				"pollOption":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"pollDurationHours": map[string]any{
					"type": "number",
				},
				"pollMulti": map[string]any{
					"type": "boolean",
				},
				"channelId":  map[string]any{"type": "string"},
				"channelIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"guildId":    map[string]any{"type": "string"},
				"userId":     map[string]any{"type": "string"},
				"authorId":   map[string]any{"type": "string"},
				"authorIds":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"roleId":     map[string]any{"type": "string"},
				"roleIds":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"participant": map[string]any{
					"type": "string",
				},
				"emojiName": map[string]any{
					"type": "string",
				},
				"stickerId": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"stickerName": map[string]any{
					"type": "string",
				},
				"stickerDesc": map[string]any{
					"type": "string",
				},
				"stickerTags": map[string]any{
					"type": "string",
				},
				"threadName": map[string]any{
					"type": "string",
				},
				"autoArchiveMin": map[string]any{
					"type": "number",
				},
				"query":     map[string]any{"type": "string"},
				"eventName": map[string]any{"type": "string"},
				"eventType": map[string]any{"type": "string"},
				"startTime": map[string]any{"type": "string"},
				"endTime":   map[string]any{"type": "string"},
				"desc":      map[string]any{"type": "string"},
				"location":  map[string]any{"type": "string"},
				"durationMin": map[string]any{
					"type": "number",
				},
				"until": map[string]any{
					"type": "string",
				},
				"reason": map[string]any{
					"type": "string",
				},
				"deleteDays": map[string]any{
					"type": "number",
				},
				"gatewayUrl":   map[string]any{"type": "string"},
				"gatewayToken": map[string]any{"type": "string"},
				"timeoutMs":    map[string]any{"type": "number"},
				"name":         map[string]any{"type": "string"},
				"type":         map[string]any{"type": "number"},
				"parentId":     map[string]any{"type": "string"},
				"topic":        map[string]any{"type": "string"},
				"position":     map[string]any{"type": "number"},
				"nsfw":         map[string]any{"type": "boolean"},
				"rateLimitPerUser": map[string]any{
					"type": "number",
				},
				"categoryId": map[string]any{
					"type": "string",
				},
				"clearParent": map[string]any{
					"type": "boolean",
				},
				"activityType": map[string]any{
					"type": "string",
				},
				"activityName": map[string]any{
					"type": "string",
				},
				"activityUrl": map[string]any{
					"type": "string",
				},
				"activityState": map[string]any{
					"type": "string",
				},
				"status": map[string]any{
					"type": "string",
				},
			},
			"required": []string{"action"},
		}),
		Enabled: true,
	}
}

func specGateway() toolSpec {
	return toolSpec{
		ID:          "gateway",
		Name:        "gateway",
		Description: "Invoke gateway control plane actions.",
		Category:    "automation",
		RiskLevel:   "high",
		Methods:     gatewayMethodSpecs(),
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": append([]string(nil), gatewayToolActions...),
				},
				"method": map[string]any{
					"type": "string",
					"enum": append([]string(nil), gatewayToolActions...),
				},
				"params": map[string]any{"type": "object"},
				"path": map[string]any{
					"type": "string",
				},
				"value": map[string]any{},
				"ops": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "object"},
				},
				"dryRun": map[string]any{
					"type": "boolean",
				},
				"expectedVersion": map[string]any{
					"type": "integer",
				},
				"config": map[string]any{
					"type": "object",
				},
				"mode": map[string]any{
					"type": "string",
				},
			},
			"required": []string{"action"},
		}),
		Enabled: true,
	}
}

func specCron() toolSpec {
	return toolSpec{
		ID:          "cron",
		Name:        "cron",
		Description: "Cron manager. Input: {action,params}. Actions: status|list|add|update|remove|run|runs|wake. Pairing: main=>systemEvent+text; isolated=>agentTurn+message. schedule: every/everyMs | cron/expr | at/at. announce channel: default|app|telegram. add auto-inherits runtime sessionKey when omitted.",
		Category:    "automation",
		RiskLevel:   "high",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"status", "list", "add", "update", "remove", "run", "runs", "wake"},
				},
				"params": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
			"required": []string{"action"},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "status"},
						},
					},
					"then": map[string]any{
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "list"},
						},
					},
					"then": map[string]any{
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"properties": map[string]any{
									"includeDisabled": map[string]any{"type": "boolean"},
									"enabled":         map[string]any{"type": "string", "enum": []string{"all", "enabled", "disabled"}},
									"query":           map[string]any{"type": "string"},
									"sortBy":          map[string]any{"type": "string", "enum": []string{"nextRunAtMs", "updatedAtMs", "name"}},
									"sortDir":         map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
									"limit":           map[string]any{"type": "integer", "minimum": 0},
									"offset":          map[string]any{"type": "integer", "minimum": 0},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "add"},
						},
					},
					"then": map[string]any{
						"required": []string{"params"},
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"name", "enabled", "schedule", "sessionTarget", "wakeMode", "payload"},
								"properties": map[string]any{
									"id":             map[string]any{"type": "string"},
									"name":           map[string]any{"type": "string"},
									"description":    map[string]any{"type": "string"},
									"enabled":        map[string]any{"type": "boolean"},
									"deleteAfterRun": map[string]any{"type": "boolean"},
									"sessionTarget":  map[string]any{"type": "string", "enum": []string{"main", "isolated"}},
									"wakeMode":       map[string]any{"type": "string", "enum": []string{"now", "next-heartbeat"}},
									"sessionKey":     map[string]any{"type": "string"},
									"schedule":       cronScheduleSchema(),
									"payload":        cronPayloadSchema(),
									"delivery":       cronDeliverySchema(),
								},
								"allOf": []any{
									map[string]any{
										"if": map[string]any{
											"properties": map[string]any{
												"sessionTarget": map[string]any{"const": "main"},
											},
										},
										"then": map[string]any{
											"properties": map[string]any{
												"payload": map[string]any{
													"required": []string{"kind", "text"},
													"properties": map[string]any{
														"kind": map[string]any{"const": "systemEvent"},
													},
												},
											},
										},
									},
									map[string]any{
										"if": map[string]any{
											"properties": map[string]any{
												"sessionTarget": map[string]any{"const": "isolated"},
											},
										},
										"then": map[string]any{
											"properties": map[string]any{
												"payload": map[string]any{
													"required": []string{"kind", "message"},
													"properties": map[string]any{
														"kind": map[string]any{"const": "agentTurn"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "update"},
						},
					},
					"then": map[string]any{
						"required": []string{"params"},
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"id", "patch"},
								"properties": map[string]any{
									"id": map[string]any{"type": "string"},
									"patch": map[string]any{
										"type":                 "object",
										"additionalProperties": false,
										"minProperties":        1,
										"properties": map[string]any{
											"name":           map[string]any{"type": "string"},
											"description":    map[string]any{"type": "string"},
											"enabled":        map[string]any{"type": "boolean"},
											"deleteAfterRun": map[string]any{"type": "boolean"},
											"sessionTarget":  map[string]any{"type": "string", "enum": []string{"main", "isolated"}},
											"wakeMode":       map[string]any{"type": "string", "enum": []string{"now", "next-heartbeat"}},
											"sessionKey":     map[string]any{"type": "string"},
											"schedule":       cronScheduleSchema(),
											"payload":        cronPayloadSchema(),
											"delivery":       cronDeliverySchema(),
										},
									},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "remove"},
						},
					},
					"then": map[string]any{
						"required": []string{"params"},
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"id"},
								"properties": map[string]any{
									"id": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "run"},
						},
					},
					"then": map[string]any{
						"required": []string{"params"},
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"id", "mode"},
								"properties": map[string]any{
									"id":   map[string]any{"type": "string"},
									"mode": map[string]any{"type": "string", "enum": []string{"due", "force"}},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "runs"},
						},
					},
					"then": map[string]any{
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"properties": map[string]any{
									"scope":            map[string]any{"type": "string", "enum": []string{"job", "all"}},
									"id":               map[string]any{"type": "string"},
									"statuses":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
									"deliveryStatuses": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
									"query":            map[string]any{"type": "string"},
									"sortDir":          map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
									"limit":            map[string]any{"type": "integer", "minimum": 0},
									"offset":           map[string]any{"type": "integer", "minimum": 0},
								},
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "wake"},
						},
					},
					"then": map[string]any{
						"required": []string{"params"},
						"properties": map[string]any{
							"params": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []string{"mode", "text"},
								"properties": map[string]any{
									"mode":       map[string]any{"type": "string", "enum": []string{"now", "next-heartbeat"}},
									"text":       map[string]any{"type": "string"},
									"sessionKey": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
		}),
		Enabled: true,
	}
}

func cronScheduleSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"kind"},
		"properties": map[string]any{
			"kind":      map[string]any{"type": "string", "enum": []string{"at", "every", "cron"}},
			"at":        map[string]any{"type": "string"},
			"everyMs":   map[string]any{"type": "integer", "minimum": 1},
			"anchorMs":  map[string]any{"type": "integer"},
			"expr":      map[string]any{"type": "string"},
			"tz":        map[string]any{"type": "string"},
			"staggerMs": map[string]any{"type": "integer", "minimum": 0},
		},
		"oneOf": []any{
			map[string]any{
				"required": []string{"kind", "at"},
				"properties": map[string]any{
					"kind": map[string]any{"const": "at"},
				},
			},
			map[string]any{
				"required": []string{"kind", "everyMs"},
				"properties": map[string]any{
					"kind": map[string]any{"const": "every"},
				},
			},
			map[string]any{
				"required": []string{"kind", "expr"},
				"properties": map[string]any{
					"kind": map[string]any{"const": "cron"},
				},
			},
		},
	}
}

func cronPayloadSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"kind"},
		"properties": map[string]any{
			"kind":           map[string]any{"type": "string", "enum": []string{"systemEvent", "agentTurn"}},
			"text":           map[string]any{"type": "string", "description": "Use only when payload.kind=systemEvent."},
			"message":        map[string]any{"type": "string", "description": "Use only when payload.kind=agentTurn."},
			"model":          map[string]any{"type": "string"},
			"thinking":       map[string]any{"type": "string"},
			"timeoutSeconds": map[string]any{"type": "integer", "minimum": 0},
			"lightContext":   map[string]any{"type": "boolean"},
		},
		"oneOf": []any{
			map[string]any{
				"required": []string{"kind", "text"},
				"properties": map[string]any{
					"kind": map[string]any{"const": "systemEvent"},
				},
			},
			map[string]any{
				"required": []string{"kind", "message"},
				"properties": map[string]any{
					"kind": map[string]any{"const": "agentTurn"},
				},
			},
		},
	}
}

func cronDeliverySchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"mode"},
		"properties": map[string]any{
			"mode":               map[string]any{"type": "string", "enum": []string{"none", "announce", "webhook"}},
			"channel":            map[string]any{"type": "string", "enum": []string{"default", "app", "telegram"}},
			"to":                 map[string]any{"type": "string"},
			"accountId":          map[string]any{"type": "string"},
			"bestEffort":         map[string]any{"type": "boolean"},
			"failureDestination": cronFailureDestinationSchema(),
		},
		"allOf": []any{
			map[string]any{
				"if": map[string]any{
					"properties": map[string]any{
						"mode": map[string]any{"const": "webhook"},
					},
				},
				"then": map[string]any{
					"required": []string{"mode", "to"},
				},
			},
		},
	}
}

func cronFailureDestinationSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"mode":      map[string]any{"type": "string", "enum": []string{"announce", "webhook"}},
			"channel":   map[string]any{"type": "string", "enum": []string{"default", "app", "telegram"}},
			"to":        map[string]any{"type": "string"},
			"accountId": map[string]any{"type": "string"},
		},
		"allOf": []any{
			map[string]any{
				"if": map[string]any{
					"properties": map[string]any{
						"mode": map[string]any{"const": "webhook"},
					},
				},
				"then": map[string]any{
					"required": []string{"to"},
				},
			},
		},
	}
}

func specAgentsList() toolSpec {
	return toolSpec{
		ID:          "agents_list",
		Name:        "agents_list",
		Description: "List available agent profiles for subagent spawning.",
		Category:    "sessions",
		Enabled:     true,
	}
}

func specSessionsList() toolSpec {
	return toolSpec{
		ID:          "sessions_list",
		Name:        "sessions_list",
		Description: "List current sessions.",
		Category:    "sessions",
		Enabled:     true,
	}
}

func specSessionsHistory() toolSpec {
	return toolSpec{
		ID:          "sessions_history",
		Name:        "sessions_history",
		Description: "Read message history for a session.",
		Category:    "sessions",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sessionId": map[string]any{"type": "string"},
				"threadId":  map[string]any{"type": "string"},
				"limit":     map[string]any{"type": "integer"},
			},
		}),
		Enabled: true,
	}
}

func specSessionsSend() toolSpec {
	return toolSpec{
		ID:          "sessions_send",
		Name:        "sessions_send",
		Description: "Append a message to an existing session.",
		Category:    "sessions",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sessionId": map[string]any{"type": "string"},
				"threadId":  map[string]any{"type": "string"},
				"role":      map[string]any{"type": "string"},
				"content":   map[string]any{"type": "string"},
			},
			"required": []string{"content"},
		}),
		Enabled: true,
	}
}

func specSessionsSpawn() toolSpec {
	return toolSpec{
		ID:          "sessions_spawn",
		Name:        "sessions_spawn",
		Description: "Spawn an isolated subagent run.",
		Category:    "sessions",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task":              map[string]any{"type": "string"},
				"label":             map[string]any{"type": "string"},
				"agentId":           map[string]any{"type": "string"},
				"model":             map[string]any{"type": "string"},
				"thinking":          map[string]any{"type": "string"},
				"runTimeoutSeconds": map[string]any{"type": "integer"},
				"cleanup":           map[string]any{"type": "string", "enum": []string{"keep", "delete"}},
			},
			"required": []string{"task"},
		}),
		Enabled: true,
	}
}

func specSessionStatus() toolSpec {
	return toolSpec{
		ID:          "session_status",
		Name:        "session_status",
		Description: "Get session metadata and status.",
		Category:    "sessions",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sessionId": map[string]any{"type": "string"},
			},
		}),
		Enabled: true,
	}
}

func specExternalToolsQuery() toolSpec {
	return toolSpec{
		ID:          "external_tools_query",
		Name:        "external_tools_query",
		Description: "Query external tools state, progress, and updates.",
		Category:    "external_tools",
		RiskLevel:   "low",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"list", "status", "install_state", "updates"},
				},
				"name": map[string]any{"type": "string"},
			},
		}),
		Enabled: true,
	}
}

func specExternalToolsManage() toolSpec {
	return toolSpec{
		ID:          "external_tools_manage",
		Name:        "external_tools_manage",
		Description: "Manage external tools lifecycle (install/verify/reinstall/remove/set_path).",
		Category:    "external_tools",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"install", "verify", "reinstall", "remove", "set_path"},
				},
				"name":    map[string]any{"type": "string"},
				"version": map[string]any{"type": "string"},
				"manager": map[string]any{"type": "string", "enum": []string{"npm", "pnpm", "bun"}},
				"execPath": map[string]any{
					"type": "string",
				},
			},
			"required": []string{"action", "name"},
		}),
		Enabled: true,
	}
}

func specSkills() toolSpec {
	actions := []string{"status", "bins", "install", "update"}
	return toolSpec{
		ID:          "skills",
		Name:        "skills",
		Description: "Inspect skills runtime status and update per-skill runtime dependencies or configuration.",
		Category:    "skills",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"action"},
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": actions,
				},
				"assistantId": map[string]any{"type": "string"},
				"providerId":  map[string]any{"type": "string"},
				"skill":       map[string]any{"type": "string"},
				"skillKey":    map[string]any{"type": "string"},
				"id":          map[string]any{"type": "string"},
				"name":        map[string]any{"type": "string"},
				"enabled":     map[string]any{"type": "boolean"},
				"apiKey":      map[string]any{"type": "string"},
				"env": map[string]any{
					"type":                 "object",
					"additionalProperties": map[string]any{"type": "string"},
				},
				"config": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
				"installId": map[string]any{"type": "string"},
				"timeoutMs": map[string]any{"type": "integer", "minimum": 1},
			},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{
								"const": "install",
							},
						},
					},
					"then": map[string]any{
						"anyOf": []any{
							map[string]any{"required": []string{"skill"}},
							map[string]any{"required": []string{"skillKey"}},
							map[string]any{"required": []string{"name"}},
							map[string]any{"required": []string{"id"}},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{
								"const": "update",
							},
						},
					},
					"then": map[string]any{
						"allOf": []any{
							map[string]any{
								"anyOf": []any{
									map[string]any{"required": []string{"skill"}},
									map[string]any{"required": []string{"skillKey"}},
									map[string]any{"required": []string{"name"}},
									map[string]any{"required": []string{"id"}},
								},
							},
							map[string]any{
								"anyOf": []any{
									map[string]any{"required": []string{"enabled"}},
									map[string]any{"required": []string{"apiKey"}},
									map[string]any{"required": []string{"env"}},
									map[string]any{"required": []string{"config"}},
								},
							},
						},
					},
				},
			},
		}),
		Enabled: true,
	}
}

func specSkillsManage() toolSpec {
	actions := []string{"list", "search", "install", "update", "remove", "sync"}
	return toolSpec{
		ID:          "skills_manage",
		Name:        "skills_manage",
		Description: "Search, install, update, remove, and sync skill packages via ClawHub.",
		Category:    "skills",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"action"},
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": actions,
				},
				"assistantId": map[string]any{"type": "string"},
				"providerId":  map[string]any{"type": "string"},
				"query":       map[string]any{"type": "string"},
				"limit":       map[string]any{"type": "integer", "minimum": 1},
				"skill":       map[string]any{"type": "string"},
				"skillKey":    map[string]any{"type": "string"},
				"id":          map[string]any{"type": "string"},
				"name":        map[string]any{"type": "string"},
				"version":     map[string]any{"type": "string"},
				"force":       map[string]any{"type": "boolean"},
			},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{
								"const": "search",
							},
						},
					},
					"then": map[string]any{
						"required": []string{"query"},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{
								"enum": []string{"install", "update", "remove"},
							},
						},
					},
					"then": map[string]any{
						"anyOf": []any{
							map[string]any{"required": []string{"skill"}},
							map[string]any{"required": []string{"skillKey"}},
							map[string]any{"required": []string{"name"}},
							map[string]any{"required": []string{"id"}},
						},
					},
				},
			},
		}),
		Enabled: true,
	}
}

func specSubagents() toolSpec {
	return toolSpec{
		ID:          "subagents",
		Name:        "subagents",
		Description: "Manage existing subagent runs (list/info/log/kill/steer/send).",
		Category:    "sessions",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action":  map[string]any{"type": "string"},
				"target":  map[string]any{"type": "string"},
				"message": map[string]any{"type": "string"},
				"limit":   map[string]any{"type": "integer"},
			},
		}),
		Enabled: true,
	}
}

func specNodes() toolSpec {
	return toolSpec{
		ID:          "nodes",
		Name:        "nodes",
		Description: "Experimental low-level RPC to a registered node. Temporarily unavailable until remote node runtime support is implemented; prefer specialized tools like canvas or browser when available.",
		Category:    "nodes",
		RiskLevel:   "medium",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nodeId":     map[string]any{"type": "string"},
				"capability": map[string]any{"type": "string"},
				"action":     map[string]any{"type": "string"},
				"args":       map[string]any{"type": "string"},
				"payload":    map[string]any{"type": "object"},
				"timeoutMs":  map[string]any{"type": "integer"},
			},
			"required": []string{"nodeId", "capability"},
		}),
		Enabled: false,
	}
}

func specTTS() toolSpec {
	return toolSpec{
		ID:          "tts",
		Name:        "tts",
		Description: "Synthesize speech audio from text with the configured voice provider.",
		Category:    "voice",
		SchemaJSON: schemaJSON(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text":       map[string]any{"type": "string"},
				"providerId": map[string]any{"type": "string"},
				"voiceId":    map[string]any{"type": "string"},
				"modelId":    map[string]any{"type": "string"},
				"format":     map[string]any{"type": "string"},
			},
		}),
		Enabled: true,
	}
}

func specMemoryQuery() toolSpec {
	return toolSpec{
		ID:          "memory_query",
		Name:        "memory_query",
		Description: "Query long-term memory with recall, list, and stats actions.",
		Category:    "memory",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"recall", "list", "stats"},
				},
				"query":       map[string]any{"type": "string"},
				"limit":       map[string]any{"type": "integer"},
				"topK":        map[string]any{"type": "integer"},
				"assistantId": map[string]any{"type": "string"},
				"threadId":    map[string]any{"type": "string"},
				"category":    map[string]any{"type": "string"},
				"scope":       map[string]any{"type": "string"},
				"channel":     map[string]any{"type": "string"},
				"accountId":   map[string]any{"type": "string"},
				"userId":      map[string]any{"type": "string"},
				"groupId":     map[string]any{"type": "string"},
				"peerKind":    map[string]any{"type": "string"},
				"peerId":      map[string]any{"type": "string"},
			},
			"required": []string{"action"},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "recall"},
						},
					},
					"then": map[string]any{
						"required": []string{"query"},
					},
				},
			},
		}),
		Enabled: true,
	}
}

func specMemoryManage() toolSpec {
	return toolSpec{
		ID:          "memory_manage",
		Name:        "memory_manage",
		Description: "Create, update, or delete long-term memory entries.",
		Category:    "memory",
		SchemaJSON: schemaJSON(map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"store", "update", "forget"},
				},
				"memoryId":    map[string]any{"type": "string"},
				"id":          map[string]any{"type": "string"},
				"text":        map[string]any{"type": "string"},
				"content":     map[string]any{"type": "string"},
				"category":    map[string]any{"type": "string"},
				"confidence":  map[string]any{"type": "number"},
				"assistantId": map[string]any{"type": "string"},
				"threadId":    map[string]any{"type": "string"},
				"scope":       map[string]any{"type": "string"},
				"metadata":    map[string]any{"type": "object"},
				"query":       map[string]any{"type": "string"},
				"limit":       map[string]any{"type": "integer"},
				"channel":     map[string]any{"type": "string"},
				"accountId":   map[string]any{"type": "string"},
				"userId":      map[string]any{"type": "string"},
				"groupId":     map[string]any{"type": "string"},
				"peerKind":    map[string]any{"type": "string"},
				"peerId":      map[string]any{"type": "string"},
			},
			"required": []string{"action"},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "store"},
						},
					},
					"then": map[string]any{
						"anyOf": []any{
							map[string]any{"required": []string{"text"}},
							map[string]any{"required": []string{"content"}},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "update"},
						},
					},
					"then": map[string]any{
						"anyOf": []any{
							map[string]any{"required": []string{"memoryId"}},
							map[string]any{"required": []string{"id"}},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{
							"action": map[string]any{"const": "forget"},
						},
					},
					"then": map[string]any{
						"anyOf": []any{
							map[string]any{"required": []string{"memoryId"}},
							map[string]any{"required": []string{"id"}},
							map[string]any{"required": []string{"query"}},
						},
					},
				},
			},
		}),
		Enabled: true,
	}
}

func specLibrary() toolSpec {
	return toolSpec{
		ID:          "library",
		Name:        "library",
		Description: "Read-only library inspector. Use action=overview|files|operations|records|operation_status to list libraries, inspect full library details, browse operations/history, or fetch a single operation by id.",
		Category:    "library",
		RiskLevel:   "medium",
		Methods:     libraryMethodSpecs(),
		SchemaJSON:  libraryToolSchema(),
		Enabled:     true,
	}
}

func specLibraryManage() toolSpec {
	return toolSpec{
		ID:          "library_manage",
		Name:        "library_manage",
		Description: "Create and manage asynchronous library jobs for download, transcode, and subtitle processing. Job-creating actions enqueue work and return operationId immediately. Do not wait, loop, or blind-retry in the same call; query progress later with library action=operation_status or library action=operations.",
		Category:    "library",
		RiskLevel:   "high",
		Methods:     libraryManageMethodSpecs(),
		SchemaJSON:  libraryManageToolSchema(),
		Enabled:     true,
	}
}
