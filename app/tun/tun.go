//go:build !confonly
// +build !confonly

package tun

import (
	"context"

	core "github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/app/tun/device"
	"github.com/v2fly/v2ray-core/v4/app/tun/device/tun"
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/features/policy"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

type TUN struct {
	ctx           context.Context
	dispatcher    routing.Dispatcher
	policyManager policy.Manager
	config        *Config

	stack *stack.Stack
}

func (t *TUN) Type() interface{} {
	return (*TUN)(nil)
}

func (t *TUN) Start() error {
	var newDeviceFunc device.NewTUNFunc
	newDeviceFunc = tun.New
	device, err := newDeviceFunc(device.Options{
		Name: t.config.Name,
		MTU:  t.config.Mtu,
	})
	if err != nil {
		return newError("failed to create device").Base(err).AtError()
	}

	stack, err := CreateStack(device)
	if err != nil {
		return newError("failed to create stack").Base(err).AtError()
	}
	t.stack = stack

	tcpHandler := &TCPHandler{
		ctx:           t.ctx,
		dispatcher:    t.dispatcher,
		policyManager: t.policyManager,
		config:        t.config,
		stack:         stack,
	}

	tcpHandler.SetHandler()

	return nil
}

func (t *TUN) Close() error {
	if t.stack != nil {
		t.stack.Close()
		t.stack.Wait()
	}
	return nil
}

func NewTUN(ctx context.Context, config *Config, dispatcher routing.Dispatcher) *TUN {
	v := core.MustFromContext(ctx)
	return &TUN{
		ctx:           ctx,
		dispatcher:    dispatcher,
		config:        config,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		tun := core.RequireFeatures(ctx, func(d routing.Dispatcher) *TUN {
			return NewTUN(ctx, config.(*Config), d)
		})

		return tun, nil
	}))
}
