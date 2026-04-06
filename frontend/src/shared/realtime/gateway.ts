import { Call } from "@wailsio/runtime";

export type GatewayStatus = "disconnected" | "connecting" | "connected";

export type GatewayEvent = {
  event: string;
  payload?: unknown;
  timestamp: number;
  sessionId?: string;
  sessionKey?: string;
  runId?: string;
  seq?: number;
  stateVersion?: number;
};

type GatewayClientOptions = {
  reconnectIntervalMs?: number;
  connectTimeoutMs?: number;
  requestTimeoutMs?: number;
  onStatusChange?: (status: GatewayStatus) => void;
};

const DEFAULT_GATEWAY_SCOPES = [
  "gateway.ping",
  "config.get",
  "config.set",
  "config.patch",
  "config.apply",
  "config.schema",
  "usage.status",
  "usage.cost",
  "tts.status",
  "tts.config.set",
  "tts.convert",
  "talk.config",
  "talk.config.set",
  "talk.mode",
  "voicewake.get",
  "voicewake.set",
  "channels.list",
  "channels.status",
  "channels.logout",
  "channels.probe",
  "channels.debug",
  "channels.menu.sync",
  "channels.pairing.list",
  "channels.pairing.approve",
  "channels.pairing.reject",
  "commands.list",
  "commands.run",
  "node.list",
  "node.describe",
  "node.invoke",
  "node.pair.request",
  "node.pair.approve",
  "node.pair.reject",
  "node.token.rotate",
  "node.token.revoke",
  "exec.approval.request",
  "exec.approval.resolve",
  "exec.approval.wait",
  "agents.list",
  "agents.create",
  "agents.update",
  "agents.delete",
  "agents.files.list",
  "agents.files.get",
  "agents.files.set",
  "models.list",
  "runtime.run",
  "runtime.abort",
  "heartbeat.read",
  "heartbeat.trigger",
  "heartbeat.toggle",
  "cron.status",
  "cron.list",
  "cron.add",
  "cron.update",
  "cron.remove",
  "cron.run",
  "cron.runs",
  "cron.runDetail",
  "cron.runEvents",
  "cron.wake",
];

const buildClientInfo = () => {
  const platform = typeof navigator === "undefined" ? "" : navigator.userAgent || "";
  return {
    id: "dreamcreator-ui",
    displayName: "DreamCreator UI",
    version: "",
    platform,
    mode: "desktop",
  };
};

const buildConnectFrame = (scopes: string[]) => ({
  type: "req",
  id: `connect-${Date.now()}`,
  method: "connect",
  params: {
    minProtocol: 1,
    maxProtocol: 1,
    client: buildClientInfo(),
    role: "operator",
    scopes,
    auth: {},
  },
});

class GatewayClient {
  private socket: WebSocket | null = null;
  private status: GatewayStatus = "disconnected";
  private readonly reconnectIntervalMs: number;
  private readonly connectTimeoutMs: number;
  private readonly requestTimeoutMs: number;
  private readonly onStatusChange?: (status: GatewayStatus) => void;
  private readonly pending = new Map<
    string,
    { resolve: (value: any) => void; reject: (reason?: any) => void; timeoutId: number }
  >();
  private readonly eventHandlers = new Set<(event: GatewayEvent) => void>();
  private reconnectTimer: number | null = null;
  private shouldReconnect = true;
  private handshakePromise: Promise<void> | null = null;
  private handshakeResolve: (() => void) | null = null;
  private handshakeReject: ((error: Error) => void) | null = null;

  constructor(private readonly url: string, options?: GatewayClientOptions) {
    this.reconnectIntervalMs = options?.reconnectIntervalMs ?? 3_000;
    this.connectTimeoutMs = options?.connectTimeoutMs ?? 5_000;
    this.requestTimeoutMs = options?.requestTimeoutMs ?? 5_000;
    this.onStatusChange = options?.onStatusChange;
  }

  connect(): Promise<void> {
    if (this.status === "connected" && this.socket) {
      return Promise.resolve();
    }
    if (this.handshakePromise) {
      return this.handshakePromise;
    }
    this.shouldReconnect = true;
    this.setStatus("connecting");

    this.handshakePromise = new Promise<void>((resolve, reject) => {
      this.handshakeResolve = resolve;
      this.handshakeReject = reject;
    });

    const timeout = window.setTimeout(() => {
      this.failHandshake(new Error("gateway connect timeout"));
      this.socket?.close();
    }, this.connectTimeoutMs);

    try {
      this.socket = new WebSocket(this.url);
      this.socket.addEventListener("open", () => {
        window.clearTimeout(timeout);
        this.sendFrame(buildConnectFrame(DEFAULT_GATEWAY_SCOPES));
      });
      this.socket.addEventListener("message", (event) => this.handleMessage(event));
      this.socket.addEventListener("close", () => this.handleClose());
      this.socket.addEventListener("error", () => this.handleClose());
    } catch (error) {
      window.clearTimeout(timeout);
      this.failHandshake(error instanceof Error ? error : new Error("gateway connect failed"));
      this.handleClose();
    }

    return this.handshakePromise;
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.socket?.close();
    this.socket = null;
    this.setStatus("disconnected");
    this.clearPending(new Error("gateway socket disconnected"));
  }

  async sendRequest(method: string, params?: unknown, timeoutMs?: number): Promise<any> {
    await this.connect();
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const frame = { type: "req", id, method, params };
    const timeout = window.setTimeout(() => {
      const entry = this.pending.get(id);
      if (entry) {
        this.pending.delete(id);
        entry.reject(new Error("gateway request timeout"));
      }
    }, timeoutMs ?? this.requestTimeoutMs);
    const promise = new Promise<any>((resolve, reject) => {
      this.pending.set(id, { resolve, reject, timeoutId: timeout });
    });
    this.sendFrame(frame);
    return promise;
  }

  subscribe(handler: (event: GatewayEvent) => void): () => void {
    this.eventHandlers.add(handler);
    return () => {
      this.eventHandlers.delete(handler);
    };
  }

  private scheduleReconnect() {
    if (this.reconnectTimer || !this.shouldReconnect) {
      return;
    }
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.handshakePromise = null;
      void this.connect();
    }, this.reconnectIntervalMs);
  }

  private handleClose() {
    this.setStatus("disconnected");
    this.failHandshake(new Error("gateway socket closed"));
    this.clearPending(new Error("gateway socket closed"));
    this.scheduleReconnect();
  }

  private handleMessage(event: MessageEvent<string>) {
    let parsed: any;
    try {
      parsed = JSON.parse(event.data);
    } catch {
      return;
    }

    if (parsed?.type === "hello-ok") {
      this.resolveHandshake();
      this.setStatus("connected");
      return;
    }

    if (this.status === "connecting" && parsed?.code && parsed?.message) {
      this.failHandshake(new Error(String(parsed.message)));
      this.socket?.close();
      return;
    }

    if (parsed?.type === "res" && parsed?.id) {
      const entry = this.pending.get(parsed.id);
      if (entry) {
        window.clearTimeout(entry.timeoutId);
        this.pending.delete(parsed.id);
        if (parsed.ok) {
          entry.resolve(parsed.payload);
        } else {
          entry.reject(parsed.error ?? new Error("gateway request failed"));
        }
      }
      return;
    }

    if (parsed?.type === "event" && parsed?.event) {
      const tsValue = parsed?.timestamp ?? parsed?.ts ?? parsed?.Timestamp ?? parsed?.TS;
      const timestamp =
        typeof tsValue === "number"
          ? tsValue
          : typeof tsValue === "string"
          ? Date.parse(tsValue) || Date.now()
          : Date.now();
      const gatewayEvent: GatewayEvent = {
        event: String(parsed.event),
        payload: parsed.payload,
        timestamp,
        sessionId: parsed.sessionId ?? parsed.SessionID,
        sessionKey: parsed.sessionKey ?? parsed.SessionKey,
        runId: parsed.runId ?? parsed.RunID,
        seq: parsed.seq ?? parsed.Seq,
        stateVersion: parsed.stateVersion ?? parsed.StateVersion,
      };
      this.eventHandlers.forEach((handler) => handler(gatewayEvent));
    }
  }

  private sendFrame(frame: unknown) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }
    this.socket.send(JSON.stringify(frame));
  }

  private setStatus(status: GatewayStatus) {
    if (this.status === status) {
      return;
    }
    this.status = status;
    this.onStatusChange?.(status);
  }

  private resolveHandshake() {
    if (this.handshakeResolve) {
      this.handshakeResolve();
    }
    this.handshakeResolve = null;
    this.handshakeReject = null;
    this.handshakePromise = null;
  }

  private failHandshake(error: Error) {
    if (this.handshakeReject) {
      this.handshakeReject(error);
    }
    this.handshakeResolve = null;
    this.handshakeReject = null;
    this.handshakePromise = null;
  }

  private clearPending(error: Error) {
    for (const entry of this.pending.values()) {
      window.clearTimeout(entry.timeoutId);
      entry.reject(error);
    }
    this.pending.clear();
  }
}

let gatewayClient: GatewayClient | null = null;
let gatewayStartPromise: Promise<GatewayClient> | null = null;

async function resolveGatewayURL(): Promise<string> {
  try {
    const base = await Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.HTTPBaseURL");
    const trimmed = typeof base === "string" ? base.trim() : String(base ?? "").trim();
    if (!trimmed) {
      return "";
    }
    const url = new URL(trimmed);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = "/gateway/ws";
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch {
    return "";
  }
}

export async function startGateway(): Promise<GatewayClient> {
  if (gatewayStartPromise) {
    return gatewayStartPromise;
  }
  gatewayStartPromise = (async () => {
    const url = await resolveGatewayURL();
    if (!url) {
      gatewayStartPromise = null;
      throw new Error("gateway url unavailable");
    }
    const client = new GatewayClient(url);
    await client.connect();
    gatewayClient = client;
    gatewayStartPromise = null;
    return client;
  })();
  return gatewayStartPromise;
}

export async function requestGateway(method: string, params?: unknown, timeoutMs?: number) {
  const client = gatewayClient ?? (await startGateway());
  return client.sendRequest(method, params, timeoutMs);
}

export function subscribeGatewayEvents(handler: (event: GatewayEvent) => void): () => void {
  let active = true;
  let unsubscribe: (() => void) | null = null;
  startGateway()
    .then((client) => {
      if (!active) {
        return;
      }
      unsubscribe = client.subscribe(handler);
    })
    .catch(() => {
      // best-effort: keep silent until gateway becomes available
    });
  return () => {
    active = false;
    unsubscribe?.();
  };
}
