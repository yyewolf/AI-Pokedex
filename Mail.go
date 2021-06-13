package main

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
)

func sendMail(subject, body, toEmail string) {
	var host = "127.0.0.1"
	var port = "25"
	var addr = host + ":" + port
	fromName := "AI Pokedex Verification"
	fromEmail := "no-reply@aipokedex.com"
	toEmails := []string{toEmail}
	// Build RFC-2822 email
	toAddresses := []string{}

	for i := range toEmails {
		to := mail.Address{
			Address: toEmails[i],
		}
		toAddresses = append(toAddresses, to.String())
	}

	toHeader := strings.Join(toAddresses, ", ")
	from := mail.Address{
		Name:    fromName,
		Address: fromEmail,
	}
	fromHeader := from.String()
	subjectHeader := subject
	header := make(map[string]string)
	header["To"] = toHeader
	header["From"] = fromHeader
	header["Subject"] = subjectHeader
	header["Content-Type"] = `text/html; charset="UTF-8"`
	msg := ""
	for k, v := range header {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body
	bMsg := []byte(msg)
	// Send using local postfix service
	c, err := smtp.Dial(addr)

	if err != nil {
		return
	}
	defer c.Close()
	if err = c.Mail(fromHeader); err != nil {
		return
	}
	for _, addr := range toEmails {
		if err = c.Rcpt(addr); err != nil {
			return
		}
	}
	w, err := c.Data()
	if err != nil {
		return
	}
	_, err = w.Write(bMsg)
	if err != nil {
		return
	}
	err = w.Close()
	if err != nil {
		return
	}
	err = c.Quit()
	if err != nil {
		return
	}
}

func mailVerif(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get(":id")
	user := getUserByID(userID)
	if user.Email == "" {
		fmt.Fprint(w, "There has been an error.")
		return
	}
	if user.Verified {
		fmt.Fprint(w, "You already verified your email.")
		return
	}
	_, err := database.Update("accounts").
		Set("verified", true).
		Where("id=$1", user.ID).
		Exec()
	if err != nil {
		fmt.Fprint(w, "There has been an error, try again later.")
	}
	fmt.Fprint(w, "Your email has been verified, thank you !")
}

func resendMail(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
	}
	var val interface{}
	var ok bool
	if val, ok = session.Values["user"]; !ok {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
		return
	}

	user := val.(cookieUser)
	u := getUserByEmail(user.Email)

	if u.Email == "" {
		return
	}

	sendMail("[AIPokedex] Email verification", "Confirm your email by clicking this link : https://aipokedex.com/emailverification/"+u.ID+"/", u.Email)
	fmt.Fprint(w, "Sent.")
}
