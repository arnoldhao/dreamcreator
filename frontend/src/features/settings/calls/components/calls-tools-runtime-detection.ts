import { isRecord } from "../utils/calls-utils";
import { readBoolValue, readStringValue } from "../utils/web-tool-settings-utils";

export type RuntimeBrowserCandidate = {
  id: string;
  label: string;
  available: boolean;
  execPath: string;
  error: string;
};

export type RuntimeDetectionRow = {
  label: string;
  value: string;
  badge?: "not_installed" | "not_detected";
};

const BROWSER_LABELS: Record<string, string> = {
  chrome: "Chrome",
  chromium: "Chromium",
  edge: "Edge",
  brave: "Brave",
};

export const normalizeRuntimeBrowserCandidates = (value: unknown): RuntimeBrowserCandidate[] => {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!isRecord(item)) {
      return [];
    }
    const id = readStringValue(item, "id", "").trim().toLowerCase();
    const fallbackLabel = id ? (BROWSER_LABELS[id] ?? id) : "Browser";
    return [{
      id,
      label: readStringValue(item, "label", fallbackLabel).trim() || fallbackLabel,
      available: readBoolValue(item, "available", false),
      execPath: readStringValue(item, "execPath", "").trim(),
      error: readStringValue(item, "error", "").trim(),
    }];
  });
};
