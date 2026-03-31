package usecase

import (
	"math"
	"safe_takeoff/domain"
)

// MatchUAVsToFormation matches UAVs to target positions based on the selected strategy.
func MatchUAVsToFormation(uavs map[int]domain.Location, targets map[int]domain.Location, strategy string) map[int]domain.Location {
	if strategy == "Hungarian" {
		return HungarianMatch(uavs, targets)
	}
	return SimplifiedMatch(uavs, targets)
}

// SimplifiedMatch is a greedy nearest-neighbor algorithm.
func SimplifiedMatch(uavs map[int]domain.Location, targets map[int]domain.Location) map[int]domain.Location {
	assignment := make(map[int]domain.Location)
	usedTargets := make(map[int]bool)

	// Sort UAV IDs for deterministic behavior
	uavIds := make([]int, 0, len(uavs))
	for id := range uavs {
		uavIds = append(uavIds, id)
	}

	for _, uavId := range uavIds {
		uavLoc := uavs[uavId]
		minDist := math.MaxFloat64
		bestTargetIdx := -1

		for targetIdx, targetLoc := range targets {
			if usedTargets[targetIdx] {
				continue
			}
			dist := calculateDist(uavLoc, targetLoc)
			if dist < minDist {
				minDist = dist
				bestTargetIdx = targetIdx
			}
		}

		if bestTargetIdx != -1 {
			assignment[uavId] = targets[bestTargetIdx]
			usedTargets[bestTargetIdx] = true
		}
	}

	return assignment
}

// HungarianMatch uses the Hungarian algorithm for optimal assignment.
func HungarianMatch(uavs map[int]domain.Location, targets map[int]domain.Location) map[int]domain.Location {
	n := len(uavs)
	if n == 0 || len(targets) != n {
		return SimplifiedMatch(uavs, targets)
	}

	uavIds := make([]int, 0, n)
	for id := range uavs {
		uavIds = append(uavIds, id)
	}
	targetIds := make([]int, 0, n)
	for id := range targets {
		targetIds = append(targetIds, id)
	}

	costs := make([][]float64, n)
	for i := 0; i < n; i++ {
		costs[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			costs[i][j] = calculateDist(uavs[uavIds[i]], targets[targetIds[j]])
		}
	}

	h := NewHungarianAlgorithm(costs)
	matches := h.Solve()

	assignment := make(map[int]domain.Location)
	for i, j := range matches {
		assignment[uavIds[i]] = targets[targetIds[j]]
	}
	return assignment
}

func calculateDist(l1, l2 domain.Location) float64 {
	dx := l1.Lon - l2.Lon
	dy := l1.Lat - l2.Lat
	return math.Sqrt(dx*dx + dy*dy)
}

// HungarianAlgorithm implementation ported from ArduSim
type HungarianAlgorithm struct {
	n      int
	costs  [][]float64
	mask   [][]int // 0: none, 1: starred, 2: primed
	rowCov []bool
	colCov []bool
}

func NewHungarianAlgorithm(costs [][]float64) *HungarianAlgorithm {
	n := len(costs)
	h := &HungarianAlgorithm{
		n:      n,
		costs:  make([][]float64, n),
		mask:   make([][]int, n),
		rowCov: make([]bool, n),
		colCov: make([]bool, n),
	}
	for i := range costs {
		h.costs[i] = make([]float64, n)
		copy(h.costs[i], costs[i])
		h.mask[i] = make([]int, n)
	}
	return h
}

func (h *HungarianAlgorithm) Solve() []int {
	// Step 1: Row reduction
	for i := 0; i < h.n; i++ {
		min := h.costs[i][0]
		for j := 1; j < h.n; j++ {
			if h.costs[i][j] < min {
				min = h.costs[i][j]
			}
		}
		for j := 0; j < h.n; j++ {
			h.costs[i][j] -= min
		}
	}

	// Step 2: Star zeroes
	for i := 0; i < h.n; i++ {
		for j := 0; j < h.n; j++ {
			if h.costs[i][j] == 0 && !h.rowCov[i] && !h.colCov[j] {
				h.mask[i][j] = 1
				h.rowCov[i] = true
				h.colCov[j] = true
			}
		}
	}
	h.resetCover()

	// Step 3: Cover columns
	for i := 0; i < h.n; i++ {
		for j := 0; j < h.n; j++ {
			if h.mask[i][j] == 1 {
				h.colCov[j] = true
			}
		}
	}

	for {
		count := 0
		for j := 0; j < h.n; j++ {
			if h.colCov[j] {
				count++
			}
		}
		if count >= h.n {
			break
		}

		// Step 4: Prime zeroes
		row, col := h.findZero()
		if row == -1 {
			// Step 6: Add/subtract min
			h.adjustMatrix()
			continue
		}

		h.mask[row][col] = 2
		starCol := h.findStarInRow(row)
		if starCol != -1 {
			h.rowCov[row] = true
			h.colCov[starCol] = false
		} else {
			// Step 5: Augment path
			h.augmentPath(row, col)
			h.resetCover()
			h.resetPrimes()
			// Re-cover starred columns
			for i := 0; i < h.n; i++ {
				for j := 0; j < h.n; j++ {
					if h.mask[i][j] == 1 {
						h.colCov[j] = true
					}
				}
			}
		}
	}

	res := make([]int, h.n)
	for i := 0; i < h.n; i++ {
		for j := 0; j < h.n; j++ {
			if h.mask[i][j] == 1 {
				res[i] = j
			}
		}
	}
	return res
}

func (h *HungarianAlgorithm) findZero() (int, int) {
	for i := 0; i < h.n; i++ {
		if !h.rowCov[i] {
			for j := 0; j < h.n; j++ {
				if !h.colCov[j] && h.costs[i][j] == 0 {
					return i, j
				}
			}
		}
	}
	return -1, -1
}

func (h *HungarianAlgorithm) findStarInRow(row int) int {
	for j := 0; j < h.n; j++ {
		if h.mask[row][j] == 1 {
			return j
		}
	}
	return -1
}

func (h *HungarianAlgorithm) findStarInCol(col int) int {
	for i := 0; i < h.n; i++ {
		if h.mask[i][col] == 1 {
			return i
		}
	}
	return -1
}

func (h *HungarianAlgorithm) findPrimeInRow(row int) int {
	for j := 0; j < h.n; j++ {
		if h.mask[row][j] == 2 {
			return j
		}
	}
	return -1
}

func (h *HungarianAlgorithm) augmentPath(row, col int) {
	path := [][]int{{row, col}}
	for {
		r := h.findStarInCol(path[len(path)-1][1])
		if r == -1 {
			break
		}
		path = append(path, []int{r, path[len(path)-1][1]})
		c := h.findPrimeInRow(path[len(path)-1][0])
		path = append(path, []int{path[len(path)-1][0], c})
	}

	for _, p := range path {
		if h.mask[p[0]][p[1]] == 1 {
			h.mask[p[0]][p[1]] = 0
		} else {
			h.mask[p[0]][p[1]] = 1
		}
	}
}

func (h *HungarianAlgorithm) adjustMatrix() {
	min := math.MaxFloat64
	for i := 0; i < h.n; i++ {
		if !h.rowCov[i] {
			for j := 0; j < h.n; j++ {
				if !h.colCov[j] && h.costs[i][j] < min {
					min = h.costs[i][j]
				}
			}
		}
	}
	for i := 0; i < h.n; i++ {
		for j := 0; j < h.n; j++ {
			if h.rowCov[i] {
				h.costs[i][j] += min
			}
			if !h.colCov[j] {
				h.costs[i][j] -= min
			}
		}
	}
}

func (h *HungarianAlgorithm) resetCover() {
	for i := range h.rowCov {
		h.rowCov[i] = false
		h.colCov[i] = false
	}
}

func (h *HungarianAlgorithm) resetPrimes() {
	for i := 0; i < h.n; i++ {
		for j := 0; j < h.n; j++ {
			if h.mask[i][j] == 2 {
				h.mask[i][j] = 0
			}
		}
	}
}
