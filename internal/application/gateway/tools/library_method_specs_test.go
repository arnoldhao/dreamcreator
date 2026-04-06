package tools

import (
	"encoding/json"
	"testing"
)

func TestLibraryMethodSpecsMatchDefinitions(t *testing.T) {
	specs := libraryMethodSpecs()
	if len(specs) != len(libraryMethodDefinitions) {
		t.Fatalf("expected %d method specs, got %d", len(libraryMethodDefinitions), len(specs))
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
		if outputExample["status"] != "ok" {
			t.Fatalf("expected ok status for %s, got %#v", spec.Name, outputExample["status"])
		}
		if _, ok := outputExample["outputJson"].(string); !ok {
			t.Fatalf("expected outputJson string for %s", spec.Name)
		}
	}

	for _, definition := range libraryMethodDefinitions {
		if _, ok := byName[definition.name]; !ok {
			t.Fatalf("missing method spec for action %q", definition.name)
		}
	}
}

func TestLibraryToolSchemaCoversAllMethodActions(t *testing.T) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(libraryToolSchema()), &parsed); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Fatalf("expected top-level object schema, got %#v", parsed["type"])
	}
	properties, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object")
	}
	actionProp, ok := properties["action"].(map[string]any)
	if !ok {
		t.Fatalf("expected action property schema")
	}
	actionEnum, ok := actionProp["enum"].([]any)
	if !ok {
		t.Fatalf("expected action enum")
	}

	found := make(map[string]struct{}, len(actionEnum))
	for _, candidate := range actionEnum {
		name, _ := candidate.(string)
		if name != "" {
			found[name] = struct{}{}
		}
	}
	for _, definition := range libraryMethodDefinitions {
		if _, ok := found[definition.name]; !ok {
			t.Fatalf("missing schema action for %q", definition.name)
		}
	}
}
