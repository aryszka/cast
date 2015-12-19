package cast

import "testing"

func TestBetween(t *testing.T) {
	for i := 0; i < 18; i++ {
		g := newGenerator(i)
		n := g.between(36, 42)
		if n < 36 || n >= 42 {
			t.Error("invalid range")
		}
	}
}

func TestBetweenFloat(t *testing.T) {
	for i := 0; i < 18; i++ {
		g := newGenerator(i)
		n := g.betweenFloat(36, 42)
		if n < 36 || n >= 42 {
			t.Error("invalid range")
		}
	}
}

func TestDelta(t *testing.T) {
	for i := 0; i < 18; i++ {
		g := newGenerator(i)
		n := g.delta(24, -0.1, 0.2)
		if n < -2 || n >= 5 {
			t.Error("invalid range", n)
		}
	}
}

func TestGeneratedCarNumbers(t *testing.T) {
	const n = 60
	numberRange := numberRangeRate * n
	for i := 0; i < 3; i++ {
		g := newGenerator(i)
		cars := g.generateCars(n)
		numbers := make(map[int]bool)
		for _, c := range cars {
			if numbers[c.number] {
				t.Error("conflicting number")
				return
			}

			if c.number < 1 || c.number >= numberRange {
				t.Error("invalid number")
			}

			numbers[c.number] = true
		}
	}
}

func TestGenerateDriverNames(t *testing.T) {
	const n = 60
	for i := 0; i < 3; i++ {
		g := newGenerator(i)
		cars := g.generateCars(n)
		names := make(map[string]bool)
		for _, c := range cars {
			if names[c.driver] || names[c.codriver] {
				t.Error("failed to generate unique names")
				return
			}

			names[c.driver] = true
			names[c.codriver] = true
		}
	}
}

func TestGenerateCarAttributes(t *testing.T) {
	const n = 60
	for i := 0; i < 3; i++ {
		g := newGenerator(i)
		cars := g.generateCars(n)
		for _, c := range cars {
			if c.speedRate < minSpeedRate || c.speedRate >= maxSpeedRate {
				t.Error("failed to generate speed rate in range")
				return
			}

			if c.crashRate < minCarCrashRate || c.crashRate >= maxCarCrashRate {
				t.Error("failed to generate crash rate in range")
				return
			}

			if c.condition != 1 {
				t.Error("failed to generate right car condition")
			}
		}
	}
}

func TestStageLength(t *testing.T) {
	for i := 0; i < 3; i++ {
		tm := newTimer(1)
		d := newDispatcher()
		g := newGenerator(0)
		s := g.createStage(tm, d)

		total := 0
		for _, si := range s {
			total += si.average
		}

		if total < minStageAverage || total >= maxStageAverage+maxMeasurePoint {
			t.Error("failed to generate stage in range")
		}
	}
}

func TestMeasurePointsLength(t *testing.T) {
	for i := 0; i < 3; i++ {
		tm := newTimer(1)
		d := newDispatcher()
		g := newGenerator(0)
		s := g.createStage(tm, d)

		for _, mp := range s {
			if mp.average < minMeasurePoint || mp.average >= maxMeasurePoint {
				t.Error("failed to generate stage measure points in range")
				return
			}
		}
	}
}

func TestMeasurePointDifficulty(t *testing.T) {
	for i := 0; i < 3; i++ {
		tm := newTimer(1)
		d := newDispatcher()
		g := newGenerator(0)
		s := g.createStage(tm, d)

		for _, mp := range s {
			if mp.difficulty < minSectionDifficulty || mp.difficulty >= maxSectionDifficulty {
				t.Error("failed to generate stage measure point difficulties in range")
				return
			}
		}
	}
}

func TestMeasurePointConnection(t *testing.T) {
	for i := 0; i < 3; i++ {
		tm := newTimer(1)
		d := newDispatcher()
		g := newGenerator(0)
		s := g.createStage(tm, d)

		for _, mp := range s {
			if mp.radio.(*network).latency < minNetworkLatency ||
				mp.radio.(*network).latency >= maxNetworkLatency ||
				mp.radio.(*network).strength < minNetworkStrength ||
				mp.radio.(*network).strength >= maxNetworkStrength {
				t.Error("failed to generate stage measure point network attributes in range")
				return
			}
		}
	}
}
