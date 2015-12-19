package cast

import "math/rand"

const (
	// cars:
	standardMinTimeRate = -0.12
	standardMaxTimeRate = 0.12
	minSpeedRate        = -0.24
	maxSpeedRate        = 0.24
	minCarCrashRate     = 0.066
	maxCarCrashRate     = 0.33
	minAmortizationRate = 0.1
	maxAmortizationRate = 0.3
	numberRangeRate     = 6

	// stage:
	minStageAverage      = 1200000
	maxStageAverage      = 1500000
	minMeasurePoint      = 30000
	maxMeasurePoint      = 90000
	minSectionDifficulty = 0.066
	maxSectionDifficulty = 0.33
	minNetworkLatency    = 12
	maxNetworkLatency    = 300
	minNetworkStrength   = 0.66
	maxNetworkStrength   = 0.99
)

type generator struct{ rand *rand.Rand }

func newGenerator(seed int) *generator {
	return &generator{rand.New(rand.NewSource(int64(seed)))}
}

func (g *generator) between(min, max int) int {
	return min + g.rand.Intn(max-min)
}

func (g *generator) betweenFloat(min, max float64) float64 {
	return min + (max-min)*g.rand.Float64()
}

func (g *generator) delta(of int, minRate, maxRate float64) int {
	return int(float64(of) * g.betweenFloat(minRate, maxRate))
}

func (g *generator) generateCars(n int) []*car {
	var cars []*car

	numberRange := numberRangeRate * n
	takenNumbers := make(map[int]bool)

	names := make([]string, len(randomNames))
	copy(names, randomNames)

	for {
		if n == 0 {
			return cars
		}

		var number int
		for number == 0 || takenNumbers[number] {
			number = g.between(1, numberRange)
		}
		takenNumbers[number] = true

		ni := g.rand.Intn(len(names))
		driver := names[ni]
		names = append(names[:ni], names[ni+1:]...)

		ni = g.rand.Intn(len(names))
		codriver := names[ni]
		names = append(names[:ni], names[ni+1:]...)

		speedRate := g.betweenFloat(minSpeedRate, maxSpeedRate)
		crashRate := g.betweenFloat(minCarCrashRate, maxCarCrashRate)
		amortizationRate := g.betweenFloat(minAmortizationRate, maxAmortizationRate)

		cars = append(cars, &car{
			number:           number,
			gen:              g,
			driver:           driver,
			codriver:         codriver,
			speedRate:        speedRate,
			crashRate:        crashRate,
			amortizationRate: amortizationRate,
			condition:        1})

		n--
	}
}

func (g *generator) createStage(t *timer, d *Dispatcher) stage {
	var s stage
	desired := g.between(minStageAverage, maxStageAverage)
	total := 0
	counter := 0
	for {
		if total >= desired {
			break
		}

		mp := &measurePoint{number: counter + 1}
		s = append(s, mp)

		mp.average = g.between(minMeasurePoint, maxMeasurePoint)
		mp.difficulty = g.betweenFloat(minSectionDifficulty, maxSectionDifficulty)
		mp.radio = &network{
			latency:  g.between(minNetworkLatency, maxNetworkLatency),
			strength: g.betweenFloat(minNetworkStrength, maxNetworkStrength),
			timer:    t,
			gen:      g,
			remote:   d}

		total += mp.average
		counter++
	}

	return s
}
