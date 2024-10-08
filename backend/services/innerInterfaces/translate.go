package innerinterfaces

type TranslateServiceInterface interface {
	AddTranslation(id, originalSubId, lang string) (err error)
}
