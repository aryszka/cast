package main

import (
	"errors"
	"github.com/aryszka/keyval"
)

type Message keyval.Entry

type Sender chan<- *Message

type Receiver <-chan *Message

type Connection interface {
	Send() Sender
	Receive() Receiver
	Close()
}

type Listener <-chan Connection

type Node interface {
	Listen(Listener) error
	Join(Connection)
	Send() Sender
	Receive() Receiver
	Errors() <-chan error
	Close()
}

var (
	ErrDisconnected = errors.New("disconnected")
	ErrCannotListen = errors.New("node cannot listen")
)

// report timeout
// test all
// self healing network
// sockets
// write a cmd client
// consider handling states
