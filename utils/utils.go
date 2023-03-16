package utils
import (
	"fmt"
	"encoding/json"
	"net/http"
	"database/sql"
	"bytes"
	"io/ioutil"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mailgun/mailgun-go/v4"
	lineblocs "github.com/Lineblocs/go-helpers"
	models "lineblocs.com/crontabs/models"
)


var db* sql.DB;

type BillingParams struct {
	Provider string
	Data map[string]string
}

func GetDBConnection() (*sql.DB, error) {
	if db != nil {
		return db, nil
	}
	var err error
	db, err = lineblocs.CreateDBConn()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func CheckRowCount(rows *sql.Rows) (int, error) {
	var count int;
 	for rows.Next() {
		err:= rows.Scan(&count)
		if err != nil {
			return 0, err
		}
    }   
    return count, nil
}



func DispatchEmail(emailType string, user* lineblocs.User, workspace* lineblocs.Workspace, emailArgs map[string]string) (error) {
    url := "http://lineblocs-email/send"

	email := models.Email{User: *user, Workspace: *workspace, EmailType: emailType, Args: emailArgs};
	b, err := json.Marshal(email)
	if err != nil {
		fmt.Println("error:", err)
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

    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return nil
}


func GetBillingParams() (*BillingParams, error) {
	data := make(map[string]string)
	params := BillingParams{
		Provider: "stripe",
		Data: data }
	return &params, nil
}

