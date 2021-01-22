// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// Seals is an autogenerated mock type for the Seals type
type Seals struct {
	mock.Mock
}

// ByBlockID provides a mock function with given fields: sealedID
func (_m *Seals) ByBlockID(sealedID flow.Identifier) (*flow.Seal, error) {
	ret := _m.Called(sealedID)

	var r0 *flow.Seal
	if rf, ok := ret.Get(0).(func(flow.Identifier) *flow.Seal); ok {
		r0 = rf(sealedID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.Seal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(sealedID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ByID provides a mock function with given fields: sealID
func (_m *Seals) ByID(sealID flow.Identifier) (*flow.Seal, error) {
	ret := _m.Called(sealID)

	var r0 *flow.Seal
	if rf, ok := ret.Get(0).(func(flow.Identifier) *flow.Seal); ok {
		r0 = rf(sealID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.Seal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(sealID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Store provides a mock function with given fields: seal
func (_m *Seals) Store(seal *flow.Seal) error {
	ret := _m.Called(seal)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.Seal) error); ok {
		r0 = rf(seal)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
