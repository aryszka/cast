package main

import (
	"fmt"
	"time"
)

func createNode(label string) Node {
	n := NewNode(Opt{})
	go func() {
		for {
			m, open := <-n.Receive()
			if !open {
				return
			}

			fmt.Printf("%-9s: %s\n", label, m.Val)
		}
	}()

	return n
}

func createChildNode(listener chan Connection, label string) Node {
	n := createNode(label)
	n.Join(NewInProcConnection(listener))
	return n
}

func sendMessage(n Node, label, message string) {
	fmt.Printf("\nsending message to: %s\n", label)
	n.Send() <- &Message{Val: message}
}

func sendHelloWorld(n Node, label string) {
	sendMessage(n, label, "Hello, world!")
}

func closeAll(n ...Node) {
	closed := make(chan struct{})
	for _, ni := range n {
		go func(n Node) {
			for {
				_, open := <-n.Receive()
				if !open {
					closed <- struct{}{}
				}
			}
		}(ni)

		close(ni.Send())
	}

	c := len(n)
	for {
		<-closed
		c--
		if c == 0 {
			return
		}
	}
}

func main() {
	listener := make(chan Connection)
	np := createNode("parent")
	np.Listen(listener)

	nc0 := createChildNode(listener, "child 0")
	nc1 := createChildNode(listener, "child 1")

	sendHelloWorld(nc0, "child 0")
	time.Sleep(time.Millisecond)

	sendHelloWorld(np, "parent")
	time.Sleep(time.Millisecond)

	sendHelloWorld(nc1, "child 1")
	time.Sleep(time.Millisecond)

	closeAll(np)
}
