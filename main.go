package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	cmd "lineblocs.com/crontabs/cmd"
	"os"
	//now "github.com/jinzhu/now"
)

func main() {
	var err error
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Please provide command")
		return
	}
	command := args[0]
	switch command {
	case "cleanup":
		fmt.Println("App cleanup started...")
		err = cmd.CleanupApp()
		if err != nil {
			fmt.Print(err)
		}
	case "background_emails":
		fmt.Println("sending background emails")
		err = cmd.SendBackgroundEmails()
		if err != nil {
			fmt.Print(err)
		}
	case "monthly_billing":
		fmt.Println("sending background emails")
		err = cmd.MonthlyBilling()
		if err != nil {
			fmt.Print(err)
		}
	case "remove_logs":
		fmt.Println("sending background emails")
		err = cmd.RemoveLogs()
		if err != nil {
			fmt.Print(err)
		}
	}
}
