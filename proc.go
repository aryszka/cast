package main

import (
	"fmt"
	"github.com/aryszka/keyval"
	"time"
)

type buffer struct {
	send       chan *Message
	connection Connection
}

type inProcConnection struct {
	local  chan *Message
	remote *inProcConnection
}

type TimeoutError struct {
	message *Message
}

type timeoutConnection struct {
	connection Connection
	send       Sender
	timeout    chan error
}

func NewBuffer(c Connection, size int, timeout time.Duration) Connection {
	send := make(chan *Message, size)
	go func() {
		for {
			m, open := <-send
			if !open {
				close(c.Send())
				return
			}

			if timeout <= 0 {
				c.Send() <- m
				return
			}

			select {
			case c.Send() <- m:
			case <-time.After(timeout):
			}
		}
	}()

	return &buffer{send, c}
}

func (b *buffer) Send() Sender      { return b.send }
func (b *buffer) Receive() Receiver { return b.connection.Receive() }

func NewInProcConnection(l chan Connection) Connection {
	local := &inProcConnection{local: make(chan *Message)}
	remote := &inProcConnection{local: make(chan *Message)}
	local.remote = remote
	remote.remote = local
	l <- remote
	return local
}

func (c *inProcConnection) Send() Sender      { return c.local }
func (c *inProcConnection) Receive() Receiver { return c.remote.local }

func NewBufferedConnection(size int, timeout time.Duration) Connection {
	if timeout <= 0 {
		return make(MessageChannel, size)
	}

	return NewBuffer(make(MessageChannel), size, timeout)
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout during sending to connection, message: %s",
		keyval.JoinKey(e.message.Key))
}

// wrong, does not provide error interface
func NewTimeoutConnection(c Connection, t time.Duration) Connection {
	send := make(chan *Message)
	to := make(chan error)
	go func() {
		for {
			m, open := <-send
			if !open {
				close(c.Send())
				return
			}

			// wrong, order is not guaranteed
			go func() {
				select {
				case c.Send() <- m:
				case <-time.After(t):
					to <- &TimeoutError{m}
				}
			}()
		}
	}()

	return &timeoutConnection{c, send, to}
}

func (c *timeoutConnection) Send() Sender {
	return c.send
}

func (c *timeoutConnection) Receive() Receiver { return c.connection.Receive() }
