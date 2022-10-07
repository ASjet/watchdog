package watchdog

import (
	"log"
	"net/rpc"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Clerk struct {
	name     string
	addr     string
	interval time.Duration
	timeout  time.Duration
	id       uuid.UUID
	exit     atomic.Bool
}

func NewClerk(name, addr string, interval, timeout time.Duration) *Clerk {
	clerk := new(Clerk)
	clerk.name = name
	clerk.addr = addr
	clerk.interval = interval
	clerk.timeout = timeout
	log.Printf("[Clerk]ping interval: %s, timeout: %s", interval, timeout)
	return clerk
}

func (c *Clerk) call(RPCName string, args interface{}, reply interface{}) error {
	dial, err := rpc.DialHTTP("tcp", c.addr)
	if err != nil {
		return err
	}
	defer dial.Close()
	err = dial.Call(RPCName, args, reply)
	if err != nil {
		return err
	}
	return nil
}

func (c *Clerk) register() {
	args := RegisterArgs{
		Name:    c.name,
		Timeout: c.timeout,
	}
	reply := Reply{}

	for {
		err := c.call("Watcher.Register", &args, &reply)
		if err == nil {
			break
		}
		log.Printf("[Clerk]register: %s, retrying...", err)
	}
	c.id = reply.ID
	log.Printf("[Clerk]assigned id %q", c.id)
}

func (c *Clerk) KeepAlive() {
	for !c.exit.Load() {
		args := PingArgs{
			ID: c.id,
		}
		reply := Reply{}
		err := c.call("Watcher.Ping", &args, &reply)
		if err != nil {
			if err.Error() == "ERR_EXPIRED" {
				c.register()
			} else {
				log.Printf("[Clerk]KeepAlive: %s", err)
			}
			continue
		}
		time.Sleep(c.interval)
	}
	log.Print("[Clerk]exit")
}

func (c *Clerk) Logout() {
	args := PingArgs{
		ID: c.id,
	}
	reply := Reply{}
	c.exit.Store(true)
	c.call("Watcher.Logout", &args, &reply)
}
