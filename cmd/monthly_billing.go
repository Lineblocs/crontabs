package cmd

import (
	"fmt"
	"time"

	"database/sql"
	"math"
	"strconv"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"

	//now "github.com/jinzhu/now"

	helpers "github.com/Lineblocs/go-helpers"
	billing "lineblocs.com/crontabs/handlers/billing"
	utils "lineblocs.com/crontabs/utils"
)


func chargeCustomer(billingParams *utils.BillingParams, user *helpers.User, workspace *helpers.Workspace, cents int, invoiceDesc string) (error) {
	var hndl billing.BillingHandler
	retryAttempts, err := strconv.Atoi(billingParams.Data["retry_attempts"])
	if err != nil {
		//retry attempts issue
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("variable retryAttempts is setup incorrectly. Please ensure that it is set to an integer. retryAttempts=%s setting value to 0", billingParams.Data["retry_attempts"]))
		retryAttempts = 0
	}

	switch billingParams.Provider {
	case "stripe":
		key := billingParams.Data["stripe_key"]
		hndl = billing.NewStripeBillingHandler(key, retryAttempts)
		err = hndl.ChargeCustomer(user, workspace, cents, invoiceDesc)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "braintree":
		key := billingParams.Data["braintree_api_key"]
		hndl = billing.NewBraintreeBillingHandler(key, retryAttempts)
		err = hndl.ChargeCustomer(user, workspace, cents, invoiceDesc)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	}
	return err
}

func computeAmountToCharge(fullCentsToCharge float64, usedMinutes float64, minutes float64) (float64, error) {
	minAfterDebit := usedMinutes - minutes
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge full: %f, used minutes %f, minutes %f, minAfterDebit: %f\r\n", fullCentsToCharge, usedMinutes, minutes, minAfterDebit))
	//when total goes below 0, only charge the amount that went below 0
	// ensure usedMinutes < minutes
	if usedMinutes > 0 && minAfterDebit < 0 && usedMinutes <= minutes {
		percentOfDebit, err := strconv.ParseFloat( fmt.Sprintf(".%s", strconv.FormatFloat((minutes - usedMinutes), 'f', -1, 64)), 8)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("computeAmountToCharge could not parse float %s", err.Error()))
			return 0, err
		}
		
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge percentage = %f, rounded = %f", percentOfDebit, math.Round(percentOfDebit)))
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge debit = %f", percentOfDebit))
		centsToCharge := math.Abs( float64(fullCentsToCharge) * percentOfDebit )
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", centsToCharge))
		return math.Max(1,centsToCharge), nil
	} else if usedMinutes >= minutes { // user has enough balance, no need to charge
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", 0.0))
		return 0, nil
	} else if usedMinutes <= 0 {
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", fullCentsToCharge))
		return fullCentsToCharge, nil
	}
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", 0.0))
	return 0, nil
}

// cron tab to remove unset password users
func MonthlyBilling() error {
	var id int
	var creatorId int

	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}
	billingParams, err := utils.GetBillingParams()
	if err != nil {
		return err
	}
	start := time.Now()
	start = start.AddDate(0, -1, 0)
	end := time.Now()
	currentTime := time.Now()
	startFormatted := start.Format("2006-01-02 15:04:05")
	endFormatted := end.Format("2006-01-02 15:04:05")
	results, err := db.Query("SELECT id, creator_id FROM workspaces")
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
		helpers.Log(logrus.ErrorLevel, err.Error())
		return err
	}
	plans, err := helpers.GetServicePlans()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error getting service plans\r\n")
		helpers.Log(logrus.ErrorLevel, err.Error())
		return err
	}

	defer results.Close()
	for results.Next() {
		err = results.Scan(&id, &creatorId)
		workspace, err := helpers.GetWorkspaceFromDB(id)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting workspace ID: "+strconv.Itoa(id)+"\r\n")
			continue
		}
		user, err := helpers.GetUserFromDB(creatorId)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting user ID: "+strconv.Itoa(id)+"\r\n")
			continue
		}

		var plan *helpers.ServicePlan
		for _, target := range plans {
			if target.Name == workspace.Plan {
				plan = &target
				break
			}
		}
		if plan == nil {
			helpers.Log(logrus.InfoLevel, "No plan found for user..\r\n")
			continue
		}
		billingInfo, err := helpers.GetWorkspaceBillingInfo(workspace)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "Could not get billing info..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}

		var didId int
		var monthlyCosts int
		results1, err := db.Query("SELECT id, monthly_cost  FROM did_numbers WHERE workspace_id = ?", workspace.Id)
		if err != sql.ErrNoRows && err != nil {
			helpers.Log(logrus.ErrorLevel, "Could not get dids info..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		defer results1.Close()
		for results1.Next() {
			results1.Scan(&didId, &monthlyCosts)
			stmt, err := db.Prepare("INSERT INTO users_debits (`source`, `status`, `cents`, `module_id`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?)")
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				continue
			}

			defer stmt.Close()
			_, err = stmt.Exec("NUMBER_RENTAL", "INCOMPLETE", monthlyCosts, didId, user.Id, workspace.Id, start)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error creating number rental debit..\r\n")
				continue
			}

		}

		baseCosts, err := helpers.GetBaseCosts()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting base costs..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}

		// get the amount of users in this workspace
		rows, err := db.Query("SELECT COUNT(*) as count FROM  workspaces_users WHERE workspace_id = ?", workspace.Id)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting workspace user count.\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		rows.Close()

		userCount, err := utils.CheckRowCount(rows)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error getting workspace user count.\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("Workspace total user count %d", userCount))

		totalCosts := 0.0
		membershipCosts := plan.BaseCosts * float64(userCount)
		callTolls := 0.0
		recordingCosts := 0.0
		faxCosts := 0.0
		monthlyNumberRentals := 0.0
		invoiceDesc := fmt.Sprintf("LineBlocs invoice for %s", billingInfo.InvoiceDue)

		helpers.Log(logrus.InfoLevel, fmt.Sprintf("Workspace total membership costs is %f", membershipCosts))

		results2, err := db.Query("SELECT id, source, module_id, cents, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			return err
		}
		defer results2.Close()
		var id int
		var source string
		var moduleId int
		var cents float64
		var created time.Time
		usedMonthlyMinutes := plan.MinutesPerMonth
		usedMonthlyRecordings := plan.RecordingSpace
		usedMonthlyFax := plan.Fax
		for results2.Next() {
			results2.Scan(&id, &source, &moduleId, &cents, &created)
			helpers.Log(logrus.InfoLevel, fmt.Sprintf("scanning in debit source %s\r\n", source))
			switch source {
			case "CALL":
				helpers.Log(logrus.InfoLevel, fmt.Sprintf("getting call %d\r\n", moduleId))
				call, err := helpers.GetCallFromDB(moduleId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					return err
				}
				duration := call.DurationNumber
				helpers.Log(logrus.InfoLevel, fmt.Sprintf("call duration is %d\r\n", duration))
				minutes := float64(duration / 60)
				charge, err := computeAmountToCharge(cents, usedMonthlyMinutes, minutes)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error getting charge..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}
				callTolls = callTolls + charge
				usedMonthlyMinutes = usedMonthlyMinutes - minutes

			case "NUMBER_RENTAL":
				helpers.Log(logrus.InfoLevel, fmt.Sprintf("getting DID %d\r\n", moduleId))
				did, err := helpers.GetDIDFromDB(moduleId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}

				monthlyNumberRentals += float64(did.MonthlyCost)
			}
		}
		results3, err := db.Query("SELECT id, size, created_at FROM recordings WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted)
		if err != sql.ErrNoRows && err != nil {
			helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			return err
		}
		defer results3.Close()
		var recId int
		var size float64
		var createdAt time.Time
		for results3.Next() {
			results3.Scan(&recId, &size, &createdAt)
			cents := math.Round(baseCosts.RecordingsPerByte * float64(size))
			charge, err := computeAmountToCharge(cents, usedMonthlyRecordings, size)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error getting charge..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				continue
			}
			recordingCosts += charge
			usedMonthlyRecordings -= size

		}
		results4, err := db.Query("SELECT id, created_at FROM faxes WHERE workspace_id = ? AND created_at BETWEEN ? AND ?", workspace.Id, startFormatted, endFormatted)
		if err != sql.ErrNoRows && err != nil {
			helpers.Log(logrus.ErrorLevel, "error running query..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			return err
		}
		defer results4.Close()
		var faxId int
		for results4.Next() {
			results4.Scan(&faxId, &createdAt)
			totalFax := float64( plan.Fax )
			centsForFax := baseCosts.FaxPerUsed
			charge, err := computeAmountToCharge(centsForFax, float64(usedMonthlyFax), totalFax)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error getting charge..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				continue
			}
			faxCosts += charge
			usedMonthlyFax -= 1

		}
		totalCosts += membershipCosts
		totalCosts += callTolls
		totalCosts += recordingCosts
		totalCosts += faxCosts
		totalCosts += monthlyNumberRentals

		helpers.Log(logrus.InfoLevel, fmt.Sprintf("Final costs are membership: %f, call tolls: %f, recordings: %f, fax: %f, did rentals: %f, total: %f (cents)\r\n",
			membershipCosts,
			callTolls,
			recordingCosts,
			faxCosts,
			monthlyNumberRentals,
			totalCosts))

		helpers.Log(logrus.InfoLevel, fmt.Sprintf("Creating invoice for user %d, on workspace %d, plan type %s\r\n", user.Id, workspace.Id, workspace.Plan))
		stmt, err := db.Prepare("INSERT INTO users_invoices (`cents`, `call_costs`, `recording_costs`, `fax_costs`, `membership_costs`, `number_costs`, `status`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		defer stmt.Close()
		res, err := stmt.Exec(cents, callTolls, recordingCosts, faxCosts, membershipCosts, monthlyNumberRentals, "INCOMPLETE", workspace.CreatorId, workspace.Id, currentTime)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error creating invoice..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		invoiceId, err := res.LastInsertId()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "could not get insert id..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
			continue
		}
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("Charging user %d, on workspace %d, plan type %s\r\n", user.Id, workspace.Id, workspace.Plan))
		// try to charge the debit
		//if workspace.Plan == "pay-as-you-go" {
		if plan.PayAsYouGo {
			remainingBalance := billingInfo.RemainingBalanceCents
			minRemaining := remainingBalance - totalCosts
			charge, err := computeAmountToCharge(totalCosts, remainingBalance, minRemaining)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error getting charge..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())

				continue
			}
			if remainingBalance >= totalCosts { //user has enough credits
				helpers.Log(logrus.InfoLevel, "User has enough credits. Charging balance\r\n")
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CREDITS', cents_collected = ? WHERE id = ?")
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(totalCosts, invoiceId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}
			} else {
				helpers.Log(logrus.InfoLevel, "User does not have enough credits. Charging any payment sources\r\n")
				// update debit to reflect exactly how much we can charge
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'INCOMPLETE', source ='CREDITS', cents_collected = ? WHERE id = ?")
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(charge, invoiceId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}
				// try to charge the rest using a card
				helpers.Log(logrus.InfoLevel, "Charging remainder with card..\r\n")

				cents := int(math.Ceil(charge))

				err = chargeCustomer(billingParams, user, workspace, cents, invoiceDesc)
				if err != nil {
					// could not charge card.
					// update invoice record and mark as outstanding
					stmt, err = db.Prepare("UPDATE users_invoices SET source = 'CARD', status = 'INCOMPLETE' WHERE id = ?")
					if err != nil {
						helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
						continue
					}
					_, err = stmt.Exec(invoiceId)
					if err != nil {
						helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
						helpers.Log(logrus.ErrorLevel, err.Error())
						continue
					}
					continue
				}
				stmt, err = db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ? WHERE id = ?")
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(totalCosts, invoiceId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}

			}
		} else {
			// regular membership charge. only try to charge a card
			helpers.Log(logrus.InfoLevel, "Charging recurringly with card..\r\n")
			cents := int(math.Ceil(totalCosts))
			err := chargeCustomer(billingParams, user, workspace, cents, invoiceDesc)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'INCOMPLETE', source = 'CARD', cents_collected = 0.0 WHERE id = ?")
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(invoiceId)
				if err != nil {
					helpers.Log(logrus.ErrorLevel, "error updating invoice....\r\n")
					helpers.Log(logrus.ErrorLevel, err.Error())
					continue
				}
				// TODO send email when any biliing attempts fail
				continue
			}
			stmt, err := db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ? WHERE id = ?")
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "could not prepare query..\r\n")
				continue
			}
			_, err = stmt.Exec(totalCosts, invoiceId)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error updating debit..\r\n")
				helpers.Log(logrus.ErrorLevel, err.Error())
				continue
			}

		}
	}
	return nil
}
