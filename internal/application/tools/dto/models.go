package dto

type ToolSpec struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Kind             string            `json:"kind"`
	SchemaJSON       string            `json:"schemaJson"`
	Methods          []ToolMethodSpec  `json:"methods,omitempty"`
	Requirements     []ToolRequirement `json:"requirements,omitempty"`
	SideEffectLevel  string            `json:"sideEffectLevel"`
	Category         string            `json:"category,omitempty"`
	RiskLevel        string            `json:"riskLevel,omitempty"`
	RequiresSandbox  bool              `json:"requiresSandbox,omitempty"`
	RequiresApproval bool              `json:"requiresApproval,omitempty"`
	Enabled          bool              `json:"enabled"`
}

type ToolRequirement struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type ToolMethodSpec struct {
	Name          string `json:"name"`
	InputSchema   any    `json:"inputSchema,omitempty"`
	OutputSchema  any    `json:"outputSchema,omitempty"`
	InputExample  any    `json:"inputExample,omitempty"`
	OutputExample any    `json:"outputExample,omitempty"`
}

type ToolInvocation struct {
	ID        string `json:"id"`
	ToolID    string `json:"toolId"`
	ToolName  string `json:"toolName,omitempty"`
	InputJSON string `json:"inputJson"`
}

type ToolResult struct {
	ID           string             `json:"id"`
	ToolID       string             `json:"toolId"`
	OutputJSON   string             `json:"outputJson"`
	ErrorMessage string             `json:"errorMessage"`
	Policy       ToolPolicyDecision `json:"policy,omitempty"`
}

type ToolPolicyContext struct {
	SessionKey      string `json:"sessionKey,omitempty"`
	AgentID         string `json:"agentId,omitempty"`
	ProviderID      string `json:"providerId,omitempty"`
	Source          string `json:"source,omitempty"`
	IsSubagent      bool   `json:"isSubagent,omitempty"`
	RequireSandbox  bool   `json:"requireSandbox,omitempty"`
	RequireApproval bool   `json:"requireApproval,omitempty"`
}

type ToolPolicyDecision struct {
	Decision         string   `json:"decision"`
	Reason           string   `json:"reason,omitempty"`
	MatchedRules     []string `json:"matchedRules,omitempty"`
	SandboxRequired  bool     `json:"sandboxRequired,omitempty"`
	ApprovalRequired bool     `json:"approvalRequired,omitempty"`
}

type ToolPolicySnapshot struct {
	Spec     ToolSpec           `json:"spec"`
	Decision ToolPolicyDecision `json:"decision"`
	Context  ToolPolicyContext  `json:"context,omitempty"`
}

type ToolsInvokeRequest struct {
	Tool       string `json:"tool"`
	ToolCallID string `json:"toolCallId,omitempty"`
	Action     string `json:"action,omitempty"`
	Args       string `json:"args,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
	DryRun     bool   `json:"dryRun,omitempty"`
}

type ToolsInvokeResponse struct {
	Result       ToolResult         `json:"result"`
	Policy       ToolPolicyDecision `json:"policy"`
	PolicyDetail ToolPolicySnapshot `json:"policyDetail,omitempty"`
}

type RegisterToolRequest struct {
	Spec ToolSpec `json:"spec"`
}

type EnableToolRequest struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type ExecuteToolRequest struct {
	Invocation ToolInvocation `json:"invocation"`
}

type QueryToolLogsRequest struct {
	ToolID string `json:"toolId"`
	Limit  int    `json:"limit"`
}
