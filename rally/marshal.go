package rally

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	carStartLineTimeout      = 15000
	carStartTimeout          = 60000
	maxTotalTimeRate         = 1.5
	raceStartMessage         = "race.start"
	raceCloseMessage         = "race.close"
	marshalServiceInfoFormat = "marshal.%s"
	marshalCarStatusFormat   = "marshal.car.%d.%s"
)

type carResult struct {
	started      bool
	startTime    int
	finished     bool
	finishTime   int
	disqualified bool
	safe         bool
}

func (cr carResult) ready() bool {
	return !(cr.started || cr.finished || cr.disqualified)
}

func (cr carResult) racing() bool {
	return cr.started && !cr.finished && !cr.disqualified
}

func (cr carResult) setStarted(t int) carResult {
	if !cr.ready() {
		return cr
	}

	cr.started = true
	cr.startTime = t
	return cr
}

func (cr carResult) setFinished(t int) carResult {
	if !cr.racing() {
		return cr
	}

	cr.finished = true
	cr.finishTime = t
	return cr
}

func (cr carResult) setDisqualified() carResult {
	cr.disqualified = true
	return cr
}

func (cr carResult) setSafe() carResult {
	cr.safe = true
	return cr
}

type marshal struct {
	timer            *timer
	intercom         *Dispatcher
	intercomReceiver chan *Message
	maxTotalTime     int
	markers          []*marker
	cars             map[int]carResult
	vcontactIn       chan Connection
	vcontactOut      chan Connection
	carVisuals       *Dispatcher
	fieldVisual      chan *Message
	lastCarTimeout   <-chan struct{}
}

func (m *marshal) simulate(t *timer, d *Dispatcher, stage []*marker) {
	m.timer = t
	m.intercom = d
	m.markers = stage

	m.intercomReceiver = make(chan *Message)
	m.intercom.Subscribe(Receiver(m.intercomReceiver))

	m.vcontactIn = make(chan Connection)
	m.vcontactOut = make(chan Connection)
	m.carVisuals = newDispatcher()
	m.fieldVisual = make(chan *Message)
	m.cars = make(map[int]carResult)
	m.maxTotalTime = int(maxTotalTimeRate * float64(sumAverage(stage)))

	m.dispatchServiceInfo("race-prepared")
	if !m.waitForCarsRegistered() {
		m.closeRace()
		return
	}

	m.dispatchServiceInfo("race-start", m.timer.now())
	if !m.startRace() {
		m.closeRace()
		return
	}

	if !m.waitForCarsFinish() {
		m.closeRace()
		return
	}

	m.dispatchServiceInfo("race-over", m.timer.now())
	m.waitForRaceClose()
	m.closeRace()
}

func sumAverage(stage []*marker) int {
	s := 0
	for _, si := range stage {
		s += si.average
	}

	return s
}

func (m *marshal) dispatchServiceInfo(message string, content ...interface{}) {
	m.intercom.Send(&Message{
		Key:     fmt.Sprintf(marshalServiceInfoFormat, message),
		Content: fmt.Sprint(content...)})
}

func (m *marshal) dispatchCarInfo(number int, typ string, content ...interface{}) {
	m.intercom.Send(&Message{
		Key:     fmt.Sprintf(marshalCarStatusFormat, number, typ),
		Content: fmt.Sprint(content...)})
}

func carSeenAt(msg *Message, position string) (int, bool) {
	k := strings.Split(msg.Key, ".")
	if len(k) < 3 || k[0] != "car" || k[2] != position {
		return 0, false
	}

	n, err := strconv.Atoi(k[1])
	if err != nil {
		return 0, false
	}

	return n, true
}

func (m *marshal) messageToCar(number int, message string) {
	m.carVisuals.Send(&Message{Key: message, Content: strconv.Itoa(number)})
}

func (m *marshal) sendRaceOver() {
	for n, _ := range m.cars {
		m.messageToCar(n, "race-over")
	}
}

func (m *marshal) allSafe() bool {
	for _, cr := range m.cars {
		if !cr.safe {
			return false
		}
	}

	return true
}

func (m *marshal) allFinishedOrSafe() bool {
	for _, cr := range m.cars {
		if !cr.safe && !cr.finished {
			return false
		}
	}

	return true
}

func (m *marshal) waitForAllSafe() {
	for {
		if m.allSafe() {
			return
		}

		msg := <-m.fieldVisual
		if n, ok := carSeenAt(msg, "safe"); ok {
			m.cars[n] = m.cars[n].setSafe()
		}
	}
}

func (m *marshal) waitForCarsRegistered() bool {
	for {
		select {
		case v := <-m.vcontactIn:
			m.carVisuals.Subscribe(v)
		case v := <-m.vcontactOut:
			m.carVisuals.Unsubscribe(v)
		case msg := <-m.fieldVisual:
			if number, ok := carSeenAt(msg, "start-area"); ok {
				m.cars[number] = carResult{}
				m.dispatchCarInfo(number, "registered")
			}
		case msg := <-m.intercomReceiver:
			switch msg.Key {
			case raceStartMessage:
				return true
			case raceCloseMessage:
				return false
			}
		}
	}
}

func (m *marshal) startCar(number int) (int, bool) {
	for {
		select {
		case <-m.timer.after(carStartLineTimeout):
			m.messageToCar(number, "race-over")
			return -1, false
		case msg := <-m.fieldVisual:
			if n, ok := carSeenAt(msg, "start-line"); ok && n == number {
				t := m.timer.now()
				m.messageToCar(number, "start")
				return t, false
			}
		case msg := <-m.intercomReceiver:
			if msg.Key == raceCloseMessage {
				return -1, true
			}
		}
	}
}

func (m *marshal) startRace() bool {
	for number, _ := range m.cars {
		carStarted, raceClose := m.startCar(number)
		if raceClose {
			return false
		}

		if carStarted < 0 {
			m.cars[number] = m.cars[number].setDisqualified()
			m.dispatchCarInfo(number, "disqualified")
		} else {
			m.cars[number] = m.cars[number].setStarted(carStarted)
			m.dispatchCarInfo(number, "started", carStarted)
		}

		<-m.timer.after(carStartTimeout)
	}

	lct := m.maxTotalTime - carStartTimeout
	if lct < 0 {
		lct = 0
	}

	m.lastCarTimeout = m.timer.after(lct)
	return true
}

func (m *marshal) disqualifySlowCars() {
	var racing []int
	for number, cr := range m.cars {
		if cr.racing() {
			racing = append(racing, number)
		}
	}

	for _, number := range racing {
		m.cars[number] = m.cars[number].setDisqualified()
	}
}

func (m *marshal) carFinished(number, t int) {
	cr, ok := m.cars[number]
	if ok && t-cr.startTime <= m.maxTotalTime {
		m.cars[number] = cr.setFinished(t)
		m.dispatchCarInfo(number, "finished", t)
	} else {
		m.cars[number] = cr.setDisqualified()
		m.dispatchCarInfo(number, "disqualified", t)
	}
}

func (m *marshal) waitForCarsFinish() bool {
	for {
		select {
		case <-m.lastCarTimeout:
			m.disqualifySlowCars()
			m.sendRaceOver()
		case msg := <-m.fieldVisual:
			if number, ok := carSeenAt(msg, "finish-line"); ok {
				m.carFinished(number, m.timer.now())
			} else if number, ok := carSeenAt(msg, "safe"); ok {
				m.cars[number] = m.cars[number].setSafe()
			}

			if m.allFinishedOrSafe() {
				return true
			}
		case msg := <-m.intercomReceiver:
			if msg.Key == raceCloseMessage {
				return false
			}
		case v := <-m.vcontactOut:
			m.carVisuals.Unsubscribe(v)
		}
	}
}

func resultRequest(msg *Message) (int, bool) {
	k := strings.Split(msg.Key, ".")
	if len(k) == 0 || k[0] != "result-request" {
		return -1, false
	}

	if len(k) == 1 {
		return -1, true
	}

	if n, err := strconv.Atoi(k[1]); err != nil {
		return -1, false
	} else {
		return n, true
	}
}

func (m *marshal) dispatchCarResult(number int) {
	dispatch := func(number int, cr carResult) {
		if cr.started {
			m.dispatchCarInfo(number, "started", cr.startTime)
		}

		if cr.disqualified {
			m.dispatchCarInfo(number, "disqualified")
		} else if cr.finished {
			m.dispatchCarInfo(number, "finished", cr.finishTime)
		}
	}

	if number < 0 {
		for n, cr := range m.cars {
			dispatch(n, cr)
		}

		return
	}

	if cr, ok := m.cars[number]; ok {
		dispatch(number, cr)
	}
}

func (m *marshal) waitForRaceClose() {
	for {
		msg := <-m.intercomReceiver
		if number, ok := resultRequest(msg); ok {
			m.dispatchCarResult(number)
		} else if msg.Key == raceCloseMessage {
			return
		}
	}
}

func (m *marshal) closeRace() {
	m.sendRaceOver()
	m.waitForAllSafe()
	m.carVisuals.Close()

	m.intercom.Unsubscribe(Receiver(m.intercomReceiver))
	for {
		if <-m.intercomReceiver == Disconnected {
			break
		}
	}

	m.dispatchServiceInfo("race-closed")
}

// stage control:
func (m *marshal) stage() []*marker                     { return m.markers }
func (m *marshal) receiveVisual(visual Connection)      { m.vcontactIn <- visual }
func (m *marshal) stopWatchingVisual(visual Connection) { m.vcontactOut <- visual }
func (m *marshal) visual() Connection                   { return Receiver(m.fieldVisual) }
