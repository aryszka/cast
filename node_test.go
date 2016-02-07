package cast

import "testing"

func TestNodeToParentMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "single node to parent", func(buffer int) (Connection, []Connection) {
		n := NewNode(Opt{ParentBuffer: buffer})
		local, remote := NewInProcConnection()
		n.Join(local)
		return n, []Connection{remote}
	})
}

func TestParentToNodeMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "parent to single node", func(buffer int) (Connection, []Connection) {
		n := NewNode(Opt{})
		local, remote := NewInProcConnection()
		n.Join(local)

		var pc Connection = remote
		if buffer != 0 {
			pc = NewBufferedConnection(pc, buffer)
		}

		return pc, []Connection{n}
	})
}

func TestNodeToChildren(t *testing.T) {
	testMessageChannel(t, "node to children", func(buffer int) (Connection, []Connection) {
		n := NewNode(Opt{ChildBuffer: buffer})
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
	testMessageChannel(t, "parent to everyone", func(buffer int) (Connection, []Connection) {
		p := NewNode(Opt{SendBuffer: buffer})
		l := make(chan Connection)
		p.Listen(l)

		var nodes []Connection
		makeChild := func() {
			local, remote := NewInProcConnection()
			l <- remote
			c := NewNode(Opt{})
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
	testMessageChannel(t, "parent to everyone", func(buffer int) (Connection, []Connection) {
		p := NewNode(Opt{SendBuffer: buffer})
		l := make(chan Connection)
		p.Listen(l)

		var nodes []Connection
		makeChild := func() {
			local, remote := NewInProcConnection()
			l <- remote
			c := NewNode(Opt{})
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
