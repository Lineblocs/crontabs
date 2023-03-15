package cmd

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	"time"

	utils "lineblocs.com/crontabs/utils"
)

// remove any logs older than retention period
func RemoveLogs() error {
	db, err := utils.GetDBConnection()
	if err != nil {
		return err
	}

	dateNow := time.Time{}
	// 7 day retention
	dateNow = dateNow.AddDate(0, 0, -7)
	dateFormatted := dateNow.Format("2006-01-02 15:04:05")
	_, err = db.Exec("DELETE from debugger_logs where created_at >= ?", dateFormatted)
	if err != nil {
		fmt.Printf("error occured in log removing\r\n")
		fmt.Println(err)
		return err
	}
	return nil
}
