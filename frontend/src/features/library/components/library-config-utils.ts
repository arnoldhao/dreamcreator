import { t } from "@/shared/i18n";
import type {
  LibraryBilingualStyleDTO,
  LibraryGlossaryProfileDTO,
  LibraryGlossaryTermDTO,
  LibraryMonoStyleDTO,
  LibraryPromptProfileDTO,
  LibrarySubtitleStyleFontDTO,
  LibrarySubtitleStyleSourceDTO,
  LibraryTaskRuntimeSettingsDTO,
  LibraryTranslateLanguageDTO,
} from "@/shared/contracts/library";

import { formatRelativeTime } from "../utils/time";

export type LanguageAssetTabId = "languages" | "glossary" | "prompts";
export type LanguageConfigItemKind = "builtin" | "custom";

export type RemoteFontSearchCandidate = {
  sourceId: string;
  sourceName: string;
  fontId: string;
  family: string;
  matchName: string;
  matchType: string;
  assetCount: number;
  declaredAssetCount: number;
  installable: boolean;
  availability: string;
  unavailableReason?: string;
};

function resolveLanguageLabel(
  value: string,
  options: LibraryTranslateLanguageDTO[],
) {
  const normalized = value.trim().toLowerCase();
  if (!normalized || normalized === "all") {
    return "";
  }
  const matched = options.find(
    (option) => option.code.trim().toLowerCase() === normalized,
  );
  if (!matched) {
    return value.trim().toUpperCase();
  }
  return `${matched.label} (${matched.code})`;
}

function describeLanguagePair(
  source: string | undefined,
  target: string | undefined,
  options: LibraryTranslateLanguageDTO[],
) {
  const sourceLabel = resolveLanguageLabel(source ?? "", options);
  const targetLabel = resolveLanguageLabel(target ?? "", options);
  if (!sourceLabel && !targetLabel) {
    return "";
  }
  return `${sourceLabel || t("library.config.languageAssets.anyLanguage")} -> ${
    targetLabel || t("library.config.languageAssets.anyLanguage")
  }`;
}

export function resolveScopedLanguageOptions(
  options: LibraryTranslateLanguageDTO[],
) {
  const seen = new Set<string>();
  const result: Array<{ value: string; label: string }> = [
    {
      value: "all",
      label: t("library.config.languageAssets.promptAll"),
    },
  ];
  for (const option of options) {
    const code = option.code.trim();
    if (!code) {
      continue;
    }
    const normalized = code.toLowerCase();
    if (normalized === "all" || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    result.push({
      value: code,
      label: option.label ? `${option.label} (${option.code})` : option.code,
    });
  }
  return result;
}

export function normalizeScopedLanguageValue(value: string | undefined) {
  const trimmed = value?.trim() ?? "";
  if (!trimmed || trimmed.toLowerCase() === "all") {
    return "all";
  }
  return trimmed;
}

export function syncSelectedAssetItemId(
  current: Record<LanguageAssetTabId, string>,
  key: LanguageAssetTabId,
  ids: string[],
) {
  const currentId = current[key];
  const nextId =
    currentId && ids.includes(currentId) ? currentId : (ids[0] ?? "");
  if (nextId === currentId) {
    return current;
  }
  return {
    ...current,
    [key]: nextId,
  };
}

function resolveStructuredOutputModeLabel(value: string | undefined) {
  switch (value?.trim()) {
    case "json_schema":
      return t("library.config.taskRuntime.structuredOutputSchema");
    case "prompt_only":
      return t("library.config.taskRuntime.structuredOutputPrompt");
    case "auto":
    case "":
    case undefined:
      return t("library.config.taskRuntime.structuredOutputAuto");
    default:
      return value;
  }
}

function resolveThinkingModeLabel(value: string | undefined) {
  switch (value?.trim()) {
    case "minimal":
      return t("library.config.taskRuntime.thinkingMinimal");
    case "low":
      return t("library.config.taskRuntime.thinkingLow");
    case "medium":
      return t("library.config.taskRuntime.thinkingMedium");
    case "high":
      return t("library.config.taskRuntime.thinkingHigh");
    case "xhigh":
      return t("library.config.taskRuntime.thinkingExtraHigh");
    case "off":
    case "":
    case undefined:
      return t("library.config.taskRuntime.thinkingOff");
    default:
      return value;
  }
}

export function summarizeTaskRuntimeSettings(
  value: LibraryTaskRuntimeSettingsDTO,
) {
  return [
    resolveStructuredOutputModeLabel(value.structuredOutputMode),
    resolveThinkingModeLabel(value.thinkingMode),
    `${value.maxTokensFloor}-${value.maxTokensCeiling}`,
  ].join(" · ");
}

export function resolvePromptCategoryLabel(category: string | undefined) {
  switch (normalizePromptCategory(category)) {
    case "all":
      return t("library.config.languageAssets.promptAll");
    case "proofread":
      return t("library.config.languageAssets.promptProofread");
    case "glossary":
      return t("library.config.languageAssets.promptGlossary");
    case "translate":
      return t("library.config.languageAssets.promptTranslate");
    default:
      return category ?? "";
  }
}

export function normalizePromptCategory(category: string | undefined) {
  const normalized = category?.trim().toLowerCase() ?? "";
  switch (normalized) {
    case "translate":
    case "proofread":
    case "glossary":
    case "all":
      return normalized;
    default:
      return "all";
  }
}

export function normalizeGlossaryCategory(category: string | undefined) {
  const normalized = category?.trim().toLowerCase() ?? "";
  switch (normalized) {
    case "translate":
    case "proofread":
    case "all":
      return normalized;
    default:
      return "all";
  }
}

export function resolveGlossaryCategoryLabel(category: string | undefined) {
  switch (normalizeGlossaryCategory(category)) {
    case "proofread":
      return t("library.config.languageAssets.promptProofread");
    case "translate":
      return t("library.config.languageAssets.promptTranslate");
    case "all":
    default:
      return t("library.config.languageAssets.promptAll");
  }
}

export function resolveSubtitleStyleSyncStatusLabel(status: string) {
  const normalized = status.trim().toLowerCase();
  if (!normalized || normalized === "idle") {
    return t("library.config.subtitleStyles.syncIdle");
  }
  if (normalized === "ready") {
    return t("library.config.subtitleStyles.syncReady");
  }
  if (normalized === "syncing") {
    return t("library.config.subtitleStyles.syncSyncing");
  }
  if (normalized === "error") {
    return t("library.config.subtitleStyles.syncError");
  }
  return status;
}

function compactPrompt(prompt: string) {
  const collapsed = prompt.replace(/\s+/g, " ").trim();
  if (!collapsed) {
    return "";
  }
  return collapsed.length > 120 ? `${collapsed.slice(0, 117)}...` : collapsed;
}

export function buildEditableCardID(scope: string, id: string) {
  return `${scope}:${id}`;
}

export function buildBuiltinLanguageItemID(code: string) {
  const normalized = code.trim().toLowerCase() || "builtin";
  return `builtin:${normalized}`;
}

export function buildCustomLanguageItemID(rowID: string) {
  return `custom:${rowID.trim()}`;
}

export function summarizeLanguageConfigItem(
  item: LibraryTranslateLanguageDTO,
  _kind: LanguageConfigItemKind,
) {
  const aliases = (item.aliases ?? [])
    .map((alias) => alias.trim())
    .filter(Boolean);
  return aliases.length > 0
    ? aliases.slice(0, 3).join(", ") + (aliases.length > 3 ? "..." : "")
    : "";
}

function summarizeGlossaryTerms(terms: LibraryGlossaryTermDTO[] | undefined) {
  const entries = (terms ?? [])
    .map((term) => {
      const source = term.source.trim();
      const target = term.target.trim();
      if (!source || !target) {
        return "";
      }
      return `${source} -> ${target}`;
    })
    .filter(Boolean);
  if (entries.length === 0) {
    return "";
  }
  return entries.slice(0, 2).join(" · ") + (entries.length > 2 ? "..." : "");
}

export function summarizeGlossaryProfile(
  profile: LibraryGlossaryProfileDTO,
  options: LibraryTranslateLanguageDTO[],
) {
  const languagePair = describeLanguagePair(
    profile.sourceLanguage,
    profile.targetLanguage,
    options,
  );
  return [
    profile.description?.trim(),
    summarizeGlossaryTerms(profile.terms),
    languagePair,
  ].find(Boolean);
}

export function summarizePromptProfile(profile: LibraryPromptProfileDTO) {
  return [
    profile.description?.trim(),
    compactPrompt(profile.prompt),
    resolvePromptCategoryLabel(profile.category),
  ].find(Boolean);
}

export function resolveSubtitleStyleSourceKind(
  value: string | undefined,
): "style" | "font" {
  return value?.trim().toLowerCase() === "font" ? "font" : "style";
}

export function resolveFontSourceDisplayName(
  source: LibrarySubtitleStyleSourceDTO,
) {
  return (
    source.remoteFontManifest?.sourceInfo?.name?.trim() ||
    source.name?.trim() ||
    source.id
  );
}

export function resolveFontSourceSummary(
  source: LibrarySubtitleStyleSourceDTO,
  syncing: boolean,
  language: string,
) {
  const sourceInfo = source.remoteFontManifest?.sourceInfo;
  const summaryParts: string[] = [];
  const fontCount = Math.max(
    0,
    Number(sourceInfo?.totalFonts ?? source.fontCount ?? 0),
  );
  if (fontCount > 0) {
    summaryParts.push(
      `${fontCount} ${t("library.config.subtitleStyles.fontSourceFontsUnit", language as "en" | "zh-CN")}`,
    );
  }
  const remoteUpdated = sourceInfo?.lastUpdated?.trim()
    ? formatRelativeTime(sourceInfo.lastUpdated, language)
    : "";
  if (remoteUpdated && remoteUpdated !== "-") {
    summaryParts.push(
      `${t("library.config.subtitleStyles.fontSourceUpdated", language as "en" | "zh-CN")} ${remoteUpdated}`,
    );
  }
  if (syncing) {
    summaryParts.push(
      t("library.config.subtitleStyles.syncingFontSourceDescription", language as "en" | "zh-CN"),
    );
    return summaryParts.join(" · ");
  }
  if (source.syncStatus === "error" && source.lastError?.trim()) {
    summaryParts.push(
      `${t("library.config.subtitleStyles.syncFontSourceFailedShort", language as "en" | "zh-CN")} · ${compactSummaryText(source.lastError)}`,
    );
    return summaryParts.join(" · ");
  }
  if (source.lastSyncedAt?.trim()) {
    const lastSynced = formatRelativeTime(source.lastSyncedAt, language);
    summaryParts.push(
      `${t("library.config.subtitleStyles.fontSourceLastSynced", language as "en" | "zh-CN")} ${lastSynced}`,
    );
    return summaryParts.join(" · ");
  }
  if (summaryParts.length > 0) {
    return summaryParts.join(" · ");
  }
  return t("library.config.subtitleStyles.fontSourceNeverSynced", language as "en" | "zh-CN");
}

function compactSummaryText(value: string, maxLength = 88) {
  const normalized = value.trim().replace(/\s+/g, " ");
  if (normalized.length <= maxLength) {
    return normalized;
  }
  return normalized.slice(0, Math.max(0, maxLength - 3)).trimEnd() + "...";
}

export function buildRemoteStyleDocumentSourceRef(
  source: LibrarySubtitleStyleSourceDTO,
  itemId: string,
) {
  return `${source.owner?.trim() || "-"}/${source.repo?.trim() || "-"}@${source.ref?.trim() || "main"}#${itemId.trim()}`;
}

export function normalizeFontFamilyKey(value: string) {
  return value
    .trim()
    .replace(/^['"]+|['"]+$/g, "")
    .replace(/\s+/g, " ")
    .toLowerCase();
}

function resolveEnabledSubtitleStyleFontMappings(
  fontMappings: LibrarySubtitleStyleFontDTO[],
) {
  const mappings = new Map<string, LibrarySubtitleStyleFontDTO>();
  for (const mapping of fontMappings) {
    if (mapping.enabled === false) {
      continue;
    }
    const family = mapping.family?.trim();
    const systemFamily = mapping.systemFamily?.trim();
    if (!family || !systemFamily) {
      continue;
    }
    mappings.set(normalizeFontFamilyKey(family), mapping);
  }
  return mappings;
}

export function resolveSubtitleStyleDocumentFontCoverage(
  fonts: string[],
  normalizedSystemFonts: Set<string>,
  fontMappings: LibrarySubtitleStyleFontDTO[],
) {
  const mappings = resolveEnabledSubtitleStyleFontMappings(fontMappings);
  const resolvedSet = new Set<string>();
  const managedSet = new Set<string>();
  const missingSet = new Set<string>();

  for (const font of fonts) {
    const trimmedFont = font.trim();
    if (!trimmedFont) {
      continue;
    }
    const normalizedFont = normalizeFontFamilyKey(trimmedFont);
    if (normalizedSystemFonts.has(normalizedFont)) {
      resolvedSet.add(trimmedFont);
      continue;
    }
    const mapping = mappings.get(normalizedFont);
    if (
      mapping &&
      normalizedSystemFonts.has(
        normalizeFontFamilyKey(mapping.systemFamily ?? ""),
      )
    ) {
      resolvedSet.add(trimmedFont);
      managedSet.add(trimmedFont);
      continue;
    }
    missingSet.add(trimmedFont);
  }

  return {
    resolved: [...resolvedSet].sort((left, right) => left.localeCompare(right)),
    managed: [...managedSet].sort((left, right) => left.localeCompare(right)),
    missing: [...missingSet].sort((left, right) => left.localeCompare(right)),
  };
}

function collectSubtitleStylePresetFontUsage(
  monoStyles: LibraryMonoStyleDTO[],
  bilingualStyles: LibraryBilingualStyleDTO[],
) {
  const usage = new Map<string, Set<string>>();
  const appendUsage = (rawFamily: string | undefined, styleName: string) => {
    const family = (rawFamily ?? "").trim();
    if (!family) {
      return;
    }
    if (!usage.has(family)) {
      usage.set(family, new Set<string>());
    }
    usage.get(family)?.add(styleName);
  };

  for (const mono of monoStyles) {
    const styleName = mono.name?.trim() || mono.id || "Mono";
    appendUsage(mono.style?.fontname, styleName);
  }
  for (const bilingual of bilingualStyles) {
    const styleName = bilingual.name?.trim() || bilingual.id || "Bilingual";
    appendUsage(bilingual.primary?.style?.fontname, styleName);
    appendUsage(bilingual.secondary?.style?.fontname, styleName);
  }

  return usage;
}

export function resolveSubtitleStylePresetFontCoverage(
  monoStyles: LibraryMonoStyleDTO[],
  bilingualStyles: LibraryBilingualStyleDTO[],
  normalizedSystemFonts: Set<string>,
  fontMappings: LibrarySubtitleStyleFontDTO[],
) {
  const mappings = resolveEnabledSubtitleStyleFontMappings(fontMappings);
  const requiredSet = new Set<string>();
  const resolvedSet = new Set<string>();
  const managedSet = new Set<string>();
  const missingSet = new Set<string>();
  const usage = collectSubtitleStylePresetFontUsage(monoStyles, bilingualStyles);

  for (const family of usage.keys()) {
    const normalizedFamily = normalizeFontFamilyKey(family);
    requiredSet.add(family);
    if (normalizedSystemFonts.has(normalizedFamily)) {
      resolvedSet.add(family);
      continue;
    }
    const mapping = mappings.get(normalizedFamily);
    if (
      mapping &&
      normalizedSystemFonts.has(
        normalizeFontFamilyKey(mapping.systemFamily ?? ""),
      )
    ) {
      resolvedSet.add(family);
      managedSet.add(family);
      continue;
    }
    missingSet.add(family);
  }

  return {
    required: [...requiredSet].sort((left, right) => left.localeCompare(right)),
    resolved: [...resolvedSet].sort((left, right) => left.localeCompare(right)),
    managed: [...managedSet].sort((left, right) => left.localeCompare(right)),
    missing: [...missingSet].sort((left, right) => left.localeCompare(right)),
  };
}

export function resolveSubtitleStylePresetReferencedFonts(
  monoStyles: LibraryMonoStyleDTO[],
  bilingualStyles: LibraryBilingualStyleDTO[],
  normalizedSystemFonts: Set<string>,
  fontMappings: LibrarySubtitleStyleFontDTO[],
) {
  const mappings = resolveEnabledSubtitleStyleFontMappings(fontMappings);
  const fontUsage = collectSubtitleStylePresetFontUsage(
    monoStyles,
    bilingualStyles,
  );

  return [...fontUsage.entries()]
    .map(([family, stylesUsingFont]) => {
      const normalizedFamily = normalizeFontFamilyKey(family);
      const mapping = mappings.get(normalizedFamily);
      if (normalizedSystemFonts.has(normalizedFamily)) {
        return {
          family,
          documents: [...stylesUsingFont].sort((left, right) =>
            left.localeCompare(right),
          ),
          status: "auto" as const,
          systemFamily: family,
        };
      }
      if (
        mapping &&
        normalizedSystemFonts.has(
          normalizeFontFamilyKey(mapping.systemFamily ?? ""),
        )
      ) {
        return {
          family,
          documents: [...stylesUsingFont].sort((left, right) =>
            left.localeCompare(right),
          ),
          status: "managed" as const,
          systemFamily: mapping.systemFamily?.trim() ?? "",
        };
      }
      return {
        family,
        documents: [...stylesUsingFont].sort((left, right) =>
          left.localeCompare(right),
        ),
        status: "missing" as const,
        systemFamily: mapping?.systemFamily?.trim() ?? "",
      };
    })
    .sort((left, right) => left.family.localeCompare(right.family));
}

export function formatFontList(fonts: string[], emptyLabel: string) {
  if (fonts.length === 0) {
    return emptyLabel;
  }
  if (fonts.length <= 8) {
    return fonts.join(", ");
  }
  return `${fonts.slice(0, 8).join(", ")} +${fonts.length - 8}`;
}

export function formatRemoteFontSourceNames(
  candidates: RemoteFontSearchCandidate[],
) {
  const names = [
    ...new Set(
      candidates
        .map((candidate) => candidate.sourceName.trim())
        .filter(Boolean),
    ),
  ];
  if (names.length === 0) {
    return "-";
  }
  if (names.length <= 2) {
    return names.join(", ");
  }
  return `${names.slice(0, 2).join(", ")} +${names.length - 2}`;
}

function resolveRemoteFontUnavailableReasonLabel(
  reason: string | undefined,
  language: string,
) {
  switch (reason?.trim()) {
    case "no_download_source":
      return t("library.config.subtitleStyles.remoteFontUnavailableNoDownload", language as "en" | "zh-CN");
    case "unsupported_asset":
      return t("library.config.subtitleStyles.remoteFontUnavailableUnsupported", language as "en" | "zh-CN");
    default:
      return t("library.config.subtitleStyles.remoteFontUnavailableGeneric", language as "en" | "zh-CN");
  }
}

export function formatRemoteFontUnavailableCandidates(
  candidates: RemoteFontSearchCandidate[],
  language: string,
) {
  const details = [
    ...new Set(
      candidates
        .map((candidate) => {
          const sourceName = candidate.sourceName.trim();
          const reason = resolveRemoteFontUnavailableReasonLabel(
            candidate.unavailableReason,
            language,
          );
          return sourceName ? `${sourceName}: ${reason}` : reason;
        })
        .filter(Boolean),
    ),
  ];

  if (details.length === 0) {
    return "-";
  }
  if (details.length <= 2) {
    return details.join(" · ");
  }
  return `${details.slice(0, 2).join(" · ")} +${details.length - 2}`;
}

function createProfileId(prefix: string) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function createEmptyGlossaryProfile(): LibraryGlossaryProfileDTO {
  return {
    id: createProfileId("glossary"),
    name: "",
    category: "all",
    description: "",
    sourceLanguage: "all",
    targetLanguage: "all",
    terms: [{ source: "", target: "", note: "" }],
  };
}

export function createEmptyPromptProfile(): LibraryPromptProfileDTO {
  return {
    id: createProfileId("prompt"),
    name: "",
    category: "all",
    description: "",
    prompt: "",
  };
}

export function parsePositiveInt(raw: string, fallback: number) {
  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
}

export function parseNonNegativeInt(raw: string, fallback: number) {
  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    return fallback;
  }
  return parsed;
}

export function splitAliases(raw: string) {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
