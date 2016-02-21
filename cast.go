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

the in-proc types are to be used with care, because the go channel and
goroutine primitives provide a better representation of the same concept

unifies in-process and network communication
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
//
// closing by closing sender
// detecting disconnect by receiving closed
type Connection interface {
	Send() chan<- Message
	Receive() <-chan Message
}

// shall we add errors? no: should be handled by the creator
// stop listening by closing this channel
// node should also close the child connections
type Listener interface {
	Connections() <-chan Connection
}

// represents an interface where it is possible to connect to
// how to extract an interface from a message?
type Interface interface {
	Connect() (Connection, error)
}

// ok, this is becoming enterprisy, stop here.
// the reality cannot be like this
type InterfaceTranslation interface {
	Translate(Message) Interface
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
// nodes are designed to be composable. they can add up to new node types,
// or whole networks can represent a single node. provide examples of
// composition primitives.
// disconnected node blocking or non-blocking
// node without any connections, parent or not, blocing or non-blocking
// node cannot be blocking by default, because it can be a leaf node
type Node interface {
	Connection
	Join(Connection)
	Listen(Listener)
	Error() <-chan error
}

var (
	// error sent when parent is disconnected
	ErrDisconnected = errors.New("disconnected")

	// error sent when active listener is disconnected
	ErrListenerDisconnected = errors.New("listener disconnected")
)

// timeout: either remove from node or make it timeout when not dispatched, maybe
// unify buffering, and let it up to the composition, how it is handled for different connections, maybe
// maybe put a single buffered timeout connection as a buffer inside the node
// node error routine never exits
// should the node have an error at all? should the connection have an error?
// no need to buffer the connection, the node should be buffered
// one blocking connection cannot block other connections in a node, because that would block whole segments of
// a network
// ergo buffering is a must, timeout is a consequence, preserving system health needs to be implemented
// node needs an internal outbox: default buffer size?
// there should be no default buffer size
// document: since wanting to avoid that one connection of a node blocks the others, cannot make sure that a
// message reaches all connections. parent is distinguished only for knowing that the node is part of the
// network. send blocks until at least one connection
// document: similarities and differences between node and go channel communication
// error cannot be blocking. simple goroutine is enough, but document that it leads to leaks not handling it.
// what else should the error channel be used for, and how during composition
// self healing network
// - circular connections: by enforcing tree structure or marking messages with sender address
// - what does address translation mean for this, how to identify a node?
// - message loss from parent to child
// - need a concept of the address, address space
// sockets
// document all
// write a cmd client
// it is possible to implement full network recovery by introducing the concept of a network
