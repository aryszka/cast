package cast

type Message struct {
	Key     string
	Content string
}

type Connection interface {
	Send(*Message)
}

type network struct {
	latency  int
	strength float64
	timer    *timer
	gen      *generator
	remote   Connection
}

func (c *network) Send(m *Message) {
	if c.gen.rand.Float64() > c.strength {
		return
	}

	go func() {
		c.timer.after(c.latency)
		c.remote.Send(m)
	}()
}

type Dispatcher struct {
	snd    chan *Message
	subscr chan Connection
}

func newDispatcher() *Dispatcher {
	snd := make(chan *Message)
	subscr := make(chan Connection)
	d := &Dispatcher{snd, subscr}

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
			}
		}
	}()

	return d
}

func (d *Dispatcher) Send(m *Message)        { go func() { d.snd <- m }() }
func (d *Dispatcher) Subscribe(c Connection) { go func() { d.subscr <- c }() }

type Listener struct{ Handler func(m *Message) }

func (l *Listener) Send(m *Message) { go l.Handler(m) }
