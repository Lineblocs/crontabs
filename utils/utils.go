package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"fmt"
	"strconv"

	helpers "github.com/Lineblocs/go-helpers"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"
	models "lineblocs.com/crontabs/models"
	billing "lineblocs.com/crontabs/handlers/billing"
)

var db *sql.DB

type BillingParams struct {
	Provider string
	Data     map[string]string
}

func GetDBConnection() (*sql.DB, error) {
	if db != nil {
		return db, nil
	}
	var err error
	db, err = helpers.CreateDBConn()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ChargeCustomer(dbConn *sql.DB, billingParams *BillingParams, user *helpers.User, workspace *helpers.Workspace, cents int, invoiceDesc string) (error) {
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
		hndl = billing.NewStripeBillingHandler(dbConn, key, retryAttempts)
		err = hndl.ChargeCustomer(user, workspace, cents, invoiceDesc)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "braintree":
		key := billingParams.Data["braintree_api_key"]
		hndl = billing.NewBraintreeBillingHandler(dbConn, key, retryAttempts)
		err = hndl.ChargeCustomer(user, workspace, cents, invoiceDesc)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	}
	return err
}


func CheckRowCount(rows *sql.Rows) (int, error) {
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func DispatchEmail(emailType string, user *helpers.User, workspace *helpers.Workspace, emailArgs map[string]string) error {
	url := "http://lineblocs-email/send"

	email := models.Email{User: *user, Workspace: *workspace, EmailType: emailType, Args: emailArgs}
	b, err := json.Marshal(email)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
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

	helpers.Log(logrus.InfoLevel, "response Status:"+resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	helpers.Log(logrus.InfoLevel, "response Body:"+string(body))
	return nil
}

func GetBillingParams() (*BillingParams, error) {
	conn, err := GetDBConnection()
	if err != nil {
		return nil, err
	}

	row := conn.QueryRow("SELECT payment_gateway FROM customizations")

	var paymentGateway string
	err = row.Scan(&paymentGateway)
	if err != nil {
		return nil, err
	}


	row = conn.QueryRow("SELECT stripe_private_key FROM api_credentials")
	if err != nil {
		return nil, err
	}

	var stripePrivateKey string
	err = row.Scan(&stripePrivateKey)
	if err != nil {
		return nil, err
	}

	data := make(map[string]string)
	data["stripe_key"] = stripePrivateKey
	params := BillingParams{
		Provider: "stripe",
		Data:     data}
	return &params, nil
}

func Config(key string) string {
	// load .env file
	loadDotEnv := os.Getenv("USE_DOTENV")
	if loadDotEnv != "off" {
		err := godotenv.Load(".env")
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "Error loading .env file")
		}
	}
	return os.Getenv(key)
}
