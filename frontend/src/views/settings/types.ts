export type SettingsSection =
  | "general"
  | "appearance"
  | "storage"
  | "dependencies"
  | "cookies"
  | "providers"
  | "llm_assets"
  | "acknowledgements"
  | "about"

export type LLMAssetsKind = "glossary" | "target_languages" | "profiles"

export type SettingsRoute = {
  section: SettingsSection
  // Providers section subpage: when set, show provider detail page.
  providerId?: string
  // LLM Assets section subpage: when set, show detail page.
  llmAssetsKind?: LLMAssetsKind
  llmAssetsId?: string
}
