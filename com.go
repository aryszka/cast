package cast

type Message struct {
	Key     string
	Content string
}

type Connection interface {
	Send(*Message)
}

var Disconnected = &Message{}

type Receiver chan<- *Message

func (r Receiver) Send(m *Message) {
	go func() { r <- m }()
}

type network struct {
	latency  int
	strength float64
	timer    *timer
	gen      *generator
	remote   Connection
}

func (nw *network) Send(m *Message) {
	go func() {
		if nw.gen.rand.Float64() > nw.strength {
			return
		}

		<-nw.timer.after(nw.latency)
		nw.remote.Send(m)
	}()
}

type Dispatcher struct {
	snd      chan *Message
	subscr   chan Connection
	unsubscr chan Connection
	close    chan struct{}
}

func newDispatcher() *Dispatcher {
	snd := make(chan *Message)
	subscr := make(chan Connection)
	unsubscr := make(chan Connection)
	close := make(chan struct{})

	go func() {
		var connections []Connection
		for {
			select {
			case m := <-snd:
				for _, c := range connections {
					go func(c Connection) { c.Send(m) }(c)
				}
			case c := <-subscr:
				connections = append(connections, c)
			case c := <-unsubscr:
				for i, conn := range connections {
					if conn == c {
						connections = append(connections[:i], connections[i+1:]...)
						break
					}
				}

				go func() { c.Send(Disconnected) }()
			case <-close:
				return
			}
		}
	}()

	return &Dispatcher{snd, subscr, unsubscr, close}
}

func (d *Dispatcher) Send(m *Message) {
	go func() { d.snd <- m }()
}

func (d *Dispatcher) Subscribe(c Connection) {
	go func() { d.subscr <- c }()
}

func (d *Dispatcher) Unsubscribe(c Connection) {
	go func() { d.unsubscr <- c }()
}

func (d *Dispatcher) Close() {
	close(d.close)
}
