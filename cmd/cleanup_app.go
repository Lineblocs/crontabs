package cmd

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	//now "github.com/jinzhu/now"

	utils "lineblocs.com/crontabs/utils"
)

// cron tab to remove unset password users
func CleanupApp() error {
	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}
	days := "7"
	var id int
	results, err := db.Query(`SELECT id FROM `+"`"+`users`+"`"+` WHERE needs_set_password_date <= DATE_ADD(NOW(), INTERVAL -? DAY) AND needs_password_set = 1`, days)
	if err != nil {
		return err
	}
	defer results.Close()
	for results.Next() {
		results.Scan(&id)
		fmt.Printf("Removing user %d\r\n", id)
		_, err := db.Query(`DELETE FROM `+"`"+`users`+"`"+` WHERE id = ?`, id)
		if err != nil {
			fmt.Printf("Could not remove %d\r\n", id)
			continue
		}
	}
	return nil
}
