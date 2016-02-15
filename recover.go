package cast

// type recoveryNode struct {
//     node Node
//     err chan error
// }
//
// func NewRecoveryNode(o *Opt, recovery ...Interface) Node {
//     n := &recoveryNode{NewNode(o), make(chan error)}
//     go func() {
//         err := <-n.node.Error()
//         n.err <- err
//     }()
//
//     if len(recovery) > 0 {
//         c, err := recovery[0].Connect()
//         if err == nil {
//             n.Join(c)
//         } else {
//             n.err <- err
//         }
//     }
//
//     return n
// }
//
// func (n *recoveryNode) Send() chan<- Message { return n.node.Send() }
// func (n *recoveryNode) Receive() <-chan Message { return n.node.Receive() }
// func (n *recoveryNode) Join(c Connection) { n.node.Join(c) }
// func (n *recoveryNode) Listen(l Listener) { n.node.Listen(l) }
// func (n *recoveryNode) Error() <-chan error { return n.err }
