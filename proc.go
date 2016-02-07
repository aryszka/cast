package cast

import (
	"fmt"
	"github.com/aryszka/keyval"
	"time"
)

// simple channel implementing the connection interface
// one direction
type MessageChannel chan *Message

type inProcConnection struct {
	local  chan *Message
	remote *inProcConnection
}

type TimeoutError struct {
	Message *Message
}

// connection wrapper with timeout
// it should be used only when leaking messages is fine,
// otherwise buffering is recommended
type TimeoutConnection struct {
	connection Connection
	send       chan<- *Message
	Timeout    chan error
}

type bufferedConnection struct {
	send       chan *Message
	connection Connection
}

func (c MessageChannel) Send() chan<- *Message    { return c }
func (c MessageChannel) Receive() <-chan *Message { return c }

// creates a symmetric connection
// representing an in-process communication channel
func NewInProcConnection() (Connection, Connection) {
	local := &inProcConnection{local: make(chan *Message)}
	remote := &inProcConnection{local: make(chan *Message)}
	local.remote = remote
	remote.remote = local
	return local, remote
}

func (c *inProcConnection) Send() chan<- *Message    { return c.local }
func (c *inProcConnection) Receive() <-chan *Message { return c.remote.local }

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout during sending to connection, message: %s",
		keyval.JoinKey(e.Message.Key))
}

// wraps a connection with send timeout
// takes ownership of the connection regarding closing
func NewTimeoutConnection(c Connection, t time.Duration) *TimeoutConnection {
	send := make(chan *Message)
	to := make(chan error)

	go func() {
		open := true
		sending := 0
		sendComplete := make(chan struct{})
		for {
			var m *Message
			select {
			case m, open = <-send:
				if open {
					// wrong: there is no guarantee this way
					// avoid receiving the next message before reaching the sender select
					mc := make(chan *Message)
					go func(mc <-chan *Message) {
						m := <-mc
						select {
						case c.Send() <- m:
						case <-time.After(t):
							to <- &TimeoutError{m}
						}

						sendComplete <- struct{}{}
					}(mc)
					mc <- m
					sending++
				} else {
					send = nil
					if sending == 0 {
						close(c.Send())
						return
					}
				}
			case <-sendComplete:
				sending--
				if !open && sending == 0 {
					close(c.Send())
					return
				}
			}
		}
	}()

	return &TimeoutConnection{c, send, to}
}

func (c *TimeoutConnection) Send() chan<- *Message    { return c.send }
func (c *TimeoutConnection) Receive() <-chan *Message { return c.connection.Receive() }

// wraps a connection with a buffer
// takes ownership of the connection regarding closing
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

	return &bufferedConnection{send, c}
}

func (c *bufferedConnection) Send() chan<- *Message    { return c.send }
func (c *bufferedConnection) Receive() <-chan *Message { return c.connection.Receive() }
