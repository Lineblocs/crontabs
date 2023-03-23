package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	helpers "github.com/Lineblocs/go-helpers"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/mailgun/mailgun-go/v4"
	"github.com/sirupsen/logrus"
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
	data := make(map[string]string)
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
