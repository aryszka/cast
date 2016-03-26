package cast

import (
    "time"
    "errors"
)

type recoveryNode struct {
    node Node
    timeout time.Duration
    parents []Interface
    err chan error
    incoming Connection
}

type RecoveryOpt struct {
    MessageBuffer int
    MessageTimeout time.Duration
    RecoveryTimeout time.Duration
    Parents []Interface
}

var ErrRecoveryFailed = errors.New("recovery to all connections failed")

func NewRecoveryNode(o RecoveryOpt) Node {
    n := &recoveryNode{
        node: NewNode(o.MessageBuffer, o.MessageTimeout),
        timeout: o.RecoveryTimeout,
        parents: o.Parents,
        err: make(chan error),
        incoming: make(MessageChannel)}
    go n.runRecovery()
    return n
}

func (n *recoveryNode) connect() bool {
    for i, p := range n.parents {
        if c, err := p.Connect(); err == nil {
            n.node.Join(c)
            n.parents = append(n.parents[i + 1:], n.parents[:i + 1]...)
            return true
        }
    }

    println("connection failed")
    go func() { n.err <- ErrRecoveryFailed }()
    return false
}

func (n *recoveryNode) runRecovery() {
    connected := false
    incoming := newRelay(n.node, n.incoming)

    for {
        if !connected && len(n.parents) > 0 {
            connected = n.connect()
        }

        select {
        case err := <-n.node.Error():
            if err == ErrDisconnected && len(n.parents) > 0 {
                println("disconnected")
                connected = false
            } else {
                go func() { n.err <- err }()
            }
        case m, open := <-incoming.receive():
            incoming.received(m, open)
            if !open {
                return
            }
        case incoming.send() <- incoming.message():
            incoming.sent()
        }
    }
}

func (n *recoveryNode) Send() chan<- Message { return n.incoming.Send() }
func (n *recoveryNode) Receive() <-chan Message { return n.node.Receive() }
func (n *recoveryNode) Join(c Connection) { n.node.Join(c) }
func (n *recoveryNode) Listen(l Listener) { n.node.Listen(l) }
func (n *recoveryNode) Error() <-chan error { return n.err }
