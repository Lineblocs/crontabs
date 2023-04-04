package cmd

import (

	"strconv"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"

	//now "github.com/jinzhu/now"

	helpers "github.com/Lineblocs/go-helpers"
	utils "lineblocs.com/crontabs/utils"
)

// cron tab to remove unset password users
func RetryFailedBillingAttempts() error {
	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}
	billingParams, err := utils.GetBillingParams()
	if err != nil {
		return err
	}
	results, err := db.Query(`SELECT users_invoices.id, users_invoices.workspace_id, workspaces.creator_id, users_invoices.cents 
	INNER JOIN workspaces ON workspaces.id = users_invoices.workspace_id
	FROM users_invoices WHERE users_invoices.status = 'INCOMPLETE'`)
	if err != nil {
		return err
	}
	defer results.Close()
	var invoiceId int 
	var workspaceId int 
	var userId int 
	var cents int 
	for results.Next() {
		err = results.Scan(&invoiceId, &workspaceId, &userId, &cents)
		workspace, err := helpers.GetWorkspaceFromDB(workspaceId)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting workspace ID: "+strconv.Itoa(workspaceId)+"\r\n")
			continue
		}
		user, err := helpers.GetUserFromDB(userId)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting user ID: "+strconv.Itoa(userId)+"\r\n")
			continue
		}
		// try to charge the user again.
		invoiceDesc:="Invoice for service"
		err = utils.ChargeCustomer(db, billingParams, user, workspace, cents, invoiceDesc)
		if err != nil { // failed again
			stmt, err := db.Prepare("UPDATE users_invoices SET status = 'INCOMPLETE', source = 'CARD', last_attempted = ? WHERE id = ?")
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
				continue
			}
			lastAttempted:=""
			_, err = stmt.Exec(lastAttempted, invoiceId)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error updating invoice....\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				continue
			}
			continue
		}
		// mark as paid
		stmt, err := db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CREDITS', cents_collected = ? WHERE id = ?")
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
			continue
		}
		_, err = stmt.Exec(cents, invoiceId)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
	}
	return nil
}
