package models

import (
	helpers "github.com/Lineblocs/go-helpers"
)

type Email struct {
	EmailType string            `json:"email_type"`
	Subject string            `json:"subject"`
	To string            `json:"to"`
	User      helpers.User      `json:"user"`
	Workspace helpers.Workspace `json:"workspace"`
	Args      map[string]string `json:"args"`
}
