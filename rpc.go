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

type AlertBody struct {
	Name     string
	Dura     time.Duration
	LastComm time.Time
}

type Node struct {
	name     string
	interval time.Duration

	lastComm atomic.Value
	online   atomic.Bool
	watching atomic.Bool
}

func NewNode(name string, interval time.Duration) *Node {
	n := Node{
		name:     name,
		interval: interval,
	}
	n.lastComm.Store(time.Now())
	n.online.Store(true)
	n.watching.Store(false)
	return &n
}

type Watcher struct {
	nodes    map[uuid.UUID]*Node
	mailConf *MailConf
}

func NewWatcher(conf *MailConf) *Watcher {
	return &Watcher{
		nodes:    make(map[uuid.UUID]*Node),
		mailConf: conf,
	}
}

type RegisterArgs struct {
	Name     string
	Interval time.Duration
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
	log.Printf("Start watcher server at port %d", port)
}

func (w *Watcher) Register(args *RegisterArgs, reply *Reply) error {
	n := NewNode(args.Name, args.Interval)
	id := uuid.New()
	w.nodes[id] = n
	go w.watch(id)
	reply.ID = id
	log.Printf("[INFO]new client %q online", n.name)
	return nil
}

func (w *Watcher) Ping(args *PingArgs, reply *Reply) error {
	n, ok := w.nodes[args.ID]
	if !ok {
		return fmt.Errorf("Register expired")
	}
	n.lastComm.Store(time.Now())
	if n.online.CompareAndSwap(false, true) {
		log.Printf("[INFO]client %q is reconnected", n.name)
	}
	go w.watch(args.ID)
	return nil
}

func (w *Watcher) Logout(args *PingArgs, reply *Reply) error {
	n, ok := w.nodes[args.ID]
	if ok {
		n.online.Store(false)
		log.Printf("[INFO]client %q logout", n.name)
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
		if dura > n.interval {
			n.online.Store(false)
			w.alert(id)
			break
		}
		time.Sleep(n.interval)
	}
	n.watching.Store(false)
}

func (w *Watcher) alert(id uuid.UUID) {
	n, ok := w.nodes[id]
	if !ok {
		return
	}
	log.Printf("[WARN]client %q is offline", n.name)
	lastCom := n.lastComm.Load().(time.Time)
	body := AlertBody{
		Name:     n.name,
		Dura:     time.Since(lastCom),
		LastComm: lastCom,
	}
	if err := sendMail(w.mailConf, &body); err != nil {
		log.Print(err)
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
