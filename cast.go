package main

import (
	"errors"
	"github.com/aryszka/keyval"
)

type (
	Message        keyval.Entry
	Sender         chan<- *Message
	Receiver       <-chan *Message
	MessageChannel chan *Message
)

type Connection interface {
	Send() Sender
	Receive() Receiver
}

type Listener <-chan Connection

type Node interface {
	Join(Connection)
	Listen(Listener) error
	Send() Sender
	Receive() Receiver
	Errors() <-chan error
}

var (
	ErrDisconnected = errors.New("disconnected")
	ErrCannotListen = errors.New("node cannot listen")
)

func (c MessageChannel) Send() Sender      { return Sender(chan *Message(c)) }
func (c MessageChannel) Receive() Receiver { return Receiver(chan *Message(c)) }

// figure if it is really crucial to have the timeout implemented in the node
// figure how the timeout and buffers should work together
// report timeout
// test all
// self healing network
// sockets
// write a cmd client
// consider handling states
// consider stopping listening
