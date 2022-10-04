package watchdog

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	name     string
	interval time.Duration
	online   bool
	deleted  bool
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
		log.Panicf("no such client %q", id)
	}
	return c
}

func (cfg *Config) register(name string, interval time.Duration) uuid.UUID {
	args := RegisterArgs{
		Name:     name,
		Interval: interval,
	}
	reply := Reply{}
	err := cfg.w.Register(&args, &reply)
	if err != nil {
		cfg.t.Fatal(err)
	}
	client := Client{
		name:     name,
		interval: interval,
		online:   true,
		deleted:  false,
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
		cfg.t.Fatal(err)
	}
	n, ok := cfg.w.nodes[id]
	if ok && n.online.Load() {
		cfg.t.Fatalf("client %q not logout", id)
	}
	cfg.match()
}

func (cfg *Config) offline(id uuid.UUID) {
	c := cfg.getClient(id)
	r := <-cfg.msq
	wName := r.(*Record).Name
	if c.name != wName {
		cfg.t.Fatalf("expect client %q name %s, got %s", id, c.name, wName)
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
				cfg.t.Fatalf("client %q should be deleted but not", id)
			}
			if n.name != c.name ||
				n.interval != c.interval ||
				n.online.Load() != c.online {
				cfg.t.Fatalf("client %q status not equal", id)
			}
		} else {
			if !c.deleted {
				cfg.t.Fatalf("client %q not exists", id)
			}
		}
	}
}

func TestWatcherRegister(t *testing.T) {
	name := "TestWatcherRegister"
	interval := time.Second
	cfg := makeConfig(t)
	cfg.register(name, interval)
	cfg.match()
}

func TestWatcherLogout(t *testing.T) {
	name := "TestWatcherLogout"
	interval := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, interval)
	cfg.match()
	cfg.logout(id)
}

func TestWatcherAlert(t *testing.T) {
	name := "TestWatcherAlert"
	interval := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, interval)
	cfg.offline(id)
}

func TestWatcherClean(t *testing.T) {
	name := "TestWatcherClean"
	interval := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, interval)
	cfg.offline(id)
	cfg.w.Clean(interval)
	cfg.clients[id].deleted = true
	cfg.match()
}

func TestWatcherReconnect(t *testing.T) {
	name := "TestWatcherReconnect"
	interval := time.Second
	cfg := makeConfig(t)
	id := cfg.register(name, interval)
	cfg.offline(id)
	cfg.online(id)
}
