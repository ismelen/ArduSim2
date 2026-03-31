package usecase

import (
	"math"
	"safe_takeoff/domain"
)

// GenerateCircleFormation creates a circle with UAV 0 at the center.
func GenerateCircleFormation(numUAVs int, minDistance float64) map[int]domain.Location {
	positions := make(map[int]domain.Location)
	if numUAVs == 0 {
		return positions
	}

	// UAV 0 is always at the center
	positions[0] = domain.Location{Lat: 0, Lon: 0}

	if numUAVs == 1 {
		return positions
	}

	var radius float64
	numSlaves := float64(numUAVs - 1)

	if numUAVs <= 7 {
		radius = minDistance
	} else {
		// Formula from ArduSim Circle.java
		radius = minDistance / (2 * math.Sin(math.Pi/numSlaves))
	}

	for i := 1; i < numUAVs; i++ {
		angle := float64(i-1) * 2 * math.Pi / float64(numUAVs-1)
		x := radius * math.Cos(angle)
		y := radius * math.Sin(angle)
		positions[i] = domain.Location{Lat: y, Lon: x}
	}

	return positions
}

// GenerateLinearFormation creates a line of UAVs.
func GenerateLinearFormation(numUAVs int, minDistance float64) map[int]domain.Location {
	positions := make(map[int]domain.Location)
	centerIndex := numUAVs / 2

	for i := 0; i < numUAVs; i++ {
		x := float64(i-centerIndex) * minDistance
		positions[i] = domain.Location{Lat: 0, Lon: x}
	}

	return positions
}

// GenerateMatrixFormation creates a grid growth formation.
func GenerateMatrixFormation(numUAVs int, minDistance float64) map[int]domain.Location {
	positions := make(map[int]domain.Location)
	if numUAVs == 0 {
		return positions
	}
	positions[0] = domain.Location{Lat: 0, Lon: 0}

	distance := 0
	for i := 0; i < numUAVs-1; i++ {
		currentEven := int(math.Floor(math.Sqrt(float64(i))))%2 == 0
		nextOdd := int(math.Floor(math.Sqrt(float64(i+1))))%2 == 1
		if currentEven && nextOdd {
			distance += 1
		}

		// simplified spiral logic for parity
		theta := float64(i)*math.Pi/2.0 + getStartAngle(i/4)
		x := float64(distance) * math.Cos(theta)
		y := float64(distance) * math.Sin(theta)

		positions[i+1] = domain.Location{
			Lat: math.Round(y) * minDistance,
			Lon: math.Round(x) * minDistance,
		}
	}
	return positions
}

func getStartAngle(index int) float64 {
	if index%2 == 1 {
		return math.Pi / 4.0
	}
	return 0
}

// GenerateRandomFormation creates a jittered grid.
func GenerateRandomFormation(numUAVs int, minDistance float64) map[int]domain.Location {
	positions := make(map[int]domain.Location)
	if numUAVs == 0 {
		return positions
	}
	positions[0] = domain.Location{Lat: 0, Lon: 0}

	const maxJitter = 0.40
	side := int(math.Ceil(math.Sqrt(float64(numUAVs) / 0.5)))
	halfSide := side / 2

	used := make(map[int]bool)
	used[halfSide*side+halfSide] = true // center

	// Deterministic random for parity (ArduSim uses seed 1)
	r := newDeterministicRand(1)

	for i := 1; i < numUAVs; i++ {
		var cell int
		for {
			cell = r.Intn(side * side)
			if !used[cell] {
				used[cell] = true
				break
			}
		}

		row := (cell / side) - halfSide
		col := (cell % side) - halfSide

		jitterX := r.Float64() * minDistance * maxJitter
		jitterY := r.Float64() * minDistance * maxJitter

		positions[i] = domain.Location{
			Lat: float64(row)*(1+maxJitter)*minDistance + jitterY,
			Lon: float64(col)*(1+maxJitter)*minDistance + jitterX,
		}
	}

	return positions
}

type detRand struct {
	seed int64
}

func newDeterministicRand(seed int64) *detRand {
	return &detRand{seed: seed}
}

func (r *detRand) Intn(n int) int {
	r.seed = (r.seed*1103515245 + 12345) & 0x7fffffff
	return int(r.seed) % n
}

func (r *detRand) Float64() float64 {
	r.seed = (r.seed*1103515245 + 12345) & 0x7fffffff
	return float64(r.seed) / float64(0x7fffffff)
}
