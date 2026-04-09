import type { ProxySettings } from "@/shared/contracts/settings";

export function isProxyConfigEqual(left: ProxySettings, right: ProxySettings) {
  const noProxyLeft = JSON.stringify(left.noProxy ?? []);
  const noProxyRight = JSON.stringify(right.noProxy ?? []);
  return (
    left.mode === right.mode &&
    left.scheme === right.scheme &&
    left.host === right.host &&
    left.port === right.port &&
    left.username === right.username &&
    left.password === right.password &&
    left.timeoutSeconds === right.timeoutSeconds &&
    noProxyLeft === noProxyRight
  );
}

export function mergeIncomingProxyDraft(
  currentDraft: ProxySettings | null,
  previousSavedDraft: ProxySettings | null,
  incoming: ProxySettings,
): ProxySettings {
  if (!currentDraft || !previousSavedDraft || isProxyConfigEqual(currentDraft, previousSavedDraft)) {
    return incoming;
  }
  return currentDraft;
}
