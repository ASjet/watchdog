package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"watchdog"
)

const CleanInterval = time.Minute * 1

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <config_file> <port>\n", os.Args[0])
		os.Exit(0)
	}
	conf, err := watchdog.ReadMailConf(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalln(err)
	}

	watcher := watchdog.NewWatcher(conf)
	watcher.StartServer(port)
	fmt.Printf("PubList: %v\n", conf.PubList)
	for {
		watcher.Clean(CleanInterval)
		time.Sleep(CleanInterval)
	}
}
