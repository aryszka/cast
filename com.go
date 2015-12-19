package cast

type Message struct {
	Key     string
	Content string
}

type Connection interface {
	Send(*Message)
}

type Handler interface {
	Handle(*Message)
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
		c.timer.wait(c.latency)
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

	return &Dispatcher{snd, subscr}
}

func (d *Dispatcher) Send(m *Message)        { go func() { d.snd <- m }() }
func (d *Dispatcher) Subscribe(c Connection) { go func() { d.subscr <- c }() }

type Listener struct{ Handler Handler }

func (l *Listener) Send(m *Message) { go l.Handler.Handle(m) }

type HandlerFunc func(m *Message)

func (hf HandlerFunc) Handle(m *Message) { hf(m) }
