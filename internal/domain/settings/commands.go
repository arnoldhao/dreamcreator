package settings

type CommandsSettings struct {
	flags map[string]bool
}

type CommandsSettingsParams struct {
	Flags map[string]bool
}

func NewCommandsSettings(params CommandsSettingsParams) CommandsSettings {
	return CommandsSettings{flags: cloneBoolMap(params.Flags)}
}

func (settings CommandsSettings) Flags() map[string]bool {
	return cloneBoolMap(settings.flags)
}

func (settings CommandsSettings) Enabled(key string) bool {
	if key == "" {
		return true
	}
	if settings.flags == nil {
		return true
	}
	enabled, ok := settings.flags[key]
	if !ok {
		return true
	}
	return enabled
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
