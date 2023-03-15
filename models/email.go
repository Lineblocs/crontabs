package models
import (
	lineblocs "github.com/Lineblocs/go-helpers"
)

type Email struct {
	EmailType string `json:"email_type"`
	User lineblocs.User `json:"user"`
	Workspace lineblocs.Workspace `json:"workspace"`
	Args map[string]string `json:"args"`
}