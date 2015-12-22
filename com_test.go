package cast

import (
	"testing"
)

func TestDispatcherNoSubscribers(t *testing.T) {
	d := newDispatcher()
	d.Send(&Message{})
	d.Close()
}

func TestDispatcherSubscribe(t *testing.T) {
}

// func TestNetworkFails(t *testing.T) {
// 	c := &network{
// 		latency:  0,
// 		strength: 0,
// 		timer:    newTimer(1),
// 		gen:      newGenerator(0),
// 		remote: &Listener{HandlerFunc(func(m *Message) {
// 			t.Error("unexpected message")
// 		})}}
// 	c.Send(&Message{Key: "test"})
// 	<-time.After(15 * time.Millisecond)
// }
//
// func TestConnectionSend(t *testing.T) {
// 	done := make(chan struct{})
// 	c := &network{
// 		latency:  15,
// 		strength: 1,
// 		timer:    newTimer(1),
// 		gen:      newGenerator(0),
// 		remote: &Listener{HandlerFunc(func(m *Message) {
// 			if m.Key != "test" {
// 				t.Error("failed to send message")
// 			}
//
// 			done <- struct{}{}
// 		})}}
// 	time.Sleep(15 * time.Millisecond)
// 	c.Send(&Message{Key: "test"})
// 	select {
// 	case <-done:
// 	case <-time.After(30 * time.Millisecond):
// 		t.Error("timeout")
// 	}
// }
