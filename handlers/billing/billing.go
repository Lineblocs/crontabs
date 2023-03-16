package billing
import (
	_ "github.com/go-sql-driver/mysql"

	lineblocs "github.com/Lineblocs/go-helpers"
)

type BillingHandler interface {
	ChargeCustomer(user*lineblocs.User, workspace*lineblocs.Workspace, cents int, desc string) (error) 
}

type Billing struct {
	RetryAttempts int
}
