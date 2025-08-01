// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	mock "github.com/stretchr/testify/mock"
)

// NewProjectGetter creates a new instance of ProjectGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProjectGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProjectGetter {
	mock := &ProjectGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// ProjectGetter is an autogenerated mock type for the ProjectGetter type
type ProjectGetter struct {
	mock.Mock
}

type ProjectGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *ProjectGetter) EXPECT() *ProjectGetter_Expecter {
	return &ProjectGetter_Expecter{mock: &_m.Mock}
}

// Get provides a mock function for the type ProjectGetter
func (_mock *ProjectGetter) Get(name string) (*v1alpha1.AppProject, error) {
	ret := _mock.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *v1alpha1.AppProject
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(string) (*v1alpha1.AppProject, error)); ok {
		return returnFunc(name)
	}
	if returnFunc, ok := ret.Get(0).(func(string) *v1alpha1.AppProject); ok {
		r0 = returnFunc(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.AppProject)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(string) error); ok {
		r1 = returnFunc(name)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// ProjectGetter_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type ProjectGetter_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - name string
func (_e *ProjectGetter_Expecter) Get(name interface{}) *ProjectGetter_Get_Call {
	return &ProjectGetter_Get_Call{Call: _e.mock.On("Get", name)}
}

func (_c *ProjectGetter_Get_Call) Run(run func(name string)) *ProjectGetter_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *ProjectGetter_Get_Call) Return(appProject *v1alpha1.AppProject, err error) *ProjectGetter_Get_Call {
	_c.Call.Return(appProject, err)
	return _c
}

func (_c *ProjectGetter_Get_Call) RunAndReturn(run func(name string) (*v1alpha1.AppProject, error)) *ProjectGetter_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetClusters provides a mock function for the type ProjectGetter
func (_mock *ProjectGetter) GetClusters(project string) ([]*v1alpha1.Cluster, error) {
	ret := _mock.Called(project)

	if len(ret) == 0 {
		panic("no return value specified for GetClusters")
	}

	var r0 []*v1alpha1.Cluster
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(string) ([]*v1alpha1.Cluster, error)); ok {
		return returnFunc(project)
	}
	if returnFunc, ok := ret.Get(0).(func(string) []*v1alpha1.Cluster); ok {
		r0 = returnFunc(project)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1alpha1.Cluster)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(string) error); ok {
		r1 = returnFunc(project)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// ProjectGetter_GetClusters_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClusters'
type ProjectGetter_GetClusters_Call struct {
	*mock.Call
}

// GetClusters is a helper method to define mock.On call
//   - project string
func (_e *ProjectGetter_Expecter) GetClusters(project interface{}) *ProjectGetter_GetClusters_Call {
	return &ProjectGetter_GetClusters_Call{Call: _e.mock.On("GetClusters", project)}
}

func (_c *ProjectGetter_GetClusters_Call) Run(run func(project string)) *ProjectGetter_GetClusters_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *ProjectGetter_GetClusters_Call) Return(clusters []*v1alpha1.Cluster, err error) *ProjectGetter_GetClusters_Call {
	_c.Call.Return(clusters, err)
	return _c
}

func (_c *ProjectGetter_GetClusters_Call) RunAndReturn(run func(project string) ([]*v1alpha1.Cluster, error)) *ProjectGetter_GetClusters_Call {
	_c.Call.Return(run)
	return _c
}
