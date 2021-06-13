package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

//map[ID]email
var recoveries map[string]string

//map[email]int
var cacherecoveries map[string]int

func changePassword(w http.ResponseWriter, r *http.Request) {
	passID := r.URL.Query().Get(":id")
	email, ok := recoveries[passID]
	if !ok {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
		return
	}

	if r.ContentLength == 0 {
		datb, _ := webbox.ReadFile("www/changepass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "")
		fmt.Fprint(w, dat)
		return
	}

	passwd := r.FormValue("password")

	if passwd == "" {
		datb, _ := webbox.ReadFile("www/changepass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid password")
		fmt.Fprint(w, dat)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	passwordHash := string(hash)

	_, err = database.Update("accounts").
		Set("password", passwordHash).
		Where("email=$1", email).
		Exec()

	if err != nil {
		datb, _ := webbox.ReadFile("www/changepass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", err.Error())
		fmt.Fprint(w, dat)
		return
	}

	delete(cacherecoveries, email)
	delete(recoveries, passID)
	http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
}

func forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength == 0 {
		datb, _ := webbox.ReadFile("www/forgotpass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "")
		fmt.Fprint(w, dat)
		return
	}

	email := r.FormValue("email")
	if !emailExist(email) {
		datb, _ := webbox.ReadFile("www/forgotpass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Email not in use")
		fmt.Fprint(w, dat)
		return
	}

	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		datb, _ := webbox.ReadFile("www/forgotpass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	domain := strings.Split(email, "@")[1]
	username := strings.Split(email, "@")[0]
	_, err := verifier.CheckSMTP(domain, username)
	if err != nil || verifier.IsDisposable(domain) || strings.Contains(username, "+") {
		datb, _ := webbox.ReadFile("www/forgotpass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	if cacherecoveries[email] >= 3 {
		datb, _ := webbox.ReadFile("www/forgotpass.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	id := generateSecureToken(16)
	recoveries[id] = email
	cacherecoveries[email]++

	sendMail("[AIPokedex] Password Reset", "Reset your password by using this link : https://aipokedex.com/changepass/"+id+"/", email)
	fmt.Fprint(w, "Sent your recovery email.")
}
