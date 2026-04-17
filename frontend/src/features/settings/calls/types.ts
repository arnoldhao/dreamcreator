import type { GatewayToolMethodSpec } from "@/shared/store/gatewayTools";
import type { ProviderSkillSpec, SkillSearchResult } from "@/shared/contracts/skills";

export type CallsTabId = "tools" | "skills";

export type ToolSource = "gateway" | "frontend";

export type ToolRequirementStatus = {
  id: string;
  name: string;
  available: boolean;
  reason: string;
  data?: unknown;
};

export type ToolItem = {
  id: string;
  source: ToolSource;
  available: boolean;
  requirements: ToolRequirementStatus[];
  category?: string;
  riskLevel?: string;
  schemaJson?: string;
  methods?: GatewayToolMethodSpec[];
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  labelKey: string;
  label: string;
  descriptionKey: string;
  description: string;
};

export type SkillListItem =
  | { kind: "local"; key: string; id: string; skill: ProviderSkillSpec }
  | { kind: "search"; key: string; id: string; result: SkillSearchResult };

export type LocalSkillListItem = Extract<SkillListItem, { kind: "local" }>;

export type RemoteSkillListItem = Extract<SkillListItem, { kind: "search" }>;

export type SkillDetailContentTab = "skill_md" | "files";

export type SkillsActionErrorInfo = {
  message: string;
  rateLimited: boolean;
  requiresForce: boolean;
};
