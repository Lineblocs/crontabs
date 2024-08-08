// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	lineblocs "github.com/Lineblocs/go-helpers"
	mock "github.com/stretchr/testify/mock"

	models "lineblocs.com/crontabs/models"

	utils "lineblocs.com/crontabs/utils"
)

// PaymentRepository is an autogenerated mock type for the PaymentRepository type
type PaymentRepository struct {
	mock.Mock
}

type PaymentRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *PaymentRepository) EXPECT() *PaymentRepository_Expecter {
	return &PaymentRepository_Expecter{mock: &_m.Mock}
}

// ChargeCustomer provides a mock function with given fields: billingParams, user, workspace, invoice
func (_m *PaymentRepository) ChargeCustomer(billingParams *utils.BillingParams, user *lineblocs.User, workspace *lineblocs.Workspace, invoice *models.UserInvoice) error {
	ret := _m.Called(billingParams, user, workspace, invoice)

	if len(ret) == 0 {
		panic("no return value specified for ChargeCustomer")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*utils.BillingParams, *lineblocs.User, *lineblocs.Workspace, *models.UserInvoice) error); ok {
		r0 = rf(billingParams, user, workspace, invoice)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PaymentRepository_ChargeCustomer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ChargeCustomer'
type PaymentRepository_ChargeCustomer_Call struct {
	*mock.Call
}

// ChargeCustomer is a helper method to define mock.On call
//   - billingParams *utils.BillingParams
//   - user *lineblocs.User
//   - workspace *lineblocs.Workspace
//   - invoice *models.UserInvoice
func (_e *PaymentRepository_Expecter) ChargeCustomer(billingParams interface{}, user interface{}, workspace interface{}, invoice interface{}) *PaymentRepository_ChargeCustomer_Call {
	return &PaymentRepository_ChargeCustomer_Call{Call: _e.mock.On("ChargeCustomer", billingParams, user, workspace, invoice)}
}

func (_c *PaymentRepository_ChargeCustomer_Call) Run(run func(billingParams *utils.BillingParams, user *lineblocs.User, workspace *lineblocs.Workspace, invoice *models.UserInvoice)) *PaymentRepository_ChargeCustomer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*utils.BillingParams), args[1].(*lineblocs.User), args[2].(*lineblocs.Workspace), args[3].(*models.UserInvoice))
	})
	return _c
}

func (_c *PaymentRepository_ChargeCustomer_Call) Return(_a0 error) *PaymentRepository_ChargeCustomer_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *PaymentRepository_ChargeCustomer_Call) RunAndReturn(run func(*utils.BillingParams, *lineblocs.User, *lineblocs.Workspace, *models.UserInvoice) error) *PaymentRepository_ChargeCustomer_Call {
	_c.Call.Return(run)
	return _c
}

// GetServicePlans provides a mock function with given fields:
func (_m *PaymentRepository) GetServicePlans() ([]lineblocs.ServicePlan, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetServicePlans")
	}

	var r0 []lineblocs.ServicePlan
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]lineblocs.ServicePlan, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []lineblocs.ServicePlan); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]lineblocs.ServicePlan)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PaymentRepository_GetServicePlans_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetServicePlans'
type PaymentRepository_GetServicePlans_Call struct {
	*mock.Call
}

// GetServicePlans is a helper method to define mock.On call
func (_e *PaymentRepository_Expecter) GetServicePlans() *PaymentRepository_GetServicePlans_Call {
	return &PaymentRepository_GetServicePlans_Call{Call: _e.mock.On("GetServicePlans")}
}

func (_c *PaymentRepository_GetServicePlans_Call) Run(run func()) *PaymentRepository_GetServicePlans_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *PaymentRepository_GetServicePlans_Call) Return(_a0 []lineblocs.ServicePlan, _a1 error) *PaymentRepository_GetServicePlans_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PaymentRepository_GetServicePlans_Call) RunAndReturn(run func() ([]lineblocs.ServicePlan, error)) *PaymentRepository_GetServicePlans_Call {
	_c.Call.Return(run)
	return _c
}

// NewPaymentRepository creates a new instance of PaymentRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPaymentRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *PaymentRepository {
	mock := &PaymentRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
