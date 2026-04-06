package commands

type Scope string

const (
	ScopeNative Scope = "native"
	ScopeText   Scope = "text"
	ScopeBoth   Scope = "both"
)

type CapabilityKey string

const (
	CapabilityControlPlane   CapabilityKey = "controlPlane"
	CapabilityRuntime        CapabilityKey = "runtime"
	CapabilityModels         CapabilityKey = "models"
	CapabilityAgents         CapabilityKey = "agents"
	CapabilitySubagents      CapabilityKey = "subagents"
	CapabilityNodes          CapabilityKey = "nodes"
	CapabilityChannels       CapabilityKey = "channels"
	CapabilityUsage          CapabilityKey = "usage"
	CapabilityVoice          CapabilityKey = "voice"
	CapabilityVoiceWake      CapabilityKey = "voiceWake"
	CapabilityToolPolicy     CapabilityKey = "toolPolicy"
	CapabilityExecApproval   CapabilityKey = "execApproval"
	CapabilityAutomation     CapabilityKey = "automation"
	CapabilitySandbox        CapabilityKey = "sandbox"
	CapabilitySessionCore    CapabilityKey = "sessionCore"
	CapabilityWorkspaceFiles CapabilityKey = "workspaceSnapshot"
)

type CommandArg struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Required    bool              `json:"required"`
	Choices     []CommandArgChoice `json:"choices,omitempty"`
	Help        string            `json:"help,omitempty"`
	CaptureRest bool              `json:"captureRest,omitempty"`
}

type CommandArgChoice struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

type CommandDefinition struct {
	Key              string            `json:"key"`
	NativeName       string            `json:"nativeName,omitempty"`
	Description      string            `json:"description"`
	Scope            Scope             `json:"scope"`
	AcceptsArgs      bool              `json:"acceptsArgs"`
	Args             []CommandArg      `json:"args,omitempty"`
	ProviderOverride map[string]string `json:"providerOverrides,omitempty"`
	Requires         []CapabilityKey   `json:"requires,omitempty"`
}
