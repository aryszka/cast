/*
cast stands for continously applied state transfer
this rather only to add up to the nice sounding abbreviation

cast is about sharing and applying atomic pieces of information in
distributed systems as soon as they are available regardless of the full
state of the concepts they describe

cast systems may not be consistent at any single moment in time,
measurable or not, but they should continously be convergent to a
consistent state

cast systems may provide consistency or consensus for a subset or the
entirety of the represented concepts built on top of the cast
communication primitives

this library provides a gossip style communication model and a carrier
network for cast based systems

if the error channels are left out, mention them as a best practice
*/
package cast

import (
	"errors"

	// the statement behind this import:
	// - strong statement for a protocol, but,
	// - anything can be represented by a list of structured keys and
	// free form values
	// - comments are important, once humans reason about computing
	// artefacts
	"github.com/aryszka/keyval"
)

// a message contains a key, a value and a comment
// all three are optional
// the handling of a message is completely up to the system built around
// cast
// the content of the comment field should not be used for
// programmatic logic, it can be used only  when a message or a set of
// messages is displayed in human readable form
type Message keyval.Entry

// blocking?
// closing?
// reconnection?
// errors? maybe in general, the errors should be handled by the
// implementation and initialization
//
// a simple channel actually implements this interface
type Connection interface {
	Send() chan<- *Message
	Receive() <-chan *Message
}

// shall we add errors? no: should be handled by the creator
// watch close?
type Listener <-chan Connection

// minimal implementation
// listen error? maybe it should be a panic
type Node interface {
	Join(Connection)
	Listen(Listener) error // is this error unavoidable?
	Send() chan<- *Message
	Receive() <-chan *Message
	Errors() <-chan error // is this unavoidable?
}

// ???
var (
	ErrDisconnected = errors.New("disconnected")
	ErrCannotListen = errors.New("node cannot listen") // should be panic
)

// test all
// benchmark all
// document closing connections
// self healing network
// sockets
// write a cmd client
// consider handling states: better as composite node
