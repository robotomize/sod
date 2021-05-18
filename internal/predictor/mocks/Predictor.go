// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	predictor "github.com/go-sod/sod/internal/predictor"
	mock "github.com/stretchr/testify/mock"
)

// Predictor is an autogenerated mock type for the Predictor type
type Predictor struct {
	mock.Mock
}

// Append provides a mock function with given fields: data
func (_m *Predictor) Append(data ...predictor.DataPoint) {
	_va := make([]interface{}, len(data))
	for _i := range data {
		_va[_i] = data[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// Build provides a mock function with given fields: data
func (_m *Predictor) Build(data ...predictor.DataPoint) {
	_va := make([]interface{}, len(data))
	for _i := range data {
		_va[_i] = data[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// Len provides a mock function with given fields:
func (_m *Predictor) Len() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// Predict provides a mock function with given fields: vec
func (_m *Predictor) Predict(vec predictor.Point) (*predictor.Conclusion, error) {
	ret := _m.Called(vec)

	var r0 *predictor.Conclusion
	if rf, ok := ret.Get(0).(func(predictor.Point) *predictor.Conclusion); ok {
		r0 = rf(vec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*predictor.Conclusion)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(predictor.Point) error); ok {
		r1 = rf(vec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Reset provides a mock function with given fields:
func (_m *Predictor) Reset() {
	_m.Called()
}
