package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"errors"
	"math"

	helpers "github.com/Lineblocs/go-helpers"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"
	billing "lineblocs.com/crontabs/handlers/billing"
	models "lineblocs.com/crontabs/models"
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

func ChargeCustomer(dbConn *sql.DB, billingParams *BillingParams, user *helpers.User, workspace *helpers.Workspace, invoice *models.UserInvoice) error {
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
		err = hndl.ChargeCustomer(user, workspace, invoice)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "error charging user..\r\n")
			helpers.Log(logrus.ErrorLevel, err.Error())
		}
	case "braintree":
		key := billingParams.Data["braintree_api_key"]
		hndl = billing.NewBraintreeBillingHandler(dbConn, key, retryAttempts)
		err = hndl.ChargeCustomer(user, workspace, invoice)
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

func DispatchEmail(subject string, emailType string, user *helpers.User, workspace *helpers.Workspace, emailArgs map[string]string) error {
	url := "http://com/api/sendEmail"
	to := user.Email
	email := models.Email{User: *user, Workspace: *workspace, Subject: subject, To: to, EmailType: emailType, Args: emailArgs}
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

func ComputeAmountToCharge(fullCentsToCharge float64, availMinutes float64, minutes float64) (float64, error) {
	minAfterDebit := availMinutes - minutes
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge full: %f, used minutes %f, minutes %f, minAfterDebit: %f\r\n", fullCentsToCharge, availMinutes, minutes, minAfterDebit))
	//when total goes below 0, only charge the amount that went below 0
	// ensure availMinutes < minutes
	if availMinutes > 0 && minAfterDebit < 0 && availMinutes <= minutes {
		percentOfDebit, err := strconv.ParseFloat(fmt.Sprintf(".%s", strconv.FormatFloat((minutes-availMinutes), 'f', -1, 64)), 8)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("computeAmountToCharge could not parse float %s", err.Error()))
			return 0, err
		}

		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge percentage = %f, rounded = %f", percentOfDebit, math.Round(percentOfDebit)))
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge debit = %f", percentOfDebit))
		centsToCharge := math.Abs(float64(fullCentsToCharge) * percentOfDebit)
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", centsToCharge))
		return math.Max(1, centsToCharge), nil
	} else if availMinutes >= minutes { // user has enough balance, no need to charge
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", 0.0))
		return 0, nil
	} else if availMinutes <= 0 { // no minutes remaining, charge the full amount
		helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", fullCentsToCharge))
		return fullCentsToCharge, nil
	}

	// this should not happen
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("computeAmountToCharge result: %f\r\n", 0.0))
	return 0, errors.New(fmt.Sprintf("billing ran into unexpected error. computeAmountToCharge full: %f, used minutes %f, minutes %f, minAfterDebit: %f\r\n", fullCentsToCharge, availMinutes, minutes, minAfterDebit))
}

func CreateInvoiceConfirmationNumber() (string, error) {
	return "123", nil
}
