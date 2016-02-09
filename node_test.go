package cast

import (
	"sync"
	"testing"
	"time"
)

func waitBuffer(t *testing.T, c Connection) {
	done := make(chan struct{})
	go func() {
		c.Send() <- &Message{}
		c.Send() <- &Message{}
		c.Send() <- &Message{}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Millisecond):
		t.Error("failed to buffer messages")
	}
}

func waitTimeout(t *testing.T, c, ec Connection) {
	m := &Message{}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case c.Send() <- m:
			case <-done:
				return
			}
		}
	}()

	select {
	case err := <-ec.Error():
		if terr, ok := err.(*TimeoutError); !ok || ok && terr.Message != m {
			t.Error("failed to apply timeout")
		}
	case <-time.After(120 * time.Millisecond):
		t.Error("failed to apply timeout")
	}

	close(done)
}

func makeNodeToParent(buffer int, timeout time.Duration) (Connection, []Connection) {
	n := Opt{ParentBuffer: buffer, ParentTimeout: timeout}.NewNode()
	local, remote := NewInProcConnection()
	n.Join(local)
	return n, []Connection{remote}
}

func makeParentToNode(buffer int, timeout time.Duration) (Connection, []Connection) {
	n := Opt{ReceiveBuffer: buffer}.NewNode()
	local, remote := NewInProcConnection()
	n.Join(local)
	return remote, []Connection{n}
}

func makeNodeToChildren(buffer int, timeout time.Duration) (Connection, []Connection) {
	n := Opt{ChildBuffer: buffer, ChildTimeout: timeout}.NewNode()
	l := make(InProcListener)
	n.Listen(l)

	l0, r0 := NewInProcConnection()
	l1, r1 := NewInProcConnection()
	l2, r2 := NewInProcConnection()

	l <- l0
	l <- l1
	l <- l2

	return n, []Connection{r0, r1, r2}
}

func makeNodesParentToEveryone(buffer int, _ time.Duration) (Connection, []Connection) {
	p := Opt{SendBuffer: buffer}.NewNode()
	l := make(InProcListener)
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
}

func makeNodesChildToEveryone(buffer int, _ time.Duration) (Connection, []Connection) {
	p := Opt{SendBuffer: buffer}.NewNode()
	l := make(InProcListener)
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
}

func TestNodeToParentMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "single node to parent", makeNodeToParent)
}

func TestParentToNodeMessageChannelBehavior(t *testing.T) {
	testMessageChannel(t, "parent to single node", makeParentToNode)
}

func TestNodeToChildren(t *testing.T) {
	testMessageChannel(t, "node to children", makeNodeToChildren)
}

func TestNodesParentToEveryone(t *testing.T) {
	testMessageChannel(t, "parent to everyone", makeNodesParentToEveryone)
}

func TestNodesChildToEveryone(t *testing.T) {
	testMessageChannel(t, "parent to everyone", makeNodesChildToEveryone)
}

func BenchmarkNodeToParentMessageChannelBehavior(b *testing.B) {
	benchmarkMessageChannel(b, "single node to parent", makeNodeToParent)
}

func BenchmarkNodeToParentMessageChannelBehaviorBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "single node to parent", makeNodeToParent)
}

func BenchmarkNodeToParentMessageChannelBehaviorTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "single node to parent", makeNodeToParent)
}

func BenchmarkNodeToParentMessageChannelBehaviorBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "single node to parent", makeNodeToParent)
}

func BenchmarkParentToNodeMessageChannelBehavior(b *testing.B) {
	benchmarkMessageChannel(b, "parent to single node", makeParentToNode)
}

func BenchmarkParentToNodeMessageChannelBehaviorBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "parent to single node", makeParentToNode)
}

func BenchmarkParentToNodeMessageChannelBehaviorTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "parent to single node", makeParentToNode)
}

func BenchmarkParentToNodeMessageChannelBehaviorBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "parent to single node", makeParentToNode)
}

func BenchmarkNodeToChildren(b *testing.B) {
	benchmarkMessageChannel(b, "node to children", makeNodeToChildren)
}

func BenchmarkNodeToChildrenBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "node to children", makeNodeToChildren)
}

func BenchmarkNodeToChildrenTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "node to children", makeNodeToChildren)
}

func BenchmarkNodeToChildrenBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "node to children", makeNodeToChildren)
}

func BenchmarkNodesParentToEveryone(b *testing.B) {
	benchmarkMessageChannel(b, "parent to everyone", makeNodesParentToEveryone)
}

func BenchmarkNodesParentToEveryoneBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "parent to everyone", makeNodesParentToEveryone)
}

func BenchmarkNodesParentToEveryoneTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "parent to everyone", makeNodesParentToEveryone)
}

func BenchmarkNodesParentToEveryoneBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "parent to everyone", makeNodesParentToEveryone)
}

func BenchmarkNodesChildToEveryone(b *testing.B) {
	benchmarkMessageChannel(b, "parent to everyone", makeNodesChildToEveryone)
}

func BenchmarkNodesChildToEveryoneBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "parent to everyone", makeNodesChildToEveryone)
}

func BenchmarkNodesChildToEveryoneTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "parent to everyone", makeNodesChildToEveryone)
}

func BenchmarkNodesChildToEveryoneBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "parent to everyone", makeNodesChildToEveryone)
}

func TestParentBuffer(t *testing.T) {
	n := Opt{ParentBuffer: 3}.NewNode()
	c, _ := NewInProcConnection()
	n.Join(c)
	waitBuffer(t, n)
}

func TestParentTimeout(t *testing.T) {
	n := Opt{ParentTimeout: time.Millisecond}.NewNode()
	c, _ := NewInProcConnection()
	n.Join(c)
	waitTimeout(t, n, n)
}

func TestChildBuffer(t *testing.T) {
	n := Opt{ChildBuffer: 3}.NewNode()
	l := make(InProcListener)
	n.Listen(l)
	c, _ := NewInProcConnection()
	l <- c
	waitBuffer(t, n)
}

func TestChildTimeout(t *testing.T) {
	n := Opt{ChildTimeout: time.Millisecond}.NewNode()
	l := make(InProcListener)
	n.Listen(l)
	c, _ := NewInProcConnection()
	l <- c
	waitTimeout(t, n, n)
}

func TestSendBuffer(t *testing.T) {
	n := Opt{SendBuffer: 3}.NewNode()
	c, _ := NewInProcConnection()
	n.Join(c)
	waitBuffer(t, n)
}

func TestSendTimeout(t *testing.T) {
	n := Opt{SendTimeout: time.Millisecond}.NewNode()
	c, _ := NewInProcConnection()
	n.Join(c)
	waitTimeout(t, n, n)
}

func TestReceiveBuffer(t *testing.T) {
	n := Opt{ReceiveBuffer: 3}.NewNode()
	local, remote := NewInProcConnection()
	n.Join(local)
	waitBuffer(t, remote)
}

func TestReceiveTimeout(t *testing.T) {
	n := Opt{ReceiveTimeout: time.Millisecond}.NewNode()
	local, remote := NewInProcConnection()
	n.Join(local)
	waitTimeout(t, remote, n)
}

func TestDispatch(t *testing.T) {
	for _, ti := range []struct {
		msg          string
		selectSource func(Connection, Connection, []Connection) Connection
	}{{
		"node",
		func(n Connection, _ Connection, _ []Connection) Connection { return n },
	}, {
		"parent",
		func(_ Connection, p Connection, _ []Connection) Connection { return p },
	}, {
		"child 0",
		func(_ Connection, _ Connection, c []Connection) Connection { return c[0] },
	}, {
		"child 1",
		func(_ Connection, _ Connection, c []Connection) Connection { return c[1] },
	}, {
		"child 2",
		func(_ Connection, _ Connection, c []Connection) Connection { return c[2] },
	}} {
		n := NewNode()
		plocal, premote := NewInProcConnection()
		n.Join(plocal)
		l := make(InProcListener)
		n.Listen(l)

		var children []Connection
		makeChild := func() {
			clocal, cremote := NewInProcConnection()
			l <- clocal
			children = append(children, cremote)
		}

		makeChild()
		makeChild()
		makeChild()

		source := ti.selectSource(n, premote, children)

		all := append(children, n, premote)
		var wg sync.WaitGroup
		wg.Add(len(all) - 1)

		for _, c := range all {
			go func(c Connection) {
				if c != source {
					<-c.Receive()
					wg.Done()
				}
			}(c)
		}

		done := make(chan struct{})

		go func() {
			select {
			case <-source.Receive():
				t.Error(ti.msg, "message sent back to source")
			case <-done:
			}
		}()

		go func() {
			wg.Wait()
			close(done)
		}()

		source.Send() <- &Message{}

		select {
		case <-done:
		case <-time.After(120 * time.Millisecond):
			t.Error("failed to dispatch message")
		}
	}
}

func TestParentDisconnect(t *testing.T) {
	n := NewNode()
	local, remote := NewInProcConnection()
	n.Join(local)
	close(remote.Send())
	select {
	case err := <-n.Error():
		if err != ErrDisconnected {
			t.Error("failed to report parent disconnect")
		}
	case <-time.After(120 * time.Millisecond):
		t.Error("failed to disconnect")
	}
}

func TestChildDisconnect(t *testing.T) {
	n := NewNode()
	l := make(InProcListener)
	n.Listen(l)

	local, remote := NewInProcConnection()
	l <- local

	close(remote.Send())
	select {
	case _, open := <-remote.Receive():
		if open {
			t.Error("failed to disconnect")
		}
	case <-time.After(42 * time.Millisecond):
		t.Error("failed to disconnect")
	}
}

func TestCloseNode(t *testing.T) {
	n := NewNode()
	plocal, premote := NewInProcConnection()
	n.Join(plocal)
	l := make(InProcListener)
	n.Listen(l)

	var children []Connection
	makeChild := func() {
		clocal, cremote := NewInProcConnection()
		l <- clocal
		children = append(children, cremote)
	}

	makeChild()
	makeChild()
	makeChild()

	all := append(children, premote)
	var wg sync.WaitGroup
	wg.Add(len(all))

	for _, c := range all {
		go func(c Connection) {
			select {
			case _, open := <-c.Receive():
				if open {
					t.Error("failed to disconnect from closed node")
				}
			case <-time.After(120 * time.Millisecond):
				t.Error("failed to disconnect from closed node")
			}

			wg.Done()
		}(c)
	}

	close(n.Send())
	wg.Wait()
}

func TestStopListening(t *testing.T) {
	n := NewNode()
	l := make(InProcListener)
	n.Listen(l)

	var children []Connection
	makeChild := func() {
		clocal, cremote := NewInProcConnection()
		l <- clocal
		children = append(children, cremote)
	}

	makeChild()
	makeChild()
	makeChild()

	var wg sync.WaitGroup
	wg.Add(len(children))

	for _, c := range children {
		go func(c Connection) {
			select {
			case _, open := <-c.Receive():
				if open {
					t.Error("failed to disconnect from closed node")
				}
			case <-time.After(120 * time.Millisecond):
				t.Error("failed to disconnect from closed node")
			}

			wg.Done()
		}(c)
	}

	close(l)
	wg.Wait()
}
