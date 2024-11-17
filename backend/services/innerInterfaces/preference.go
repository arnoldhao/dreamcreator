package innerinterfaces

type PreferenceServiceInterface interface {
	TestProxy(id string) (err error)
}
