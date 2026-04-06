import type {
  LibraryModuleConfigDTO,
  LibraryPromptProfileDTO,
  LibraryTranslateLanguageDTO,
} from "@/shared/contracts/library"
import { t } from "@/shared/i18n"

import type { WorkspaceConstraintOption, WorkspaceSelectOption } from "./types"

type LanguageAssets = NonNullable<LibraryModuleConfigDTO["languageAssets"]>
type GlossaryTaskScope = "translate" | "proofread"

const EMPTY_LANGUAGE_ASSETS: LanguageAssets = {
  glossaryProfiles: [],
  promptProfiles: [],
}

export function resolveTranslateLanguageOptions(config?: LibraryModuleConfigDTO): WorkspaceSelectOption[] {
  const builtin = config?.translateLanguages?.builtin ?? []
  const custom = config?.translateLanguages?.custom ?? []
  const combined = [...builtin, ...custom]
  const seen = new Set<string>()

  const options = combined.flatMap((language) => {
    const value = language.code?.trim() ?? ""
    if (!value || seen.has(value)) {
      return []
    }
    seen.add(value)
    return [
      {
        value,
        label: `${language.label || value.toUpperCase()} (${value.toUpperCase()})`,
      },
    ]
  })

  return options.length > 0
    ? options
    : [
        { value: "en", label: t("library.workspace.languages.english") },
        { value: "zh", label: t("library.workspace.languages.chinese") },
        { value: "ja", label: t("library.workspace.languages.japanese") },
      ]
}

export function resolveGlossaryConstraintOptions(
  config: LibraryModuleConfigDTO | undefined,
  scope: GlossaryTaskScope,
  sourceLanguage?: string,
  targetLanguage?: string,
): WorkspaceConstraintOption[] {
  const languages = resolveLanguageCatalog(config)
  return resolveLanguageAssets(config).glossaryProfiles
    .filter((profile) => matchesGlossaryScope(profile.category, scope))
    .filter((profile) => matchesGlossaryLanguageScope(profile.sourceLanguage, sourceLanguage))
    .filter((profile) => matchesGlossaryLanguageScope(profile.targetLanguage, targetLanguage))
    .map((profile) => {
      const termCount = profile.terms?.length ?? 0
      return {
        value: profile.id,
        label: profile.name || profile.id,
        badge: describeLanguagePair(profile.sourceLanguage, profile.targetLanguage, languages) || undefined,
        description:
          profile.description?.trim() ||
          (termCount === 1
            ? t("library.workspace.languageAssets.termSingular")
            : t("library.workspace.languageAssets.termPlural").replace("{count}", String(termCount))),
      }
    })
}

export function resolveTranslatePromptConstraintOptions(config?: LibraryModuleConfigDTO): WorkspaceConstraintOption[] {
  return resolveLanguageAssets(config).promptProfiles
    .filter((profile) => {
      const category = normalizePromptCategory(profile.category)
      return category === "all" || category === "translate" || category === "glossary"
    })
    .map(toPromptConstraintOption)
}

export function resolveProofreadPromptConstraintOptions(config?: LibraryModuleConfigDTO): WorkspaceConstraintOption[] {
  return resolveLanguageAssets(config).promptProfiles
    .filter((profile) => {
      const category = normalizePromptCategory(profile.category)
      return category === "all" || category === "proofread"
    })
    .map(toPromptConstraintOption)
}

function resolveLanguageAssets(config?: LibraryModuleConfigDTO): LanguageAssets {
  return config?.languageAssets ?? EMPTY_LANGUAGE_ASSETS
}

function resolveLanguageCatalog(config?: LibraryModuleConfigDTO): LibraryTranslateLanguageDTO[] {
  const builtin = config?.translateLanguages?.builtin ?? []
  const custom = config?.translateLanguages?.custom ?? []
  return [...builtin, ...custom]
}

function toPromptConstraintOption(profile: LibraryPromptProfileDTO): WorkspaceConstraintOption {
  return {
    value: profile.id,
    label: profile.name || profile.id,
    badge: resolvePromptCategoryBadgeLabel(profile.category),
    description: profile.description?.trim() || compactPrompt(profile.prompt),
  }
}

function describeLanguagePair(
  sourceLanguage: string | undefined,
  targetLanguage: string | undefined,
  languages: LibraryTranslateLanguageDTO[],
) {
  const sourceLabel = resolveLanguageLabel(sourceLanguage ?? "", languages)
  const targetLabel = resolveLanguageLabel(targetLanguage ?? "", languages)
  if (!sourceLabel && !targetLabel) {
    return ""
  }
  return `${sourceLabel || t("library.config.languageAssets.anyLanguage")} -> ${
    targetLabel || t("library.config.languageAssets.anyLanguage")
  }`
}

function resolveLanguageLabel(value: string, languages: LibraryTranslateLanguageDTO[]) {
  const normalized = value.trim().toLowerCase()
  if (!normalized || normalized === "all") {
    return ""
  }
  const matched = languages.find((item) => item.code.trim().toLowerCase() === normalized)
  if (!matched) {
    return value.trim().toUpperCase()
  }
  return matched.label
}

function compactPrompt(value: string) {
  const collapsed = value.replace(/\s+/g, " ").trim()
  if (!collapsed) {
    return ""
  }
  return collapsed.length > 100 ? `${collapsed.slice(0, 97)}...` : collapsed
}

function normalizePromptCategory(value: string | undefined) {
  const normalized = value?.trim().toLowerCase() ?? ""
  switch (normalized) {
    case "all":
    case "translate":
    case "proofread":
    case "glossary":
      return normalized
    default:
      return "all"
  }
}

function normalizeGlossaryCategory(value: string | undefined) {
  const normalized = value?.trim().toLowerCase() ?? ""
  switch (normalized) {
    case "all":
    case "translate":
    case "proofread":
      return normalized
    default:
      return "all"
  }
}

function normalizeGlossaryLanguageScope(value: string | undefined) {
  const normalized = value?.trim().toLowerCase() ?? ""
  if (!normalized || normalized === "all") {
    return "all"
  }
  return normalized
}

function matchesGlossaryScope(value: string | undefined, scope: GlossaryTaskScope) {
  const category = normalizeGlossaryCategory(value)
  if (scope === "translate") {
    return category === "all" || category === "translate"
  }
  return category === "all" || category === "proofread"
}

function matchesGlossaryLanguageScope(profileValue: string | undefined, taskValue: string | undefined) {
  const profileScope = normalizeGlossaryLanguageScope(profileValue)
  const taskScope = normalizeGlossaryLanguageScope(taskValue)
  if (profileScope === "all" || taskScope === "all") {
    return true
  }
  return profileScope === taskScope
}

function resolvePromptCategoryBadgeLabel(value: string | undefined) {
  switch (normalizePromptCategory(value)) {
    case "proofread":
      return t("library.config.languageAssets.promptProofread")
    case "glossary":
      return t("library.config.languageAssets.promptGlossary")
    case "translate":
      return t("library.config.languageAssets.promptTranslate")
    case "all":
    default:
      return t("library.config.languageAssets.promptAll")
  }
}
