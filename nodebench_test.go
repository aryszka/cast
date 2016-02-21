package cast

import "testing"

func receive(c Connection) {
	for {
		_, open := <-c.Receive()
		if !open {
			return
		}
	}
}

func benchmarkDispatch(b *testing.B, parent bool, childCount int) {
	n, p, _, children := createTestNode(0, 0, parent, childCount)

	if parent {
		go receive(p)
	}

	for _, c := range children {
		go receive(c)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n.Send() <- Message{}
	}

	b.StopTimer()
	close(n.Send())
}

func createTree(n int) []Node {
	if n == 0 {
		return nil
	}

	node := NewNode(0, 0)
	listener := make(InProcListener)
	node.Listen(listener)
	nodes := []Node{node}
	n--

	d, r := n/3, n%3
	for i := 0; i < 3; i++ {
		nn := d
		if i < r {
			nn++
		}

		children := createTree(nn)

		if len(children) > 0 {
			c, p := NewInProcConnection()
			listener <- p
			children[0].Join(c)
		}

		nodes = append(nodes, children...)
	}

	return nodes
}

func benchmarkTree(b *testing.B, n int) {
	nodes := createTree(n)
	for _, n := range nodes {
		go receive(n)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodes[0].Send() <- Message{}
	}

	b.StopTimer()
	for _, n := range nodes {
		close(n.Send())
	}
}

func BenchmarkNoParentNoChildren(b *testing.B) {
	benchmarkDispatch(b, false, 0)
}

func BenchmarkParentOnly(b *testing.B) {
	benchmarkDispatch(b, true, 0)
}

func BenchmarkParentOnlyChildren1(b *testing.B) {
	benchmarkDispatch(b, true, 1)
}

func BenchmarkParentOnlyChildren2(b *testing.B) {
	benchmarkDispatch(b, true, 2)
}

func BenchmarkParentOnlyChildren5(b *testing.B) {
	benchmarkDispatch(b, true, 5)
}

func BenchmarkParentOnlyChildren10(b *testing.B) {
	benchmarkDispatch(b, true, 10)
}

func BenchmarkParentOnlyChildren20(b *testing.B) {
	benchmarkDispatch(b, true, 20)
}

func BenchmarkParentOnlyChildren50(b *testing.B) {
	benchmarkDispatch(b, true, 50)
}

func BenchmarkParentOnlyChildren100(b *testing.B) {
	benchmarkDispatch(b, true, 100)
}

func BenchmarkParentOnlyChildren200(b *testing.B) {
	benchmarkDispatch(b, true, 200)
}

func BenchmarkParentOnlyChildren500(b *testing.B) {
	benchmarkDispatch(b, true, 500)
}

func BenchmarkParentOnlyChildren1000(b *testing.B) {
	benchmarkDispatch(b, true, 1000)
}

func BenchmarkParentOnlyChildren2000(b *testing.B) {
	benchmarkDispatch(b, true, 2000)
}

func BenchmarkParentOnlyChildren5000(b *testing.B) {
	benchmarkDispatch(b, true, 5000)
}

func BenchmarkParentOnlyChildren10000(b *testing.B) {
	benchmarkDispatch(b, true, 10000)
}

func BenchmarkParentOnlyChildren20000(b *testing.B) {
	benchmarkDispatch(b, true, 20000)
}

func BenchmarkParentOnlyChildren50000(b *testing.B) {
	benchmarkDispatch(b, true, 50000)
}

func BenchmarkParentOnlyChildren100000(b *testing.B) {
	benchmarkDispatch(b, true, 100000)
}

func BenchmarkParentOnlyChildren200000(b *testing.B) {
	benchmarkDispatch(b, true, 200000)
}

func BenchmarkParentOnlyChildren500000(b *testing.B) {
	benchmarkDispatch(b, true, 500000)
}

func BenchmarkParentOnlyChildren1000000(b *testing.B) {
	benchmarkDispatch(b, true, 1000000)
}

func BenchmarkTree1(b *testing.B) {
	benchmarkTree(b, 1)
}

func BenchmarkTree2(b *testing.B) {
	benchmarkTree(b, 2)
}

func BenchmarkTree5(b *testing.B) {
	benchmarkTree(b, 5)
}

func BenchmarkTree10(b *testing.B) {
	benchmarkTree(b, 10)
}

func BenchmarkTree20(b *testing.B) {
	benchmarkTree(b, 20)
}

func BenchmarkTree50(b *testing.B) {
	benchmarkTree(b, 50)
}

func BenchmarkTree100(b *testing.B) {
	benchmarkTree(b, 100)
}

func BenchmarkTree200(b *testing.B) {
	benchmarkTree(b, 200)
}

func BenchmarkTree500(b *testing.B) {
	benchmarkTree(b, 500)
}

func BenchmarkTree1000(b *testing.B) {
	benchmarkTree(b, 1000)
}

func BenchmarkTree2000(b *testing.B) {
	benchmarkTree(b, 2000)
}

func BenchmarkTree5000(b *testing.B) {
	benchmarkTree(b, 5000)
}

func BenchmarkTree10000(b *testing.B) {
	benchmarkTree(b, 10000)
}

func BenchmarkTree20000(b *testing.B) {
	benchmarkTree(b, 20000)
}

func BenchmarkTree50000(b *testing.B) {
	benchmarkTree(b, 50000)
}

func BenchmarkTree100000(b *testing.B) {
	benchmarkTree(b, 100000)
}

func BenchmarkTree200000(b *testing.B) {
	benchmarkTree(b, 200000)
}

func BenchmarkTree500000(b *testing.B) {
	benchmarkTree(b, 500000)
}

func BenchmarkTree1000000(b *testing.B) {
	benchmarkTree(b, 1000000)
}
