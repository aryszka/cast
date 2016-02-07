package cast

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

/*
In many places, using time.After(time.Millisecond) in selects instead of
default, because the Connection implementation not necessarily provides
immediate blocking.
*/

type (
	makeChannel func(int) (Connection, []Connection)
	testFunc    func(*testing.T, string, Connection, []Connection)
)

func testSend(t *testing.T, msg string, local Connection, remote []Connection) {
	m := &Message{}
	go func() { local.Send() <- m }()
	for _, r := range remote {
		mr := <-r.Receive()
		if mr != m {
			t.Error(msg+";", "failed to send message")
		}
	}
}

func testBlock(t *testing.T, msg string, local Connection, remote []Connection) {
	var wg sync.WaitGroup
	wg.Add(len(remote))
	for _, r := range remote {
		go func(r Connection) {
			select {
			case <-r.Receive():
				t.Error(msg, "failed to block channel")
			case <-time.After(time.Millisecond):
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func testBuffer(t *testing.T, msg string, local Connection, remote []Connection) {
	const buf = 3

	for i := 0; i < buf; i++ {
		local.Send() <- &Message{}
	}

	var wg sync.WaitGroup
	wg.Add(len(remote))
	for _, r := range remote {
		go func(r Connection) {
			for i := 0; i < buf; i++ {
				select {
				case <-r.Receive():
				case <-time.After(time.Millisecond):
					t.Error(msg+";", "failed to buffer messages")
				}
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func testClose(t *testing.T, msg string, local Connection, remote []Connection) {
	close(local.Send())
	var wg sync.WaitGroup
	wg.Add(len(remote))
	for _, r := range remote {
		go func(r Connection) {
			if _, ok := r.(Node); ok {
				wg.Done()
				return
			}

			select {
			case _, open := <-r.Receive():
				if open {
					t.Error(msg+";", "failed to close channel")
				}
			case <-time.After(time.Millisecond):
				t.Error(msg+";", "failed to close channel")
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func testOrder(t *testing.T, msg string, local Connection, remote []Connection) {
	const count = 3

	go func() {
		for i := 0; i < 3; i++ {
			local.Send() <- &Message{Val: strconv.Itoa(i)}
		}

		close(local.Send())
	}()

	var wg sync.WaitGroup
	wg.Add(len(remote))
	for _, r := range remote {
		go func(r Connection) {
			var prev *Message
			for i := 0; i < count; i++ {
				next := <-r.Receive()
				if prev != nil && next.Val <= prev.Val {
					t.Error(msg+";", "failed to send messages in the right order")
					return
				}

				prev = next
			}

			if _, ok := r.(Node); ok {
				wg.Done()
				return
			}

			_, open := <-r.Receive()
			if open {
				t.Error(msg+";", "failed to close channel")
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func testMessageChannel(t *testing.T, msg string, mc makeChannel) {
	for _, ti := range []struct {
		tf     testFunc
		buffer int
	}{
		{testSend, 0},
		{testBlock, 0},
		{testBuffer, 3},
		{testClose, 0},
		{testOrder, 0},
	} {
		local, remote := mc(ti.buffer)
		ti.tf(t, msg, local, remote)
	}
}
