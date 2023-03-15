package cmd

import (
	"fmt"
	"time"

	"database/sql"
	"strconv"
	"math"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	//now "github.com/jinzhu/now"

	utils "lineblocs.com/crontabs/utils"
	lineblocs "github.com/Lineblocs/go-helpers"
)

func computeAmountToCharge(fullCentsToCharge float64, monthlyAllowed float64, change float64) (float64, error) {
	fmt.Printf("computeAmountToCharge full: %f, monthly allowed: %f, change: %f\r\n", fullCentsToCharge, monthlyAllowed, change);
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
		fmt.Printf("computeAmountToCharge result: %f\r\n", centsToCharge);
		return centsToCharge, nil
	} else if (monthlyAllowed <= 0) {
		fmt.Printf("computeAmountToCharge result: %f\r\n", fullCentsToCharge);
		return fullCentsToCharge, nil
	} else if (monthlyAllowed > 0 && change >= 0) {
		fmt.Printf("computeAmountToCharge result: %f\r\n", 0.0);
		return 0, nil
	}
	fmt.Printf("computeAmountToCharge result: %f\r\n", 0.0);
	return 0, nil
}


// cron tab to remove unset password users
func MonthlyBilling() (error) {
	var id int 
	var creatorId int 

	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}



	start := time.Now()
	start = start.AddDate(0, -1, 0)
	end := time.Now()
	currentTime := time.Now()
	startFormatted := start.Format("2006-01-02 15:04:05")
	endFormatted := end.Format("2006-01-02 15:04:05")
 	results, err := db.Query("SELECT id, creator_id FROM workspaces");
    if err != nil {
		fmt.Printf("error running query..\r\n")
		fmt.Println(err)
		return err
	}
	plans, err := lineblocs.GetServicePlans()
	if err != nil {
		fmt.Printf("error getting service plans\r\n")
		fmt.Println(err)
		return err
	}

	  defer results.Close()
    for results.Next() {
		err = results.Scan(&id, &creatorId)
		workspace, err := lineblocs.GetWorkspaceFromDB(id)
	    if err != nil {
			fmt.Printf("error getting workspace ID: " + strconv.Itoa(id) + "\r\n")
			continue
		}
		user, err := lineblocs.GetUserFromDB(creatorId)
	    if err != nil {
			fmt.Printf("error getting user ID: " + strconv.Itoa(id) + "\r\n")
			continue
		}

	  	var plan *lineblocs.ServicePlan
		for _, target := range plans {
			if target.Name == workspace.Plan {
				plan = &target
				break
			}
		}
		if plan == nil {
			fmt.Printf("No plan found for user..\r\n")
			continue
		}
		billingInfo, err := lineblocs.GetWorkspaceBillingInfo(workspace)
		if err != nil {
			fmt.Printf("Could not get billing info..\r\n")
			fmt.Println(err)
			continue
		}


		var didId int
		var monthlyCosts int
		results1, err := db.Query("SELECT id, monthly_cost  FROM did_numbers WHERE workspace_id = ?", workspace.Id)
		if ( err != sql.ErrNoRows && err != nil ) {  //create conference
			fmt.Printf("Could not get dids info..\r\n")
			fmt.Println(err)
			continue
		}
		defer results1.Close()
    	for results1.Next() {
			results1.Scan(&didId, &monthlyCosts)
			stmt, err := db.Prepare("INSERT INTO users_debits (`source`, `status`, `cents`, `module_id`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?)")
			if err != nil {
				fmt.Printf("could not prepare query..\r\n")
				fmt.Println(err)
				continue
			}

			defer stmt.Close()
			_, err = stmt.Exec("NUMBER_RENTAL", "INCOMPLETE", monthlyCosts, didId, user.Id, workspace.Id, start)
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

		// get the amount of users in this workspace
		rows, err := db.Query("SELECT COUNT(*) as count FROM  workspaces_users WHERE workspace_id = ?", workspace.Id);
		if err != nil {
			fmt.Printf("error getting workspace user count.\r\n")
			fmt.Println(err)
			continue
		}
		rows.Close()


		userCount, err := utils.CheckRowCount(rows)
		if err != nil {
			fmt.Printf("error getting workspace user count.\r\n")
			fmt.Println(err)
			continue
		}
		fmt.Printf("Workspace total user count %d",userCount)

		totalCosts := 0.0
		membershipCosts := plan.BaseCosts * float64(userCount)
		callTolls := 0.0
		recordingCosts := 0.0
		faxCosts := 0.0
		monthlyNumberRentals :=  0.0
		invoiceDesc := fmt.Sprintf("LineBlocs invoice for %s", billingInfo.InvoiceDue)


		fmt.Printf("Workspace total membership costs is %f",membershipCosts)

		results2, err := db.Query("SELECT id, source, module_id, cents, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted);
		if err != nil {
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
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
			fmt.Printf("scanning in debit source %s\r\n", source);
			if source == "CALL" {
				fmt.Printf("getting call %d\r\n", moduleId)
				call, err := lineblocs.GetCallFromDB(moduleId)
				if err != nil {
					fmt.Printf("error running query..\r\n")
					fmt.Println(err)
					return err
				}
				duration := call.DurationNumber
				fmt.Printf("call duration is %d\r\n", duration)
				minutes := float64(duration / 60)
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
				fmt.Printf("getting DID %d\r\n", moduleId)
				did, err := lineblocs.GetDIDFromDB(moduleId)
				if err != nil {
					fmt.Printf("error running query..\r\n")
					fmt.Println(err)
					continue
				}

				monthlyNumberRentals += float64(did.MonthlyCost)
			} 
		}
		results3, err := db.Query("SELECT id, size, created_at FROM recordings WHERE user_id = ? AND created_at BETWEEN ? AND ?", workspace.CreatorId, startFormatted, endFormatted);
		if ( err != sql.ErrNoRows && err != nil ) { 
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
			return err
		}
		defer results3.Close()
		var recId int
		var size float64
		var createdAt time.Time
		for results3.Next() {
			results3.Scan(&recId, &size, &createdAt)
			change := usedMonthlyRecordings - size
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
		results4, err := db.Query("SELECT id, created_at FROM faxes WHERE workspace_id = ? AND created_at BETWEEN ? AND ?", workspace.Id, startFormatted, endFormatted);
		if ( err != sql.ErrNoRows && err != nil ) { 
			fmt.Printf("error running query..\r\n")
			fmt.Println(err)
			return err
		}
		defer results4.Close()
		var faxId int
		for results4.Next() {
			results4.Scan(&faxId, &createdAt)
			change := float64(usedMonthlyFax - 1)
			centsForFax := baseCosts.FaxPerUsed
			charge, err := computeAmountToCharge(centsForFax, float64(usedMonthlyFax), change)
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

		fmt.Printf("Final costs are membership: %f, call tolls: %f, recordings: %f, fax: %f, did rentals: %f, total: %f (cents)\r\n", 
			membershipCosts,
			callTolls,
			recordingCosts,
			faxCosts,
			monthlyNumberRentals,
		totalCosts)


		fmt.Printf("Creating invoice for user %d, on workspace %d, plan type %s\r\n", user.Id, workspace.Id, workspace.Plan)
		stmt, err := db.Prepare("INSERT INTO users_invoices (`cents`, `call_costs`, `recording_costs`, `fax_costs`, `membership_costs`, `number_costs`, `status`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			fmt.Printf("could not prepare query..\r\n")
			fmt.Println( err )
			continue
		}
		defer stmt.Close()
		res, err := stmt.Exec(cents, callTolls, recordingCosts, faxCosts, membershipCosts, monthlyNumberRentals, "INCOMPLETE", workspace.CreatorId, workspace.Id, currentTime)
		if err != nil {
			fmt.Printf("error creating invoice..\r\n")
			fmt.Println(err)
			continue
		}
		invoiceId, err := res.LastInsertId()
		if err != nil {
			fmt.Printf("could not get insert id..\r\n")
			fmt.Println(err)
			continue
		}
		fmt.Printf("Charging user %d, on workspace %d, plan type %s\r\n", user.Id, workspace.Id, workspace.Plan)
		// try to charge the debit
		//if workspace.Plan == "pay-as-you-go" {
		if plan.PayAsYouGo {
			remaining := billingInfo.RemainingBalanceCents
			change :=  remaining - totalCosts
			charge, err := computeAmountToCharge(totalCosts, remaining, change)
			if err != nil {
				fmt.Printf("error getting charge..\r\n")
				fmt.Println(err)
				continue
			}
			if ( charge == totalCosts ) { //user has enough credits
				fmt.Printf("User has enough credits. Charging balance\r\n")
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CREDITS', cents_collected = ? WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					 continue
				}
				_, err = stmt.Exec(totalCosts, invoiceId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}
			} else {
				fmt.Printf("User does not have enough credits. Charging balance as much as possible\r\n")
				// update debit to reflect exactly how much we can charge
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'INCOMPLETE', source ='CREDITS', cents_collected = ? WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(charge, invoiceId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}
				// try to charge the rest using a card
				fmt.Printf("Charging remainder with card..\r\n")

				cents := int(math.Ceil(charge))
				err = lineblocs.ChargeCustomer(user, workspace, cents, invoiceDesc)
				if err != nil {
					fmt.Printf("error charging user..\r\n")
					fmt.Println(err)
					continue
				}
				stmt, err = db.Prepare("UPDATE users_invoices SET status = 'complete', source ='CREDITS', cents_collected = ? WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(totalCosts, invoiceId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}

			}
            continue;

		} else {
			// regular membership charge. only try to charge a card
				fmt.Printf("Charging recurringly with card..\r\n")
				cents := int(math.Ceil(totalCosts))
				err = lineblocs.ChargeCustomer(user, workspace, cents, invoiceDesc)
				if err != nil {
					fmt.Printf("error charging user..\r\n")
					fmt.Println(err)
					stmt, err := db.Prepare("UPDATE users_invoices SET status = 'INCOMPLETE', cents_collected = 0.0 WHERE id = ?")
					if err != nil {
						fmt.Printf("could not prepare query..\r\n")
						continue
					}
					_, err = stmt.Exec(invoiceId)
					if err != nil {
						fmt.Printf("error updating invoice....\r\n")
						fmt.Println(err)
						continue
					}

					continue
				}
				stmt, err := db.Prepare("UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ? WHERE id = ?")
				if err != nil {
					fmt.Printf("could not prepare query..\r\n")
					continue
				}
				_, err = stmt.Exec(totalCosts, invoiceId)
				if err != nil {
					fmt.Printf("error updating debit..\r\n")
					fmt.Println(err)
					continue
				}


		}
	}
	return nil
}