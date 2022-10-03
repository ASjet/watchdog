package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"watchdog"
	"watchdog/sender"
)

const CleanInterval = time.Minute * 1

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <config_file> <port>\n", os.Args[0])
		os.Exit(0)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	sender := sender.NewMailSender()
	err = sender.Init(file)
	if err != nil {
		log.Fatal(err)
	}

	msq := make(chan interface{}, 10)
	go sender.Listen(msq)

	watcher := watchdog.NewWatcher(msq)
	watcher.StartServer(port)
	for {
		watcher.Clean(CleanInterval)
		time.Sleep(CleanInterval)
	}
}
