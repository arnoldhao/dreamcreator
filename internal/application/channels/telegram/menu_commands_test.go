package telegram

import (
	"testing"

	appcommands "dreamcreator/internal/application/commands"
)

func TestFilterTelegramMenuNativeCommandSpecs_ExcludesP2(t *testing.T) {
	t.Parallel()
	input := []appcommands.NativeCommandSpec{
		{Key: "help", Name: "help"},
		{Key: "model", Name: "model"},
		{Key: "restart", Name: "restart"},
		{Key: "config", Name: "config"},
		{Key: "debug", Name: "debug"},
		{Key: "exec", Name: "exec"},
		{Key: "elevated", Name: "elevated"},
	}

	filtered := FilterTelegramMenuNativeCommandSpecs(input)
	if len(filtered) != 2 {
		t.Fatalf("unexpected filtered length: got %d want %d", len(filtered), 2)
	}
	if filtered[0].Key != "help" {
		t.Fatalf("unexpected first command: %s", filtered[0].Key)
	}
	if filtered[1].Key != "model" {
		t.Fatalf("unexpected second command: %s", filtered[1].Key)
	}
}
