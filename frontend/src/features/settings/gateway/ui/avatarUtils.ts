import type { Assistant } from "@/shared/store/assistant";

export function resolveAvatarFallback(assistant: Assistant, fallbackLabel: string) {
  const name = assistant.identity?.name?.trim();
  if (!name) {
    return fallbackLabel;
  }
  return name.slice(0, 2).toUpperCase();
}

export function buildAssetPreviewUrl(baseUrl: string, path?: string) {
  if (!baseUrl || !path) {
    return "";
  }
  const trimmed = baseUrl.replace(/\/+$/, "");
  const previewName = path.replace(/\\/g, "/").split("/").pop()?.trim() || "asset";
  return `${trimmed}/api/library/asset/${encodeURIComponent(previewName)}?path=${encodeURIComponent(path)}`;
}
