package main

import (
	"errors"
	"fmt"
	"github.com/aryszka/keyval"
	"log"
	"time"
)

const (
	joinTimeout   = time.Second
	castPrefix    = "cast"
	messagePrefix = "message"
	joinTag       = "join"
	acceptTag     = "accept"
)

type Message keyval.Entry

type Connection interface {
	Send(*Message) error
	Receive() <-chan *Message
	Disconnect()
	Disconnected() <-chan struct{}
}

type Listener <-chan Connection

type Node interface {
	Listen(Listener) error
	Join(Connection) error
	Send(*Message)
	Receive() <-chan *Message
	Errors() <-chan error
	Close()
}

type inProcConnection struct {
	receiver     chan *Message
	remote       *inProcConnection
	disconnected chan struct{}
}

type listen struct {
	listener Listener
	err      chan error
}

type join struct {
	connection Connection
	err        chan error
}

type incoming struct {
	message    *Message
	connection Connection
}

type node struct {
	listen             chan listen
	listener           Listener
	childConnections   []Connection
	removeConnection   chan Connection
	join               chan join
	parentConnection   Connection
	parentDisconnected <-chan struct{}
	send               chan *Message
	parentReceive      <-chan *Message
	childReceive       chan incoming
	receive            chan *Message
	errors             chan error
	close              chan struct{}
}

var (
	keyJoin       = []string{castPrefix, joinTag}
	keyJoinAccept = []string{castPrefix, joinTag, acceptTag}

	msgJoin       = &Message{Key: keyJoin}
	msgJoinAccept = &Message{Key: keyJoinAccept}
)

var (
	ErrDisconnected = errors.New("disconnected")
	ErrJoinTimeout  = errors.New("join timeout")
	ErrNodeClosed   = errors.New("node closed")
	ErrCannotListen = errors.New("node cannot listen")
)

func newInProcConnection(l chan Connection) *inProcConnection {
	local := &inProcConnection{receiver: make(chan *Message), disconnected: make(chan struct{})}
	remote := &inProcConnection{receiver: make(chan *Message), disconnected: make(chan struct{})}
	local.remote = remote
	remote.remote = local
	l <- remote
	return local
}

func (c *inProcConnection) Send(m *Message) error {
	select {
	case <-c.disconnected:
		return ErrDisconnected
	case c.remote.receiver <- m:
		return nil
	}
}

func (c *inProcConnection) Disconnect() {
	select {
	case <-c.disconnected:
	default:
		close(c.disconnected)
		c.remote.Disconnect()
	}
}

func (c *inProcConnection) Receive() <-chan *Message      { return c.receiver }
func (c *inProcConnection) Disconnected() <-chan struct{} { return c.disconnected }

func NewNode() Node {
	n := &node{
		listen:  make(chan listen),
		join:    make(chan join),
		send:    make(chan *Message),
		receive: make(chan *Message),
		errors:  make(chan error),
		close:   make(chan struct{})}
	go n.run()
	return n
}

func (n *node) startListening(l Listener, ec chan error) {
	if n.listener != nil {
		ec <- ErrCannotListen
		return
	}

	n.listener = l
	n.removeConnection = make(chan Connection)
	n.childReceive = make(chan incoming)
	ec <- nil
}

func isPayload(key []string) bool {
	return len(key) > 0 && key[0] == messagePrefix
}

func (n *node) watchChildConnection(c Connection) {
	for {
		select {
		case <-c.Disconnected():
			n.removeConnection <- c
			return
		case m := <-c.Receive():
			switch {
			case keyval.KeyEq(m.Key, keyJoin):
				sendTo(c, msgJoinAccept, n.errors)
			case isPayload(m.Key):
				n.childReceive <- incoming{m, c}
			}
		}
	}
}

func (n *node) addChildConnection(c Connection) {
	n.childConnections = append(n.childConnections, c)
	go n.watchChildConnection(c)
}

func (n *node) removeChildConnection(c Connection) {
	cc := n.childConnections
	for i, ci := range cc {
		if ci == c {
			cc, cc[len(cc)-1] = append(cc[:i], cc[i+1:]...), nil
			n.childConnections = cc
			break
		}
	}
}

func waitMessage(c Connection, key []string, timeout time.Duration) (*Message, error) {
	to := time.After(timeout)
	for {
		select {
		case <-c.Disconnected():
			return nil, ErrDisconnected
		case <-to:
			return nil, ErrJoinTimeout
		case m := <-c.Receive():
			if keyval.KeyEq(m.Key, key) {
				return m, nil
			}
		}
	}
}

func (n *node) joinNetwork(c Connection, ec chan error) {
	pc := n.parentConnection
	if pc != nil {
		pc.Disconnect()
	}

	if !sendTo(c, msgJoin, ec) {
		return
	}

	_, err := waitMessage(c, keyJoinAccept, joinTimeout)
	if err != nil {
		ec <- err
		return
	}

	n.parentConnection = c
	n.parentReceive = c.Receive()
	n.parentDisconnected = c.Disconnected()
	ec <- nil
}

func (n *node) parentMessage(m *Message) {
	if isPayload(m.Key) {
		n.forwardMessage(m, false, n.parentConnection)
	}
}

func sendTo(c Connection, m *Message, ec chan error) bool {
	err := c.Send(m)
	if err != nil {
		ec <- err
		return false
	}

	return true
}

func (n *node) forwardMessage(m *Message, skipNode bool, skipConnections ...Connection) {
	skip := func(c Connection) bool {
		for _, ci := range skipConnections {
			if c == ci {
				return true
			}
		}

		return false
	}

	if n.parentConnection != nil && !skip(n.parentConnection) {
		sendTo(n.parentConnection, m, n.errors)
	}

	for _, c := range n.childConnections {
		if !skip(c) {
			sendTo(c, m, n.errors)
		}
	}

	if !skipNode {
		key := m.Key
		if isPayload(key) {
			key = key[1:]
		}

		n.receive <- &Message{Key: key, Val: m.Val, Comment: m.Comment}
	}
}

func (n *node) sendMessage(m *Message) {
	n.forwardMessage(&Message{
		Key:     append([]string{messagePrefix}, m.Key...),
		Val:     m.Val,
		Comment: m.Comment}, true)
}

func (n *node) disconnectAllChildren() {
	for _, c := range n.childConnections {
		c.Disconnect()
	}

	for {
		c := <-n.removeConnection
		n.removeChildConnection(c)
		if len(n.childConnections) == 0 {
			return
		}
	}
}

func (n *node) closeNode() {
	pc := n.parentConnection
	if pc != nil {
		pc.Disconnect()
	}

	n.errors <- ErrNodeClosed
	n.disconnectAllChildren()
}

func (n *node) run() {
	for {
		select {
		case lm := <-n.listen:
			n.startListening(lm.listener, lm.err)
		case c := <-n.listener:
			n.addChildConnection(c)
		case c := <-n.removeConnection:
			n.removeChildConnection(c)
		case jm := <-n.join:
			n.joinNetwork(jm.connection, jm.err)
		case m := <-n.send:
			n.sendMessage(m)
		case m := <-n.parentReceive:
			n.parentMessage(m)
		case im := <-n.childReceive:
			n.forwardMessage(im.message, false, im.connection)
		case <-n.parentDisconnected:
			n.errors <- ErrDisconnected
		case <-n.close:
			n.closeNode()
			return
		}
	}
}

func (n *node) Listen(l Listener) error {
	ec := make(chan error)
	select {
	case <-n.close:
		return ErrNodeClosed
	case n.listen <- listen{l, ec}:
		return <-ec
	}
}

func (n *node) Join(c Connection) error {
	ec := make(chan error)
	select {
	case <-n.close:
		return ErrNodeClosed
	case n.join <- join{c, ec}:
		return <-ec
	}
}

func (n *node) Send(m *Message) {
	select {
	case <-n.close:
	case n.send <- m:
	}
}

func (n *node) Receive() <-chan *Message {
	return n.receive
}

func (n *node) Errors() <-chan error { return n.errors }

func (n *node) Close() {
	select {
	case <-n.close:
	default:
		close(n.close)
	}
}

func createNode(label string) Node {
	n := NewNode()
	go func() {
		for {
			m := <-n.Receive()
			fmt.Printf("%-9s: %s\n", label, m.Val)
		}
	}()

	return n
}

func createChildNode(listener chan Connection, label string) Node {
	c := newInProcConnection(listener)
	n := createNode(label)
	err := n.Join(c)
	if err != nil {
		log.Fatal(err)
	}

	return n
}

func sendMessage(n Node, label, message string) {
	fmt.Printf("\nsending message to: %s\n", label)
	n.Send(&Message{Val: message})
	time.Sleep(time.Millisecond)
}

func sendHelloWorld(n Node, label string) {
	sendMessage(n, label, "Hello, world!")
}

func closeAll(n ...Node) {
	closed := make(chan struct{})
	for _, ni := range n {
		go func(n Node) {
			for {
				err := <-n.Errors()
				if err == ErrNodeClosed {
					closed <- struct{}{}
					return
				}
			}
		}(ni)

		ni.Close()
	}

	c := len(n)
	for {
		<-closed
		c--
		if c == 0 {
			return
		}
	}
}

func main() {
	listener := make(chan Connection)
	np := createNode("parent")
	np.Listen(listener)

	nc0 := createChildNode(listener, "child 0")
	nc1 := createChildNode(listener, "child 1")

	sendHelloWorld(nc0, "child 0")
	sendHelloWorld(np, "parent")
	sendHelloWorld(nc1, "child 1")

	closeAll(nc0, np, nc1)
}

// verify all blocking cases
// verify where timeout is required
// refactor
// test all
// self healing network
// sockets
// write a cmd client
// consider handling states
