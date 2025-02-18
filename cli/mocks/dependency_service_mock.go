package mocks

import "github.com/stretchr/testify/mock"

type MockDependencyService struct {
	mock.Mock
}

func (_m *MockDependencyService) VerifyDependencies() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
