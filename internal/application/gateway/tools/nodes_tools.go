package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

func runNodesTool(nodes *gatewaynodes.Service) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if nodes == nil {
			return "", errors.New("nodes service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		nodeID := getStringArg(payload, "nodeId", "nodeID")
		capability := getStringArg(payload, "capability", "action")
		if nodeID == "" {
			return "", errors.New("nodeId is required")
		}
		argsJSON := strings.TrimSpace(getStringArg(payload, "args"))
		if argsJSON == "" {
			if payloadMap := getMapArg(payload, "payload"); payloadMap != nil {
				if data, err := json.Marshal(payloadMap); err == nil {
					argsJSON = string(data)
				}
			}
		}
		request := gatewaynodes.NodeInvokeRequest{
			NodeID:     nodeID,
			Capability: capability,
			Action:     getStringArg(payload, "action"),
			Args:       argsJSON,
		}
		result, err := nodes.Invoke(ctx, request)
		if err != nil {
			return marshalResult(result), err
		}
		return marshalResult(result), nil
	}
}
