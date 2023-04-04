package billing

import (
	_ "github.com/go-sql-driver/mysql"

	"errors"
	"database/sql"
	helpers "github.com/Lineblocs/go-helpers"
)

type BraintreeBillingHandler struct {
	Billing
	RetryAttempts int
	BraintreeKey  string
	DbConn *sql.DB
}

func NewBraintreeBillingHandler(dbConn *sql.DB, BraintreeKey string, retryAttempts int) *BraintreeBillingHandler {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := BraintreeBillingHandler{
		DbConn: dbConn,
		BraintreeKey:  BraintreeKey,
		RetryAttempts: retryAttempts}
	return &item
}

func (hndl *BraintreeBillingHandler) ChargeCustomer(user *helpers.User, workspace *helpers.Workspace, cents int, desc string) error {
	//_ := hndl.DbConn
	// todo: implement handler
	return errors.New("not implemented yet")
}
