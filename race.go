package cast

import (
	"fmt"
	"strconv"
)

const (
	startFormat  = "car.%d.start"
	finishFormat = "car.%d.finish"
	totalFormat  = "car.%d.total"
	startDelta   = 60000
	minCars      = 18
	maxCars      = 36
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

func (r *Race) reportStart(c *car) {
	r.Dispatcher.Send(&Message{Key: fmt.Sprintf(startFormat, c.number)})
}

func (r *Race) reportComplete(c *car, t int) {
	r.Dispatcher.Send(&Message{Key: fmt.Sprintf(finishFormat, c.number)})
	r.Dispatcher.Send(&Message{
		Key:     fmt.Sprintf(totalFormat, c.number),
		Content: strconv.Itoa(t)})
}

func (r *Race) start(c *car) {
	r.startTimes[c.number] = r.timer.now()
	r.reportStart(c)
	go c.start(r)
}

func (r *Race) finish(c *car) {
	t := r.timer.now() - r.startTimes[c.number]
	r.reportComplete(c, t)
}

func (r *Race) Start() {
	r.startTimes = make(map[int]int)
	var started bool
	for _, c := range r.cars {
		if started {
			r.timer.after(startDelta)
		}

		r.start(c)
	}

	<-make(chan struct{})
}
