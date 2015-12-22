package cast

import (
	"fmt"
	"strconv"
)

const (
	carMessageFormat       = "car.%d.%s"
	environmentMinTimeRate = -0.12
	environmentMaxTimeRate = 0.12
)

type stageControl interface {
	stage() []*marker
	receiveVisual(Connection)
	stopWatchingVisual(Connection)
	visual() Connection
}

type car struct {
	number           int
	driver           string
	codriver         string
	speedRate        float64
	crashRate        float64
	amortizationRate float64
	condition        float64
	gen              *generator
	timer            *timer
	stageControl     stageControl
	visual           chan *Message
}

func (c *car) simulate(g *generator, timer *timer, sc stageControl) {
	c.gen = g
	c.timer = timer
	c.stageControl = sc

	c.visual = make(chan *Message)
	c.stageControl.receiveVisual(Receiver(c.visual))

	if !c.goAndWaitMarshalling("start-area", "start-line") {
		return
	}

	if !c.goAndWaitMarshalling("start-line", "start") {
		return
	}

	c.race()
}

func (c *car) send(conn Connection, msg string, content ...interface{}) {
	conn.Send(&Message{
		Key:     fmt.Sprintf(carMessageFormat, c.number, msg),
		Content: fmt.Sprint(content...)})
}

func (c *car) goTo(position string) {
	c.send(c.stageControl.visual(), position)
}

func (c *car) goToSafe() {
	c.stageControl.stopWatchingVisual(Receiver(c.visual))
	for {
		if <-c.visual == Disconnected {
			break
		}
	}

	c.goTo("safe")
}

func (c *car) markerTransmit(m *marker, msg string, content ...interface{}) {
	c.send(m.fieldReceiver, msg, content...)
}

func (c *car) messageOf(m *Message, typ string) bool {
	return m.Key == typ && m.Content == strconv.Itoa(c.number)
}

func (c *car) goAndWaitMarshalling(position, message string) bool {
	c.goTo(position)
	if c.messageOf(<-c.visual, message) {
		return true
	}

	c.goToSafe()
	return false
}

func (c *car) raceOverOrContinue() bool {
	select {
	case m := <-c.visual:
		if c.messageOf(m, "race-over") {
			return true
		}
	default:
	}

	return false
}

func (c *car) crash(stageLength int, m *marker) bool {
	chanceToContinue := c.gen.rand.Float64() * c.condition
	carCrashRate := c.crashRate / float64(stageLength)
	combinedCrashRate := (carCrashRate + m.difficulty) / 2
	return chanceToContinue < combinedCrashRate
}

func (c *car) amortization(m *marker) float64 {
	return c.gen.rand.Float64() * c.amortizationRate * m.difficulty
}

func (c *car) raceToMarker(m *marker) {
	carEffect := c.gen.delta(m.average, 0, c.speedRate)
	environmentEffect := c.gen.delta(m.average,
		environmentMinTimeRate, environmentMaxTimeRate)
	<-c.timer.after(m.average + carEffect + environmentEffect)
}

func (c *car) race() {
	stage := c.stageControl.stage()
	stageLength := len(stage)
	for {
		if len(stage) == 0 {
			c.goTo("finish-line")
			c.goToSafe()
			return
		}

		marker := stage[0]
		stage = stage[1:]

		crash := c.crash(stageLength, marker)
		c.condition -= c.amortization(marker)
		c.raceToMarker(marker)

		if crash {
			c.markerTransmit(marker, "crash")
			c.goToSafe()
			return
		}

		if c.condition <= 0 {
			c.markerTransmit(marker, "give-up")
			c.goToSafe()
			return
		}

		c.markerTransmit(marker, "condition", c.condition)
		c.markerTransmit(marker, "pass")

		if c.raceOverOrContinue() {
			c.goToSafe()
			return
		}
	}
}
