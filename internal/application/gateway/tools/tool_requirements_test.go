package tools

import (
	"context"
	"strings"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
)

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
	requirement, ok := findRequirement(spec.Requirements, "web_fetch.config_enabled")
	if !ok {
		t.Fatalf("expected web_fetch config requirement")
	}
	if requirement.Available {
		t.Fatalf("expected web_fetch config requirement to be unavailable")
	}
}

func TestResolveEffectiveToolSpecBrowserPlaywrightIncludesRuntimeRequirement(t *testing.T) {
	t.Parallel()

	snapshot := loadToolRequirementSnapshot(context.Background(), gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"browser": map[string]any{
					"type": "playwright",
				},
			},
		},
	})
	spec := resolveEffectiveToolSpec(specBrowser().toDTO(), snapshot)
	requirement, ok := findRequirement(spec.Requirements, "browser.playwright_runtime")
	if !ok {
		t.Fatalf("expected browser playwright runtime requirement")
	}
	_ = requirement
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
	requirement, ok := findRequirement(spec.Requirements, "browser.playwright_runtime")
	if !ok {
		t.Fatalf("expected browser playwright runtime requirement")
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
	requirement, ok := findRequirement(spec.Requirements, "browser.playwright_runtime")
	if !ok {
		t.Fatalf("expected browser playwright runtime requirement")
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
