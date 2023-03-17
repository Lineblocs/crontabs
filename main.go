package main

import (
	"os"

	lineblocs "github.com/Lineblocs/go-helpers"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"
	cmd "lineblocs.com/crontabs/cmd"
	"lineblocs.com/crontabs/utils"
	//now "github.com/jinzhu/now"
)

func main() {
	var err error

	logDestination := utils.Config("LOG_DESTINATIONS")
	lineblocs.InitLogrus(logDestination)

	args := os.Args[1:]
	if len(args) == 0 {
		lineblocs.Log(logrus.InfoLevel, "Please provide command")
		return
	}
	command := args[0]
	switch command {
	case "cleanup":
		lineblocs.Log(logrus.InfoLevel, "App cleanup started...")
		err = cmd.CleanupApp()
		if err != nil {
			lineblocs.Log(logrus.ErrorLevel, err.Error())
		}
	case "background_emails":
		lineblocs.Log(logrus.InfoLevel, "sending background emails")
		err = cmd.SendBackgroundEmails()
		if err != nil {
			lineblocs.Log(logrus.ErrorLevel, err.Error())
		}
	case "monthly_billing":
		lineblocs.Log(logrus.InfoLevel, "sending background emails")
		err = cmd.MonthlyBilling()
		if err != nil {
			lineblocs.Log(logrus.ErrorLevel, err.Error())
		}
	case "remove_logs":
		lineblocs.Log(logrus.InfoLevel, "sending background emails")
		err = cmd.RemoveLogs()
		if err != nil {
			lineblocs.Log(logrus.ErrorLevel, err.Error())
		}
	}
}
