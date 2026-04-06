export type ConnectorStatus = "connected" | "disconnected" | "expired";

export interface ConnectorCookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  expires: number;
  httpOnly: boolean;
  secure: boolean;
  sameSite?: string;
}

export interface Connector {
  id: string;
  type: string;
  group?: string;
  desc?: string;
  status: ConnectorStatus | string;
  cookiesCount?: number;
  cookies?: ConnectorCookie[];
  lastVerifiedAt?: string;
}

export interface UpsertConnectorRequest {
  id?: string;
  type?: string;
  status?: ConnectorStatus | string;
  cookiesPath?: string;
}

export interface ClearConnectorRequest {
  id: string;
}

export interface ConnectConnectorRequest {
  id: string;
}

export interface OpenConnectorSiteRequest {
  id: string;
}
