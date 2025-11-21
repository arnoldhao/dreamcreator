package types

// GlossaryEntry defines a special term behavior for translation
// - If DoNotTranslate is true, the Source term must be preserved as-is
// - Otherwise, Translations may specify desired target forms keyed by target language code
type GlossaryEntry struct {
    ID             string            `json:"id"`
    SetID          string            `json:"set_id,omitempty"`
    Source         string            `json:"source"`
    DoNotTranslate bool              `json:"do_not_translate"`
    CaseSensitive  bool              `json:"case_sensitive"`
    Translations   map[string]string `json:"translations"` // targetLang -> fixed translation
    Notes          string            `json:"notes,omitempty"`
    CreatedAt      int64             `json:"created_at"`
    UpdatedAt      int64             `json:"updated_at"`
}

// GlossarySet groups entries logically. Users may create multiple global sets
// and select one or more sets during translation.
type GlossarySet struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    CreatedAt   int64  `json:"created_at"`
    UpdatedAt   int64  `json:"updated_at"`
}
