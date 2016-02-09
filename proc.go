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

// error send in case of a timeout on a connection that handles it
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

type timeoutMessage struct {
	message *Message
	timeout chan struct{}
}

// wraps a connection with send timeout
// takes ownership of the connection regarding closing
//
// connection wrapper with timeout
// it should be used only when leaking messages is fine,
// otherwise buffering is recommended
//
// order of timeout errors not guaranteed, order of sent messages guaranteed
//
// This connection trades in memory for time, therefore it can cause
// unpredictable behavior.
func NewTimeoutConnection(c Connection, t time.Duration) Connection {
	send := make(chan *Message)
	ec := make(chan error)

	go func() {
		var (
			forward chan<- *Message
			ctm     *timeoutMessage
			cm      *Message
			cto     <-chan struct{}
			tms     []*timeoutMessage
		)

		for {
			if ctm == nil && len(tms) > 0 {
				forward = c.Send()
				ctm = tms[0]
				tms = tms[1:]
				cm = ctm.message
				cto = ctm.timeout
			} else if ctm == nil {
				forward = nil
				cto = nil
			}

			select {
			case m, open := <-send:
				if open {
					to := make(chan struct{})
					time.AfterFunc(t, func() {
						close(to)
					})
					tms = append(tms, &timeoutMessage{m, to})
				} else {
					send = nil
					if ctm == nil {
						close(c.Send())
						return
					}
				}
			case forward <- cm:
				ctm = nil
			case <-cto:
				ec <- &TimeoutError{ctm.message}
				ctm = nil
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
