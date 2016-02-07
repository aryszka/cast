package main

import (
	"fmt"
	"time"
    "github.com/aryszka/cast"
)

func createNode(label string) cast.Node {
	n := cast.NewNode(cast.Opt{})
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

func createListenerNode(label string) (cast.Node, chan cast.Connection) {
	listener := make(chan cast.Connection)
	n := createNode(label)
	n.Listen(listener)
    return n, listener
}

func createChildNode(listener chan cast.Connection, label string) cast.Node {
	n := createNode(label)
    local, remote := cast.NewInProcConnection()
    listener <- remote
	n.Join(local)
	return n
}

func sendMessage(n cast.Node, label, message string) {
	fmt.Printf("\nsending message to: %s\n", label)
	n.Send() <- &cast.Message{Val: message}
}

func sendHelloWorld(n cast.Node, label string) {
	sendMessage(n, label, "Hello, world!")
}

func closeAll(n ...cast.Node) {
	closed := make(chan struct{})
	for _, ni := range n {
		go func(n cast.Node) {
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
    np, listener := createListenerNode("parent")

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
