package rally

import (
	"fmt"
	"strconv"
)

const (
	timeFormat    = "marker.%d.%s.time"
	forwardFormat = "marker.%d.%s"
)

type marker struct {
	number           int
	average          int
	difficulty       float64
	networkLatency   int
	networkStrength  float64
	receiverStrength float64
	timer            *timer
	network          Connection
	fieldReceiver    Connection
	cls              chan struct{}
}

func (m *marker) simulate(g *generator, t *timer, d *Dispatcher) {
	receiver := make(chan *Message)
	m.cls = make(chan struct{})

	m.network = &network{
		latency:  m.networkLatency,
		strength: m.networkStrength,
		timer:    t,
		gen:      g,
		remote:   d}

	m.fieldReceiver = &network{
		latency:  0,
		strength: m.receiverStrength,
		timer:    t,
		gen:      g,
		remote:   Receiver(receiver)}

	go m.forward(receiver)
}

func (m *marker) forward(r chan *Message) {
	for {
		select {
		case msg := <-r:
			t := m.timer.now()

			m.network.Send(&Message{
				Key:     fmt.Sprintf(timeFormat, m.number, msg.Key),
				Content: strconv.Itoa(t)})

			m.network.Send(&Message{
				Key:     fmt.Sprintf(forwardFormat, m.number, msg.Key),
				Content: msg.Content})
		case <-m.cls:
			return
		}
	}
}

func (m *marker) close() {
	close(m.cls)
}
