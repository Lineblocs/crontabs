package billing

import (
	_ "github.com/go-sql-driver/mysql"

	helpers "github.com/Lineblocs/go-helpers"
)

type BillingHandler interface {
	ChargeCustomer(user *helpers.User, workspace *helpers.Workspace, cents int, desc string) error
}

type Billing struct {
	RetryAttempts int
}
