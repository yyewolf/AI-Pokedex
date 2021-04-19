package main

import (
	_ "image/jpeg"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	go hostService()
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
