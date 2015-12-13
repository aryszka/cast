package cast

import "time"

type timer struct {
	timeRate float64
	start    time.Time
}

func newTimer(timeRate float64) *timer {
	return &timer{timeRate, time.Now()}
}

func (t *timer) now() int {
	return int(float64(time.Now().Sub(t.start)/time.Millisecond) * t.timeRate)
}

func (t *timer) after(dt int) {
	time.Sleep(time.Duration(float64(dt)/t.timeRate) * time.Millisecond)
}
