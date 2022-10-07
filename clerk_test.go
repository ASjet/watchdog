package watchdog

import (
	"strconv"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	name := "TestRegister"
	interval := time.Second / 2
	timeout := time.Second

	cfg := makeConfig(t)
	port := cfg.runServer(false, 0, 0)
	addr := "localhost:" + strconv.Itoa(port)

	ck := NewClerk(name, addr, interval, timeout)
	ck.register()
	cfg.addClient(ck.name, ck.id, ck.timeout)
}

func TestKeepAlive(t *testing.T) {
	name := "TestKeepAlive"
	interval := time.Second / 2
	timeout := time.Second

	cfg := makeConfig(t)
	port := cfg.runServer(true, timeout, timeout)
	addr := "localhost:" + strconv.Itoa(port)

	iter := 5

	ck := NewClerk(name, addr, interval, timeout)
	go ck.KeepAlive()
	time.Sleep(ck.interval)
	cfg.addClient(ck.name, ck.id, ck.timeout)

	for i := 0; i < iter; i++ {
		cfg.match()
		time.Sleep(ck.timeout)
	}
}

func TestOffline(t *testing.T) {
	name := "TestOffline"
	interval := time.Second / 2
	timeout := time.Second

	cfg := makeConfig(t)
	port := cfg.runServer(true, timeout*4, timeout)
	addr := "localhost:" + strconv.Itoa(port)

	ck := NewClerk(name, addr, interval, timeout)
	go ck.KeepAlive()
	time.Sleep(ck.interval)
	cfg.addClient(ck.name, ck.id, ck.timeout)

	time.Sleep(ck.timeout)
	ck.exit.Store(true)
	cfg.offline(ck.id)

	time.Sleep(timeout * 4)
	cfg.delClient(ck.id)
}

func TestLogout(t *testing.T) {
	name := "TestLogout"
	interval := time.Second / 2
	timeout := time.Second

	cfg := makeConfig(t)
	port := cfg.runServer(false, 0, 0)
	addr := "localhost:" + strconv.Itoa(port)

	ck := NewClerk(name, addr, interval, timeout)
	go ck.KeepAlive()
	time.Sleep(ck.interval)
	cfg.addClient(ck.name, ck.id, ck.timeout)

	time.Sleep(ck.timeout)
	ck.Logout()
	cfg.offlineClient(ck.id)
}
