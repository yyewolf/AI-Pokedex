package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	_ "image/jpeg"
)

type User struct {
	ID       string `json:"id" db:"id"`
	Email    string `json:"email" db:"email"`
	Password string `json:"password" db:"password"`
	Token    string `json:"token" db:"token"`
	Paid     bool   `json:"paid" db:"paid"`
	LastPaid int64  `json:"last_paid" db:"last_paid"`
	Euros    int    `json:"euros" db:"euros"`
	Cents    int    `json:"cents" db:"cents"`
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func emailExist(email string) bool {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("email=$1", email).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return false
	}
	return true
}

func tokenExist(token string) bool {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("token=$1", token).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return false
	}
	return true
}

func getUserByToken(token string) User {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("token=$1", token).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return User{}
	}
	return u[0]
}

func getUserByEmail(email string) User {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("email=$1", email).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return User{}
	}
	return u[0]
}
