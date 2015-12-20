package cast

const ()

type Race struct {
	gen        *generator
	timer      *timer
	Dispatcher *Dispatcher
	stage      []*marker
	cars       []*car
}

func NewRace(timeRate float64) *Race {
	g := newGenerator(0)
	t := newTimer(timeRate)
	d := newDispatcher()

	s := g.createStage(g.between(minStageAverage, maxStageAverage))
	for _, mp := range s {
		mp.simulate(g, t, d)
	}

	c := g.generateCars(g.between(minCars, maxCars))

	return &Race{
		gen:        g,
		timer:      t,
		Dispatcher: d,
		stage:      s,
		cars:       c}
}
