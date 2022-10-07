package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"watchdog"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Printf("Usage: %s <addr:port> <name> <timeout> <interval>\n", os.Args[0])
		os.Exit(0)
	}

	addr := os.Args[1]
	name := os.Args[2]

	timeout, err := time.ParseDuration(os.Args[3])
	if err != nil {
		log.Fatalf("while parsing timeout %q: %s", os.Args[3], err)
	}

	interval, err := time.ParseDuration(os.Args[4])
	if err != nil {
		log.Fatalf("while parsing interval %q: %s", os.Args[4], err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	clerk := watchdog.NewClerk(name, addr, interval, timeout)
	go clerk.KeepAlive()

	<-c
	clerk.Logout()
}
