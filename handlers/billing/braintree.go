package billing

import (
	_ "github.com/go-sql-driver/mysql"

	"errors"

	helpers "github.com/Lineblocs/go-helpers"
	utils "lineblocs.com/crontabs/utils"
)

type BraintreeBillingHandler struct {
	Billing
	RetryAttempts int
	BraintreeKey  string
}

func NewBraintreeBillingHandler(BraintreeKey string, retryAttempts int) *BraintreeBillingHandler {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := BraintreeBillingHandler{
		BraintreeKey:  BraintreeKey,
		RetryAttempts: retryAttempts}
	return &item
}

func (hndl *BraintreeBillingHandler) ChargeCustomer(user *helpers.User, workspace *helpers.Workspace, cents int, desc string) error {
	_, err := utils.GetDBConnection()
	if err != nil {
		return err
	}
	// todo: implement handler
	return errors.New("not implemented yet")
}
