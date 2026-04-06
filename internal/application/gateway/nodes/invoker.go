package nodes

import (
	"context"
	"errors"
	"strings"

	nodedevice "dreamcreator/internal/application/gateway/nodes/device"
)

type InvokerFunc func(ctx context.Context, request NodeInvokeRequest) (NodeInvokeResult, error)

func (fn InvokerFunc) Invoke(ctx context.Context, request NodeInvokeRequest) (NodeInvokeResult, error) {
	if fn == nil {
		return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: "node invoker unavailable"}, errors.New("node invoker unavailable")
	}
	return fn(ctx, request)
}

type DeviceInvoker struct {
	host *nodedevice.Host
}

func NewDeviceInvoker(host *nodedevice.Host) *DeviceInvoker {
	if host == nil {
		host = nodedevice.NewHost()
	}
	return &DeviceInvoker{host: host}
}

func (invoker *DeviceInvoker) Invoke(ctx context.Context, request NodeInvokeRequest) (NodeInvokeResult, error) {
	if invoker == nil || invoker.host == nil {
		return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: "node invoker unavailable"}, errors.New("node invoker unavailable")
	}
	envelope := nodedevice.CapabilityInvokeEnvelope{
		Capability: strings.TrimSpace(request.Capability),
		Action:     strings.TrimSpace(request.Action),
		Args:       strings.TrimSpace(request.Args),
	}
	result, err := invoker.host.Invoke(ctx, envelope)
	response := NodeInvokeResult{
		InvokeID: strings.TrimSpace(request.InvokeID),
		Ok:       result.OK,
		Output:   strings.TrimSpace(result.Output),
		Error:    strings.TrimSpace(result.Error),
	}
	if err != nil {
		response.Ok = false
		if response.Error == "" {
			response.Error = err.Error()
		}
	}
	return response, err
}
