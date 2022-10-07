package watchdog

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
)

func randPort() int {
	return 30000 + (rand.Int() % 30000)
}

type Client struct {
	name    string
	timeout time.Duration
	online  bool
	deleted bool
}

type Config struct {
	t       *testing.T
	w       *Watcher
	msq     chan interface{}
	clients map[uuid.UUID]*Client
}

func makeConfig(t *testing.T) *Config {
	ch := make(chan interface{}, 10)
	cfg := Config{
		t:       t,
		w:       NewWatcher(ch),
		msq:     ch,
		clients: make(map[uuid.UUID]*Client),
	}
	return &cfg
}

func (cfg *Config) getClient(id uuid.UUID) *Client {
	c, ok := cfg.clients[id]
	if !ok {
		log.Panicf("[%s]getClient: no such client", id)
	}
	return c
}

func (cfg *Config) addClient(name string, id uuid.UUID, timeout time.Duration) {
	client := &Client{
		name:    name,
		timeout: timeout,
		online:  true,
		deleted: false,
	}
	cfg.clients[id] = client
	cfg.match()
}

func (cfg *Config) offlineClient(id uuid.UUID) {
	client := cfg.getClient(id)
	client.online = false
	cfg.match()
}

func (cfg *Config) delClient(id uuid.UUID) {
	client := cfg.getClient(id)
	client.online = false
	client.deleted = true
	cfg.match()
}

func (cfg *Config) runServer(clean bool, interval, bound time.Duration) int {
	port := randPort()
	cfg.w.StartServer(port)
	if clean {
		go func(interval time.Duration) {
			for {
				cfg.w.Clean(bound)
				time.Sleep(interval)
			}
		}(interval)
	}
	return port
}

func (cfg *Config) register(name string, timeout time.Duration) uuid.UUID {
	args := RegisterArgs{
		Name:    name,
		Timeout: timeout,
	}
	reply := Reply{}
	err := cfg.w.Register(&args, &reply)
	if err != nil {
		log.Panicf("register: %s", err)
	}
	client := Client{
		name:    name,
		timeout: timeout,
		online:  true,
		deleted: false,
	}
	cfg.clients[reply.ID] = &client
	return reply.ID
}

func (cfg *Config) logout(id uuid.UUID) {
	c := cfg.getClient(id)
	args := PingArgs{
		ID: id,
	}
	reply := Reply{}
	c.online = false
	err := cfg.w.Logout(&args, &reply)
	if err != nil {
		log.Panicf("logout: %s", err)
	}
	n, ok := cfg.w.nodes[id]
	if ok && n.online.Load() {
		cfg.t.Fatalf("[%s]expect offline but not", id)
	}
	cfg.match()
}

func (cfg *Config) offline(id uuid.UUID) {
	c := cfg.getClient(id)
	r := <-cfg.msq
	wName := r.(*Record).Name
	if c.name != wName {
		cfg.t.Fatalf("[%s]expect client name %s, got %s", id, c.name, wName)
	}
	c.online = false
	cfg.match()
}

func (cfg *Config) online(id uuid.UUID) {
	c := cfg.getClient(id)
	args := PingArgs{
		ID: id,
	}
	reply := Reply{}
	cfg.w.Ping(&args, &reply)
	c.online = true
	c.deleted = false
	cfg.match()
}

func (cfg *Config) match() {
	for id, c := range cfg.clients {
		n, ok := cfg.w.nodes[id]
		if ok {
			if c.deleted {
				cfg.t.Fatalf("[%s]match(deleted): expect %v, got %v", id, c.deleted, !ok)
			}
			if n.name != c.name {
				cfg.t.Fatalf("[%s]match(name): expect %s, got %s", id, c.name, n.name)
			}
			if n.timeout != c.timeout {
				cfg.t.Fatalf("[%s]match(timeout): expect %s, got %s", id, c.timeout, n.timeout)
			}
			if n.online.Load() != c.online {
				cfg.t.Fatalf("[%s]match(online): expect %v, got %v", id, c.online, n.online.Load())
			}
		} else {
			if !c.deleted {
				cfg.t.Fatalf("[%s]match(deleted): expect %v, got %v", id, c.deleted, !ok)
			}
		}
	}
}

func TestWatcherRegister(t *testing.T) {
	name := "TestWatcherRegister"
	timeout := time.Second
	cfg := makeConfig(t)
	cfg.register(name, timeout)
	cfg.match()
}

func TestWatcherLogout(t *testing.T) {
	name := "TestWatcherLogout"
	timeout := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, timeout)
	cfg.match()
	cfg.logout(id)
}

func TestWatcherAlert(t *testing.T) {
	name := "TestWatcherAlert"
	timeout := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, timeout)
	cfg.offline(id)
}

func TestWatcherClean(t *testing.T) {
	name := "TestWatcherClean"
	timeout := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, timeout)
	cfg.offline(id)
	cfg.w.Clean(timeout)
	cfg.clients[id].deleted = true
	cfg.match()
}

func TestWatcherReconnect(t *testing.T) {
	name := "TestWatcherReconnect"
	timeout := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, timeout)
	cfg.offline(id)
	cfg.online(id)
}
