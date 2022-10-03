package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"watchdog"

	"github.com/google/uuid"
)

const RETRY_TIME = 5

func main() {
	if len(os.Args) != 5 {
		fmt.Printf("Usage: %s <addr> <port> <name> <interval>\n", os.Args[0])
		os.Exit(0)
	}
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalln(err)
	}
	interval, err := time.ParseDuration(os.Args[4])
	if err != nil {
		log.Fatalln(err)
	}

	addr := fmt.Sprintf("%s:%d", os.Args[1], port)
	id := register(addr, os.Args[3], interval)
	go cleanup(addr, id)

	for {
		if ping(addr, id) != nil {
			id = register(addr, os.Args[3], interval)
		}
		time.Sleep(interval / 2)
	}
}

func cleanup(addr string, id uuid.UUID) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logout(addr, id)
	os.Exit(0)
}

func register(addr, name string, interval time.Duration) uuid.UUID {
	d := dial(addr)
	defer d.Close()

	args := watchdog.RegisterArgs{
		Name:     name,
		Interval: interval,
	}
	reply := watchdog.Reply{}

	for i := 0; i < RETRY_TIME; i++ {
		err := d.Call("Watcher.Register", &args, &reply)
		if err == nil {
			log.Printf("assign id %q", reply.ID)
			return reply.ID
		}
	}
	log.Fatalf("unable to acquire uuid")
	return uuid.UUID{}
}

func ping(addr string, id uuid.UUID) error {
	d := dial(addr)
	defer d.Close()

	args := watchdog.PingArgs{
		ID: id,
	}
	reply := watchdog.Reply{}

	return d.Call("Watcher.Ping", &args, &reply)
}

func logout(addr string, id uuid.UUID) {
	d := dial(addr)
	defer d.Close()

	args := watchdog.PingArgs{
		ID: id,
	}
	reply := watchdog.Reply{}
	d.Call("Watcher.Logout", &args, &reply)
}

func dial(addr string) *rpc.Client {
	dial, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return dial
}
