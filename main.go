package main

import (
	"fmt"
	"database/sql"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"bytes"
	"strconv"
	"time"
	"os"
	math "math"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	lineblocs "bitbucket.org/infinitet3ch/lineblocs-go-helpers"
	now "github.com/jinzhu/now"
)

var db* sql.DB;


type Email struct {
	EmailType string `json:"email_type"`
	UserId int `json:"user_id"`
	WorkspaceId int `json:"workspace_id"`
	Args map[string]string `json:"args"`
}

// Abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
// call email service to send email
func dispatchEmail(emailType string, user* lineblocs.User, workspace* lineblocs.Workspace, emailArgs map[string]string) (error) {
    url := "http://lineblocs-email/send"
    fmt.Println("URL:>", url)

	email := Email{UserId: user.Id, WorkspaceId: workspace.Id, EmailType: emailType, Args: emailArgs};
	b, err := json.Marshal(email)
	if err != nil {
		fmt.Println("error:", err)
		return err
	}
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
    req.Header.Set("X-Lineblocs-Key", "xxx")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return err
    }
    defer resp.Body.Close()

    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return nil
}
// cron tab to remove unset password users
func runDeleteUnsetPasswordUsers() {
	days := "7"
	var id int 
	results, err := db.Query(`SELECT id FROM ` + "`" + `users` + "`" + ` WHERE needs_set_password_date <= DATE_ADD(NOW(), INTERVAL -? DAY) AND needs_password_set = 1`, days)
    if err != nil {
		return
	}
    defer results.Close()
    for results.Next() {
		results.Scan(&id)
		fmt.Printf("Removing user %d\r\n", id)
		_, err := db.Query(`DELETE FROM ` + "`" + `users` + "`" + ` WHERE id = ?`, id)
		if err != nil {
			fmt.Printf("Could not remove %d\r\n", id)
			continue
		}
	}
}

// cron tab to email users to tell them that their free trial will be ending soon
func runFreeTrialEnding() {
	days := "7"
	var id int 
	var creatorId int 
	results, err := db.Query(`SELECT id, creator_id FROM ` + "`" + `workspaces` + "`" + ` WHERE free_trial_started <= DATE_ADD(NOW(), INTERVAL -? DAY) AND free_trial_reminder_sent = 0`, days)
    if err != nil {
		return
	}
    defer results.Close()
    for results.Next() {
		results.Scan(&id)
		results.Scan(&creatorId)
		user, err := lineblocs.GetUserFromDB(creatorId)
		if err != nil {
			fmt.Printf("could not get user from DB\r\n")
			continue
		}
		workspace, err := lineblocs.GetWorkspaceFromDB(id)
		if err != nil {
			fmt.Printf("could not get workspace from DB\r\n")
			continue
		}
		args := make(map[string]string)
		err = dispatchEmail("free-trial-ending", user, workspace, args)
		if err != nil {
			fmt.Printf("could not send email\r\n")
			fmt.Println(err)
			continue
		}
	}

}

// cron tab to email users to tell them that their free trial will be ending soon
func runMonthlyBilling() {
	var id int 
	var creatorId int 
	start := now.BeginningOfMonth()    // 2013-11-01 00:00:00 Fri
	end := now.EndOfMonth()          // 2013-11-30 23:59:59.999999999 Sat
	currentTime := time.Now()
	startFormatted := start.Format("2006-01-02 15:04:05")
	endFormatted := end.Format("2006-01-02 15:04:05")
 	results, err := db.Query("SELECT id, creator_id FROM workspaces");
    if err != nil {
		fmt.Printf("error running query..\r\n")
		fmt.Println(err)
		return
	}
	plans, err := lineblocs.GetServicePlans()
	if err != nil {
		fmt.Printf("error getting service plans\r\n")
		fmt.Println(err)
		return
	}

	  defer results.Close()
    for results.Next() {
		err = results.Scan(&id, &creatorId)
		workspace, err := lineblocs.GetWorkspaceFromDB(id)
	    if err != nil {
			fmt.Printf("error getting workspace ID: " + strconv.Itoa(id) + "\r\n")
			continue
		}
		user, err := lineblocs.GetUserFromDB(id)
	    if err != nil {
			fmt.Printf("error getting user ID: " + strconv.Itoa(id) + "\r\n")
			continue
		}

	  	var plan *lineblocs.ServicePlan
		for _, aPlan := range plans {
			if aPlan.Name == workspace.Plan {
				plan = &aPlan
				break
			}
		}
		if plan == nil {
			fmt.Printf("No plan found for user..\r\n")
		}
		billingInfo, err := lineblocs.GetWorkspaceBillingInfo(workspace)
		if err != nil {
			fmt.Printf("Could not get billing info..\r\n")
			continue
		}


		var didId int
		var monthlyCosts int
		results1, err := db.Query("SELECT id, monthly_costs  FROM did_numbers WHERE workspace_id = ?", workspace.Id)
		if err != nil {
			fmt.Printf("Could not get dids info..\r\n")
			continue
		}

    	for results.Next() {
			results.Scan(&didId, &monthlyCosts)
			stmt, err := db.Prepare("INSERT INTO users_logs (`type`, `cents`, `module_id`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?)")
			if err != nil {
				fmt.Printf("could not prepare query..\r\n")
				continue
			}

			defer stmt.Close()
			_, err = stmt.Exec("NUMBER_RENTAL", monthlyCosts, didId, user.Id, workspace.Id, currentTime)
			if err != nil {
				fmt.Printf("error creating number rental debit..\r\n")
				continue
			}

		}



		baseCosts, err := lineblocs.GetBaseCosts()
		if err != nil {
			fmt.Printf("error getting base costs..\r\n")
			fmt.Println(err)
			continue
		}

		totalCosts := 0.0
		membershipCosts := 0.0
		callTolls := 0.0
		recordingCosts := 0.0
		faxCosts := 0.0
		monthlyNumberRentals :=  0.0
		invoiceDesc := fmt.Sprintf("LineBlocs invoice for %s", billingInfo.InvoiceDue)
		results1, err = db.Query("SELECT id, source, module_id, cents, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted);
		defer results1.Close()
		if err != nil {
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
			return
		}
		var id int
		var source string
		var moduleId int
		var cents float64
		var created time.Time
		usedMonthlyMinutes := plan.MinutesPerMonth
		usedMonthlyRecordings := plan.RecordingSpace
		usedMonthlyFax := plan.Fax
    	for results.Next() {
			results.Scan(&id, &source, &moduleId, &cents, &created)
			if source == "CALL" {
				call, err := lineblocs.GetCallFromDB(moduleId)
				if err != nil {
					fmt.Printf("error running query..\r\n")
					fmt.Println(err)
					return
				}
				duration := call.DurationNumber
				minutes := duration / 60
				change := usedMonthlyMinutes - minutes;
				charge, err := computeAmountToCharge(cents, usedMonthlyMinutes, float64(change))
				if err != nil {
					fmt.Printf("error getting charge..\r\n")
					fmt.Println(err)
					continue
				}
				callTolls = callTolls + charge
				usedMonthlyMinutes = usedMonthlyMinutes - minutes;

			} else if source == "NUMBER_RENTAL" {
				did, err := lineblocs.GetDIDFromDB(moduleId)
				if err != nil {
					fmt.Printf("error running query..\r\n")
					fmt.Println(err)
					return
				}

				monthlyNumberRentals += float64(did.MonthlyCosts)
			} 
		}
		results2, err := db.Query("SELECT id, size, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted);
		defer results1.Close()
		if err != nil {
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
			return
		}
		var recId int
		var size int
		var createdAt time.Time
		for results2.Next() {
			results2.Scan(&recId, &size, &createdAt)
			change := float64( usedMonthlyRecordings - size )
			cents := math.Round(baseCosts.RecordingsPerByte * float64(size))
			charge, err := computeAmountToCharge(cents, usedMonthlyRecordings, change)
			if err != nil {
				fmt.Printf("error getting charge..\r\n")
				fmt.Println(err)
				continue
			}
			recordingCosts += charge 
			usedMonthlyRecordings -= size

		}
		results3, err := db.Query("SELECT id, created_at FROM faxess WHERE workspace_id = ? AND created_at BETWEEN ? AND ?", workspace.Id, startFormatted, endFormatted);
		defer results3.Close()
		if err != nil {
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
			return
		}
		var faxId int
		for results3.Next() {
			results3.Scan(&faxId, &createdAt)
			change := float64(usedMonthlyFax - 1)
			centsForFax := baseCosts.FaxPerUsed
			charge, err := computeAmountToCharge(centsForFax, usedMonthlyFax, change)
			if err != nil {
				fmt.Printf("error getting charge..\r\n")
				fmt.Println(err)
				continue
			}
			faxCosts += charge 
			usedMonthlyFax -= 1

		}
		totalCosts += membershipCosts
		totalCosts += callTolls;
		totalCosts += recordingCosts;
		totalCosts += faxCosts;
		totalCosts += monthlyNumberRentals;
		stmt, err := db.Prepare("INSERT INTO users_debits (`cents`, `status`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?)")
		if err != nil {
			fmt.Printf("could not prepare query..\r\n")
			continue
		}
		defer stmt.Close()
		res, err := stmt.Exec(cents, "INCOMPLETE", workspace.CreatorId, workspace.Id)
		if err != nil {
			fmt.Printf("error creating debit..\r\n")
			continue
		}
		debitId, err := res.LastInsertId()
		if err != nil {
			fmt.Printf("could not get insert id..\r\n")
			fmt.Println(err)
			continue
		}
		// try to charge the debit
		if workspace.Plan == "pay-as-you-go" {
			remaining := billingInfo.RemainingBalanceCents
			change :=  float64(remaining) - totalCosts
			charge, err := computeAmountToCharge(totalCosts, remaining, change)
			if err != nil {
				fmt.Printf("error getting charge..\r\n")
				fmt.Println(err)
				continue
			}
			if ( charge == totalCosts ) { //user has enough credits
				stmt, err := db.Prepare("UPDATE users_debits SET status = 'complete', source ='CREDITS' WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					 continue
				}
				_, err = stmt.Exec(debitId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}
			} else {
				// update debit to reflect exactly how much we can charge
				stmt, err := db.Prepare("UPDATE users_debits SET status = 'complete', source ='CREDITS', cents = ? WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(billingInfo.RemainingBalanceCents, debitId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}
				// try to charge the rest using a card


				cents := int(math.Ceil(charge))
				err = lineblocs.ChargeCustomer(user, workspace, cents, invoiceDesc)
				if err != nil {
					fmt.Printf("error charging user..\r\n")
					fmt.Println(err)
					continue
				}

			}
            continue;

		} else {
// try to charge the rest using a card
				cents := int(math.Ceil(totalCosts))
				err = lineblocs.ChargeCustomer(user, workspace, cents, invoiceDesc)
				if err != nil {
					fmt.Printf("error charging user..\r\n")
					fmt.Println(err)
					continue
				}

		}
	}
}

func computeAmountToCharge(fullCentsToCharge float64, monthlyAllowed int, change float64) (float64, error) {
	//when total goes below 0, only charge the amount that went below 0
	if (monthlyAllowed > 0 && change < 0) {
		percentOfDebit := 1.0;
		//change =  -5;
		//usedMonthlyMinutes =  10;
		positive := math.Abs(change)

		set1 := float64(monthlyAllowed) + positive
		percentage := set1 / positive
		percentOfDebit, err := strconv.ParseFloat(".%d", int(math.Round(percentage)))
		if err != nil {
			fmt.Printf("error using ParseFloat on .%d\r\n", percentage)
			fmt.Println(err)
			return 0, err
		}

		centsToCharge := math.Ceil(float64(fullCentsToCharge) * percentOfDebit)
		return centsToCharge, nil
	} else if (monthlyAllowed <= 0) {
		return fullCentsToCharge, nil
	} else if (monthlyAllowed > 0 && change >= 0) {
		return 0, nil
	}
	return 0, nil
}
func runRemoveOldLogs() {
	now := time.Time{}
	// 7 day retention
	now.AddDate(0, 0, -7)
	dateFormatted := now.Format("2006-01-02 15:04:05")
	_, err := db.Exec("DELETE from debugger_logs where created_at >= ?", dateFormatted)
	if err != nil {
		fmt.Printf("error occured in log removing\r\n")
		fmt.Println(err)
		return
	}
}
func runSendBackgroundEmails() {
	ago := time.Time{}
	ago.AddDate(0, 0, -14)
	reminded := time.Time{}
	reminded.AddDate(0, 0, -28)
	dateFormatted := ago.Format("2006-01-02 15:04:05")
	results, err := db.Query("SELECT workspaces.id, workspaces.creator_id from workspaces inner join users on users.id = workspaces.creator_id where users.last_login >= ? AND users.last_login_reminded IS NULL", dateFormatted)
	if err != nil {
		fmt.Printf("error getting workspaces..\r\n")
		fmt.Println(err)
		return
	}

	defer results.Close()
	var id int
	var creatorId int
    for results.Next() {
		results.Scan(&id, &creatorId)

		fmt.Printf("Reminding user %d to use Lineblocs!\r\n", creatorId)
		user, err := lineblocs.GetUserFromDB(creatorId)
		if err != nil {
			fmt.Printf("could not get user from DB\r\n")
			continue
		}
		workspace, err := lineblocs.GetWorkspaceFromDB(id)
		if err != nil {
			fmt.Printf("could not get workspace from DB\r\n")
			continue
		}

		args := make(map[string]string)
		err = dispatchEmail("inactive-user", user, workspace, args)
		if err != nil {
			fmt.Printf("could not send email\r\n")
			fmt.Println(err)
			continue
		}
		stmt, err := db.Prepare("UPDATE users SET last_login_reminded = NOW()")
		if err != nil {
			fmt.Printf("could not prepare query..\r\n")
				continue
		}
		_, err = stmt.Exec()
		if err != nil {
			fmt.Printf("error updating users table..\r\n")
			fmt.Println(err)
			continue
		}
	}

	// usage triggers
	results, err = db.Query("SELECT workspaces.id, workspaces.creator_id from workspaces inner join users on users.id = workspaces.creator_id")
	if err != nil {
		fmt.Printf("error getting workspaces..\r\n")
		fmt.Println(err)
		return
	}

	defer results.Close()
	var creditId int
	var balance int
	var triggerId int
	var percentage int
    for results.Next() {
		results.Scan(&id, &creatorId)
		fmt.Printf("working with id: %d, creator %d\r\n", id, creatorId)
		user, err := lineblocs.GetUserFromDB(creatorId)
		if err != nil {
			fmt.Printf("could not get user from DB\r\n")
			continue
		}
		workspace, err := lineblocs.GetWorkspaceFromDB(id)
		if err != nil {
			fmt.Printf("could not get workspace from DB\r\n")
			continue
		}
		row := db.QueryRow(`SELECT id, balance FROM users_credits WHERE workspace_id=?`, workspace.Id)
		err = row.Scan(&creditId, &balance)
		if err != nil {
			fmt.Printf("could not get last balance of user..\r\n")
			fmt.Println(err)
			continue
		}
		billingInfo, err := lineblocs.GetWorkspaceBillingInfo(workspace)
		if err != nil {
			fmt.Printf("Could not get billing info..\r\n")
			continue
		}


		results2, err := db.Query("SELECT id, percentage from usage_triggers where workspace_id = ?", workspace.Id)
		defer results2.Close()
    	for results2.Next() {
			results2.Scan(&triggerId, &percentage)
			var triggerUsageId int
			row := db.QueryRow(`SELECT id FROM users WHERE id=?`, triggerId)
			err := row.Scan(&triggerUsageId)
			if ( err == sql.ErrNoRows ) {  //create conference
				fmt.Printf("Trigger reminder already sent..\r\n")
				continue
			}
			if ( err != nil ) {  //another error
				fmt.Printf("SQL error\r\n")
				fmt.Println(err)
				continue
			}


			percentOfTrigger, err := strconv.ParseFloat(".%d", percentage)
			if err != nil {
				fmt.Printf("error using ParseFloat on .%d\r\n", percentage)
				fmt.Println(err)
				continue
			}
			amount := int(math.Round(float64(balance) * percentOfTrigger))
		
			if billingInfo.RemainingBalanceCents <= amount {
				args := make(map[string]string)
				args["triggerPercent"] = fmt.Sprintf("%f", percentOfTrigger)
				args["triggerBalance"] = fmt.Sprintf("%d", balance)

				err = dispatchEmail("usage-trigger", user, workspace, args)
				if err != nil {
					fmt.Printf("could not send email\r\n")
					fmt.Println(err)
					continue
				}

				stmt, err := db.Prepare("INSERT INTO usage_triggers_results (usage_trigger_id) VALUES (?)")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					continue
				}

				defer stmt.Close()
				_, err = stmt.Exec(triggerId)
				if err != nil {
					fmt.Printf("error create usage trigger result..\r\n")
					continue
				}
			}
		}
	}


}
func main() {
	var err error
	db, err = lineblocs.CreateDBConn()
	if err != nil {
		fmt.Println(err)
		return
	}
	argsWithoutProg := os.Args[1:]
	if argsWithoutProg[0] == "half-hour" {
		runDeleteUnsetPasswordUsers()
		runFreeTrialEnding()
		runRemoveOldLogs()
		runSendBackgroundEmails()
	} else if argsWithoutProg[0] == "monthly" {
		runMonthlyBilling()
	}

}