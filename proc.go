package cast

import (
	"fmt"
	"github.com/aryszka/keyval"
	"time"
)

type MessageChannel chan *Message

type inProcConnection struct {
	local  chan *Message
	remote *inProcConnection
}

type TimeoutError struct {
	message *Message
}

type TimeoutConnection struct {
	connection Connection
	send       chan<- *Message
	Timeout    chan error
}

func (c MessageChannel) Send() chan<- *Message    { return c }
func (c MessageChannel) Receive() <-chan *Message { return c }

func NewInProcConnection(l chan Connection) Connection {
	local := &inProcConnection{local: make(chan *Message)}
	remote := &inProcConnection{local: make(chan *Message)}
	local.remote = remote
	remote.remote = local
	l <- remote
	return local
}

func (c *inProcConnection) Send() chan<- *Message    { return c.local }
func (c *inProcConnection) Receive() <-chan *Message { return c.remote.local }

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout during sending to connection, message: %s",
		keyval.JoinKey(e.message.Key))
}

func NewTimeoutConnection(c Connection, t time.Duration) *TimeoutConnection {
	send := make(chan *Message)
	to := make(chan error)

	go func() {
		for {
			m, open := <-send
			if !open {
				close(c.Send())
				return
			}

			// avoid receiving the next message before reaching the sender select
			mc := make(chan *Message)
			go func(mc <-chan *Message) {
				m := <-mc
				select {
				case c.Send() <- m:
				case <-time.After(t):
					to <- &TimeoutError{m}
				}
			}(mc)
			mc <- m
		}
	}()

	return &TimeoutConnection{c, send, to}
}

func (c *TimeoutConnection) Send() chan<- *Message    { return c.send }
func (c *TimeoutConnection) Receive() <-chan *Message { return c.connection.Receive() }

func NewBufferedConnection(c Connection, size int) Connection {
	send := make(MessageChannel, size)
	go func() {
		for {
			m, open := <-send
			if !open {
				close(c.Send())
				return
			}

			c.Send() <- m
		}
	}()

	return send
}
