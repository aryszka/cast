package cast

import (
	"fmt"
	"github.com/aryszka/keyval"
	"time"
)

// simple channel implementing the connection interface
// one direction
// error channel always blocking
type MessageChannel chan *Message

type inProcConnection struct {
	local  chan *Message
	remote *inProcConnection
}

type TimeoutError struct {
	Message *Message
}

type bufferedConnection struct {
	send       chan *Message
	connection Connection
	err        chan error
}

type timeoutConnection struct {
	connection Connection
	send       chan<- *Message
	err        chan error
}

func (c MessageChannel) Send() chan<- *Message    { return c }
func (c MessageChannel) Receive() <-chan *Message { return c }
func (c MessageChannel) Error() <-chan error      { return nil }

// creates a symmetric connection
// representing an in-process communication channel
// error channel always blocking
func NewInProcConnection() (Connection, Connection) {
	local := &inProcConnection{local: make(chan *Message)}
	remote := &inProcConnection{local: make(chan *Message)}
	local.remote = remote
	remote.remote = local
	return local, remote
}

func (c *inProcConnection) Send() chan<- *Message    { return c.local }
func (c *inProcConnection) Receive() <-chan *Message { return c.remote.local }
func (c *inProcConnection) Error() <-chan error      { return nil }

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout during sending to connection, message: %s",
		keyval.JoinKey(e.Message.Key))
}

// wraps a connection with a buffer
// takes ownership of the connection regarding closing
// takes over error reporting of embedded channel
func NewBufferedConnection(c Connection, size int) Connection {
	send := make(MessageChannel, size)
	ec := make(chan error)
	go func() {
		for {
			select {
			case m, open := <-send:
				if !open {
					close(c.Send())
					return
				}

				c.Send() <- m
			case err, open := <-c.Error():
				if open {
					ec <- err
				} else {
					panic("error channel closed")
				}
			}
		}
	}()

	return &bufferedConnection{send, c, ec}
}

func (c *bufferedConnection) Send() chan<- *Message    { return c.send }
func (c *bufferedConnection) Receive() <-chan *Message { return c.connection.Receive() }
func (c *bufferedConnection) Error() <-chan error      { return c.err }

// wraps a connection with send timeout
// takes ownership of the connection regarding closing
//
// connection wrapper with timeout
// it should be used only when leaking messages is fine,
// otherwise buffering is recommended
//
// order of timeout errors not guaranteed, order of sent messages guaranteed
func NewTimeoutConnection(c Connection, t time.Duration) Connection {
	send := make(chan *Message)
	ec := make(chan error)

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
							ec <- &TimeoutError{m}
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
			case err, open := <-c.Error():
				if open {
					ec <- err
				} else {
					panic("error channel closed")
				}
			}
		}
	}()

	return &timeoutConnection{c, send, ec}
}

func (c *timeoutConnection) Send() chan<- *Message    { return c.send }
func (c *timeoutConnection) Receive() <-chan *Message { return c.connection.Receive() }
func (c *timeoutConnection) Error() <-chan error      { return c.err }
