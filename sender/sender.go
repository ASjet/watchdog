package sender

import "io"

type Sender interface {
	Init(conf io.Reader) error
	Listen(msq chan interface{})
}
