package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
)

type User struct {
	ID             string `json:"id" db:"id"`
	Email          string `json:"email" db:"email"`
	Password       string `json:"password" db:"password"`
	Token          string `json:"token" db:"token"`
	Paid           bool   `json:"paid" db:"paid"`
	LastPaid       int64  `json:"last_paid" db:"last_paid"`
	Euros          int    `json:"euros" db:"euros"`
	Cents          int    `json:"cents" db:"cents"`
	SubscriptionID string `json:"sub_id" db:"sub_id"`
	Verified       bool   `json:"verified" db:"verified"`
}

func generateRecoveryToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (u *User) generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	email := base64.StdEncoding.EncodeToString([]byte(u.Email))
	return fmt.Sprintf("%s.%s", email, hex.EncodeToString(b))
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
	d, e := database.Select("*").
		From("accounts").
		Where("token=$1", token).
		QueryJSON()
	if e != nil {
		return false
	}

	e = json.Unmarshal(d, &u)
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

func getUserBySubID(id string) User {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("sub_id=$1", id).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return User{}
	}
	return u[0]
}

func getUserByID(id string) User {
	var u []User
	d, _ := database.Select("*").
		From("accounts").
		Where("id=$1", id).
		QueryJSON()

	e := json.Unmarshal(d, &u)
	if e != nil {
		return User{}
	}
	return u[0]
}
