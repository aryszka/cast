package cast

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
}

type node struct {
	opt           Opt
	control       chan control
	connections   <-chan Connection
	children      []Connection
	parent        Connection
	parentReceive <-chan *Message
	parentError   <-chan error
	send          Connection
	receive       Connection
	errors        chan error
}

// Node init options
// timeouts should be used only when leaking messages is fine,
// otherwise buffers are recommended
type Opt struct {
	// send buffer to parent connection
	ParentBuffer int

	// send timeout to parent connection
	ParentTimeout time.Duration

	// send buffer to child connections
	ChildBuffer int

	// send timeout to child connections
	ChildTimeout time.Duration

	// buffer on Node.Send
	SendBuffer int

	// timeout on Node.Send
	SendTimeout time.Duration

	// buffer on Node.Receive
	ReceiveBuffer int

	// timeout on Node.Receive
	ReceiveTimeout time.Duration

	// buffer on Node.Error
	ErrorBuffer int
}

// wrap a connection with timeout and send timeout errors to ec
func connectionTimeout(c Connection, timeout time.Duration, ec chan error) Connection {
	tc := NewTimeoutConnection(c, timeout)
	go func() {
		err := <-tc.Error()
		ec <- err
	}()

	return tc
}

// if buffer and/or timeout > 0, wrap a connection with buffer and/or timeout
func timeoutBufferConnection(c Connection, buffer int, timeout time.Duration, ec chan error) Connection {
	if buffer > 0 {
		c = NewBufferedConnection(c, buffer)
	}

	if timeout > 0 {
		c = connectionTimeout(c, timeout, ec)
	}

	return c
}

// make a connection. if buffer and/or timeout > 0, wrap a connection with buffer and/or timeout
func makeTimeoutBufferConnection(buffer int, timeout time.Duration, ec chan error) Connection {
	var c Connection = make(MessageChannel, buffer)
	if timeout > 0 {
		c = connectionTimeout(c, timeout, ec)
	}

	return c
}

// create a Node with the provided options
func (o Opt) NewNode() Node {
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

// create a Node without buffering or timeout
func NewNode() Node { return Opt{}.NewNode() }

// dispatch a message
// if omitNode true, don't send it onto Node.Receive
// don't send it connections listed as to omit
func (n *node) dispatchMessage(m *Message, omitNode bool, omit ...Connection) {
	omitConnection := func(c Connection) bool {
		for _, ci := range omit {
			if c == ci {
				return true
			}
		}

		return false
	}

	if n.parent != nil && !omitConnection(n.parent) {
		n.parent.Send() <- m
	}

	for _, c := range n.children {
		if !omitConnection(c) {
			c.Send() <- m
		}
	}

	if !omitNode {
		n.receive.Send() <- m
	}
}

// join a parent connection
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
	n.parentError = c.Error()
}

// listen for child connections
func (n *node) listen(l Listener) error {
	if n.connections != nil {
		panic("already listening")
	}

	n.connections = l.Connections()
	return nil
}

// stops listening and closes child connections
func (n *node) stopListening() {
	n.connections = nil
	n.closeChildren()
}

// process a message from a child
// if the child connection was closed, remove it
func (n *node) receiveFromChild(c Connection) {
	for {
		select {
		case m, open := <-c.Receive():
			if open {
				n.control <- control{typ: childMessage, connection: c, message: m}
			} else {
				n.control <- control{typ: removeChild, connection: c}
				return
			}
		case err, open := <-c.Error():
			if open {
				n.errors <- err
			} else {
				panic("error channel closed")
			}
		}
	}
}

// adds a new child connection and starts receiving messages from it
func (n *node) addChild(c Connection) {
	if n.opt.ChildBuffer > 0 || n.opt.ChildTimeout > 0 {
		c = timeoutBufferConnection(c, n.opt.ChildBuffer, n.opt.ChildTimeout, n.errors)
	}

	n.children = append(n.children, c)
	go n.receiveFromChild(c)
}

// closes and removes a child connection
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

// closes all child connections
func (n *node) closeChildren() {
	for _, c := range n.children {
		close(c.Send())
	}

	n.children = nil
}

// closes the node, including the parent and child connections if any
func (n *node) closeNode() {
	if n.parent != nil {
		close(n.parent.Send())
		n.parentReceive = nil
	}

	close(n.receive.Send())
	n.closeChildren()
}

// processes a control message
func (n *node) receiveControl(c control) {
	switch c.typ {
	case join:
		n.join(c.connection)
	case listen:
		n.listen(c.listener)
	case childMessage:
		n.dispatchMessage(c.message, false, c.connection)
	case removeChild:
		n.removeChild(c.connection)
	}
}

// runs the node's main processing loop
func (n *node) run() {
	for {
		select {
		case c := <-n.control:
			n.receiveControl(c)
		case m, open := <-n.parentReceive:
			if open {
				n.dispatchMessage(m, false, n.parent)
			} else {
				n.errors <- ErrDisconnected
				n.parentReceive = nil
			}
		case err, open := <-n.parentError:
			if open {
				n.errors <- err
			} else {
				panic("error channel closed")
			}
		case m, open := <-n.send.Receive():
			if !open {
				n.closeNode()
				return
			}

			n.dispatchMessage(m, true)
		case c, open := <-n.connections:
			if !open {
				n.stopListening()
			} else {
				n.addChild(c)
			}
		}
	}
}

func (n *node) Listen(l Listener)        { n.control <- control{typ: listen, listener: l} }
func (n *node) Join(c Connection)        { n.control <- control{typ: join, connection: c} }
func (n *node) Send() chan<- *Message    { return n.send.Send() }
func (n *node) Receive() <-chan *Message { return n.receive.Receive() }
func (n *node) Error() <-chan error      { return n.errors }
