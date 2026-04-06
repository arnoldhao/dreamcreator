package app

import "testing"

func TestCurrentStartupContext(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{name: "no marker", args: []string{"--verbose"}, expected: false},
		{name: "exact marker", args: []string{"--autostart"}, expected: true},
		{name: "marker with spaces and mixed case", args: []string{"  --AutoStart  "}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := currentStartupContext(tt.args)
			if got.launchedByAutoStart != tt.expected {
				t.Fatalf("launchedByAutoStart = %v, want %v", got.launchedByAutoStart, tt.expected)
			}
		})
	}
}
