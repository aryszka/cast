package main

import (
	"errors"
	"fmt"
	"github.com/aryszka/keyval"
	"log"
	"time"
)

const (
	castPrefix    = "cast"
	messagePrefix = "message"
	acceptTag     = "accept"
)

type Message keyval.Entry

type Connection interface {
	Send() chan<- *Message
	Receive() <-chan *Message
	Close()
}

type Listener <-chan Connection

type Node interface {
	Listen(Listener) error
	Join(Connection) error
	Send() chan<- *Message
	Receive() <-chan *Message
	Errors() <-chan error
	Close()
}

type messageChannel chan *Message

type buffer struct {
    send chan *Message
    connection Connection
}

type inProcConnection struct {
    local chan *Message
    remote *inProcConnection
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
    opt Opt
	listen             chan listen
	listener           Listener
	childConnections   []Connection
	removeConnection   chan Connection
	join               chan join
	parentConnection   Connection
	send               chan<- *Message
	sender               <-chan *Message
	parentReceive      <-chan *Message
	childReceive       chan incoming
	receive            <-chan *Message
	receiver            chan<- *Message
	errors             chan error
	close              chan struct{}
}

type Opt struct {
    ParentTimeout time.Duration
    ChildTimeout time.Duration
    SendTimeout time.Duration
    ReceiveTimeout time.Duration
    ParentBuffer int
    ChildBuffer int
    SendBuffer int
    ReceiveBuffer int
    ErrorBuffer int
}

var (
	ErrDisconnected = errors.New("disconnected")
	ErrNodeClosed   = errors.New("node closed")
	ErrCannotListen = errors.New("node cannot listen")
)

func (c messageChannel) Send() chan<- *Message { return c }
func (c messageChannel) Receive() <-chan *Message { return c }
func (c messageChannel) Close() { close(c) }

func newBuffer(c Connection, size int, timeout time.Duration) Connection {
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

func (b *buffer) Send() chan<- *Message { return b.send }
func (b *buffer) Receive() <-chan *Message { return b.connection.Receive() }
func (b *buffer) Close() { close(b.send) }

func newBufferedConnection(size int, timeout time.Duration) Connection {
    if timeout <= 0 {
        return make(messageChannel, size)
    }

    return newBuffer(make(messageChannel), size, timeout)
}

func newInProcConnection(l chan Connection) Connection {
	local := &inProcConnection{local: make(chan *Message)}
	remote := &inProcConnection{local: make(chan *Message)}
	local.remote = remote
	remote.remote = local
	l <- remote
	return local
}

func (c *inProcConnection) Send() chan<- *Message { return c.local }
func (c *inProcConnection) Receive() <-chan *Message { return c.remote.local }
func (c *inProcConnection) Close() { close(c.local) }

func NewNode(o Opt) Node {
	n := &node{
        opt: o,
		listen:  make(chan listen),
		join:    make(chan join),
		errors:  make(chan error, o.ErrorBuffer),
		close:   make(chan struct{})}

    s := newBufferedConnection(o.SendBuffer, o.SendTimeout)
    n.send = s.Send()
    n.sender = s.Receive()

    r := newBufferedConnection(o.ReceiveBuffer, o.ReceiveTimeout)
    n.receive = r.Receive()
    n.receiver = r.Send()

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
		m, open := <-c.Receive()
        switch {
        case !open:
            n.removeConnection <- c
            return
        default:
            n.childReceive <- incoming{m, c}
        }
	}
}

func (n *node) addChildConnection(c Connection) {
    if n.opt.ChildBuffer > 0 || n.opt.ChildTimeout > 0 {
        c = newBuffer(c, n.opt.ChildBuffer, n.opt.ChildTimeout)
    }

	n.childConnections = append(n.childConnections, c)
	go n.watchChildConnection(c)
}

func (n *node) removeChildConnection(c Connection) {
    c.Close()
	cc := n.childConnections
	for i, ci := range cc {
		if ci == c {
			cc, cc[len(cc)-1] = append(cc[:i], cc[i+1:]...), nil
			n.childConnections = cc
			break
		}
	}
}

func (n *node) joinNetwork(c Connection, ec chan error) {
	pc := n.parentConnection
	if pc != nil {
		pc.Close()
	}

    if n.opt.ParentBuffer > 0 || n.opt.ParentTimeout > 0 {
        c = newBuffer(c, n.opt.ParentBuffer, n.opt.ParentTimeout)
    }

	n.parentConnection = c
	n.parentReceive = c.Receive()
	ec <- nil
}

func (n *node) parentMessage(m *Message) {
	if isPayload(m.Key) {
		n.forwardMessage(m, false, n.parentConnection)
	}
}

func sendTo(c Connection, m *Message) {
    c.Send() <- m
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
		sendTo(n.parentConnection, m)
	}

	for _, c := range n.childConnections {
		if !skip(c) {
			sendTo(c, m)
		}
	}

	if !skipNode {
		key := m.Key
		if isPayload(key) {
			key = key[1:]
		}

		n.receiver <- &Message{Key: key, Val: m.Val, Comment: m.Comment}
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
		c.Close()
	}

    n.childConnections = nil
}

func (n *node) closeNode() {
	if n.parentConnection != nil {
		n.parentConnection.Close()
        n.parentReceive = nil
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
		case m := <-n.sender:
			n.sendMessage(m)
		case m, open := <-n.parentReceive:
            if !open {
                n.errors <- ErrDisconnected
                n.parentReceive = nil
            } else {
                n.parentMessage(m)
            }
		case im := <-n.childReceive:
            if isPayload(im.message.Key) {
                n.forwardMessage(im.message, false, im.connection)
            }
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

func (n *node) Send() chan<- *Message { return n.send }
func (n *node) Receive() <-chan *Message { return n.receive }
func (n *node) Errors() <-chan error { return n.errors }
func (n *node) Close() { close(n.close) }

func createNode(label string) Node {
	n := NewNode(Opt{})
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
	n.Send() <- &Message{Val: message}
}

func sendHelloWorld(n Node, label string) {
	sendMessage(n, label, "Hello, world!")
}

func closeAll(n ...Node) {
	closed := make(chan error)
	for _, ni := range n {
		go func(n Node) {
			for {
				err := <-n.Errors()
				if err == ErrNodeClosed {
					closed <- err
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
	time.Sleep(time.Millisecond)
	sendHelloWorld(np, "parent")
	time.Sleep(time.Millisecond)
	sendHelloWorld(nc1, "child 1")
	time.Sleep(time.Millisecond)

	closeAll(np)
}

// refactor
// test all
// self healing network
// sockets
// write a cmd client
// consider handling states
