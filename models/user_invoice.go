package models

type UserInvoice struct {
	Id                 int    `json:"id"`
	InvoiceDesc        string `json:"invoice_desc"`
	Cents              int    `json:"cents"`
	ConfirmationNumber int    `json:"confirmation_number"`
}
