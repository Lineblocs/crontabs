package cmd

import (
	"fmt"
	"time"

	"database/sql"
	"math"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	//now "github.com/jinzhu/now"

	lineblocs "github.com/Lineblocs/go-helpers"
	utils "lineblocs.com/crontabs/utils"
)

// cron tab to email users to tell them that their free trial will be ending soon
func SendBackgroundEmails() error {
	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}

	ago := time.Time{}
	ago = ago.AddDate(0, 0, -14)
	reminded := time.Time{}
	reminded = reminded.AddDate(0, 0, -28)
	dateFormatted := ago.Format("2006-01-02 15:04:05")
	results, err := db.Query("SELECT workspaces.id, workspaces.creator_id from workspaces inner join users on users.id = workspaces.creator_id where users.last_login >= ? AND users.last_login_reminded IS NULL", dateFormatted)
	if err != nil {
		fmt.Printf("error getting workspaces..\r\n")
		fmt.Println(err)
		return err
	}

	defer results.Close()
	// declare some common variables
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
		err = utils.DispatchEmail("inactive-user", user, workspace, args)
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
		return err
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
			fmt.Println(err)
			continue
		}

		results2, err := db.Query("SELECT id, percentage from usage_triggers where workspace_id = ?", workspace.Id)
		defer results2.Close()
		for results2.Next() {
			results2.Scan(&triggerId, &percentage)
			var triggerUsageId int
			row := db.QueryRow(`SELECT id FROM users WHERE id=?`, triggerId)
			err := row.Scan(&triggerUsageId)
			if err == sql.ErrNoRows {
				fmt.Printf("Trigger reminder already sent..\r\n")
				continue
			}
			if err != nil { //another error
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
			amount := math.Round(float64(balance) * percentOfTrigger)

			if billingInfo.RemainingBalanceCents <= amount {
				args := make(map[string]string)
				args["triggerPercent"] = fmt.Sprintf("%f", percentOfTrigger)
				args["triggerBalance"] = fmt.Sprintf("%d", balance)

				err = utils.DispatchEmail("usage-trigger", user, workspace, args)
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

	days := "7"
	results, err = db.Query(`SELECT id, creator_id FROM `+"`"+`workspaces`+"`"+` WHERE free_trial_started <= DATE_ADD(NOW(), INTERVAL -? DAY) AND free_trial_reminder_sent = 0`, days)
	if err != nil {
		return err
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
		err = utils.DispatchEmail("free-trial-ending", user, workspace, args)
		if err != nil {
			fmt.Printf("could not send email\r\n")
			fmt.Println(err)
			continue
		}
		stmt, err := db.Prepare("UPDATE workspaces SET free_trial_reminder_sent = 1 WHERE id = ?")
		if err != nil {
			fmt.Printf("could not prepare query..\r\n")
			continue
		}
		_, err = stmt.Exec(workspace.Id)
		if err != nil {
			fmt.Printf("error updating DB..\r\n")
			fmt.Println(err)
			continue
		}
	}
	return nil
}
