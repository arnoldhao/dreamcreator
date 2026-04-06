package usage

import (
	"math"
	"testing"
)

func TestParsePricingFromCapabilitiesCostObject(t *testing.T) {
	raw := `{"cost":{"input":1,"output":3.2,"cache_read":0.2,"reasoning":0.5,"input_audio":12,"output_audio":24}}`
	parsed, ok := ParsePricingFromCapabilities(raw)
	if !ok {
		t.Fatal("expected pricing to be parsed")
	}
	assertApproxEqual(t, parsed.InputPerMillion, 1)
	assertApproxEqual(t, parsed.OutputPerMillion, 3.2)
	assertApproxEqual(t, parsed.CachedInputPerMillion, 0.2)
	assertApproxEqual(t, parsed.ReasoningPerMillion, 0.5)
	assertApproxEqual(t, parsed.AudioInputPerMillion, 12)
	assertApproxEqual(t, parsed.AudioOutputPerMillion, 24)
	assertApproxEqual(t, parsed.PerRequest, 0)
}

func TestParsePricingFromCapabilitiesPricingObject(t *testing.T) {
	raw := `{"pricing":{"input":"0.000002","output":"0.000006","cache_read_per_1k":"0.0015","request":"0.001"}}`
	parsed, ok := ParsePricingFromCapabilities(raw)
	if !ok {
		t.Fatal("expected pricing to be parsed")
	}
	assertApproxEqual(t, parsed.InputPerMillion, 2)
	assertApproxEqual(t, parsed.OutputPerMillion, 6)
	assertApproxEqual(t, parsed.CachedInputPerMillion, 1.5)
	assertApproxEqual(t, parsed.PerRequest, 0.001)
}

func TestParsePricingFromCapabilitiesInvalid(t *testing.T) {
	if _, ok := ParsePricingFromCapabilities("not-json"); ok {
		t.Fatal("expected invalid payload to return ok=false")
	}
	if _, ok := ParsePricingFromCapabilities(`{"name":"glm-5"}`); ok {
		t.Fatal("expected payload without pricing to return ok=false")
	}
}

func assertApproxEqual(t *testing.T, actual float64, expected float64) {
	t.Helper()
	if math.Abs(actual-expected) > 1e-9 {
		t.Fatalf("expected %f, got %f", expected, actual)
	}
}
