package main

import (
	"os"

	helpers "github.com/Lineblocs/go-helpers"
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
	helpers.InitLogrus(logDestination)

	args := os.Args[1:]
	if len(args) == 0 {
		helpers.Log(logrus.InfoLevel, "Please provide command")
		return
	}
	command := args[0]
	switch command {
	case "cleanup":
		helpers.Log(logrus.InfoLevel, "App cleanup started...")
		err = cmd.CleanupApp()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "background_emails":
		helpers.Log(logrus.InfoLevel, "sending background emails")
		err = cmd.SendBackgroundEmails()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "monthly_billing":
		helpers.Log(logrus.InfoLevel, "running monthly billing routines")
		err = cmd.MonthlyBilling()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "remove_logs":
		helpers.Log(logrus.InfoLevel, "removing old logs"0
		err = cmd.RemoveLogs()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	}
}
