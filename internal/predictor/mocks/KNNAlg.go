// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	predictor "github.com/go-sod/sod/internal/predictor"
	mock "github.com/stretchr/testify/mock"
)

// KNNAlg is an autogenerated mock type for the KNNAlg type
type KNNAlg struct {
	mock.Mock
}

// Append provides a mock function with given fields: data
func (_m *KNNAlg) Append(data ...predictor.DataPoint) {
	_va := make([]interface{}, len(data))
	for _i := range data {
		_va[_i] = data[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// Build provides a mock function with given fields: data
func (_m *KNNAlg) Build(data ...predictor.DataPoint) {
	_va := make([]interface{}, len(data))
	for _i := range data {
		_va[_i] = data[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// KNN provides a mock function with given fields: vec, k
func (_m *KNNAlg) KNN(vec predictor.Point, k int) ([]predictor.Point, error) {
	ret := _m.Called(vec, k)

	var r0 []predictor.Point
	if rf, ok := ret.Get(0).(func(predictor.Point, int) []predictor.Point); ok {
		r0 = rf(vec, k)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]predictor.Point)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(predictor.Point, int) error); ok {
		r1 = rf(vec, k)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Len provides a mock function with given fields:
func (_m *KNNAlg) Len() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// Reset provides a mock function with given fields:
func (_m *KNNAlg) Reset() {
	_m.Called()
}
