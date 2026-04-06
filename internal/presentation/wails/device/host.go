package device

import (
	"context"

	nodedevice "dreamcreator/internal/application/gateway/nodes/device"
)

type RuntimeHost struct {
	host *nodedevice.Host
}

func NewRuntimeHost() *RuntimeHost {
	return &RuntimeHost{host: nodedevice.NewHost()}
}

func (runtime *RuntimeHost) Hello(ctx context.Context) (nodedevice.DeviceRuntimeHello, error) {
	if runtime == nil || runtime.host == nil {
		return nodedevice.DeviceRuntimeHello{}, nil
	}
	return runtime.host.Hello(ctx), nil
}

func (runtime *RuntimeHost) Invoke(ctx context.Context, envelope nodedevice.CapabilityInvokeEnvelope) (nodedevice.CapabilityResult, error) {
	if runtime == nil || runtime.host == nil {
		return nodedevice.CapabilityResult{OK: false, Error: "runtime host unavailable"}, nil
	}
	return runtime.host.Invoke(ctx, envelope)
}
