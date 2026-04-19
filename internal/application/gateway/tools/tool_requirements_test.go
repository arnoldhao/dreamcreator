package tools

import (
	"context"
	"strings"
	"testing"

	gatewayvoice "dreamcreator/internal/application/gateway/voice"
	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
	"dreamcreator/internal/domain/providers"
)

type voiceStatusStub struct {
	status gatewayvoice.TTSStatusResponse
	err    error
}

func (stub voiceStatusStub) Status(context.Context) (gatewayvoice.TTSStatusResponse, error) {
	if stub.err != nil {
		return gatewayvoice.TTSStatusResponse{}, stub.err
	}
	return stub.status, nil
}

func TestResolveEffectiveToolSpecWebSearchAPIMissingKeyDisablesTool(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"web": map[string]any{
					"search": map[string]any{
						"type":     "api",
						"provider": "brave",
					},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specWebSearch().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected web_search disabled when API key is missing")
	}
	requirement, ok := findRequirement(spec.Requirements, "web_search.provider_api_key")
	if !ok {
		t.Fatalf("expected provider api key requirement")
	}
	if requirement.Available {
		t.Fatalf("expected provider api key requirement to be unavailable")
	}
}

func TestResolveEffectiveToolSpecWebSearchAPIWithKeyKeepsToolEnabled(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"web": map[string]any{
					"search": map[string]any{
						"type":     "api",
						"provider": "brave",
						"providers": map[string]any{
							"brave": map[string]any{
								"apiKey": "token",
							},
						},
					},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specWebSearch().toDTO(), snapshot)
	if !spec.Enabled {
		t.Fatalf("expected web_search enabled when provider API key is configured")
	}
}

func TestResolveEffectiveToolSpecWebSearchExternalToolsDisablesTool(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"web": map[string]any{
					"search": map[string]any{
						"type": "external_tools",
					},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specWebSearch().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected web_search disabled in external_tools mode stub")
	}
	requirement, ok := findRequirement(spec.Requirements, "web_search.external_tools_supported")
	if !ok {
		t.Fatalf("expected external tools requirement")
	}
	if requirement.Available {
		t.Fatalf("expected external tools requirement to be unavailable")
	}
}

func TestResolveEffectiveToolSpecWebFetchDisabledByConfig(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"web_fetch": map[string]any{
					"enabled": false,
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specWebFetch().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected web_fetch disabled when config enabled flag is false")
	}
	requirement, ok := findRequirement(spec.Requirements, "web_fetch.local_browser")
	if !ok {
		t.Fatalf("expected web_fetch local browser requirement")
	}
	if _, ok := findRequirement(spec.Requirements, "web_fetch.config_enabled"); ok {
		t.Fatalf("did not expect web_fetch config enabled requirement")
	}
	_ = requirement
}

func TestResolveEffectiveToolSpecBrowserIncludesCDPRuntimeRequirement(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"browser": map[string]any{
					"preferredBrowser": "brave",
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specBrowser().toDTO(), snapshot)
	requirement, ok := findRequirement(spec.Requirements, "browser.cdp_runtime")
	if !ok {
		t.Fatalf("expected browser cdp runtime requirement")
	}
	data, ok := requirement.Data.(map[string]any)
	if !ok || data == nil {
		t.Fatalf("expected browser runtime requirement data")
	}
	if data["selectedBrowser"] != "brave" {
		t.Fatalf("expected selected browser brave, got %#v", data["selectedBrowser"])
	}
}

func TestResolveEffectiveToolSpecBrowserDisabledByConfig(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"browser": map[string]any{
					"enabled": false,
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specBrowser().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected browser disabled when browser.enabled is false")
	}
	requirement, ok := findRequirement(spec.Requirements, "browser.cdp_runtime")
	if !ok {
		t.Fatalf("expected browser cdp runtime requirement")
	}
	if _, ok := findRequirement(spec.Requirements, "browser.config_enabled"); ok {
		t.Fatalf("did not expect browser config enabled requirement")
	}
	_ = requirement
}

func TestResolveEffectiveToolSpecBrowserLegacyTypeIsIgnored(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"browser": map[string]any{
					"type": "terminal",
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specBrowser().toDTO(), snapshot)
	requirement, ok := findRequirement(spec.Requirements, "browser.cdp_runtime")
	if !ok {
		t.Fatalf("expected browser cdp runtime requirement")
	}
	if _, ok := findRequirement(spec.Requirements, "browser.type_supported"); ok {
		t.Fatalf("did not expect browser type requirement")
	}
	_ = requirement
}

func TestResolveEffectiveToolSpecGatewayRequiresControlPlane(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: false},
		},
	})
	spec := resolveEffectiveToolSpec(specGateway().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected gateway tool disabled when control plane is off")
	}
	requirement, ok := findRequirement(spec.Requirements, "gateway.control_plane_enabled")
	if !ok {
		t.Fatalf("expected gateway control plane requirement")
	}
	if requirement.Available {
		t.Fatalf("expected gateway control plane requirement to be unavailable")
	}
}

func TestResolveEffectiveToolSpecNodesRemainsDisabledUntilRemoteRuntimeExists(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	spec := resolveEffectiveToolSpec(specNodes().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected nodes tool disabled while remote node runtime is unavailable")
	}
	requirement, ok := findRequirement(spec.Requirements, "nodes.remote_runtime")
	if !ok {
		t.Fatalf("expected remote node runtime requirement")
	}
	if requirement.Available {
		t.Fatalf("expected remote node runtime requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "not implemented") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestResolveEffectiveToolSpecCanvasRemainsDisabledUntilRemoteRuntimeExists(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	spec := resolveEffectiveToolSpec(specCanvas().toDTO(), snapshot)
	if spec.Enabled {
		t.Fatalf("expected canvas tool disabled while remote node runtime is unavailable")
	}
	requirement, ok := findRequirement(spec.Requirements, "canvas.remote_runtime")
	if !ok {
		t.Fatalf("expected canvas remote runtime requirement")
	}
	if requirement.Available {
		t.Fatalf("expected canvas remote runtime requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "not implemented") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestResolveEffectiveToolSpecImageRequiresConfiguredModel(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Providers: imageProviderRepoStub{items: map[string]providers.Provider{}},
		Models:    imageModelRepoStub{items: map[string][]providers.Model{}},
		Secrets:   imageSecretRepoStub{items: map[string]providers.ProviderSecret{}},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specImage().toDTO(), snapshot, resolver)
	if spec.Enabled {
		t.Fatalf("expected image tool disabled without a configured model")
	}
	requirement, ok := findRequirement(spec.Requirements, imageRequirementID)
	if !ok {
		t.Fatalf("expected image model requirement")
	}
	if requirement.Available {
		t.Fatalf("expected image model requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "image model") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestResolveEffectiveToolSpecImageStaysEnabledWithConfiguredModel(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Providers: imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Enabled:  true,
				Type:     providers.ProviderTypeOpenAI,
				Endpoint: "https://api.openai.com/v1",
			},
		}},
		Models: imageModelRepoStub{items: map[string][]providers.Model{
			"openai": {
				{
					ID:             "gpt-4o",
					Name:           "gpt-4o",
					Enabled:        true,
					SupportsVision: ptrBool(true),
				},
			},
		}},
		Secrets: imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {ProviderID: "openai", APIKey: "test-key"},
		}},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specImage().toDTO(), snapshot, resolver)
	if !spec.Enabled {
		t.Fatalf("expected image tool to stay enabled with a configured model")
	}
	requirement, ok := findRequirement(spec.Requirements, imageRequirementID)
	if !ok {
		t.Fatalf("expected image model requirement")
	}
	if !requirement.Available {
		t.Fatalf("expected image model requirement to be available")
	}
}

func TestResolveEffectiveToolSpecTTSRequiresRunnableProvider(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Voice: voiceStatusStub{
			status: gatewayvoice.TTSStatusResponse{
				Enabled: true,
				Config: gatewayvoice.TTSConfig{
					ProviderID: "edge",
				},
				Providers: []gatewayvoice.TTSProviderCatalogItem{
					{ProviderID: "edge", DisplayName: "Edge-TTS", Available: true},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specTTS().toDTO(), snapshot, resolver)
	if spec.Enabled {
		t.Fatalf("expected tts tool disabled when only edge placeholder provider is selected")
	}
	requirement, ok := findRequirement(spec.Requirements, ttsProviderRequirementID)
	if !ok {
		t.Fatalf("expected tts provider requirement")
	}
	if requirement.Available {
		t.Fatalf("expected tts provider requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "edge-tts") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestResolveEffectiveToolSpecTTSStaysEnabledWithConfiguredProvider(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Voice: voiceStatusStub{
			status: gatewayvoice.TTSStatusResponse{
				Enabled: true,
				Config: gatewayvoice.TTSConfig{
					ProviderID: "openai",
				},
				Providers: []gatewayvoice.TTSProviderCatalogItem{
					{ProviderID: "openai", DisplayName: "OpenAI", Available: true},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specTTS().toDTO(), snapshot, resolver)
	if !spec.Enabled {
		t.Fatalf("expected tts tool to stay enabled with a configured provider")
	}
	requirement, ok := findRequirement(spec.Requirements, ttsProviderAPIKeyRequirementID)
	if !ok {
		t.Fatalf("expected tts provider api key requirement")
	}
	if !requirement.Available {
		t.Fatalf("expected tts provider api key requirement to be available")
	}
}

func TestResolveEffectiveToolSpecTTSVoiceFeatureDisabled(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Voice: voiceStatusStub{
			status: gatewayvoice.TTSStatusResponse{
				Enabled: false,
				Config: gatewayvoice.TTSConfig{
					ProviderID: "openai",
				},
				Providers: []gatewayvoice.TTSProviderCatalogItem{
					{ProviderID: "openai", DisplayName: "OpenAI", Available: true},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specTTS().toDTO(), snapshot, resolver)
	if spec.Enabled {
		t.Fatalf("expected tts tool disabled when voice feature is disabled")
	}
	requirement, ok := findRequirement(spec.Requirements, ttsVoiceEnabledRequirementID)
	if !ok {
		t.Fatalf("expected tts voice feature requirement")
	}
	if requirement.Available {
		t.Fatalf("expected tts voice feature requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "voice is disabled") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestResolveEffectiveToolSpecTTSElevenLabsRequiresVoiceID(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	resolver := NewBuiltinRequirementResolver(BuiltinRequirementDeps{
		Voice: voiceStatusStub{
			status: gatewayvoice.TTSStatusResponse{
				Enabled: true,
				Config: gatewayvoice.TTSConfig{
					ProviderID: "elevenlabs",
				},
				Providers: []gatewayvoice.TTSProviderCatalogItem{
					{ProviderID: "elevenlabs", DisplayName: "ElevenLabs", Available: true},
				},
			},
		},
	})
	spec := resolveEffectiveToolSpecWithResolver(context.Background(), specTTS().toDTO(), snapshot, resolver)
	if spec.Enabled {
		t.Fatalf("expected tts tool disabled when elevenlabs voice id is missing")
	}
	requirement, ok := findRequirement(spec.Requirements, ttsVoiceIDRequirementID)
	if !ok {
		t.Fatalf("expected tts voice id requirement")
	}
	if requirement.Available {
		t.Fatalf("expected tts voice id requirement to be unavailable")
	}
	if !strings.Contains(strings.ToLower(requirement.Reason), "voice id") {
		t.Fatalf("unexpected requirement reason: %q", requirement.Reason)
	}
}

func TestPolicyPipelineDeniesUnavailableWebSearch(t *testing.T) {
	t.Parallel()

	pipeline := NewPolicyPipeline(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"web": map[string]any{
					"search": map[string]any{
						"type":     "api",
						"provider": "brave",
					},
				},
			},
		},
	})
	decision, err := pipeline.Decide(context.Background(), tooldto.ToolSpec{
		ID:      "web_search",
		Name:    "web_search",
		Enabled: true,
	}, tooldto.ToolPolicyContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Decision != "deny" {
		t.Fatalf("expected deny decision, got %q", decision.Decision)
	}
	if !strings.Contains(strings.ToLower(decision.Reason), "api key") {
		t.Fatalf("expected api key reason, got %q", decision.Reason)
	}
}

func TestPolicyPipelineHighRiskToolRequiresApprovalByDefault(t *testing.T) {
	t.Parallel()

	pipeline := NewPolicyPipeline(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	})
	decision, err := pipeline.Decide(context.Background(), tooldto.ToolSpec{
		ID:        "exec",
		Name:      "exec",
		Enabled:   true,
		RiskLevel: "high",
	}, tooldto.ToolPolicyContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Decision != "ask" {
		t.Fatalf("expected ask decision, got %q", decision.Decision)
	}
	if !decision.ApprovalRequired {
		t.Fatalf("expected approval required for high-risk tool")
	}
	if !decision.SandboxRequired {
		t.Fatalf("expected sandbox required for high-risk tool")
	}
}

func TestPolicyPipelineAllAccessDisablesApprovalAndSandbox(t *testing.T) {
	t.Parallel()

	pipeline := NewPolicyPipeline(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"execPermissionMode": "full access",
			},
		},
	})
	decision, err := pipeline.Decide(context.Background(), tooldto.ToolSpec{
		ID:        "exec",
		Name:      "exec",
		Enabled:   true,
		RiskLevel: "high",
	}, tooldto.ToolPolicyContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Decision != "allow" {
		t.Fatalf("expected allow decision, got %q", decision.Decision)
	}
	if decision.ApprovalRequired {
		t.Fatalf("expected approval disabled in full access mode")
	}
	if decision.SandboxRequired {
		t.Fatalf("expected sandbox disabled in full access mode")
	}
}

func TestGatewayServiceListToolsAppliesRequirements(t *testing.T) {
	t.Parallel()

	toolSvc := toolservice.NewToolService()
	_, err := toolSvc.RegisterTool(context.Background(), tooldto.RegisterToolRequest{Spec: specWebSearch().toDTO()})
	if err != nil {
		t.Fatalf("register web_search: %v", err)
	}
	service := NewService(
		toolSvc,
		nil,
		nil,
		gatewayToolSettingsStub{
			settings: settingsdto.Settings{
				Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
				Tools: map[string]any{
					"web": map[string]any{
						"search": map[string]any{
							"type":     "api",
							"provider": "brave",
						},
					},
				},
			},
		},
		nil,
		nil,
	)
	specs := service.ListTools(context.Background())
	if len(specs) == 0 {
		t.Fatalf("expected web_search spec")
	}
	if specs[0].Enabled {
		t.Fatalf("expected web_search disabled when API key requirement is not met")
	}
}

func findRequirement(requirements []tooldto.ToolRequirement, id string) (tooldto.ToolRequirement, bool) {
	for _, requirement := range requirements {
		if strings.EqualFold(requirement.ID, id) {
			return requirement, true
		}
	}
	return tooldto.ToolRequirement{}, false
}
