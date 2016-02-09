/*
cast stands for continously applied state transfer
this rather only to add up to the nice sounding abbreviation

cast is about sharing and applying atomic pieces of information in
distributed systems as soon as they are available regardless of the full
state of the concepts they are describing

cast systems may not be consistent at any single moment in time,
measurable or not, but they should continously be convergent to a
consistent state

cast systems may provide consistency or consensus for a subset or the
entirety of the represented concepts, but the cast communication
primitives don't target consistency or consensus

this library provides a gossip style communication model and a carrier
network for cast based systems

other network topologies can still comply with the cast communication
model

close to notify failure
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

// objects representing one enpoint of a communcation channel
//
// blocking?
// closing?
// reconnection?
// errors? maybe in general, the errors should be handled by the
// implementation and initialization
//
// a simple channel actually can implement this interface
// one way
//
// it's an error to close the error channel
//
// error handling asynchronous
type Connection interface {
	Send() chan<- *Message
	Receive() <-chan *Message
	Error() <-chan error
}

// shall we add errors? no: should be handled by the creator
// stop listening by closing this channel
// node should also close the child connections
type Listener interface {
	Connections() <-chan Connection
}

// minimal implementation
// listen error? maybe it should be a panic
// many to many communication
// building block for network topography
// parent child system
// disconnected when parent connection disconnected
// closing by closing send
// join takes over ownership regarding closing
// listen takes over ownership of incoming connections
// it is an error to call listen twice without closing the listener channel
// first
// does not recieve its own messages
// parent is the point where a node joins a network of nodes
// takes over error reporting from connections
type Node interface {
	Connection
	Join(Connection)
	Listen(Listener)
}

// error sent when parent is disconnected
var ErrDisconnected = errors.New("disconnected")

// benchmark all
// self healing network
// sockets
// document all
// write a cmd client
