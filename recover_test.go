package cast

import (
    "testing"
    "time"
)

type tiResponse struct {
    conn Connection
    err error
}

type testInterface struct {
    connections chan Connection
}

var errTestInterface = errors.New("test interface disabled")

func split(Connection c) (Connection, Connection) {
}

func join(left, right Connection) Connection {
}

func tap(left, right Connection) (Connection, Connection, Connection) {
}

func newTestInterface() *testInterface {
    enabled := true
    connections := make(chan Connection)
    connect := make(chan chan tiResponse)

    go func() {
        var (
            conn Connection
            outbox []Connection
            connsOut chan<- Connection
        )

        if len(outbox) > 0 && conn == nil {
            conn = conns[0]
            outbox = outbox[1:]
            connsOut = connections
        }

        for {
            select {
            case e := <-enabledChan:
                if !e && enabled {
                    // how to disconnect existing connections?
                    // need a test breakable connection
                }
            case connsOut <- conn:
                conn = nil
                connsOut = nil
            case req := <-connect:
                if enabled {
                    req <- tiResponse{err: errTestInterface}
                } else {
                    local, remote := NewInProcConnection()
                    outbox = append(outbox, local)
                    req <- tiResponse{conn: remote}
                }
            }
        }
    }()

    return &testInterface{connections}
}

func (i *testInterface) disable() {
    i.enabled <- false
}

func (i *testInterface) enable() {
    i.enabled <- true
}

func (i *testInterface) Connect() (Connection, error) {
    req := make(chan tiResponse)
    i.req <- req
    rsp := <-req
    return rsp.conn, rsp.err
}

func (i *testInterface) Connections() <-chan Connection {
    return i.connections
}

func TestRecoverTakesOptions(t *testing.T) {
    c := make(MessageChannel)
    n := NewRecoveryNode(RecoveryOpt{MessageTimeout: time.Millisecond})
    n.Join(c)

    go func() { n.Send() <- Message{} }()
    go func() { n.Send() <- Message{} }()
    go func() { n.Send() <- Message{} }()
    go func() { n.Send() <- Message{} }()

    select {
    case err := <-n.Error():
        if _, ok := err.(*TimeoutError); !ok {
            t.Error("failed to pass options, invalid error")
        }
    case <-time.After(120 * time.Millisecond):
        t.Error("failed to pass options, unexpected timeout")
    }
}

func TestRecoverInitialConnectOne(t *testing.T) {
    n1 := NewRecoveryNode(RecoveryOpt{})
    l1 := make(InProcListener)
    n1.Listen(l1)

    n2 := NewRecoveryNode(RecoveryOpt{Parents: []Interface{l1}})

    min := Message{Val: "Hello, world!"}
    n2.Send() <- min

    select {
    case mout := <-n1.Receive():
        if mout.Val != min.Val {
            t.Error("failed to forward message")
        }
    case <-time.After(120 * time.Millisecond):
        t.Error("forwarding timeout")
    }
}

func TestRecoverActAsAnyNode(t *testing.T) {
    n1 := NewRecoveryNode(RecoveryOpt{})
    l1 := make(InProcListener)
    n1.Listen(l1)

    c, err := l1.Connect()
    if err != nil {
        t.Error(err)
        return
    }

    n2 := NewRecoveryNode(RecoveryOpt{})
    n2.Join(c)

    testTimeout(t, func() {
        n1.Send() <- Message{}
        <-n2.Receive()
    })

    testTimeout(t, func() {
        n2.Send() <- Message{}
        <-n1.Receive()
    })

    testTimeout(t, func() {
        n1.Send() <- Message{}
        <-n2.Receive()
    })

    close(n1.Send())
    testTimeout(t, func() { <-n2.Error() })
    time.Sleep(3 * time.Millisecond)
}

func TestRecoverSendRecieveMessages(t *testing.T) {
    tn := NewNode(0, 0)
    ln := make(InProcListener)
    tn.Listen(ln)

    p := NewNode(0, 0)
    go receiveAll(p)
    c, err := ln.Connect()
    if err != nil {
        t.Error(err)
    }

    p.Join(c)
    i := make(InProcListener)
    p.Listen(i)

    rn := NewRecoveryNode(RecoveryOpt{Parents: []Interface{i}})

    testTimeout(t, func() {
        rn.Send() <- Message{}
        <-tn.Receive()
    })

    testTimeout(t, func() {
        tn.Send() <- Message{}
        <-rn.Receive()
    })

    testTimeout(t, func() {
        rn.Send() <- Message{}
        <-tn.Receive()
    })
}

func TestRecoverDoRecover(t *testing.T) {
    tn := NewNode(0, 0)
    ln := make(InProcListener)
    tn.Listen(ln)

    recoveryParent := func() (Node, Interface) {
        n := NewNode(0, 0)
        go receiveAll(n)
        c, err := ln.Connect()
        if err != nil {
            t.Error(err)
        }

        n.Join(c)
        l := make(InProcListener)
        n.Listen(l)
        return n, l
    }

    _, i1 := recoveryParent()
    _, i2 := recoveryParent()
    _, i3 := recoveryParent()

    rn := NewRecoveryNode(RecoveryOpt{Parents: []Interface{i1, i2, i3}})

    testTimeout(t, func() {
        for {
            select {
            case rn.Send() <- Message{}:
            case <-tn.Receive():
                return
            }
        }
    })

    close(p1.Send())

    time.Sleep(999 * time.Millisecond)
    testTimeout(t, func() {
        for {
            select {
            case rn.Send() <- Message{}:
            case <-tn.Receive():
                return
            }
        }
    })

    close(p2.Send())
    time.Sleep(999 * time.Millisecond)

    testTimeout(t, func() {
        for {
            select {
            case rn.Send() <- Message{}:
            case <-tn.Receive():
                return
            }
        }
    })

    close(p3.Send())
    time.Sleep(999 * time.Millisecond)

    testTimeout(t, func() {
        if err := <-rn.Error(); err != ErrRecoveryFailed {
            t.Error(err)
        }
    })
}
