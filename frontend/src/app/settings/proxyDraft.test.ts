import { describe, expect, it } from "bun:test";

import type { ProxySettings } from "@/shared/contracts/settings";
import { isProxyConfigEqual, mergeIncomingProxyDraft } from "./proxyDraft";

function createProxy(overrides: Partial<ProxySettings> = {}): ProxySettings {
  return {
    mode: "none",
    scheme: "http",
    host: "",
    port: 0,
    username: "",
    password: "",
    noProxy: [],
    timeoutSeconds: 30,
    testedAt: "",
    testSuccess: false,
    testMessage: "",
    ...overrides,
  };
}

describe("proxy draft sync", () => {
  it("treats test metadata as non-dirty when proxy config is unchanged", () => {
    const saved = createProxy({ mode: "system" });
    const current = createProxy({
      mode: "system",
      testedAt: "2026-04-08T17:59:41+08:00",
      testSuccess: true,
      testMessage: "status 204",
    });

    expect(isProxyConfigEqual(current, saved)).toBe(true);
  });

  it("adopts an incoming backend proxy snapshot when the local draft still matches the previous saved snapshot", () => {
    const previousSaved = createProxy({ mode: "none" });
    const incoming = createProxy({ mode: "system" });

    expect(mergeIncomingProxyDraft(previousSaved, previousSaved, incoming)).toEqual(incoming);
  });

  it("keeps the local draft when the user has unsaved proxy edits", () => {
    const previousSaved = createProxy({ mode: "none" });
    const currentDraft = createProxy({
      mode: "manual",
      host: "127.0.0.1",
      port: 7890,
    });
    const incoming = createProxy({ mode: "system" });

    expect(mergeIncomingProxyDraft(currentDraft, previousSaved, incoming)).toEqual(currentDraft);
  });
});
