package cast

import "fmt"

const carReportFormat = "measure-point.%d.car.%d.%s"

type car struct {
	number    int
	gen       *generator
	driver    string
	codriver  string
	speedRate float64
	crashRate float64
	condition float64
}

func (c *car) report(mp *measurePoint, msg string, content interface{}) {
	mp.radio.Send(&Message{
		Key:     fmt.Sprintf(carReportFormat, mp.number, c.number, msg),
		Content: fmt.Sprint(content)})
}

func (c *car) start(r *Race) {
	stage := r.stage
	for {
		if len(stage) == 0 {
			r.finish(c)
			return
		}

		measurePoint := stage[0]
		stage = stage[1:]

		crash := c.gen.rand.Float64()*c.condition < (c.crashRate/float64(len(r.stage))+measurePoint.difficulty)/2
		c.condition -= amortizationRate * measurePoint.difficulty

		t := measurePoint.average +
			c.gen.delta(measurePoint.average, 0, c.speedRate) +
			c.gen.delta(measurePoint.average, standardMinTimeRate, standardMaxTimeRate)
		r.timer.wait(t)

		if crash {
			c.report(measurePoint, "crash", "")
			return
		}

		if c.condition <= 0 {
			c.report(measurePoint, "give-up", "")
			return
		}

		c.report(measurePoint, "condition", c.condition)

		if len(stage) > 0 {
			c.report(measurePoint, "pass", r.timer.now())
		}
	}
}
