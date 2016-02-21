package cast

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func timeoutOrBlock(block bool, a func()) error {
	done := make(chan struct{})
	go func() {
		a()
		close(done)
	}()

	select {
	case <-done:
		if block {
			return errors.New("failed to block")
		}
	case <-time.After(120 * time.Millisecond):
		if !block {
			return errors.New("test timeout")
		}
	}

	return nil
}

func callTimeout(a func()) error { return timeoutOrBlock(false, a) }
func callBlock(a func()) error   { return timeoutOrBlock(true, a) }

func testMessage(from Connection, to ...Connection) error {
	msend := []Message{{Val: "one"}, {Val: "two"}, {Val: "three"}}
	for _, m := range msend {
		err := callTimeout(func() { from.Send() <- m })
		if err != nil {
			return err
		}

		terr := callTimeout(func() {
			for _, r := range to {
				mreceive := <-r.Receive()
				if mreceive.Val != m.Val {
					err = errors.New("failed to send message")
					break
				}
			}
		})

		if terr != nil {
			return terr
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func testMessageBlock(n Node, buffer int, timeout time.Duration, from, block Connection, to ...Connection) error {
	/* 3 is kind of a magic value */
	const defaultBuffer = 3

	send := func(count int) {
		for i := 0; i < count; i++ {
			from.Send() <- Message{}
		}
	}

	err := callTimeout(func() { send(buffer) })
	if err != nil {
		return err
	}

	if timeout == 0 {
		return callBlock(func() { send(defaultBuffer) })
	}

	var (
		wg         sync.WaitGroup
		cerr, terr error
	)

	wg.Add(2)
	go func() {
		cerr = callTimeout(func() { send(defaultBuffer) })
		wg.Done()
	}()
	go func() {
		terr = callTimeout(func() {
			for {
				err := <-n.Error()
				if _, ok := err.(*TimeoutError); ok {
					return
				}
			}
		})
		wg.Done()
	}()

	wg.Wait()

	if cerr != nil {
		return cerr
	}

	return terr
}

func createTestNode(buffer int, timeout time.Duration, hasParent bool, childrenCount int) (
	n Node, parent Connection, l Listener, children []Connection) {

	n = NewNode(buffer, timeout)

	if hasParent {
		var parentRemote Connection
		parent, parentRemote = NewInProcConnection()
		n.Join(parentRemote)
	}

	if childrenCount > 0 {
		l = make(InProcListener)
		n.Listen(l)
	}

	for i := 0; i < childrenCount; i++ {
		c, err := l.(InProcListener).Connect()
		if err != nil {
			panic(err)
		}

		children = append(children, c)
	}

	return
}

func TestMessaging(t *testing.T) {
	for _, ti := range []struct {
		msg             string
		buffer          int
		timeout         time.Duration
		parent          bool
		children        int
		from, to, block string
	}{{
		"no block when no connection",
		0, 0, false, 0, "node", "", "",
	}, {
		"node to parent",
		0, 0, true, 0, "node", "parent", "",
	}, {
		"node to parent, parent block",
		0, 0, true, 0, "node", "parent", "parent",
	}, {
		"node to parent, parent block, buffered",
		12, 0, true, 0, "node", "parent", "parent",
	}, {
		"node to parent, parent block, timeout",
		0, 12 * time.Millisecond, true, 0, "node", "parent", "parent",
	}, {
		"node to parent, parent block, buffered timeout",
		12, 12 * time.Millisecond, true, 0, "node", "parent", "parent",
	}, {
		"parent to node",
		0, 0, true, 0, "parent", "node", "",
	}, {
		"parent to node, node block",
		0, 0, true, 0, "parent", "node", "node",
	}, {
		"parent to node, node block, buffered",
		12, 0, true, 0, "parent", "node", "node",
	}, {
		"parent to node, node block, timeout",
		0, 12 * time.Millisecond, true, 0, "parent", "node", "node",
	}, {
		"parent to node, node block, buffered timeout",
		12, 12 * time.Millisecond, true, 0, "parent", "node", "node",
	}, {
		"node to child",
		0, 0, false, 1, "node", "child0", "",
	}, {
		"node to child, child block",
		0, 0, false, 1, "node", "child0", "child0",
	}, {
		"node to child, child block, buffered",
		12, 0, false, 1, "node", "child0", "child0",
	}, {
		"node to child, child block, timeout",
		0, 12 * time.Millisecond, false, 1, "node", "child0", "child0",
	}, {
		"node to child, child block, buffered timeout",
		12, 12 * time.Millisecond, false, 1, "node", "child0", "child0",
	}, {
		"child to node",
		0, 0, false, 1, "child0", "node", "",
	}, {
		"child to node, node block",
		0, 0, false, 1, "child0", "node", "node",
	}, {
		"child to node, node block, buffered",
		12, 0, false, 1, "child0", "node", "node",
	}, {
		"child to node, node block, timeout",
		0, 12 * time.Millisecond, false, 1, "child0", "node", "node",
	}, {
		"child to node, node block, buffered timeout",
		12, 12 * time.Millisecond, false, 1, "child0", "node", "node",
	}, {
		"node to multiple children",
		0, 0, false, 3, "node", "child0 child1 child2", "",
	}, {
		"node to multiple children, child block",
		0, 0, false, 3, "node", "child0 child1 child2", "child0",
	}, {
		"node to multiple children, child block, buffered",
		12, 0, false, 3, "node", "child0 child1 child2", "child0",
	}, {
		"node to multiple children, child block, timeout",
		0, 12 * time.Millisecond, false, 3, "node", "child0 child1 child2", "child0",
	}, {
		"node to multiple children, child block, buffered timeout",
		12, 12 * time.Millisecond, false, 3, "node", "child0 child1 child2", "child0",
	}, {
		"child to node and children",
		0, 0, false, 3, "child0", "node child1 child2", "",
	}, {
		"child to node and children, node block",
		0, 0, false, 3, "child0", "node child1 child2", "node",
	}, {
		"child to node and children, node block, buffered",
		12, 0, false, 3, "child0", "node child1 child2", "node",
	}, {
		"child to node and children, node block, timeout",
		0, 12 * time.Millisecond, false, 3, "child0", "node child1 child2", "node",
	}, {
		"child to node and children, node block, buffered timeout",
		12, 12 * time.Millisecond, false, 3, "child0", "node child1 child2", "node",
	}, {
		"child to node and children, child block",
		0, 0, false, 3, "child0", "node child1 child2", "child1",
	}, {
		"child to node and children, child block, buffered",
		12, 0, false, 3, "child0", "node child1 child2", "child1",
	}, {
		"child to node and children, child block, timeout",
		0, 12 * time.Millisecond, false, 3, "child0", "node child1 child2", "child1",
	}, {
		"child to node and children, child block, buffered timeout",
		12, 12 * time.Millisecond, false, 3, "child0", "node child1 child2", "child1",
	}, {
		"node to all",
		0, 0, true, 3, "node", "parent child0 child1 child2", "",
	}, {
		"node to all, parent block",
		0, 0, true, 3, "node", "parent child0 child1 child2", "parent",
	}, {
		"node to all, parent block, buffered",
		12, 0, true, 3, "node", "parent child0 child1 child2", "parent",
	}, {
		"node to all, parent block, timeout",
		0, 12 * time.Millisecond, true, 3, "node", "parent child0 child1 child2", "parent",
	}, {
		"node to all, parent block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "node", "parent child0 child1 child2", "parent",
	}, {
		"node to all, child block",
		0, 0, true, 3, "node", "parent child0 child1 child2", "child0",
	}, {
		"node to all, child block, buffered",
		12, 0, true, 3, "node", "parent child0 child1 child2", "child0",
	}, {
		"node to all, child block, timeout",
		0, 12 * time.Millisecond, true, 3, "node", "parent child0 child1 child2", "child0",
	}, {
		"node to all, child block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "node", "parent child0 child1 child2", "child0",
	}, {
		"parent to all",
		0, 0, true, 3, "parent", "node child0 child1 child2", "",
	}, {
		"parent to all, node block",
		0, 0, true, 3, "parent", "node child0 child1 child2", "node",
	}, {
		"parent to all, node block, buffered",
		12, 0, true, 3, "parent", "node child0 child1 child2", "node",
	}, {
		"parent to all, node block, timeout",
		0, 12 * time.Millisecond, true, 3, "parent", "node child0 child1 child2", "node",
	}, {
		"parent to all, node block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "parent", "node child0 child1 child2", "node",
	}, {
		"parent to all, child block",
		0, 0, true, 3, "parent", "node child0 child1 child2", "child0",
	}, {
		"parent to all, child block, buffered",
		12, 0, true, 3, "parent", "node child0 child1 child2", "child0",
	}, {
		"parent to all, child block, timeout",
		0, 12 * time.Millisecond, true, 3, "parent", "node child0 child1 child2", "child0",
	}, {
		"parent to all, child block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "parent", "node child0 child1 child2", "child0",
	}, {
		"child to all",
		0, 0, true, 3, "child0", "node parent child1 child2", "",
	}, {
		"child to all, node block",
		0, 0, true, 3, "child0", "node parent child1 child2", "node",
	}, {
		"child to all, node block, buffered",
		12, 0, true, 3, "child0", "node parent child1 child2", "node",
	}, {
		"child to all, node block, timeout",
		0, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "node",
	}, {
		"child to all, node block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "node",
	}, {
		"child to all, parent block",
		0, 0, true, 3, "child0", "node parent child1 child2", "parent",
	}, {
		"child to all, parent block, buffered",
		12, 0, true, 3, "child0", "node parent child1 child2", "parent",
	}, {
		"child to all, parent block, timeout",
		0, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "parent",
	}, {
		"child to all, parent block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "parent",
	}, {
		"child to all, child block",
		0, 0, true, 3, "child0", "node parent child1 child2", "child1",
	}, {
		"child to all, child block, buffered",
		12, 0, true, 3, "child0", "node parent child1 child2", "child1",
	}, {
		"child to all, child block, timeout",
		0, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "child1",
	}, {
		"child to all, child block, buffered timeout",
		12, 12 * time.Millisecond, true, 3, "child0", "node parent child1 child2", "child1",
	}} {
		n, parent, _, children := createTestNode(ti.buffer, ti.timeout, ti.parent, ti.children)

		getConnection := func(name string) (Connection, error) {
			var c Connection
			switch name {
			case "node":
				c = n
			case "parent":
				c = parent
			}

			if strings.HasPrefix(name, "child") {
				ci, err := strconv.Atoi(name[len("child"):])
				if err != nil {
					return nil, err
				}

				c = children[ci]
			}

			var err error
			if c == nil {
				err = errors.New("connection not found")
			}

			return c, err
		}

		from, err := getConnection(ti.from)
		if err != nil {
			t.Error(ti.msg, err)
			continue
		}

		var to []Connection
		if ti.to != "" {
			for _, t := range strings.Split(ti.to, " ") {
				var tc Connection
				tc, err = getConnection(t)
				if err != nil {
					break
				}

				to = append(to, tc)
			}
		}

		if err != nil {
			t.Error(ti.msg, err)
			continue
		}

		if ti.block == "" {
			err = testMessage(from, to...)
		} else if ti.timeout > 0 || !testing.Short() {
			var block Connection
			block, err = getConnection(ti.block)
			if err == nil {
				err = testMessageBlock(n, ti.buffer, ti.timeout, from, block, to...)
			}
		}

		if err != nil {
			t.Error(ti.msg, err)
		}
	}
}
