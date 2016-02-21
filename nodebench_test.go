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
	if childCount > 500 && testing.Short() {
		b.Skip()
	}

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

func benchmarkTree(b *testing.B, n int) {
	if n > 200 && testing.Short() {
		b.Skip()
	}

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

func BenchmarkNoParentNoChildren(b *testing.B)  { benchmarkDispatch(b, false, 0) }
func BenchmarkParentOnly(b *testing.B)          { benchmarkDispatch(b, true, 0) }
func BenchmarkParentChildren1(b *testing.B)     { benchmarkDispatch(b, true, 1) }
func BenchmarkParentChildren2(b *testing.B)     { benchmarkDispatch(b, true, 2) }
func BenchmarkParentChildren5(b *testing.B)     { benchmarkDispatch(b, true, 5) }
func BenchmarkParentChildren10(b *testing.B)    { benchmarkDispatch(b, true, 10) }
func BenchmarkParentChildren20(b *testing.B)    { benchmarkDispatch(b, true, 20) }
func BenchmarkParentChildren50(b *testing.B)    { benchmarkDispatch(b, true, 50) }
func BenchmarkParentChildren100(b *testing.B)   { benchmarkDispatch(b, true, 100) }
func BenchmarkParentChildren200(b *testing.B)   { benchmarkDispatch(b, true, 200) }
func BenchmarkParentChildren500(b *testing.B)   { benchmarkDispatch(b, true, 500) }
func BenchmarkParentChildren1000(b *testing.B)  { benchmarkDispatch(b, true, 1000) }
func BenchmarkParentChildren2000(b *testing.B)  { benchmarkDispatch(b, true, 2000) }
func BenchmarkParentChildren5000(b *testing.B)  { benchmarkDispatch(b, true, 5000) }
func BenchmarkParentChildren10000(b *testing.B) { benchmarkDispatch(b, true, 10000) }
func BenchmarkParentChildren20000(b *testing.B) { benchmarkDispatch(b, true, 20000) }
func BenchmarkParentChildren50000(b *testing.B) { benchmarkDispatch(b, true, 50000) }

func BenchmarkTree1(b *testing.B)     { benchmarkTree(b, 1) }
func BenchmarkTree2(b *testing.B)     { benchmarkTree(b, 2) }
func BenchmarkTree5(b *testing.B)     { benchmarkTree(b, 5) }
func BenchmarkTree10(b *testing.B)    { benchmarkTree(b, 10) }
func BenchmarkTree20(b *testing.B)    { benchmarkTree(b, 20) }
func BenchmarkTree50(b *testing.B)    { benchmarkTree(b, 50) }
func BenchmarkTree100(b *testing.B)   { benchmarkTree(b, 100) }
func BenchmarkTree200(b *testing.B)   { benchmarkTree(b, 200) }
func BenchmarkTree500(b *testing.B)   { benchmarkTree(b, 500) }
func BenchmarkTree1000(b *testing.B)  { benchmarkTree(b, 1000) }
func BenchmarkTree2000(b *testing.B)  { benchmarkTree(b, 2000) }
func BenchmarkTree5000(b *testing.B)  { benchmarkTree(b, 5000) }
func BenchmarkTree10000(b *testing.B) { benchmarkTree(b, 10000) }
func BenchmarkTree20000(b *testing.B) { benchmarkTree(b, 20000) }
func BenchmarkTree50000(b *testing.B) { benchmarkTree(b, 50000) }
