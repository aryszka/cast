package cast

import (
	"fmt"
	"testing"
	"time"
)

func TestCompletesWhenStageIsEmpty(t *testing.T) {
	r := NewRace(300)
	c := r.gen.generateCars(1)[0]
	r.stage = nil

	var (
		completeReceived bool
		totalReceived    bool
	)

	done := make(chan struct{})
	r.Dispatcher.Subscribe(&Listener{func(m *Message) {
		if m.Key == fmt.Sprintf(finishFormat, c.number) {
			if completeReceived {
				t.Error("complete received twice")
				close(done)
				return
			}

			completeReceived = true
			if totalReceived {
				close(done)
			}
		} else if m.Key == fmt.Sprintf(totalFormat, c.number) {
			if totalReceived {
				t.Error("total received twice")
				close(done)
				return
			}

			totalReceived = true
			if completeReceived {
				close(done)
			}
		} else {
			t.Error("unexpected message")
			close(done)
		}
	}})

	time.Sleep(15 * time.Millisecond)
	go c.start(r)

	select {
	case <-done:
		return
	case <-time.After(15 * time.Millisecond):
		t.Error("timeout")
		return
	}
}

func TestCarCrashes(t *testing.T) {
	r := NewRace(10000)
	c := r.gen.generateCars(1)[0]

	r.stage = r.stage[:1]
	c.crashRate = 1
	for _, mp := range r.stage {
		mp.difficulty = 1
		mp.radio.(*network).latency = 0
		mp.radio.(*network).strength = 1
	}

	done := make(chan struct{})
	r.Dispatcher.Subscribe(&Listener{func(m *Message) {
		if m.Key != fmt.Sprintf(carReportFormat, 1, c.number, "crash") {
			t.Error("failed to report crash")
		}

		close(done)
	}})

	time.Sleep(15 * time.Millisecond)
	go c.start(r)
	select {
	case <-done:
		return
	case <-time.After(1200 * time.Millisecond):
		t.Error("timeout")
		return
	}
}

func TestCarGivesUp(t *testing.T) {
	r := NewRace(10000)
	c := r.gen.generateCars(1)[0]

	c.crashRate = 0
	c.condition = 0
	for _, mp := range r.stage {
		mp.difficulty = 0
		mp.radio.(*network).latency = 0
		mp.radio.(*network).strength = 1
	}

	done := make(chan struct{})
	r.Dispatcher.Subscribe(&Listener{func(m *Message) {
		if m.Key != fmt.Sprintf(carReportFormat, 1, c.number, "give-up") {
			t.Error("failed to report give up", m.Key)
		}

		close(done)
	}})

	time.Sleep(15 * time.Millisecond)
	go c.start(r)
	select {
	case <-done:
	case <-time.After(1200 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestCarFinishesRace(t *testing.T) {
	r := NewRace(10000)
	c := r.gen.generateCars(1)[0]

	c.crashRate = 0
	for _, mp := range r.stage {
		mp.difficulty = 0
		mp.radio.(*network).latency = 0
		mp.radio.(*network).strength = 1
	}

	done := make(chan struct{})
	r.Dispatcher.Subscribe(&Listener{func(m *Message) {
		if m.Key == fmt.Sprintf(finishFormat, c.number) {
			close(done)
		}
	}})

	time.Sleep(15 * time.Millisecond)
	go c.start(r)

	select {
	case <-done:
		return
	case <-time.After(1200 * time.Millisecond):
		t.Error("timeout")
		return
	}
}

func TestReport(t *testing.T) {
	l := &Listener{}
	mp := &measurePoint{
		number: 36,
		radio:  l}
	c := &car{number: 42, condition: 0.66}
	done := make(chan struct{})

	l.Handler = func(m *Message) {
		if m.Key != fmt.Sprintf(carReportFormat, 36, 42, "crash") {
			t.Error("failed to report crash")
		}

		close(done)
	}

	c.report(mp, "crash", "")
	select {
	case <-done:
	case <-time.After(30 * time.Millisecond):
		t.Error("timeout")
		return
	}
}
