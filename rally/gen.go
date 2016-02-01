package rally

import "math/rand"

const (
	// cars:
	minSpeedRate        = -0.24
	maxSpeedRate        = 0.24
	minCarCrashRate     = 0.066
	maxCarCrashRate     = 0.33
	minAmortizationRate = 0.1
	maxAmortizationRate = 0.3
	numberRangeRate     = 6

	// stage:
	minMeasurePoint           = 30000
	maxMeasurePoint           = 90000
	minSectionDifficulty      = 0.066
	maxSectionDifficulty      = 0.33
	minNetworkLatency         = 12
	maxNetworkLatency         = 300
	minNetworkStrength        = 0.66
	maxNetworkStrength        = 0.99
	minMarkerReceiverStrength = 0.45
	maxMarkerReceiverStrength = 0.72
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

func (g *generator) name(names []string) (string, []string) {
	i := g.rand.Intn(len(names))
	return names[i], append(names[:i], names[i+1:]...)
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

		var driver, codriver string
		driver, names = g.name(names)
		codriver, names = g.name(names)

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

func (g *generator) createStage(desiredLength int) []*marker {
	var s []*marker
	total := 0
	counter := 0
	for {
		if total >= desiredLength {
			break
		}

		number := counter + 1
		average := g.between(minMeasurePoint, maxMeasurePoint)
		difficulty := g.betweenFloat(minSectionDifficulty, maxSectionDifficulty)
		networkLatency := g.between(minNetworkLatency, maxNetworkLatency)
		networkStrength := g.betweenFloat(minNetworkStrength, maxNetworkStrength)
		receiverStrength := g.betweenFloat(minMarkerReceiverStrength, maxMarkerReceiverStrength)

		s = append(s, &marker{
			number:           number,
			average:          average,
			difficulty:       difficulty,
			networkLatency:   networkLatency,
			networkStrength:  networkStrength,
			receiverStrength: receiverStrength})

		total += average
		counter++
	}

	return s
}
