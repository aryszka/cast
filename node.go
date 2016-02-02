package main

import "time"

const messagePrefix = "message"

type controlType int

const (
	none controlType = iota
	listen
	join
	removeChild
	childMessage
)

type control struct {
	typ        controlType
	listener   Listener
	connection Connection
	message    *Message
	err        chan error
}

type node struct {
	opt           Opt
	control       chan control
	listener      Listener
	children      []Connection
	parent        Connection
	parentReceive Receiver
	send          Connection
	receive       Connection
	errors        chan error
}

type Opt struct {
	ParentTimeout  time.Duration
	ChildTimeout   time.Duration
	SendTimeout    time.Duration
	ReceiveTimeout time.Duration
	ParentBuffer   int
	ChildBuffer    int
	SendBuffer     int
	ReceiveBuffer  int
	ErrorBuffer    int
}

func NewNode(o Opt) Node {
	n := &node{
		opt:     o,
		control: make(chan control),
		send:    NewBufferedConnection(o.SendBuffer, o.SendTimeout),
		receive: NewBufferedConnection(o.ReceiveBuffer, o.ReceiveTimeout),
		errors:  make(chan error, o.ErrorBuffer)}
	go n.run()
	return n
}

func isPayload(key []string) bool {
	return len(key) > 0 && key[0] == messagePrefix
}

func (n *node) dispatchMessage(m *Message, omitNode bool, omit ...Connection) {
	omitContains := func(c Connection) bool {
		for _, ci := range omit {
			if c == ci {
				return true
			}
		}

		return false
	}

	if n.parent != nil && !omitContains(n.parent) {
		n.parent.Send() <- m
	}

	for _, c := range n.children {
		if !omitContains(c) {
			c.Send() <- m
		}
	}

	if !omitNode {
		key := m.Key
		if isPayload(key) {
			key = key[1:]
		}

		n.receive.Send() <- &Message{Key: key, Val: m.Val, Comment: m.Comment}
	}
}

func (n *node) incomingMessage(c Connection, m *Message) {
	if isPayload(m.Key) {
		n.dispatchMessage(m, false, c)
	}
}

func (n *node) sendMessage(m *Message) {
	n.dispatchMessage(&Message{
		Key:     append([]string{messagePrefix}, m.Key...),
		Val:     m.Val,
		Comment: m.Comment}, true)
}

func (n *node) listen(l Listener) error {
	if n.listener != nil {
		return ErrCannotListen
	}

	n.listener = l
	return nil
}

func (n *node) receiveFromChild(c Connection) {
	for {
		m, open := <-c.Receive()
		switch {
		case !open:
			n.control <- control{typ: removeChild, connection: c}
			return
		default:
			n.control <- control{typ: childMessage, connection: c, message: m}
		}
	}
}

func (n *node) addChild(c Connection) {
	if n.opt.ChildBuffer > 0 || n.opt.ChildTimeout > 0 {
		c = NewBuffer(c, n.opt.ChildBuffer, n.opt.ChildTimeout)
	}

	n.children = append(n.children, c)
	go n.receiveFromChild(c)
}

func (n *node) removeChild(c Connection) {
	c.Close()
	cc := n.children
	for i, ci := range cc {
		if ci == c {
			cc, cc[len(cc)-1] = append(cc[:i], cc[i+1:]...), nil
			n.children = cc
			break
		}
	}
}

func (n *node) closeChildren() {
	for _, c := range n.children {
		c.Close()
	}

	n.children = nil
}

func (n *node) join(c Connection) {
	pc := n.parent
	if pc != nil {
		pc.Close()
	}

	if n.opt.ParentBuffer > 0 || n.opt.ParentTimeout > 0 {
		c = NewBuffer(c, n.opt.ParentBuffer, n.opt.ParentTimeout)
	}

	n.parent = c
	n.parentReceive = c.Receive()
}

func (n *node) closeNode() {
	if n.parent != nil {
		n.parent.Close()
		n.parentReceive = nil
	}

	n.receive.Close()
	n.closeChildren()
}

func (n *node) receiveControl(c control) {
	switch c.typ {
	case listen:
		c.err <- n.listen(c.listener)
	case join:
		n.join(c.connection)
	case removeChild:
		n.removeChild(c.connection)
	case childMessage:
		n.incomingMessage(c.connection, c.message)
	}
}

func (n *node) run() {
	for {
		select {
		case c := <-n.control:
			n.receiveControl(c)
		case c := <-n.listener:
			n.addChild(c)
		case m, open := <-n.send.Receive():
			if !open {
				n.closeNode()
				return
			}

			n.sendMessage(m)
		case m, open := <-n.parentReceive:
			if !open {
				n.errors <- ErrDisconnected
				n.parentReceive = nil
			} else {
				n.incomingMessage(n.parent, m)
			}
		}
	}
}

func (n *node) Listen(l Listener) error {
	ec := make(chan error)
	n.control <- control{typ: listen, listener: l, err: ec}
	return <-ec
}

func (n *node) Join(c Connection) {
	n.control <- control{typ: join, connection: c}
}

func (n *node) Send() Sender         { return n.send.Send() }
func (n *node) Receive() Receiver    { return n.receive.Receive() }
func (n *node) Errors() <-chan error { return n.errors }
func (n *node) Close()               { n.send.Close() }
