package main

import (
	"embed"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/badoux/checkmail"
	"github.com/gorilla/pat"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

//go:embed www
var webbox embed.FS

type cookieUser struct {
	Email         string
	Authenticated bool
}

var store *sessions.CookieStore

func hostService() {
	mux := pat.New()
	srv := http.Server{
		Addr:    ":5000",
		Handler: mux,
	}
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	store = sessions.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	store.Options = &sessions.Options{
		MaxAge:   60 * 15,
		HttpOnly: true,
	}

	gob.Register(cookieUser{})

	//Find Poke
	mux.Post("/getPoke", compressHandler(http.HandlerFunc(findPoke)))
	mux.Post("/login", compressHandler(http.HandlerFunc(login)))
	mux.Post("/signup", compressHandler(http.HandlerFunc(signup)))
	mux.Get("/login", compressHandler(http.HandlerFunc(login)))
	mux.Get("/signup", compressHandler(http.HandlerFunc(signup)))
	mux.Get("/refreshToken", compressHandler(http.HandlerFunc(refreshToken)))

	mux.Get("/", compressHandler(http.HandlerFunc(indexHandler)))

	go srv.ListenAndServe()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	datb, _ := webbox.ReadFile("www/index.html")
	dat := string(datb)
	Path := r.URL.Path[1:]
	Path = "www/" + Path
	if Path == "www/" {
		dat = strings.ReplaceAll(dat, "{Calls}", strconv.Itoa(calls))
		dat = strings.ReplaceAll(dat, `<p style="color:red">{Error}</p>`, "")
		fmt.Fprint(w, dat)
	} else {
		dat, err := webbox.ReadFile(Path)
		if err != nil {
			fmt.Fprint(w, err)
		}
		if strings.HasSuffix(Path, ".css") {
			w.Header().Add("Content-Type", "text/css")
		}
		if strings.HasSuffix(Path, ".svg") {
			w.Header().Add("Content-Type", "image/svg+xml")
		}
		if strings.HasSuffix(Path, ".ico") {
			w.Header().Add("Content-Type", "image/x-icon")
		}
		fmt.Fprint(w, string(dat))
	}
}

func signup(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err == nil {
		if val, ok := session.Values["user"]; ok {
			user := val.(cookieUser)
			if !emailExist(user.Email) {
				session.Values["user"] = User{}
				session.Options.MaxAge = -1
				err = session.Save(r, w)
				http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
				return
			}
			u := getUserByEmail(user.Email)
			datb, _ := webbox.ReadFile("www/connected.html")
			dat := string(datb)
			dat = strings.ReplaceAll(dat, "{Token}", u.Token)
			dat = strings.ReplaceAll(dat, "{Email}", u.Email)
			dat = strings.ReplaceAll(dat, "{Paid}", strconv.FormatBool(u.Paid))
			fmt.Fprint(w, dat)
			return
		}
	}
	email := r.FormValue("email")
	passwd := r.FormValue("password")

	if emailExist(email) {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Email already in use")
		fmt.Fprint(w, dat)
		return
	}

	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	err = checkmail.ValidateHost(email)
	if err != nil {
		fmt.Println(err)
	}
	var (
		serverHostName    = "ssl0.ovh.net"        // set your SMTP server here
		serverMailAddress = "tristan@smagghe.com" // set your valid mail address here
	)
	err = checkmail.ValidateHostAndUser(serverHostName, serverMailAddress, email)
	if _, ok := err.(checkmail.SmtpError); ok && err != nil {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	if passwd == "" {
		datb, _ := webbox.ReadFile("www/signup.html")
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

	newUser := User{
		ID:       node.Generate().String(),
		Email:    email,
		Password: passwordHash,
		Token:    generateSecureToken(30),
	}

	_, err = database.InsertInto("accounts").
		Columns("*").
		Record(newUser).
		Exec()

	if err != nil {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", err.Error())
		fmt.Fprint(w, dat)
		return
	}

	userSession := &cookieUser{
		Email:         newUser.Email,
		Authenticated: true,
	}

	session.Values["user"] = userSession

	err = session.Save(r, w)

	datb, _ := webbox.ReadFile("www/connected.html")
	dat := string(datb)
	dat = strings.ReplaceAll(dat, "{Token}", newUser.Token)
	dat = strings.ReplaceAll(dat, "{Email}", newUser.Email)
	dat = strings.ReplaceAll(dat, "{Paid}", strconv.FormatBool(newUser.Paid))
	fmt.Fprint(w, dat)
}

func login(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err == nil {
		if val, ok := session.Values["user"]; ok {
			user := val.(cookieUser)
			if !emailExist(user.Email) {
				session.Values["user"] = User{}
				session.Options.MaxAge = -1
				err = session.Save(r, w)
				http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
				return
			}
			u := getUserByEmail(user.Email)
			datb, _ := webbox.ReadFile("www/connected.html")
			dat := string(datb)
			dat = strings.ReplaceAll(dat, "{Token}", u.Token)
			dat = strings.ReplaceAll(dat, "{Email}", u.Email)
			dat = strings.ReplaceAll(dat, "{Paid}", strconv.FormatBool(u.Paid))
			fmt.Fprint(w, dat)
			return
		}
	}
	email := r.FormValue("email")
	passwd := r.FormValue("password")

	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		datb, _ := webbox.ReadFile("www/login.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid email")
		fmt.Fprint(w, dat)
		return
	}

	if passwd == "" {
		datb, _ := webbox.ReadFile("www/login.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid password")
		fmt.Fprint(w, dat)
		return
	}

	user := getUserByEmail(email)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwd)); err != nil {
		datb, _ := webbox.ReadFile("www/login.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "Invalid password")
		fmt.Fprint(w, dat)
		return
	}

	userSession := &cookieUser{
		Email:         user.Email,
		Authenticated: true,
	}

	session.Values["user"] = userSession

	u := getUserByEmail(user.Email)

	err = session.Save(r, w)

	datb, _ := webbox.ReadFile("www/connected.html")
	dat := string(datb)
	dat = strings.ReplaceAll(dat, "{Token}", u.Token)
	dat = strings.ReplaceAll(dat, "{Email}", u.Email)
	dat = strings.ReplaceAll(dat, "{Paid}", strconv.FormatBool(u.Paid))
	fmt.Fprint(w, dat)
}

func refreshToken(w http.ResponseWriter, r *http.Request) {
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

	//Change token
	u.Token = generateSecureToken(30)

	database.Update("accounts").
		Set("token", u.Token).
		Where("email=$1", user.Email).
		Exec()

	http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
}
