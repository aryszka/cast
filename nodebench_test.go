package cast

import "testing"

func benchmarkDispatch(b *testing.B, parent bool, childCount int) {
	receive := func(c Connection) {
		for {
			_, open := <-c.Receive()
			if !open {
				return
			}
		}
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
