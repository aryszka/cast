package cast

// import (
//     "testing"
//     "time"
// )
//
// func TestRecoverTakesOptions(t *testing.T) {
//     c := make(MessageChannel)
//     o := Opt{SendTimeout: time.Millisecond}
//     n := NewRecoveryNode(&o)
//     n.Join(c)
//
//     go func() { n.Send() <- Message{} }()
//     go func() { n.Send() <- Message{} }()
//     go func() { n.Send() <- Message{} }()
//     go func() { n.Send() <- Message{} }()
//
//     select {
//     case err := <-n.Error():
//         if _, ok := err.(*TimeoutError); !ok {
//             t.Error("failed to pass options, invalid error")
//         }
//     case <-time.After(120 * time.Millisecond):
//         t.Error("failed to pass options, unexpected timeout")
//     }
// }
//
// func TestRecoverInitialConnectOne(t *testing.T) {
//     n1 := NewRecoveryNode(nil)
//     l1 := make(InProcListener)
//     n1.Listen(l1)
//
//     n2 := NewRecoveryNode(nil, l1)
//
//     min := Message{Val: "Hello, world!"}
//     n2.Send() <- min
//
//     select {
//     case mout := <-n1.Receive():
//         if mout.Val != min.Val {
//             t.Error("failed to forward message")
//         }
//     case <-time.After(120 * time.Millisecond):
//         t.Error("forwarding timeout")
//     }
// }
