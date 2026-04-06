package settings

type ChannelsSettings struct {
	config map[string]any
}

type ChannelsSettingsParams struct {
	Config map[string]any
}

func DefaultChannelsConfig() map[string]any {
	return map[string]any{
		"telegram": map[string]any{
			"enabled": false,
		},
		"web": map[string]any{
			"enabled": false,
		},
	}
}

func NewChannelsSettings(params ChannelsSettingsParams) ChannelsSettings {
	return ChannelsSettings{config: cloneChannelsMap(params.Config)}
}

func (channels ChannelsSettings) Config() map[string]any {
	if len(channels.config) == 0 {
		return cloneChannelsMap(DefaultChannelsConfig())
	}
	return cloneChannelsMap(channels.config)
}

func cloneChannelsMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
