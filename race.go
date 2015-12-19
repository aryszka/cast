package cast

import "fmt"

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

func (r *Race) sendFormat(content interface{}, format string, args ...interface{}) {
	r.Dispatcher.Send(&Message{
		Key:     fmt.Sprintf(format, args...),
		Content: fmt.Sprint(content)})
}

func (r *Race) reportStart(c *car, t int) {
	r.sendFormat(t, startFormat, c.number)
}

func (r *Race) reportComplete(c *car, t, dt int) {
	r.sendFormat(t, finishFormat, c.number)
	r.sendFormat(dt, totalFormat, c.number)
}

func (r *Race) start(c *car) {
	t := r.timer.now()
	r.startTimes[c.number] = t
	r.reportStart(c, t)
	go c.start(r)
}

func (r *Race) finish(c *car) {
	n := r.timer.now()
	t := r.startTimes[c.number]
	dt := n - t
	r.reportComplete(c, n, dt)
}

func (r *Race) Start() {
	r.startTimes = make(map[int]int)
	var started bool
	for _, c := range r.cars {
		if started {
			r.timer.wait(startDelta)
		}

		r.start(c)
		started = true
	}
}
