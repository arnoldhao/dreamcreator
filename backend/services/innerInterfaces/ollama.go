package innerinterfaces

type OllamaServiceInterface interface {
	Pull(key string, model string) (err error)
}
