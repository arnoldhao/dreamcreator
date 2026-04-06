import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";

const runtimeState = {
  bootstrap: {
    enabled: true,
    appId: "app-123",
    appVersion: "dev",
    installId: "install-1",
    sessionId: "session-1",
    testMode: true,
  },
  calls: [] as string[],
  eventHandlers: new Map<string, (event: unknown) => void>(),
  flushSignal: null as unknown,
};

mock.module("@wailsio/runtime", () => ({
  Call: {
    ByName(name: string) {
      runtimeState.calls.push(name);
      if (name.endsWith(".Bootstrap")) {
        return Promise.resolve(runtimeState.bootstrap);
      }
      if (name.endsWith(".TrackAppLaunch")) {
        return Promise.resolve(0);
      }
      if (name.endsWith(".FlushSessionSummary")) {
        const handler = runtimeState.eventHandlers.get("telemetry:signal");
        if (handler && runtimeState.flushSignal) {
          handler({ data: runtimeState.flushSignal });
        }
        return Promise.resolve(undefined);
      }
      return Promise.resolve(undefined);
    },
  },
  Events: {
    On(name: string, callback: (event: unknown) => void) {
      runtimeState.eventHandlers.set(name, callback);
      return () => {
        if (runtimeState.eventHandlers.get(name) === callback) {
          runtimeState.eventHandlers.delete(name);
        }
      };
    },
  },
}));

const { TelemetryManager } = await import("./manager");

type Listener = () => void;

const originalFetch = globalThis.fetch;
const originalWindow = (globalThis as { window?: unknown }).window;
const windowListeners = new Map<string, Set<Listener>>();
const fetchCalls: Array<{ url: string; init: RequestInit }> = [];

const emitWindowEvent = (eventName: string) => {
  for (const listener of Array.from(windowListeners.get(eventName) ?? [])) {
    listener();
  }
};

const waitFor = async (predicate: () => boolean) => {
  for (let attempt = 0; attempt < 50; attempt++) {
    if (predicate()) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  throw new Error("timed out waiting for telemetry call");
};

beforeEach(() => {
  runtimeState.bootstrap = {
    enabled: true,
    appId: "app-123",
    appVersion: "dev",
    installId: "install-1",
    sessionId: "session-1",
    testMode: true,
  };
  runtimeState.calls = [];
  runtimeState.eventHandlers.clear();
  runtimeState.flushSignal = null;
  windowListeners.clear();
  fetchCalls.length = 0;

  (globalThis as { window?: unknown }).window = {
    addEventListener(eventName: string, listener: Listener) {
      const listeners = windowListeners.get(eventName) ?? new Set<Listener>();
      listeners.add(listener);
      windowListeners.set(eventName, listeners);
    },
    removeEventListener(eventName: string, listener: Listener) {
      const listeners = windowListeners.get(eventName);
      if (!listeners) {
        return;
      }
      listeners.delete(listener);
      if (listeners.size === 0) {
        windowListeners.delete(eventName);
      }
    },
    setTimeout,
  };

  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    fetchCalls.push({ url: String(input), init: init ?? {} });
    return new Response(null, { status: 202 });
  }) as typeof fetch;
});

afterEach(() => {
  globalThis.fetch = originalFetch;
  if (originalWindow === undefined) {
    delete (globalThis as { window?: unknown }).window;
  } else {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
});

describe("TelemetryManager", () => {
  it("subscribes only to telemetry signals and real unload events", async () => {
    const manager = new TelemetryManager();

    await manager.start();

    expect(runtimeState.eventHandlers.has("telemetry:signal")).toBe(true);
    expect(runtimeState.eventHandlers.has("common:WindowClosing")).toBe(false);
    expect(windowListeners.has("pagehide")).toBe(true);
    expect(windowListeners.has("beforeunload")).toBe(true);

    manager.stop();

    expect(windowListeners.has("pagehide")).toBe(false);
    expect(windowListeners.has("beforeunload")).toBe(false);
  });

  it("posts session summaries with top-level floatValue and keepalive during unload", async () => {
    runtimeState.flushSignal = {
      type: "DreamCreator.Session.summaryRecorded",
      floatValue: 600,
      payload: {
        "DreamCreator.Session.durationBucket": "5m-15m",
      },
    };
    const manager = new TelemetryManager();

    await manager.start();
    emitWindowEvent("pagehide");
    await waitFor(() => fetchCalls.length === 1);

    const flushCalls = runtimeState.calls.filter((name) => name.endsWith(".FlushSessionSummary"));
    expect(flushCalls).toHaveLength(1);
    expect(fetchCalls).toHaveLength(1);
    expect(fetchCalls[0]?.init.keepalive).toBe(true);

    const payload = JSON.parse(String(fetchCalls[0]?.init.body)) as Array<Record<string, unknown>>;
    const body = payload[0] ?? {};
    const metadata = body.payload as Record<string, unknown>;

    expect(body.type).toBe("DreamCreator.Session.summaryRecorded");
    expect(body.sessionID).toBe("session-1");
    expect(body.isTestMode).toBe(true);
    expect(body.floatValue).toBe(600);
    expect(metadata["DreamCreator.Session.durationBucket"]).toBe("5m-15m");
    expect(metadata.floatValue).toBeUndefined();

    emitWindowEvent("beforeunload");
    expect(runtimeState.calls.filter((name) => name.endsWith(".FlushSessionSummary"))).toHaveLength(1);

    manager.stop();
  });
});
