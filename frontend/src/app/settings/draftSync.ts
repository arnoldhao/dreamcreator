type DraftSyncParams = {
  draftSignature: string;
  currentRemoteSignature: string;
  previousRemoteSignature: string;
  lastPersistedSignature: string;
};

type DraftPersistState = {
  draftSignature: string;
  lastPersistedSignature: string;
  pendingSubmittedSignature: string;
};

export function canAdoptIncomingDraftSnapshot({
  draftSignature,
  currentRemoteSignature,
  previousRemoteSignature,
  lastPersistedSignature,
}: DraftSyncParams) {
  if (draftSignature === currentRemoteSignature) {
    return true;
  }
  if (draftSignature === previousRemoteSignature) {
    return true;
  }
  return (
    lastPersistedSignature !== "" &&
    draftSignature === lastPersistedSignature &&
    currentRemoteSignature !== previousRemoteSignature
  );
}

export function shouldSkipDraftPersist({
  draftSignature,
  lastPersistedSignature,
  pendingSubmittedSignature,
}: DraftPersistState) {
  if (draftSignature === lastPersistedSignature) {
    return true;
  }
  return pendingSubmittedSignature !== "" && draftSignature === pendingSubmittedSignature;
}
