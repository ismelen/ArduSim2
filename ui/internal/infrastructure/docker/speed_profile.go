package docker

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// ParseSpeedProfile reads a .csv file containing one speed value (m/s) per line.
// It ignores empty lines and comments starting with '#'.
func ParseSpeedProfile(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var speeds []float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		speed, err := strconv.ParseFloat(line, 64)
		if err != nil {
			// If a line is malformed, we can either error out entirely or skip it.
			// Given it's a configuration file, erroring out entirely is safer to alert the user.
			return nil, err
		}
		
		speeds = append(speeds, speed)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return speeds, nil
}
