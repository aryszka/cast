package cast

import (
	"testing"
	"time"
)

func TestNodeToParentMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "single node to parent", func(buffer int, timeout time.Duration) (Connection, []Connection) {
		n := Opt{ParentBuffer: buffer, ParentTimeout: timeout}.NewNode()
		local, remote := NewInProcConnection()
		n.Join(local)
		return n, []Connection{remote}
	})
}

func TestParentToNodeMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "parent to single node", func(buffer int, _ time.Duration) (Connection, []Connection) {
		n := Opt{ReceiveBuffer: buffer}.NewNode()
		local, remote := NewInProcConnection()
		n.Join(local)
		return remote, []Connection{n}
	})
}

func TestNodeToChildren(t *testing.T) {
	testMessageChannel(t, "node to children", func(buffer int, timeout time.Duration) (Connection, []Connection) {
		n := Opt{ChildBuffer: buffer, ChildTimeout: timeout}.NewNode()
		l := make(chan Connection)
		n.Listen(l)

		l0, r0 := NewInProcConnection()
		l1, r1 := NewInProcConnection()
		l2, r2 := NewInProcConnection()

		l <- l0
		l <- l1
		l <- l2

		return n, []Connection{r0, r1, r2}
	})
}

func TestNodesParentToEveryone(t *testing.T) {
	testMessageChannel(t, "parent to everyone", func(buffer int, _ time.Duration) (Connection, []Connection) {
		p := Opt{SendBuffer: buffer}.NewNode()
		l := make(chan Connection)
		p.Listen(l)

		var nodes []Connection
		makeChild := func() {
			local, remote := NewInProcConnection()
			l <- remote
			c := NewNode()
			c.Join(local)
			nodes = append(nodes, c)
		}

		makeChild()
		makeChild()
		makeChild()

		return p, nodes
	})
}

func TestNodesChildToEveryone(t *testing.T) {
	testMessageChannel(t, "parent to everyone", func(buffer int, _ time.Duration) (Connection, []Connection) {
		p := Opt{SendBuffer: buffer}.NewNode()
		l := make(chan Connection)
		p.Listen(l)

		var nodes []Connection
		makeChild := func() {
			local, remote := NewInProcConnection()
			l <- remote
			c := NewNode()
			c.Join(local)
			nodes = append(nodes, c)
		}

		makeChild()
		makeChild()
		makeChild()

		c := nodes[1]
		nodes[1] = p
		return c, nodes
	})
}
