// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocknetwork

import mock "github.com/stretchr/testify/mock"

// Decoder is an autogenerated mock type for the Decoder type
type Decoder struct {
	mock.Mock
}

// Decode provides a mock function with given fields:
func (_m *Decoder) Decode() (interface{}, error) {
	ret := _m.Called()

	var r0 interface{}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
