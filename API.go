package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/juju/ratelimit"
)

func findPoke(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	token := r.FormValue("token")
	calls++

	if token == "" {
		token = r.Header.Get("X-Real-Ip")
		if _, ok := ratelimits[token]; !ok {
			ratelimits[token] = ratelimit.NewBucket(60*time.Second, 2)
		}
	} else {
		if !tokenExist(token) {
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write([]byte("403 - Access denied."))
			return
		}
		if _, ok := ratelimits[token]; !ok {
			ratelimits[token] = ratelimit.NewBucket(60*time.Second, 5)
		}
	}

	u := getUserByToken(token)

	if !u.Paid {
		if ratelimits[token].Capacity() == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("1015 - You are being rate limited."))
			return
		}
		d := ratelimits[token].Take(1)
		if d > 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("1015 - You are being rate limited."))
			return
		}
	}

	if !strings.HasPrefix(url, "http") || strings.Contains(url, " ") {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something bad happened!"))
		return
	}

	fmt.Println("Sarting request : " + url)

	resp, err := http.Post("http://127.0.0.1:5300", "", strings.NewReader(url))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Try again later."))
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
		w.Write([]byte("500 - Something bad happened!"))
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
