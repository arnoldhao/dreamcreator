package runtime

import (
	"testing"

	"dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/domain/providers"
)

func TestParseModelTokenPricingOpenRouter(t *testing.T) {
	raw := `{"pricing":{"prompt":"0.00003","completion":"0.00006","request":"0.001"}}`
	pricing, ok := parseModelTokenPricing(raw)
	if !ok {
		t.Fatal("expected pricing to be parsed")
	}
	if pricing.PromptUSDPerToken != 0.00003 {
		t.Fatalf("expected prompt price 0.00003, got %f", pricing.PromptUSDPerToken)
	}
	if pricing.CompletionUSDPerToken != 0.00006 {
		t.Fatalf("expected completion price 0.00006, got %f", pricing.CompletionUSDPerToken)
	}
	if pricing.RequestUSD != 0.001 {
		t.Fatalf("expected request price 0.001, got %f", pricing.RequestUSD)
	}

	cost := calculateUsageCostMicros(dto.RuntimeUsage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}, pricing)
	if cost != 61000 {
		t.Fatalf("expected cost micros 61000, got %d", cost)
	}
}

func TestParseModelTokenPricingPerMillion(t *testing.T) {
	raw := `{"input_cost_per_million":2,"output_cost_per_million":8}`
	pricing, ok := parseModelTokenPricing(raw)
	if !ok {
		t.Fatal("expected pricing to be parsed")
	}

	cost := calculateUsageCostMicros(dto.RuntimeUsage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}, pricing)
	if cost != 6000 {
		t.Fatalf("expected cost micros 6000, got %d", cost)
	}
}

func TestParseModelTokenPricingCostObjectPerMillion(t *testing.T) {
	raw := `{"cost":{"input":1.25,"output":10}}`
	pricing, ok := parseModelTokenPricing(raw)
	if !ok {
		t.Fatal("expected pricing to be parsed from cost object")
	}

	cost := calculateUsageCostMicros(dto.RuntimeUsage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}, pricing)
	if cost != 6250 {
		t.Fatalf("expected cost micros 6250, got %d", cost)
	}
}

func TestFindModelByNameFallbackToSuffix(t *testing.T) {
	models := []providers.Model{
		{Name: "openai/gpt-4.1", CapabilitiesJSON: "{}"},
		{Name: "anthropic/claude-3-7-sonnet", CapabilitiesJSON: "{}"},
	}

	model, ok := findModelByName(models, "gpt-4.1")
	if !ok {
		t.Fatal("expected model to be matched by suffix")
	}
	if model.Name != "openai/gpt-4.1" {
		t.Fatalf("unexpected model matched: %s", model.Name)
	}
}
