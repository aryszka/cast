package cast

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

/*
In many places, using time.After(time.Millisecond) in selects instead of
`default`, because the Connection implementation not necessarily provides
immediate blocking.
*/

type (
	makeChannel func(int, time.Duration) (Connection, []Connection)
	testFunc    func(*testing.T, string, string, Connection, []Connection)
)

func errmsg(msg, smsg, ssmsg string) string {
	return fmt.Sprintf("%s, %s; %s", msg, smsg, ssmsg)
}

func testSend(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	m := &Message{}
	go func() { local.Send() <- m }()
	for _, r := range remote {
		mr := <-r.Receive()
		if mr != m {
			t.Error(errmsg(msg, smsg, "failed to send message"))
		}
	}
}

func testBlock(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	var wg sync.WaitGroup
	wg.Add(len(remote))

	for _, r := range remote {
		go func(r Connection) {
			defer wg.Done()

			select {
			case <-r.Receive():
				t.Error(errmsg(msg, smsg, "failed to block channel"))
			case <-time.After(time.Millisecond):
			}
		}(r)
	}

	wg.Wait()
}

func testBuffer(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	const buf = 3

	for i := 0; i < buf; i++ {
		local.Send() <- &Message{}
	}

	var wg sync.WaitGroup
	wg.Add(len(remote))

	for _, r := range remote {
		go func(r Connection) {
			defer wg.Done()

			for i := 0; i < buf; i++ {
				select {
				case <-r.Receive():
				case <-time.After(120 * time.Millisecond):
					t.Error(errmsg(msg, smsg, "failed to buffer messages"))
				}
			}
		}(r)
	}

	wg.Wait()
}

func testTimeout(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	const timeout = time.Millisecond

	for _, c := range remote {
		if _, ok := c.(Node); ok {
			return
		}
	}

	m := &Message{}
	go func() { local.Send() <- m }()
	select {
	case err := <-local.Error():
		if terr, ok := err.(*TimeoutError); ok && terr.Message != m {
			t.Error(errmsg(msg, smsg, "invalid message in timeout error"))
		} else if !ok {
			t.Error(errmsg(msg, smsg, "invalid error"))
		}
	case <-time.After(120 * time.Millisecond):
		t.Error(errmsg(msg, smsg, "timeout failed"))
	}
}

func testBufferAndTimeout(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	const (
		buf     = 3
		timeout = time.Millisecond
	)

	for _, c := range remote {
		if _, ok := c.(Node); ok {
			return
		}
	}

	go func() {
		for {
			local.Send() <- &Message{}
		}
	}()

	select {
	case err := <-local.Error():
		if _, ok := err.(*TimeoutError); !ok {
			t.Error(errmsg(msg, smsg, "invalid error"))
		}
	case <-time.After(120 * time.Millisecond):
		t.Error(errmsg(msg, smsg, "timeout failed"))
	}
}

func testClose(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
	close(local.Send())

	var wg sync.WaitGroup
	wg.Add(len(remote))

	for _, r := range remote {
		go func(r Connection) {
			defer wg.Done()

			if _, ok := r.(Node); ok {
				return
			}

			select {
			case _, open := <-r.Receive():
				if open {
					t.Error(errmsg(msg, smsg, "failed to close channel"))
				}
			case <-time.After(120 * time.Millisecond):
				t.Error(errmsg(msg, smsg, "failed to close channel"))
			}
		}(r)
	}

	wg.Wait()
}

func testOrder(t *testing.T, msg, smsg string, local Connection, remote []Connection) {
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
			defer wg.Done()

			var prev *Message
			for i := 0; i < count; i++ {
				next := <-r.Receive()
				if prev != nil && next.Val <= prev.Val {
					t.Error(errmsg(msg, smsg, "failed to send messages in the right order"))
					return
				}

				prev = next
			}

			if _, ok := r.(Node); ok {
				return
			}

			_, open := <-r.Receive()
			if open {
				t.Error(errmsg(msg, smsg, "failed to close channel"))
			}
		}(r)
	}

	wg.Wait()
}

func testMessageChannel(t *testing.T, msg string, mc makeChannel) {
	for _, ti := range []struct {
		tf      testFunc
		msg     string
		buffer  int
		timeout time.Duration
	}{
		{testSend, "send", 0, 0},
		{testBlock, "block", 0, 0},
		{testBuffer, "buffer", 3, 0},
		{testTimeout, "timeout", 0, time.Millisecond},
		{testBufferAndTimeout, "buffer and timeout", 3, time.Millisecond},
		{testClose, "close", 0, 0},
		{testOrder, "order", 0, 0},
	} {
		local, remote := mc(ti.buffer, ti.timeout)
		ti.tf(t, msg, ti.msg, local, remote)
	}
}
