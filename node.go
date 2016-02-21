package cast

import "time"

type connControlType int

const (
	newOutgoing connControlType = iota
	cancelOutgoing
	closeNodeConn
)

type nodeControlType int

const (
	outgoingTimeout nodeControlType = iota
	connOutgoingDone
	nodeConnClosed
	joinParent
	listenChildren
)

type connControl struct {
	typ     connControlType
	message *outgoingMessage
}

type nodeConn chan<- *connControl

type nodeControl struct {
	typ      nodeControlType
	message  *outgoingMessage
	nodeConn nodeConn
	listener Listener
	conn     Connection
}

type incomingMessage struct {
	source  nodeConn
	message *Message
}

type outgoingMessage struct {
	message *Message
	conns   []nodeConn
	discard chan struct{}
}

type node struct {
	extern  Connection
	control chan *nodeControl
	err     chan error
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

func removeNodeConn(cs []nodeConn, c nodeConn) []nodeConn {
	for i, ci := range cs {
		if ci == c {
			cs, cs[len(cs)-1] = append(cs[:i], cs[i+1:]...), nil
			break
		}
	}

	return cs
}

func discardOutgoing(om *outgoingMessage) {
	close(om.discard)
	for _, conn := range om.conns {
		conn <- &connControl{typ: cancelOutgoing, message: om}
	}
}

func runConnection(
	c Connection,
	im chan<- *incomingMessage,
	nctl chan<- *nodeControl,
	control chan *connControl) {

	var (
		in         *incomingMessage
		out        *outgoingMessage
		outm       Message
		outbox     []*outgoingMessage
		receiver   <-chan Message
		fwdReceive chan<- *incomingMessage
		fwdSend    chan<- Message
		nodeQueue  []*nodeControl
		sendnc     chan<- *nodeControl
		nc         *nodeControl
	)

	for {
		// receive incoming from outside or forward it to the node.
		// when there is an incoming message to be forwarded,
		// block the receiver by setting it to nil.
		if in == nil {
			receiver = c.Receive()
			fwdReceive = nil
		} else {
			receiver = nil
			fwdReceive = im
		}

		// when there is something in the outbox, forward it out.
		// when there is nothing to send, block the sender
		// by setting it to nil.
		if out == nil && len(outbox) > 0 {
			fwdSend = c.Send()
			out = outbox[0]
			outbox = outbox[1:]
			outm = *out.message
		} else if out == nil {
			fwdSend = nil
		}

		// when there is a node control message in the queue,
		// set it to be sent.
		// when there is no node control message, block the
		// node control channel by setting it to nil.
		if nc == nil && len(nodeQueue) > 0 {
			nc = nodeQueue[0]
			nodeQueue = nodeQueue[1:]
			sendnc = nctl
		} else if nc == nil {
			sendnc = nil
		}

		// never block at other places than this select
		// to avoid blocking the main node process that
		// may send control messages anytime.
		select {
		case m, open := <-receiver:
			if open {
				in = &incomingMessage{control, &m}
			} else {
				nodeQueue = append(nodeQueue, &nodeControl{
					typ:      nodeConnClosed,
					nodeConn: control})
			}
		case fwdReceive <- in:
			in = nil
		case fwdSend <- outm:
			nodeQueue = append(nodeQueue, &nodeControl{
				typ:      connOutgoingDone,
				nodeConn: control,
				message:  out})
			out = nil
		case sendnc <- nc:
			nc = nil
		case ctl := <-control:
			switch ctl.typ {
			case newOutgoing:
				outbox = append(outbox, ctl.message)
			case cancelOutgoing:
				if out == ctl.message {
					out = nil
				} else {
					outbox = removeOutgoing(outbox, ctl.message)
				}
			case closeNodeConn:
				close(c.Send())
				return
			}
		}
	}
}

// process for communicating between the node and a single connection
func newNodeConn(c Connection, im chan<- *incomingMessage, nctl chan<- *nodeControl) nodeConn {
	control := make(chan *connControl)
	go runConnection(c, im, nctl, control)
	return control
}

func targetConns(source nodeConn, conns []nodeConn) []nodeConn {
	// don't send the message when not coming from an
	// existing connection, or there is no connection
	// to send it to. don't send the message to the
	// source connection.

	var (
		targetConns  []nodeConn
		existingConn bool
	)

	for _, c := range conns {
		if c != nil && c != source {
			targetConns = append(targetConns, c)
		}

		if c == source {
			existingConn = true
		}
	}

	if !existingConn {
		return nil
	}

	return targetConns
}

func waitTimeoutOrDiscard(om *outgoingMessage, timeout time.Duration, nctl chan<- *nodeControl) {
	var (
		timeoutc <-chan time.Time
		tout     *nodeControl
		sendctl  chan<- *nodeControl
	)

	if timeout > 0 {
		timeoutc = time.After(timeout)
		tout = &nodeControl{typ: outgoingTimeout, message: om}
	}

	for {
		select {
		case <-timeoutc:
			// block the nodeControl channel until timeout
			// by leaving it nil
			sendctl = nctl
		case sendctl <- tout:
			sendctl = nil
		case <-om.discard:
			return
		}
	}
}

func dispatchMessage(
	m *incomingMessage,
	timeout time.Duration,
	control chan<- *nodeControl,
	conns []nodeConn) *outgoingMessage {

	conns = targetConns(m.source, conns)
	if len(conns) == 0 {
		return nil
	}

	om := &outgoingMessage{
		message: m.message,
		conns:   conns,
		discard: make(chan struct{})}

	if timeout > 0 {
		go waitTimeoutOrDiscard(om, timeout, control)
	}

	for _, ci := range conns {
		ci <- &connControl{typ: newOutgoing, message: om}
	}

	return om
}

func findConnMessages(c nodeConn, ms []*outgoingMessage) []*outgoingMessage {
	var result []*outgoingMessage
	for _, om := range ms {
		for _, ci := range om.conns {
			if ci == c {
				result = append(result, om)
				break
			}
		}
	}

	return result
}

func closeNode(intern, parent nodeConn, children []nodeConn, outbox []*outgoingMessage) {
	cls := &connControl{typ: closeNodeConn}
	intern <- cls

	if parent != nil {
		parent <- cls
	}

	for _, ci := range children {
		ci <- cls
	}

	for _, m := range outbox {
		close(m.discard)
	}
}

func runNode(
	buffer int,
	timeout time.Duration,
	control chan *nodeControl,
	incoming chan *incomingMessage,
	ownConn nodeConn,
	err chan error) {

	var (
		receiveIncoming <-chan *incomingMessage
		outbox          []*outgoingMessage
		parent          nodeConn
		children        []nodeConn
		listen          <-chan Connection
	)

	for {
		// when the outbox is full, block all
		// incoming messages by setting the
		// incoming channel to nil.
		if len(outbox) > buffer {
			receiveIncoming = nil
		} else {
			receiveIncoming = incoming
		}

		select {
		case m := <-receiveIncoming:
			om := dispatchMessage(m, timeout, control,
				append([]nodeConn{ownConn, parent}, children...))
			if om != nil {
				outbox = append(outbox, om)
			}
		case c := <-control:
			switch c.typ {
			case outgoingTimeout:
				discardOutgoing(c.message)
				outbox = removeOutgoing(outbox, c.message)
				go func() { err <- &TimeoutError{*c.message.message} }()
			case connOutgoingDone:
				c.message.conns = removeNodeConn(c.message.conns, c.nodeConn)
				if len(c.message.conns) == 0 {
					discardOutgoing(c.message)
					outbox = removeOutgoing(outbox, c.message)
				}
			case nodeConnClosed:
				// closing the node's internal connection means that the node is closed
				if c.nodeConn == ownConn {
					closeNode(ownConn, parent, children, outbox)
					return
				}

				oms := findConnMessages(c.nodeConn, outbox)
				for _, om := range oms {
					om.conns = removeNodeConn(om.conns, c.nodeConn)
					if len(om.conns) == 0 {
						discardOutgoing(om)
						outbox = removeOutgoing(outbox, om)
					}
				}

				c.nodeConn <- &connControl{typ: closeNodeConn}
				if c.nodeConn == parent {
					parent = nil
					go func() { err <- ErrDisconnected }()
				} else {
					children = removeNodeConn(children, c.nodeConn)
				}
			case joinParent:
				if parent != nil {
					parent <- &connControl{typ: closeNodeConn}
				}

				parent = newNodeConn(c.conn, incoming, control)
			case listenChildren:
				if listen != nil {
					panic("already listening")
				}

				listen = c.listener.Connections()
			}
		case c, open := <-listen:
			if !open {
				listen = nil
				for _, c := range children {
					c <- &connControl{typ: closeNodeConn}
				}

				children = nil
				go func() { err <- ErrListenerDisconnected }()
			} else {
				children = append(children, newNodeConn(c, incoming, control))
			}
		}
	}
}

func NewNode(buffer int, timeout time.Duration) Node {
	intern, extern := NewInProcConnection()
	control := make(chan *nodeControl)
	incoming := make(chan *incomingMessage)
	ownConn := newNodeConn(intern, incoming, control)
	err := make(chan error)
	go runNode(buffer, timeout, control, incoming, ownConn, err)
	return &node{extern: extern, control: control, err: err}
}

func (n *node) Send() chan<- Message    { return n.extern.Send() }
func (n *node) Receive() <-chan Message { return n.extern.Receive() }
func (n *node) Join(c Connection)       { n.control <- &nodeControl{typ: joinParent, conn: c} }
func (n *node) Listen(l Listener)       { n.control <- &nodeControl{typ: listenChildren, listener: l} }
func (n *node) Error() <-chan error     { return n.err }
