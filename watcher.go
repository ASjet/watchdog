package watchdog

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	name    string
	timeout time.Duration

	lastComm atomic.Value
	online   atomic.Bool
	watching atomic.Bool
}

func NewNode(name string, timeout time.Duration) *Node {
	n := Node{
		name:    name,
		timeout: timeout,
	}
	n.lastComm.Store(time.Now())
	n.online.Store(true)
	n.watching.Store(false)
	return &n
}

type Watcher struct {
	nodes    map[uuid.UUID]*Node
	notifier chan interface{}
}

func NewWatcher(notifier chan interface{}) *Watcher {
	return &Watcher{
		nodes:    make(map[uuid.UUID]*Node),
		notifier: notifier,
	}
}

type Record struct {
	Name     string
	Dura     time.Duration
	LastComm time.Time
}

type RegisterArgs struct {
	Name    string
	Timeout time.Duration
}

type PingArgs struct {
	ID uuid.UUID
}

type Reply struct {
	ID uuid.UUID
}

func (w *Watcher) StartServer(port int) {
	rpc.Register(w)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	log.Printf("[Watcher]Listening on port %d", port)
}

// This is a RPC
func (w *Watcher) Register(args *RegisterArgs, reply *Reply) error {
	n := NewNode(args.Name, args.Timeout)
	id := uuid.New()
	w.nodes[id] = n
	go w.watch(id)
	reply.ID = id
	log.Printf("[Watcher]new client %q online, timeout %s", n.name, args.Timeout)
	return nil
}

// This is a RPC
func (w *Watcher) Ping(args *PingArgs, reply *Reply) error {
	n, ok := w.nodes[args.ID]
	if !ok {
		return fmt.Errorf("ERR_EXPIRED")
	}
	n.lastComm.Store(time.Now())
	if n.online.CompareAndSwap(false, true) {
		log.Printf("[Watcher]client %q is reconnected", n.name)
	}
	go w.watch(args.ID)
	return nil
}

// This is a RPC
func (w *Watcher) Logout(args *PingArgs, reply *Reply) error {
	n, ok := w.nodes[args.ID]
	if ok {
		n.online.Store(false)
		n.lastComm.Store(time.Now())
		log.Printf("[Watcher]client %q logout", n.name)
	}
	return nil
}

func (w *Watcher) watch(id uuid.UUID) {
	n := w.nodes[id]
	success := n.watching.CompareAndSwap(false, true)
	if !success {
		return
	}
	for n.online.Load() {
		dura := time.Since(n.lastComm.Load().(time.Time))
		if dura > n.timeout {
			n.online.Store(false)
			go w.alert(n)
			break
		}
		time.Sleep(n.timeout)
	}
	n.watching.Store(false)
}

func (w *Watcher) alert(n *Node) {
	log.Printf("[WARN]client %q is offline", n.name)
	lastCom := n.lastComm.Load().(time.Time)
	w.notifier <- &Record{
		Name:     n.name,
		Dura:     time.Since(lastCom),
		LastComm: lastCom,
	}
}

func (w *Watcher) Clean(bound time.Duration) {
	for id, n := range w.nodes {
		dura := time.Since(n.lastComm.Load().(time.Time))
		if dura > bound {
			delete(w.nodes, id)
		}
	}
}
