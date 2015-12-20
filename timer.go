package cast

import "time"

type timer struct {
	timeRate float64
	start    time.Time
}

func newTimer(timeRate float64) *timer {
	return &timer{timeRate, time.Now()}
}

func (t *timer) scale(d time.Duration) int {
	return int(float64(d/time.Millisecond) * t.timeRate)
}

func (t *timer) unscale(dt int) time.Duration {
	return time.Duration(float64(dt)/t.timeRate) * time.Millisecond
}

func (t *timer) now() int {
	return t.scale(time.Now().Sub(t.start))
}

func (t *timer) after(dt int) <-chan struct{} {
	to := make(chan struct{})
	go func() {
		<-time.After(t.unscale(dt))
		close(to)
	}()

	return to
}
