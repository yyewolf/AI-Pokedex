package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/juju/ratelimit"
)

func adminArea(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	paid := r.FormValue("paid")
	pswd := r.FormValue("pswd")

	if pswd != "***REMOVED***" {
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
	calls++

	u := getUserByToken(token)
	/*
		Applies the first rate limiter : IP based
	*/
	if u.Email != "***REMOVED***" && u.Email != "***REMOVED***" {
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
			ratelimits[token] = ratelimit.NewBucket(30*time.Second, 1)
		}
	} else {
		if !tokenExist(token) {
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("403 - Access denied."))
			return
		}
		if _, ok := ratelimits[token]; !ok {
			if u.Paid {
				ratelimits[token] = ratelimit.NewBucket(5*time.Second, 1)
			} else {
				ratelimits[token] = ratelimit.NewBucket(10*time.Second, 1)
			}
		}
	}

	/*
		Applies second rate limiter : token based
	*/

	if u.Email != "***REMOVED***" && u.Email != "***REMOVED***" {
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

	resp, err := http.Post("http://127.0.0.1:5300", "", strings.NewReader(url))
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
