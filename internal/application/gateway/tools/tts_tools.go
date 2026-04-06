package tools

import (
	"context"
	"errors"

	gatewayvoice "dreamcreator/internal/application/gateway/voice"
)

func runTTSTool(voice *gatewayvoice.Service) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if voice == nil {
			return "", errors.New("voice service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		text := getStringArg(payload, "text")
		if text == "" {
			return "", errors.New("text is required")
		}
		request := gatewayvoice.TTSConvertRequest{
			Text:       text,
			ProviderID: getStringArg(payload, "providerId", "providerID"),
			VoiceID:    getStringArg(payload, "voiceId", "voiceID"),
			ModelID:    getStringArg(payload, "modelId", "modelID"),
			Format:     getStringArg(payload, "format"),
			Channel:    getStringArg(payload, "channel"),
			RequestID:  getStringArg(payload, "requestId", "requestID"),
		}
		response, err := voice.Convert(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalResult(response), nil
	}
}
