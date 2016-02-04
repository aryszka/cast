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

func timeoutConnection(c Connection, timeout time.Duration, ec chan error) Connection {
	tc := NewTimeoutConnection(c, timeout)
	go func() {
		err := <-tc.Timeout
		ec <- err
	}()

	return tc
}

func timeoutBufferConnection(c Connection, buffer int, timeout time.Duration, ec chan error) Connection {
	if buffer > 0 {
		c = NewBufferedConnection(c, buffer)
	}

	if timeout > 0 {
		c = timeoutConnection(c, timeout, ec)
	}

	return c
}

func makeTimeoutBufferConnection(buffer int, timeout time.Duration, ec chan error) Connection {
	c := make(MessageChannel, buffer)
	if timeout <= 0 {
		return c
	}

	return timeoutConnection(c, timeout, ec)
}

func NewNode(o Opt) Node {
	err := make(chan error, o.ErrorBuffer)
	n := &node{
		opt:     o,
		control: make(chan control),
		send:    makeTimeoutBufferConnection(o.SendBuffer, o.SendTimeout, err),
		receive: makeTimeoutBufferConnection(o.ReceiveBuffer, o.ReceiveTimeout, err),
		errors:  err}
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

func (n *node) join(c Connection) {
	pc := n.parent
	if pc != nil {
		close(pc.Send())
	}

	if n.opt.ParentBuffer > 0 || n.opt.ParentTimeout > 0 {
		c = timeoutBufferConnection(c, n.opt.ParentBuffer, n.opt.ParentTimeout, n.errors)
	}

	n.parent = c
	n.parentReceive = c.Receive()
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
		c = timeoutBufferConnection(c, n.opt.ChildBuffer, n.opt.ChildTimeout, n.errors)
	}

	n.children = append(n.children, c)
	go n.receiveFromChild(c)
}

func (n *node) removeChild(c Connection) {
	close(c.Send())
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
		close(c.Send())
	}

	n.children = nil
}

func (n *node) closeNode() {
	if n.parent != nil {
		close(n.parent.Send())
		n.parentReceive = nil
	}

	close(n.receive.Send())
	n.closeChildren()
}

func (n *node) receiveControl(c control) {
	switch c.typ {
	case join:
		n.join(c.connection)
	case listen:
		c.err <- n.listen(c.listener)
	case childMessage:
		n.incomingMessage(c.connection, c.message)
	case removeChild:
		n.removeChild(c.connection)
	}
}

func (n *node) run() {
	for {
		select {
		case c := <-n.control:
			n.receiveControl(c)
		case m, open := <-n.parentReceive:
			if !open {
				n.errors <- ErrDisconnected
				n.parentReceive = nil
			} else {
				n.incomingMessage(n.parent, m)
			}
		case m, open := <-n.send.Receive():
			if !open {
				n.closeNode()
				return
			}

			n.sendMessage(m)
		case c := <-n.listener:
			n.addChild(c)
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
