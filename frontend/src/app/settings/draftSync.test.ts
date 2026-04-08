import { describe, expect, it } from "bun:test";

import { canAdoptIncomingDraftSnapshot, shouldSkipDraftPersist } from "./draftSync";

describe("draft sync", () => {
  it("adopts the latest remote snapshot when the draft already matches it", () => {
    expect(
      canAdoptIncomingDraftSnapshot({
        draftSignature: "current",
        currentRemoteSignature: "current",
        previousRemoteSignature: "previous",
        lastPersistedSignature: "",
      })
    ).toBe(true);
  });

  it("adopts a remote change when the draft still matches the previous saved snapshot", () => {
    expect(
      canAdoptIncomingDraftSnapshot({
        draftSignature: "previous",
        currentRemoteSignature: "current",
        previousRemoteSignature: "previous",
        lastPersistedSignature: "",
      })
    ).toBe(true);
  });

  it("does not treat a submitted draft as persisted before the remote snapshot changes", () => {
    expect(
      canAdoptIncomingDraftSnapshot({
        draftSignature: "submitted",
        currentRemoteSignature: "server-old",
        previousRemoteSignature: "server-old",
        lastPersistedSignature: "submitted",
      })
    ).toBe(false);
  });

  it("adopts the remote snapshot after a successful save changes the backend state", () => {
    expect(
      canAdoptIncomingDraftSnapshot({
        draftSignature: "submitted",
        currentRemoteSignature: "server-normalized",
        previousRemoteSignature: "server-old",
        lastPersistedSignature: "submitted",
      })
    ).toBe(true);
  });

  it("skips autosave while the same draft is already pending", () => {
    expect(
      shouldSkipDraftPersist({
        draftSignature: "draft",
        lastPersistedSignature: "",
        pendingSubmittedSignature: "draft",
      })
    ).toBe(true);
  });

  it("does not skip autosave for a new draft after the user edits again", () => {
    expect(
      shouldSkipDraftPersist({
        draftSignature: "draft-next",
        lastPersistedSignature: "draft-previous",
        pendingSubmittedSignature: "draft-previous",
      })
    ).toBe(false);
  });
});
