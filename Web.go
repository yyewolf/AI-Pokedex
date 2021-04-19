package main

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/pat"
)

//go:embed www
var webbox embed.FS

func hostService() {
	mux := pat.New()
	srv := http.Server{
		Addr:    ":5000",
		Handler: mux,
	}

	//Find Poke
	mux.Post("/getPoke", compressHandler(http.HandlerFunc(findPoke)))

	mux.Get("/", compressHandler(http.HandlerFunc(indexHandler)))

	go srv.ListenAndServe()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	datb, _ := webbox.ReadFile("www/index.html")
	dat := string(datb)
	Path := r.URL.Path[1:]
	Path = "www/" + Path
	if Path == "www/" {
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
