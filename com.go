package cast

type (
	Message struct {
		Key   string
		Value string
	}

	Sender interface {
		Send() chan<- *Message
        Close()
	}

	Receiver interface {
		Receive() <-chan *Message
	}

    Connection interface {
        Sender
        Receiver
    }

	ChanSender   chan<- *Message
	ChanReceiver <-chan *Message

	node struct {
		message *Message
		next    *node
	}

	queue struct {
		first *node
		last  *node
	}

	Repeater struct {
		closed chan struct{}
	}

	Buffer struct {
		in     chan *Message
		out    chan *Message
		repeat *Repeater
	}

	Network struct {
		in  chan *Message
		out chan *Message
	}

	Dispatcher struct {
		subscribe   chan chan Receiver
		unsubscribe chan Receiver
		in          chan *Message
		closed      chan struct{}
	}
)

func (cs ChanSender) Send() chan<- *Message      { return chan<- *Message(cs) }
func (cs ChanReceiver) Receive() <-chan *Message { return <-chan *Message(cs) }

func (q *queue) push(m *Message) {
	n := &node{message: m}
	if q.first == nil {
		q.first, q.last = n, n
	} else {
		q.last.next, q.last = n, n
	}
}

func (q *queue) shift() *Message {
	n, q.first = q.first, q.first.next
	if q.first == nil {
		q.last = nil
	}

	return n.message
}

func (q *queue) front() *Message { return q.first.message }
func (q *queue) empty() bool     { return q.first == nil }

func NewRepeater(s Sender, r Receiver) *Repeater {
	closed := make(chan struct{})

	go func() {
		var (
			all     queue
			m       *Message
			send    chan<- *Message
			receive chan<- *Message
			open    bool
		)

		receive = r.Receive()
		for {
			select {
			case m, open = <-receive:
				if !open {
					receive = nil
				} else {
					all.push(m)
				}
			case send <- m:
				all.shift()
			}

			if all.empty() {
				if !open {
					close(closed)
					return
				}

                send = nil
			} else {
				m = all.front()
				send = s.Send()
			}
		}
	}()

	return &Repeater{closed}
}

func (c *Repeater) Closed() <-chan struct{} { return r.closed }

func NewBuffer() *Buffer {
	in := make(chan *Message)
	out := make(chan *Message)
	repeat := NewRepeater(ChanSender(out), ChanSender(in))
	return &Buffer{in, out, repeat}
}

func (b *Buffer) Send() chan<- *Message    { return b.in }
func (b *Buffer) Receive() <-chan *Message { return b.out }

func (b *Buffer) Close() {
	close(b.in)
	go func() {
		<-b.repeat.Closed()
		close(b.out)
	}()
}

func NewNetwork(latency int, strength float64, t *timer, g *generator) *Network {
	in := make(chan *Message)
	out := make(chan *Message)
	outRepeat := make(chan *Message)
	repeat := NewRepeater(ChanSender(out), ChanReceiver(outRepeat))

	go func() {
		for {
			select {
			case m, open := <-in:
				if open {
					transferring++
					go func(m *Message) {
						if g.rand.Float64() > strength {
							done <- struct{}{}
							return
						}

						<-t.after(latency)
						outRepeat <- m
						done <- struct{}{}
					}(m)
				} else {
					in = nil
				}
			case <-done:
				transferring--
			}

			if transferring == 0 && !open {
				close(outRepeat)
				<-repeat.Closed()
				close(out)
				return
			}
		}
	}()

	return &Network{in, out}
}

func (nw *Network) Send() chan<- *Message    { return nw.in }
func (nw *Network) Receive() <-chan *Message { return nw.out }
func (nw *Network) Close()                   { close(nw.in) }

func NewDispatcher() *Dispatcher {
	subscribe := make(chan chan Receiver)
	unsubscribe := make(chan Receiver)
	in := make(chan *Message)
	closed := make(chan struct{})

	go func() {
		var receivers []*Buffer
		for {
			select {
			case sc := <-subscribe:
				r := NewBuffer()
				receivers = append(receivers, r)
				sc <- r
			case r := <-unsubscribe:
				rb, ok := r.(*Buffer)
				if !ok {
					return
				}

				for i, ri := range receivers {
					if ri == rb {
						rb.Close()
						receivers = append(receivers[:i], receivers[i+1:]...)
						break
					}
				}
			case m, open := <-in:
				if open {
					for _, r := range receivers {
						r.Send() <- m
					}
				} else {
					close(closed)
					for _, r := range receivers {
						r.Close()
					}

					return
				}
			}
		}
	}()

	return &Dispatcher{subscribe, unsubscribe, in, closed}
}

func (d *Dispatcher) Subscribe() Receiver {
	c := make(chan Receiver)
	select {
	case d.subscribe <- c:
		return <-c
	case <-d.closed:
		r := make(chan *Message)
		close(r)
		return ChanReceiver(r)
	}
}

func (d *Dispatcher) Unsubscribe(r Receiver) {
	select {
	case d.unsubscribe <- r:
	case <-d.closed:
	}
}

func (d *Dispatcher) Send() chan<- *Message { return d.in }
func (d *Dispatcher) Close()                { close(d.in) }
