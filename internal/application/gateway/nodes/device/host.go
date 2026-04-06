package device

import (
	"context"
	"errors"
	"runtime"
	"strings"
)

type DeviceRuntimeHello struct {
	DeviceID     string                 `json:"deviceId,omitempty"`
	DisplayName  string                 `json:"displayName,omitempty"`
	Platform     string                 `json:"platform,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Capabilities []CapabilityDescriptor `json:"capabilities,omitempty"`
}

type CapabilityDescriptor struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

type CapabilityInvokeEnvelope struct {
	Capability string `json:"capability"`
	Action     string `json:"action,omitempty"`
	Args       string `json:"args,omitempty"`
}

type CapabilityResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Host struct{}

func NewHost() *Host {
	return &Host{}
}

func (host *Host) Hello(_ context.Context) DeviceRuntimeHello {
	capabilities := []CapabilityDescriptor{
		{Name: "screen"},
		{Name: "system.run"},
	}
	return DeviceRuntimeHello{
		Platform:     runtime.GOOS,
		Version:      runtime.Version(),
		Capabilities: capabilities,
	}
}

func (host *Host) Invoke(_ context.Context, envelope CapabilityInvokeEnvelope) (CapabilityResult, error) {
	capability := strings.TrimSpace(envelope.Capability)
	if capability == "" {
		return CapabilityResult{OK: false, Error: "capability is required"}, errors.New("capability is required")
	}
	switch capability {
	case "screen", "system.run":
		return CapabilityResult{OK: true, Output: "{}"}, nil
	default:
		return CapabilityResult{OK: false, Error: "capability not supported"}, errors.New("capability not supported")
	}
}
