export type GatewayToolMethodSpec = {
  name: string;
  inputSchema?: unknown;
  outputSchema?: unknown;
  inputExample?: unknown;
  outputExample?: unknown;
};

export type GatewayToolRequirement = {
  id: string;
  name?: string;
  available: boolean;
  reason?: string;
};

export type GatewayToolSpec = {
  id: string;
  name: string;
  description?: string;
  kind?: string;
  schemaJson?: string;
  methods?: GatewayToolMethodSpec[];
  requirements?: GatewayToolRequirement[];
  sideEffectLevel?: string;
  category?: string;
  riskLevel?: string;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  enabled?: boolean;
};
