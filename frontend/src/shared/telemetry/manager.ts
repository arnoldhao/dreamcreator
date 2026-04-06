import TelemetryDeck from "@telemetrydeck/sdk";
import { Call, Events } from "@wailsio/runtime";

const TELEMETRY_SIGNAL_EVENT = "telemetry:signal";

type TelemetryBootstrap = {
  enabled: boolean;
  appId: string;
  appVersion: string;
  installId: string;
  sessionId: string;
  testMode: boolean;
};

type TelemetrySignal = {
  type: string;
  floatValue?: number;
  payload?: Record<string, unknown>;
};

type TelemetryDeckPrivateClient = TelemetryDeck & {
  target: string;
  _build: (
    type: string,
    payload?: Record<string, unknown>,
    options?: Record<string, unknown>,
    receivedAt?: string
  ) => Promise<Record<string, unknown>>;
};

const resolveTimeZone = () => {
  if (typeof Intl === "undefined" || typeof Intl.DateTimeFormat !== "function") {
    return "";
  }
  return Intl.DateTimeFormat().resolvedOptions().timeZone?.trim() ?? "";
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const stringOrEmpty = (value: unknown) => (typeof value === "string" ? value.trim() : "");
const finiteNumberOrUndefined = (value: unknown) =>
  typeof value === "number" && Number.isFinite(value) ? value : undefined;

const normalizeBootstrap = (value: unknown): TelemetryBootstrap => {
  const raw = isRecord(value) ? value : {};
  return {
    enabled: raw.enabled === true,
    appId: stringOrEmpty(raw.appId),
    appVersion: stringOrEmpty(raw.appVersion),
    installId: stringOrEmpty(raw.installId),
    sessionId: stringOrEmpty(raw.sessionId),
    testMode: raw.testMode === true,
  };
};

const normalizeSignal = (value: unknown): TelemetrySignal | null => {
  const raw = isRecord(value) ? value : {};
  const type = stringOrEmpty(raw.type);
  if (!type) {
    return null;
  }
  const payload = isRecord(raw.payload) ? raw.payload : undefined;
  const floatValue = finiteNumberOrUndefined(raw.floatValue);
  return { type, floatValue, payload };
};

export class TelemetryManager {
  private client: TelemetryDeckPrivateClient | null = null;
  private stopFns: Array<() => void> = [];
  private pendingSignals = new Set<Promise<unknown>>();
  private sessionSummaryRequested = false;
  private readonly timeZone = resolveTimeZone();
  private unloading = false;

  async start() {
    if (typeof window === "undefined") {
      return;
    }

    const bootstrap = normalizeBootstrap(
      await Call.ByName("dreamcreator/internal/presentation/wails.TelemetryHandler.Bootstrap").catch(
        (error) => {
          console.warn("[telemetry] bootstrap failed", error);
          return null;
        }
      )
    );
    if (!bootstrap.enabled || !bootstrap.appId || !bootstrap.installId) {
      return;
    }

    try {
      this.client = new TelemetryDeck({
        appID: bootstrap.appId,
        clientUser: bootstrap.installId,
        sessionID: bootstrap.sessionId || undefined,
        testMode: bootstrap.testMode,
      }) as TelemetryDeckPrivateClient;
    } catch (error) {
      console.warn("[telemetry] sdk init failed", error);
      return;
    }

    const offSignal = Events.On(TELEMETRY_SIGNAL_EVENT, (event: unknown) => {
      const signal = normalizeSignal((event as { data?: unknown } | null)?.data ?? event);
      if (signal) {
        void this.sendSignal(signal);
      }
    });
    this.stopFns.push(offSignal);

    window.addEventListener("pagehide", this.handlePageHide);
    window.addEventListener("beforeunload", this.handleBeforeUnload);

    const emittedLaunchSignals = await Call.ByName("dreamcreator/internal/presentation/wails.TelemetryHandler.TrackAppLaunch").catch(
      (error) => {
        console.warn("[telemetry] app launch tracking failed", error);
        return 0;
      }
    );
    void emittedLaunchSignals;
  }

  stop() {
    for (const stop of this.stopFns.splice(0)) {
      stop();
    }
    window.removeEventListener("pagehide", this.handlePageHide);
    window.removeEventListener("beforeunload", this.handleBeforeUnload);
  }

  private handlePageHide = () => {
    void this.requestSessionSummary();
  };

  private handleBeforeUnload = () => {
    void this.requestSessionSummary();
  };

  private async requestSessionSummary() {
    if (this.sessionSummaryRequested) {
      return;
    }
    this.sessionSummaryRequested = true;
    this.unloading = true;
    await Call.ByName("dreamcreator/internal/presentation/wails.TelemetryHandler.FlushSessionSummary").catch(
        (error) => {
          console.warn("[telemetry] session summary failed", error);
        }
    );
    await this.waitForPendingSignals(750);
  }

  private async sendSignal(signal: TelemetrySignal) {
    if (!this.client) {
      return;
    }
    const body = await this.buildSignalBody(signal);
    if (!body) {
      return;
    }
    let pending: Promise<unknown> | null = null;
    pending = this.postSignalBody(body, this.unloading)
      .then((response) => response)
      .catch((error) => {
        console.warn("[telemetry] signal failed", signal.type, error);
      })
      .finally(() => {
        if (pending) {
          this.pendingSignals.delete(pending);
        }
      });
    this.pendingSignals.add(pending);
    await pending;
  }

  private async buildSignalBody(signal: TelemetrySignal) {
    if (!this.client) {
      return null;
    }
    const payload = signal.payload ? { ...signal.payload } : {};
    if (this.timeZone) {
      payload["DreamCreator.Locale.timeZone"] = this.timeZone;
    }
    const body = await this.client._build(signal.type, payload);
    if (signal.floatValue !== undefined) {
      body["floatValue"] = signal.floatValue;
      if (isRecord(body.payload)) {
        delete body.payload["floatValue"];
      }
    }
    return body;
  }

  private postSignalBody(body: Record<string, unknown>, keepalive: boolean) {
    if (!this.client) {
      return Promise.resolve(undefined);
    }
    return fetch(this.client.target, {
      method: "POST",
      mode: "cors",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify([body]),
      keepalive,
    });
  }

  private async waitForPendingSignals(timeoutMs: number) {
    if (this.pendingSignals.size === 0) {
      return;
    }
    await Promise.race([
      Promise.allSettled(Array.from(this.pendingSignals)),
      new Promise((resolve) => window.setTimeout(resolve, timeoutMs)),
    ]);
  }
}
