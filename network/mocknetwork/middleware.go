// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocknetwork

import (
	datastore "github.com/ipfs/go-datastore"
	flow "github.com/onflow/flow-go/model/flow"

	irrecoverable "github.com/onflow/flow-go/module/irrecoverable"

	mock "github.com/stretchr/testify/mock"

	network "github.com/onflow/flow-go/network"

	protocol "github.com/libp2p/go-libp2p-core/protocol"
)

// Middleware is an autogenerated mock type for the Middleware type
type Middleware struct {
	mock.Mock
}

// Done provides a mock function with given fields:
func (_m *Middleware) Done() <-chan struct{} {
	ret := _m.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// IsConnected provides a mock function with given fields: nodeID
func (_m *Middleware) IsConnected(nodeID flow.Identifier) (bool, error) {
	ret := _m.Called(nodeID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(flow.Identifier) bool); ok {
		r0 = rf(nodeID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(nodeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewBlobService provides a mock function with given fields: channel, store, opts
func (_m *Middleware) NewBlobService(channel network.Channel, store datastore.Batching, opts ...network.BlobServiceOption) network.BlobService {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, channel, store)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 network.BlobService
	if rf, ok := ret.Get(0).(func(network.Channel, datastore.Batching, ...network.BlobServiceOption) network.BlobService); ok {
		r0 = rf(channel, store, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(network.BlobService)
		}
	}

	return r0
}

// NewPingService provides a mock function with given fields: pingProtocol, provider
func (_m *Middleware) NewPingService(pingProtocol protocol.ID, provider network.PingInfoProvider) network.PingService {
	ret := _m.Called(pingProtocol, provider)

	var r0 network.PingService
	if rf, ok := ret.Get(0).(func(protocol.ID, network.PingInfoProvider) network.PingService); ok {
		r0 = rf(pingProtocol, provider)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(network.PingService)
		}
	}

	return r0
}

// Publish provides a mock function with given fields: msg, channel
func (_m *Middleware) Publish(msg interface{}, channel network.Channel) error {
	ret := _m.Called(msg, channel)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}, network.Channel) error); ok {
		r0 = rf(msg, channel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Ready provides a mock function with given fields:
func (_m *Middleware) Ready() <-chan struct{} {
	ret := _m.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// SendDirect provides a mock function with given fields: channel, msg, targetID
func (_m *Middleware) SendDirect(channel network.Channel, msg interface{}, targetID flow.Identifier) error {
	ret := _m.Called(channel, msg, targetID)

	var r0 error
	if rf, ok := ret.Get(0).(func(network.Channel, interface{}, flow.Identifier) error); ok {
		r0 = rf(channel, msg, targetID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetDirectMessageHandler provides a mock function with given fields: channel, handler
func (_m *Middleware) SetDirectMessageHandler(channel network.Channel, handler network.DirectMessageHandler) {
	_m.Called(channel, handler)
}

// SetOverlay provides a mock function with given fields: _a0
func (_m *Middleware) SetOverlay(_a0 network.Overlay) {
	_m.Called(_a0)
}

// Start provides a mock function with given fields: _a0
func (_m *Middleware) Start(_a0 irrecoverable.SignalerContext) {
	_m.Called(_a0)
}

// Subscribe provides a mock function with given fields: channel
func (_m *Middleware) Subscribe(channel network.Channel) error {
	ret := _m.Called(channel)

	var r0 error
	if rf, ok := ret.Get(0).(func(network.Channel) error); ok {
		r0 = rf(channel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Unsubscribe provides a mock function with given fields: channel
func (_m *Middleware) Unsubscribe(channel network.Channel) error {
	ret := _m.Called(channel)

	var r0 error
	if rf, ok := ret.Get(0).(func(network.Channel) error); ok {
		r0 = rf(channel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateNodeAddresses provides a mock function with given fields:
func (_m *Middleware) UpdateNodeAddresses() {
	_m.Called()
}
