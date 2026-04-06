package tools

import "testing"

func TestGatewayMethodSpecsMatchActions(t *testing.T) {
	specs := gatewayMethodSpecs()
	if len(specs) != len(gatewayToolActions) {
		t.Fatalf("expected %d method specs, got %d", len(gatewayToolActions), len(specs))
	}

	byName := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		byName[spec.Name] = struct{}{}

		inputExample, ok := spec.InputExample.(map[string]any)
		if !ok {
			t.Fatalf("expected input example object for %s", spec.Name)
		}
		if inputExample["action"] != spec.Name {
			t.Fatalf("expected input action %q for %s, got %#v", spec.Name, spec.Name, inputExample["action"])
		}

		outputExample, ok := spec.OutputExample.(map[string]any)
		if !ok {
			t.Fatalf("expected output example object for %s", spec.Name)
		}
		if outputExample["action"] != spec.Name {
			t.Fatalf("expected output action %q for %s, got %#v", spec.Name, spec.Name, outputExample["action"])
		}
	}

	for _, action := range gatewayToolActions {
		if _, ok := byName[action]; !ok {
			t.Fatalf("missing method spec for action %q", action)
		}
	}
}
