package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/juju/ratelimit"
	"github.com/pmylund/go-cache"
)

func adminArea(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	paid := r.FormValue("paid")
	pswd := r.FormValue("pswd")

	if pswd != adminPass {
		return
	}

	u := getUserByEmail(email)

	b, err := strconv.ParseBool(paid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Wrong boolean value."))
		return
	}

	database.Update("accounts").
		Set("paid", b).
		Where("email=$1", u.Email).
		Exec()
}

func findPoke(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	token := r.FormValue("token")
	model := r.FormValue("model")

	if model != "background" {
		model = "classic"
	}

	u := getUserByToken(token)

	/*
		If the person is using the web version and connected it will use their token
	*/

	session, err := store.Get(r, "cookie-name")
	if err == nil {
		var val interface{}
		var ok bool
		if val, ok = session.Values["user"]; ok {
			user := val.(cookieUser)
			u = getUserByEmail(user.Email)
			token = u.Token
		}
	}

	/*
		If the request is from Pokeboat
	*/

	if r.Header.Get(specialHeader) != "" {
		u.Email = specialEmail
	}

	if r.Header.Get(specialHeader) == "dataset" {
		u.Email = adminMail
	}

	/*
		Applies the first rate limiter : IP based
	*/
	if !priviledgedIP.Has(u.Email) {
		IP := r.Header.Get("X-Real-Ip")
		if _, ok := iplimits[IP]; !ok {
			iplimits[IP] = ratelimit.NewBucket(2*time.Second, 1)
		}
		if iplimits[IP].Available() == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("1015 - You are being rate limited."))
			return
		}
		d := iplimits[IP].Take(1)
		if d > 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Add("RateLimit-Reset", strconv.FormatInt(int64(d.Seconds()), 10))
			w.Write([]byte("1015 - You are being rate limited (" + strconv.FormatInt(int64(d.Seconds()), 10) + ")."))
			return
		}
	}

	/*
		Check for token or for guest
	*/

	if token == "" {
		token = r.Header.Get("X-Real-Ip")
		if _, ok := ratelimits[token]; !ok {
			ratelimits[token] = ratelimit.NewBucket(140*time.Second, 1)
		}
	} else {
		if !tokenExist(token) && u.Email != specialEmail {
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("403 - Access denied."))
			return
		}
		//People can use their account as well as pokeboat
		if u.Email == specialEmail {
			token = token + specialEmail
		}
		if _, ok := ratelimits[token]; !ok {
			if u.Paid {
				ratelimits[token] = ratelimit.NewBucket(30*time.Second, 1)
			} else {
				ratelimits[token] = ratelimit.NewBucket(100*time.Second, 1)
			}
		}
	}

	/*
		Applies second rate limiter : token based
	*/

	if !priviledgedToken.Has(u.Email) {
		if ratelimits[token].Available() == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("1015 - You are being rate limited."))
			return
		}
		d := ratelimits[token].Take(1)
		if d > 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Add("RateLimit-Reset", strconv.FormatInt(int64(d.Seconds()), 10))
			w.Write([]byte("1015 - You are being rate limited (" + strconv.FormatInt(int64(d.Seconds()), 10) + ")."))
			return
		}
	}

	w.Header().Add("RateLimit-Remaining", strconv.FormatInt(ratelimits[token].Available(), 10))

	if !strings.HasPrefix(url, "http") || strings.Contains(url, " ") {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Wrong URL Format."))
		return
	}

	if x, found := urlcache.Get(url); found {
		calls++
		response := x.(string)

		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, response)
		return
	}

	req, err := http.NewRequest("POST", "http://127.0.0.1:5300", strings.NewReader(url))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Recognition API is offline."))
		return
	}
	req.Header.Set("Content-Type", "")
	req.Header.Set("Model", model)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Recognition API is offline."))
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	response := string(body)
	response = strings.ReplaceAll(response, "b'", "")
	response = strings.ReplaceAll(response, "'", "")

	if response != "" {
		calls++
		urlcache.Set(url, response, cache.DefaultExpiration)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, response)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Recognition API is offline."))
	}

	/*
		//GET POKENAME HERE
		result := new(bytes.Buffer)
		cmd := exec.Command("python3", "./static/script.py", url)
		cmd.Stdout = result
		cmd.Run()

		str := result.String()
		if str != "" {
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, result.String())
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Something bad happened!"))
		}
	*/
}
