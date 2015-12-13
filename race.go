package cast

import (
	"fmt"
	"strconv"
)

const (
	completeFormat = "car.%d.complete"
	totalFormat    = "car.%d.total"
	startDelta     = 60000
	minCars        = 18
	maxCars        = 36
)

type measurePoint struct {
	number     int
	average    int
	difficulty float64
	radio      Connection
}

type stage []*measurePoint

type Race struct {
	gen        *generator
	timer      *timer
	Dispatcher *Dispatcher
	stage      stage
	cars       []*car
	startTimes map[int]int
}

func NewRace(timeRate float64) *Race {
	g := newGenerator(0)
	t := newTimer(timeRate)
	d := newDispatcher()
	s := g.createStage(t, d)
	c := g.generateCars(g.between(minCars, maxCars))
	return &Race{
		gen:        g,
		timer:      t,
		Dispatcher: d,
		stage:      s,
		cars:       c}
}

func (r *Race) reportComplete(c *car) {
	r.Dispatcher.Send(&Message{Key: fmt.Sprintf(completeFormat, c.number)})
	r.Dispatcher.Send(&Message{
		Key:     fmt.Sprintf(totalFormat, c.number),
		Content: strconv.Itoa(r.timer.now() - r.startTimes[c.number])})
}

func (r *Race) Start() {
	var started bool
	for _, c := range r.cars {
		if started {
			r.timer.after(startDelta)
		}

		c.start(r)
	}

	<-make(chan struct{})
}
