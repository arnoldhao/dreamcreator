package commands

import (
	settingsdto "dreamcreator/internal/application/settings/dto"
	domaincommands "dreamcreator/internal/domain/commands"
)

type Capabilities struct {
	ControlPlane      bool
	Runtime           bool
	Models            bool
	Agents            bool
	Subagents         bool
	Nodes             bool
	Channels          bool
	Usage             bool
	Voice             bool
	VoiceWake         bool
	ToolPolicy        bool
	ExecApproval      bool
	Automation        bool
	Sandbox           bool
	SessionCore       bool
	WorkspaceSnapshot bool
}

func ResolveCapabilities(settings settingsdto.Settings) Capabilities {
	controlPlane := settings.Gateway.ControlPlaneEnabled
	sessionCore := controlPlane
	voiceEnabled := controlPlane && settings.Gateway.VoiceEnabled
	return Capabilities{
		ControlPlane:      controlPlane,
		Runtime:           controlPlane,
		Models:            controlPlane,
		Agents:            controlPlane,
		Subagents:         sessionCore,
		Nodes:             controlPlane,
		Channels:          controlPlane,
		Usage:             controlPlane,
		Voice:             voiceEnabled,
		VoiceWake:         voiceEnabled && settings.Gateway.VoiceWakeEnabled,
		ToolPolicy:        controlPlane,
		ExecApproval:      controlPlane,
		Automation:        controlPlane,
		Sandbox:           controlPlane && settings.Gateway.SandboxEnabled,
		SessionCore:       sessionCore,
		WorkspaceSnapshot: controlPlane,
	}
}

func (caps Capabilities) Has(capability domaincommands.CapabilityKey) bool {
	switch capability {
	case domaincommands.CapabilityControlPlane:
		return caps.ControlPlane
	case domaincommands.CapabilityRuntime:
		return caps.Runtime
	case domaincommands.CapabilityModels:
		return caps.Models
	case domaincommands.CapabilityAgents:
		return caps.Agents
	case domaincommands.CapabilitySubagents:
		return caps.Subagents
	case domaincommands.CapabilityNodes:
		return caps.Nodes
	case domaincommands.CapabilityChannels:
		return caps.Channels
	case domaincommands.CapabilityUsage:
		return caps.Usage
	case domaincommands.CapabilityVoice:
		return caps.Voice
	case domaincommands.CapabilityVoiceWake:
		return caps.VoiceWake
	case domaincommands.CapabilityToolPolicy:
		return caps.ToolPolicy
	case domaincommands.CapabilityExecApproval:
		return caps.ExecApproval
	case domaincommands.CapabilityAutomation:
		return caps.Automation
	case domaincommands.CapabilitySandbox:
		return caps.Sandbox
	case domaincommands.CapabilitySessionCore:
		return caps.SessionCore
	case domaincommands.CapabilityWorkspaceFiles:
		return caps.WorkspaceSnapshot
	default:
		return false
	}
}
