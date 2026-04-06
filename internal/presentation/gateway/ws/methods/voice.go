package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayvoice "dreamcreator/internal/application/gateway/voice"
)

const (
	ScopeTTSStatus  = "tts.status"
	ScopeTTSConfig  = "tts.config.set"
	ScopeTTSConvert = "tts.convert"

	ScopeTalkConfig = "talk.config"
	ScopeTalkSet    = "talk.config.set"
	ScopeTalkMode   = "talk.mode"

	ScopeVoiceWakeGet = "voicewake.get"
	ScopeVoiceWakeSet = "voicewake.set"
)

func RegisterVoice(router *controlplane.Router, voiceService *gatewayvoice.Service) {
	if router == nil || voiceService == nil {
		return
	}
	router.Register("tts.status", []string{ScopeTTSStatus}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		resp, err := voiceService.Status(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("tts.config.set", []string{ScopeTTSConfig}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.TTSConfig
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid tts.config.set params")
		}
		resp, err := voiceService.SetTTSConfig(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("tts.convert", []string{ScopeTTSConvert}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.TTSConvertRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid tts.convert params")
		}
		resp, err := voiceService.Convert(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("talk.config", []string{ScopeTalkConfig}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.TalkConfigRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid talk.config params")
			}
		}
		resp, err := voiceService.TalkConfig(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("talk.config.set", []string{ScopeTalkSet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.TalkConfigSetRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid talk.config.set params")
		}
		resp, err := voiceService.SetTalkConfig(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("talk.mode", []string{ScopeTalkMode}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.TalkModeRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid talk.mode params")
		}
		return voiceService.TalkMode(payload), nil
	})
	router.Register("voicewake.get", []string{ScopeVoiceWakeGet}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		resp, err := voiceService.VoiceWakeGet(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("voicewake.set", []string{ScopeVoiceWakeSet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayvoice.VoiceWakeSetRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid voicewake.set params")
		}
		resp, err := voiceService.VoiceWakeSet(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
}
