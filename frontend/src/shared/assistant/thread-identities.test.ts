import { describe, expect, test } from "bun:test";

import {
  clearThreadStreamCursor,
  getThreadStreamCursor,
  registerThreadAlias,
  resolvePersistedThreadId,
  resolveRemoteThreadId,
  updateThreadStreamCursor,
} from "./thread-identities";

describe("thread identity helpers", () => {
  test("prefers the remote thread id when present", () => {
    registerThreadAlias("local-thread", "remote-thread");

    expect(resolvePersistedThreadId("remote-thread", "local-thread")).toBe("remote-thread");
  });

  test("falls back to the aliased local thread id when remote id is not ready yet", () => {
    registerThreadAlias("local-only-thread", "remote-from-alias");

    expect(resolvePersistedThreadId("", "local-only-thread")).toBe("remote-from-alias");
    expect(resolveRemoteThreadId("local-only-thread")).toBe("remote-from-alias");
  });

  test("returns the local thread id when no alias exists", () => {
    expect(resolvePersistedThreadId("", "standalone-thread")).toBe("standalone-thread");
  });

  test("clears stream cursors for both local aliases and remote ids", () => {
    registerThreadAlias("cursor-local-thread", "cursor-remote-thread");
    updateThreadStreamCursor("cursor-local-thread", 42);

    expect(getThreadStreamCursor("cursor-local-thread")).toBe(42);
    expect(getThreadStreamCursor("cursor-remote-thread")).toBe(42);

    clearThreadStreamCursor("cursor-local-thread");

    expect(getThreadStreamCursor("cursor-local-thread")).toBe(0);
    expect(getThreadStreamCursor("cursor-remote-thread")).toBe(0);
  });
});
