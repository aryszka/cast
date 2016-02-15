package cast

import "time"

type nodeConnControlType int

const (
	newOutgoing nodeConnControlType = iota
	cancelOutgoing
	closeNodeConn
)

type nodeControlType int

const (
	outgoingTimeout nodeControlType = iota
	connOutgoingDone
	nodeConnClosed
)

type nodeConnControl struct {
	typ     nodeConnControlType
	message *outgoingMessage
	receive bool
}

type nodeControl struct {
	typ     nodeControlType
	message *outgoingMessage
	conn    *nodeConn
}

type incomingMessage struct {
	source  *nodeConn
	message *Message
}

type outgoingMessage struct {
	message     *Message
	timeout     <-chan time.Time
	nodeControl chan<- *nodeControl
	done        chan struct{}
	conns       []*nodeConn
}

type nodeConn struct {
	conn        Connection
	control     chan *nodeConnControl
	nodeControl chan<- *nodeControl
	incoming    chan<- *incomingMessage
	currentIn   *incomingMessage
	currentOut  *outgoingMessage
	outbox      []*outgoingMessage
}

type node struct {
	buffer   int
	timeout  time.Duration
	intern   *nodeConn
	extern   Connection
	parent   *nodeConn
	children []*nodeConn
	incoming chan *incomingMessage
	outbox   []*outgoingMessage
	control  chan *nodeControl
	err      chan error
}

func removeNodeConn(cs []*nodeConn, c *nodeConn) []*nodeConn {
	for i, ci := range cs {
		if ci == c {
			cs, cs[len(cs)-1] = append(cs[:i], cs[i+1:]...), nil
			break
		}
	}

	return cs
}

func (m *outgoingMessage) waitDone() {
	var (
		tonc    *nodeControl
		nodectl chan<- *nodeControl
	)

	for {
		if tonc == nil {
			nodectl = m.nodeControl
		} else {
			nodectl = nil
		}

		select {
		case <-m.timeout:
			tonc = &nodeControl{typ: outgoingTimeout, message: m}
		case nodectl <- tonc:
			tonc = nil
		case <-m.done:
			return
		}
	}
}

func newNodeConn(c Connection, im chan<- *incomingMessage, nctl chan<- *nodeControl) *nodeConn {
	nc := &nodeConn{
		conn:        c,
		control:     make(chan *nodeConnControl),
		incoming:    im,
		nodeControl: nctl}
	go nc.sendReceive()
	return nc
}

func (nc *nodeConn) relayReceive() (receiver <-chan Message, fwdReceive chan<- *incomingMessage) {
	if nc.currentIn == nil {
		receiver = nc.conn.Receive()
		fwdReceive = nil
	} else {
		receiver = nil
		fwdReceive = nc.incoming
	}

	return
}

func (nc *nodeConn) relaySend() (currentm Message, fwdSend chan<- Message) {
	if nc.currentOut == nil && len(nc.outbox) > 0 {
		fwdSend = nc.conn.Send()
		nc.currentOut = nc.outbox[0]
		nc.outbox = nc.outbox[1:]
		currentm = *(nc.currentOut.message)
	} else if nc.currentOut == nil {
		fwdSend = nil
	}

	return
}

func (nc *nodeConn) nodeConnControl(c *nodeConnControl) {
	switch c.typ {
	case newOutgoing:
		nc.outbox = append(nc.outbox, c.message)
	case cancelOutgoing:
		if nc.currentOut == c.message {
			nc.currentOut = nil
		} else {
			nc.outbox = removeOutgoing(nc.outbox, c.message)
		}
	case closeNodeConn:
		close(nc.conn.Send())
	}
}

func (nc *nodeConn) sendReceive() {
	var (
		currentm     Message
		receiver     <-chan Message
		fwdReceive   chan<- *incomingMessage
		fwdSend      chan<- Message
		nodeControls []*nodeControl
		currnc       *nodeControl
		nodectl      chan<- *nodeControl
	)

	for {
		receiver, fwdReceive = nc.relayReceive()
		currentm, fwdSend = nc.relaySend()

		if currnc == nil && len(nodeControls) > 0 {
			currnc = nodeControls[0]
			nodeControls = nodeControls[1:]
			nodectl = nc.nodeControl
		} else if currnc == nil {
			nodectl = nil
		}

		select {
		case m, open := <-receiver:
			if open {
				nc.currentIn = &incomingMessage{nc, &m}
			} else {
				nodeControls = append(nodeControls, &nodeControl{typ: nodeConnClosed, conn: nc})
			}
		case fwdReceive <- nc.currentIn:
			nc.currentIn = nil
		case fwdSend <- currentm:
			nodeControls = append(nodeControls, &nodeControl{
				typ:     connOutgoingDone,
				conn:    nc,
				message: nc.currentOut})
			nc.currentOut = nil
		case nodectl <- currnc:
			currnc = nil
		case c := <-nc.control:
			nc.nodeConnControl(c)
			if c.typ == closeNodeConn {
				return
			}
		}
	}
}

func NewNode(buffer int, timeout time.Duration) Node {
	intern, extern := NewInProcConnection()
	control := make(chan *nodeControl)
	incoming := make(chan *incomingMessage)
	n := &node{
		buffer:   buffer,
		timeout:  timeout,
		intern:   newNodeConn(intern, incoming, control),
		extern:   extern,
		control:  control,
		incoming: incoming,
		err:      make(chan error)}
	go n.run()
	return n
}

func (n *node) dispatchMessage(m *incomingMessage) {
	var conns []*nodeConn

	if n.intern != m.source {
		conns = append(conns, n.intern)
	}

	if n.parent != nil && n.parent != m.source {
		conns = append(conns, n.parent)
	}

	for _, c := range n.children {
		if c != m.source {
			conns = append(conns, c)
		}
	}

	if len(conns) == 0 {
		return
	}

	om := &outgoingMessage{
		message:     m.message,
		nodeControl: n.control,
		conns:       conns}
	if n.timeout > 0 {
		om.timeout = time.After(n.timeout)
	}

	go om.waitDone()
	for _, ci := range conns {
		ci.control <- &nodeConnControl{typ: newOutgoing, message: om}
	}
}

func (n *node) relayIncoming() <-chan *incomingMessage {
	if len(n.outbox) > n.buffer {
		return nil
	}

	return n.incoming
}

func removeOutgoing(ms []*outgoingMessage, m *outgoingMessage) []*outgoingMessage {
	for i, mi := range ms {
		if mi == m {
			ms, ms[len(ms)-1] = append(ms[:i], ms[i+1:]...), nil
			break
		}
	}

	return ms
}

func (n *node) nodeControl(c *nodeControl) {
	switch c.typ {
	case outgoingTimeout:
		close(c.message.done)
		for _, conn := range c.message.conns {
			conn.control <- &nodeConnControl{typ: cancelOutgoing, message: c.message}
		}

		n.outbox = removeOutgoing(n.outbox, c.message)
	case connOutgoingDone:
		for i, ci := range c.message.conns {
			if ci == c.conn {
				c.message.conns, c.message.conns[len(c.message.conns)-1] =
					append(c.message.conns[:i], c.message.conns[i+1:]...), nil
				if len(c.message.conns) == 0 {
					close(c.message.done)
					n.outbox = removeOutgoing(n.outbox, c.message)
				}

				break
			}
		}
	case nodeConnClosed:
		for i, m := range n.outbox {
			for j, ci := range m.conns {
				if ci == c.conn {
					m.conns, m.conns[len(m.conns)-1] = append(m.conns[:j], m.conns[j+1:]...), nil
					if len(m.conns) == 0 {
						close(m.done)
						n.outbox, n.outbox[len(n.outbox)-1] = append(n.outbox[:i], n.outbox[i+1:]...), nil
					}

					break
				}
			}
		}

		c.conn.control <- &nodeConnControl{typ: closeNodeConn}

		if c.conn == n.intern {
			if n.parent != nil {
				n.parent.control <- &nodeConnControl{typ: closeNodeConn}
				n.parent = nil
			}

			for _, ci := range n.children {
				ci.control <- &nodeConnControl{typ: closeNodeConn}
			}

			n.children = nil

			for _, m := range n.outbox {
				close(m.done)
			}

			n.outbox = nil
		} else if c.conn == n.parent {
			n.err <- ErrDisconnected
			n.parent = nil
		} else {
			n.children = removeNodeConn(n.children, c.conn)
		}
	}
}

func (n *node) run() {
	for {
		select {
		case m := <-n.relayIncoming():
			n.dispatchMessage(m)
		case c := <-n.control:
			n.nodeControl(c)
		}
	}
}

func (n *node) Send() chan<- Message    { return n.extern.Send() }
func (n *node) Receive() <-chan Message { return n.extern.Receive() }
func (n *node) Join(c Connection)       { n.control <- &nodeControl{} }
func (n *node) Listen(l Listener)       { n.control <- &nodeControl{} }
func (n *node) Error() <-chan error     { return n.err }
