package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

func findPoke(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")

	if !strings.HasPrefix(url, "http") || strings.Contains(url, " ") {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something bad happened!"))
		return
	}

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
}
