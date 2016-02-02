package main

import "time"

type MessageChannel chan *Message

type buffer struct {
	send       chan *Message
	connection Connection
}

type inProcConnection struct {
	local  chan *Message
	remote *inProcConnection
}

func (c MessageChannel) Send() Sender      { return Sender(chan *Message(c)) }
func (c MessageChannel) Receive() Receiver { return Receiver(chan *Message(c)) }
func (c MessageChannel) Close()            { close(c) }

func NewBuffer(c Connection, size int, timeout time.Duration) Connection {
	send := make(chan *Message, size)
	go func() {
		for {
			m, open := <-send
			if !open {
				c.Close()
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
func (b *buffer) Close()            { close(b.send) }

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
func (c *inProcConnection) Close()            { close(c.local) }

func NewBufferedConnection(size int, timeout time.Duration) Connection {
	if timeout <= 0 {
		return make(MessageChannel, size)
	}

	return NewBuffer(make(MessageChannel), size, timeout)
}
