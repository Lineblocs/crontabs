package billing

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/charge"
	models "lineblocs.com/crontabs/models"

	"database/sql"

	helpers "github.com/Lineblocs/go-helpers"
)

type StripeBillingHandler struct {
	Billing
	RetryAttempts int
	StripeKey     string
	DbConn *sql.DB
}

func NewStripeBillingHandler(dbConn *sql.DB, stripeKey string, retryAttempts int) *StripeBillingHandler {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := StripeBillingHandler{
		DbConn: dbConn,
		StripeKey:     stripeKey,
		RetryAttempts: retryAttempts}
	return &item
}

func (hndl *StripeBillingHandler) ChargeCustomer(user *helpers.User, workspace *helpers.Workspace, invoice *models.UserInvoice) error {
	db := hndl.DbConn
	stripe.Key = hndl.StripeKey

	var id int
	var tokenId string
	row := db.QueryRow(`SELECT id, stripe_id FROM users_cards WHERE workspace_id=? AND primary =1`, workspace.Id)

	err := row.Scan(&id, &tokenId)
	if err != nil {
		return err
	}

	// `source` is obtained with Stripe.js; see https://stripe.com/docs/payments/accept-a-payment-charges#web-create-token
	params := &stripe.ChargeParams{Amount: stripe.Int64(int64(invoice.Cents)),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String(invoice.InvoiceDesc),
		Source:      &stripe.SourceParams{Token: stripe.String(tokenId)}}
	_, err = charge.New(params)
	if err != nil {
		return err
	}
	return nil
}
