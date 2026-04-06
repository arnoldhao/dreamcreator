export type ChannelState =
  | "online"
  | "offline"
  | "degraded"
  | "reconnecting"
  | "unknown"
  | "disabled"
  | string;

export interface ChannelOverview {
  channelId: string;
  displayName: string;
  kind: string;
  capabilities?: string[];
  enabled: boolean;
  state: ChannelState;
  accountId?: string;
  latencyMs?: number;
  lastError?: string;
  updatedAt?: string;
}

export interface ChannelProbeRequest {
  channelId: string;
}

export interface ChannelProbeResult {
  channelId: string;
  state?: string;
  success: boolean;
  error?: string;
  checkedAt?: string;
}

export interface ChannelLogoutRequest {
  channelId: string;
}

export interface ChannelLogoutResult {
  channelId: string;
  success: boolean;
  error?: string;
}

export interface ChannelMenuSyncRequest {
  channelId: string;
}

export interface ChannelMenuSyncResult {
  channelId: string;
  ready: boolean;
  synced: boolean;
  commands: number;
  issues?: string[];
  overflowCount?: number;
  error?: string;
  syncedAt?: string;
}

export interface ChannelPairingRequest {
  id: string;
  code: string;
  createdAt: string;
  lastSeenAt?: string;
  meta?: Record<string, string>;
}

export interface ChannelPairingListRequest {
  channelId: string;
  accountId?: string;
}

export interface ChannelPairingListResult {
  channelId: string;
  requests: ChannelPairingRequest[];
}

export interface ChannelPairingApproveRequest {
  channelId: string;
  code: string;
  accountId?: string;
  notify?: boolean;
}

export interface ChannelPairingApproveResult {
  channelId: string;
  approved: boolean;
  requestId?: string;
  error?: string;
}

export interface ChannelPairingRejectRequest {
  channelId: string;
  code: string;
  accountId?: string;
}

export interface ChannelPairingRejectResult {
  channelId: string;
  rejected: boolean;
  requestId?: string;
  error?: string;
}

export interface ChannelDebugAccount {
  accountId: string;
  enabled: boolean;
  configured: boolean;
  running: boolean;
  mode?: string;
  botUsername?: string;
  botId?: number;
  webhookUrl?: string;
  webhookSecretSet: boolean;
  dmPolicy?: string;
  groupPolicy?: string;
  allowFromCount: number;
  groupAllowFromCount: number;
  groupsCount: number;
  lastInboundAt?: string;
  lastInboundType?: string;
  lastInboundUpdateId?: number;
  lastInboundMessageId?: number;
  lastInboundChatId?: string;
  lastInboundUserId?: string;
  lastInboundCommand?: string;
  lastDeniedReason?: string;
  lastDeniedAt?: string;
  lastRunAt?: string;
  lastRunId?: string;
  lastRunError?: string;
  lastOutboundAt?: string;
  lastOutboundMessageId?: number;
  lastOutboundError?: string;
  inboundCount: number;
  outboundCount: number;
  deniedCount: number;
  errorCount: number;
  notes?: string[];
}

export interface ChannelDebugSnapshot {
  channelId: string;
  displayName: string;
  kind: string;
  enabled: boolean;
  state: string;
  updatedAt: string;
  lastError?: string;
  accounts?: ChannelDebugAccount[];
  notes?: string[];
}
