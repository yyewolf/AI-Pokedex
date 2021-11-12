package main

import (
	"embed"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"
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
	mux.Post("/changepass/{id}", compressHandler(http.HandlerFunc(changePassword)))
	mux.Post("/forgotpass", compressHandler(http.HandlerFunc(forgotPassword)))
	mux.Post("/admin", compressHandler(http.HandlerFunc(adminArea)))
	mux.Get("/login", compressHandler(http.HandlerFunc(login)))
	mux.Get("/signup", compressHandler(http.HandlerFunc(signup)))
	mux.Get("/refreshToken", compressHandler(http.HandlerFunc(refreshToken)))
	mux.Get("/resendMail", compressHandler(http.HandlerFunc(resendMail)))
	mux.Get("/pay", compressHandler(http.HandlerFunc(payHandler)))
	mux.Get("/emailverification/{id}", compressHandler(http.HandlerFunc(mailVerif)))
	mux.Get("/changepass/{id}", compressHandler(http.HandlerFunc(changePassword)))
	mux.Get("/forgotpass", compressHandler(http.HandlerFunc(forgotPassword)))
	mux.Get("/logout", compressHandler(http.HandlerFunc(logout)))
	mux.Get("/counts", compressHandler(http.HandlerFunc(showCounts)))
	mux.Get("/", compressHandler(http.HandlerFunc(indexHandler)))

	go srv.ListenAndServe()
}

func formatConnected(u User, data string) string {
	data = strings.ReplaceAll(data, "{Token}", u.Token)
	data = strings.ReplaceAll(data, "{Email}", u.Email)

	paidstring := ""
	if !u.Paid {
		paidstring = `<a href ="pay"><i class="fa fa-paypal"></i></a>`
	}
	data = strings.ReplaceAll(data, "{Paid}", strconv.FormatBool(u.Paid)+". "+paidstring)

	verifstring := ""
	if !u.Verified {
		verifstring = `<a href ="resendMail"><i class="fa fa-refresh"></i></a>`
	}
	data = strings.ReplaceAll(data, "{Verified}", strconv.FormatBool(u.Verified)+". "+verifstring)
	return data
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

var (
	verifier = emailverifier.
		NewVerifier().
		EnableSMTPCheck().
		EnableAutoUpdateDisposable()
)

var (
	ips list
)

func signup(w http.ResponseWriter, r *http.Request) {
	IP := strings.Split(r.Header.Get("X-Real-Ip"), ":")[0]
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
			dat := formatConnected(u, string(datb))
			fmt.Fprint(w, dat)
			return
		}
	}
	email := r.FormValue("email")
	passwd := r.FormValue("password")

	if r.ContentLength == 0 {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "")
		fmt.Fprint(w, dat)
		return
	}

	if ips.Has(IP) {
		datb, _ := webbox.ReadFile("www/signup.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "We've detected that you might already have an account, use it instead")
		fmt.Fprint(w, dat)
		return
	}

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

	domain := strings.Split(email, "@")[1]
	username := strings.Split(email, "@")[0]
	_, err = verifier.CheckSMTP(domain, username)
	if err != nil || verifier.IsDisposable(domain) || strings.Contains(username, "+") {
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
	}
	newUser.Token = newUser.generateSecureToken(30)

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

	ips.Add(IP)

	datb, _ := webbox.ReadFile("www/connected.html")
	dat := formatConnected(newUser, string(datb))
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
			dat := formatConnected(u, string(datb))
			fmt.Fprint(w, dat)
			return
		}
	}
	if r.ContentLength == 0 {
		datb, _ := webbox.ReadFile("www/login.html")
		dat := string(datb)
		dat = strings.ReplaceAll(dat, "{Error}", "")
		fmt.Fprint(w, dat)
		return
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
	dat := formatConnected(u, string(datb))
	fmt.Fprint(w, dat)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")
	session.Values["user"] = User{}
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.SetCookie(w, &http.Cookie{Name: "cookie-name"})
	http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
}

func refreshToken(w http.ResponseWriter, r *http.Request) {
	IP := strings.Split(r.Header.Get("X-Real-Ip"), ":")[0]
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
	old := u.Token
	//Change token
	u.Token = u.generateSecureToken(30)

	ratelimits[u.Token] = ratelimits[old]

	database.Update("accounts").
		Set("token", u.Token).
		Where("email=$1", user.Email).
		Exec()

	http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)

	delete(ratelimits, old)
	iptokens[IP].Remove(old)
	iptokens[IP].Add(u.Token)
}

func showCounts(w http.ResponseWriter, r *http.Request) {
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
	if user.Email != "yyewolf@gmail.com" {
		return
	}
	n := map[int][]string{}
	var a []int
	for k, v := range counts {
		n[v] = append(n[v], k)
	}
	for k := range n {
		a = append(a, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(a)))
	for _, k := range a {
		for _, s := range n[k] {
			fmt.Fprintf(w, "%s, %d\n", s, k)
		}
	}
}
