package main

import (
	"errors"
	"github.com/aryszka/keyval"
)

type (
	Message  keyval.Entry
	Sender   chan<- *Message
	Receiver <-chan *Message
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

// test all
// benchmark all
// self healing network
// sockets
// write a cmd client
// consider handling states
// consider stopping listening
