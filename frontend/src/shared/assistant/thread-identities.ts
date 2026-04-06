const threadIdAliases = new Map<string, string>();
const threadStreamCursors = new Map<string, number>();

const normalizeThreadId = (value: string | null | undefined) =>
  typeof value === "string" ? value.trim() : "";

export const registerThreadAlias = (localId: string, remoteId: string) => {
  if (!localId || !remoteId || localId === remoteId) {
    return;
  }
  threadIdAliases.set(localId, remoteId);
};

export const removeThreadAlias = (remoteId: string) => {
  if (!remoteId) {
    return;
  }
  threadIdAliases.delete(remoteId);
  for (const [localId, mappedId] of threadIdAliases.entries()) {
    if (mappedId === remoteId) {
      threadIdAliases.delete(localId);
    }
  }
};

export const resolveRemoteThreadId = (threadId: string) => {
  if (!threadId) {
    return "";
  }
  return threadIdAliases.get(threadId) ?? threadId;
};

export const resolvePersistedThreadId = (
  remoteId: string | null | undefined,
  localId: string | null | undefined
) => {
  const normalizedRemoteId = normalizeThreadId(remoteId);
  if (normalizedRemoteId) {
    return resolveRemoteThreadId(normalizedRemoteId);
  }
  const normalizedLocalId = normalizeThreadId(localId);
  if (!normalizedLocalId) {
    return "";
  }
  return resolveRemoteThreadId(normalizedLocalId);
};

export const getThreadStreamCursor = (threadId: string) => {
  const resolved = resolveRemoteThreadId(threadId);
  return threadStreamCursors.get(resolved) ?? 0;
};

export const updateThreadStreamCursor = (threadId: string, eventId: number) => {
  const resolved = resolveRemoteThreadId(threadId);
  if (!resolved || !Number.isFinite(eventId) || eventId <= 0) {
    return;
  }
  const current = threadStreamCursors.get(resolved) ?? 0;
  if (eventId > current) {
    threadStreamCursors.set(resolved, eventId);
  }
};

export const clearThreadStreamCursor = (threadId: string) => {
  if (!threadId) {
    return;
  }
  const resolved = resolveRemoteThreadId(threadId);
  threadStreamCursors.delete(resolved);
  if (resolved != threadId) {
    threadStreamCursors.delete(threadId);
  }
};
