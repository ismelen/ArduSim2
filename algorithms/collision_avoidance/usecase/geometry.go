package usecase

import (
	"math"

	"collision_avoidance/domain"
)

// Geometry variables matched from ArduSim
var (
	FunctionDistanceVsSpeed = [][]float64{
		{0.0, 1.0, 0.5, 0.0},
		{5.0, 2.0, 0.8, 0.0},
	}
	FunctionDistanceVsAlpha = [][]float64{
		{0.0, 1.0, 0.5, 0.0},
		{math.Pi, 2.0, 0.8, 0.0},
	}
	FunctionWaypointThresHold = []float64{1.0, 0.5, 0.0}
)

func getWaypointThreshold(speed float64) float64 {
	f := FunctionWaypointThresHold
	return f[0] + f[1]*speed + f[2]*speed*speed
}

func getAngle(start, end domain.Location2DUTM) *float64 {
	incX := end.X - start.X
	incY := end.Y - start.Y
	var angle float64

	if incX == 0 {
		if incY > 0 {
			angle = math.Pi / 2
		} else if incY < 0 {
			angle = -math.Pi / 2
		} else {
			return nil
		}
	} else {
		angle = math.Atan(incY / incX)
		if incX < 0 {
			if angle > 0 {
				angle = angle - math.Pi
			} else {
				angle = angle + math.Pi
			}
		}
	}
	return &angle
}

func getAngleDifference(l0Start, l0End, l1Start, l1End domain.Location2DUTM) *float64 {
	a0 := getAngle(l0Start, l0End)
	a1 := getAngle(l1Start, l1End)
	if a0 == nil || a1 == nil {
		return nil
	}
	res := *a1 - *a0
	if res < -math.Pi {
		res += math.Pi * 2
	}
	if res > math.Pi {
		res -= math.Pi * 2
	}
	return &res
}

func getCurveDistance(speed, angle float64) float64 {
	alpha := math.Abs(angle)

	fSpeed := FunctionDistanceVsSpeed
	fAlpha := FunctionDistanceVsAlpha

	prevSpeed := 0
	for i := 1; i < len(fSpeed); i++ {
		if fSpeed[i][0] < alpha {
			prevSpeed = i
		}
	}
	postSpeed := prevSpeed
	inc := 0.0
	if prevSpeed < len(fSpeed)-1 {
		postSpeed = prevSpeed + 1
		inc = (alpha - fSpeed[prevSpeed][0]) / (fSpeed[postSpeed][0] - fSpeed[prevSpeed][0])
	}

	dS0 := fSpeed[prevSpeed][1] + fSpeed[prevSpeed][2]*speed + fSpeed[prevSpeed][3]*speed*speed
	dS1 := fSpeed[postSpeed][1] + fSpeed[postSpeed][2]*speed + fSpeed[postSpeed][3]*speed*speed
	dSpeed := dS0 + (dS1-dS0)*inc

	prevAngle := 0
	for i := 1; i < len(fAlpha); i++ {
		if fAlpha[i][0] < speed {
			prevAngle = i
		}
	}
	postAngle := prevAngle
	inc = 0.0
	if prevAngle < len(fAlpha)-1 {
		postAngle = prevAngle + 1
		inc = (speed - fAlpha[prevAngle][0]) / (fAlpha[postAngle][0] - fAlpha[prevAngle][0])
	}

	dA0 := fAlpha[prevAngle][1] + fAlpha[prevAngle][2]*alpha + fAlpha[prevAngle][3]*alpha*alpha
	dA1 := fAlpha[postAngle][1] + fAlpha[postAngle][2]*alpha + fAlpha[postAngle][3]*alpha*alpha
	dAngle := dA0 + (dA1-dA0)*inc

	return math.Max(dSpeed, dAngle)
}

func getIntersection(p domain.Location2DUTM, l1, l2 domain.Location3DUTM) domain.Location2DUTM {
	// A simple perpendicular projection of point p onto line (l1, l2)
	// Extracted from MBCAP logic
	dx := l2.X - l1.X
	dy := l2.Y - l1.Y
	if dx == 0 && dy == 0 {
		return domain.Location2DUTM{X: l1.X, Y: l1.Y}
	}

	u := ((p.X-l1.X)*dx + (p.Y-l1.Y)*dy) / (dx*dx + dy*dy)
	return domain.Location2DUTM{
		X: l1.X + u*dx,
		Y: l1.Y + u*dy,
	}
}

func distance2D(p1, p2 domain.Location2DUTM) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// hasCollisionRisk checks if given two beacons there is risk of collision (Space-Time correlation)
func hasCollisionRisk(selfBeacon, receivedBeacon domain.Beacon, params *domain.MBCAPParam) *domain.Location3DUTM {
	checkTime := (receivedBeacon.State == domain.NORMAL) && (receivedBeacon.Speed >= params.MinAdvertismentSpeed) && (len(receivedBeacon.Points) > 1) && (selfBeacon.Speed >= params.MinAdvertismentSpeed) && (len(selfBeacon.Points) > 1)

	for i, selfPoint := range selfBeacon.Points {
		selfTime := selfBeacon.Time + int64(i)*params.HopTimeNS

		for j, recPoint := range receivedBeacon.Points {
			if j == 1 && (receivedBeacon.State == domain.GO_ON_PLEASE || receivedBeacon.State == domain.STAND_STILL) {
				break
			}

			risky := true
			if checkTime {
				beaconTime := receivedBeacon.Time + int64(j)*params.HopTimeNS
				diff := selfTime - beaconTime
				if diff < 0 {
					diff = -diff
				}
				if diff >= params.CollisionWarningTimeOffset {
					risky = false
				}
			}

			if risky {
				distXY := math.Sqrt(math.Pow(selfPoint.X-recPoint.X, 2) + math.Pow(selfPoint.Y-recPoint.Y, 2))
				distZ := math.Abs(selfPoint.Z - recPoint.Z)

				if distXY < params.CollisionWarningDistance && distZ < params.CollisionWarningAltitudeOffset {
					return &selfPoint
				}
			}
		}
	}
	return nil
}
