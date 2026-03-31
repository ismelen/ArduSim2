package infrastructure

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"collision_avoidance/domain"
)

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (f *FileLoader) LoadAppConfig(filePath string) (*domain.AppConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &domain.AppConfig{}
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (f *FileLoader) LoadMBCAPParams(filePath string) (*domain.MBCAPParam, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Defaults
	params := &domain.MBCAPParam{
		CollisionWarningDistance:       20.0,
		CollisionWarningAltitudeOffset: 50.0,
		CollisionWarningTimeOffset:     1000000000,
		RiskCheckPeriod:                1000000000,
		BeaconExpirationTime:           3000000000,
		HoveringTimeout:                5000000000,
		DefaultFlightModeResumeDelay:   4000000000,
		CheckRiskSameUAVDelay:          5000,
		OvertakeDelayTimeout:           10000000000,
		DeadlockBaseTimeout:            120, // seconds
		SafePlaceDistance:              10.0,
		SafetyDistanceRange:            2.0,
		HopTimeNS:                      1000000000,
		MinAdvertismentSpeed:           0.5,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Some properties files have trailing text with spaces (e.g. `120 mcab.properties`)
			value = strings.Split(value, " ")[0]

			switch key {
			case "collisionWarningDistance":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.CollisionWarningDistance = val
				}
			case "collisionWarningAltitudeOffset":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.CollisionWarningAltitudeOffset = val
				}
			case "collisionWarningTimeOffset":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.CollisionWarningTimeOffset = int64(val * 1e9)
				}
			case "riskCheckPeriod":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.RiskCheckPeriod = int64(val * 1e9)
				}
			case "beaconExpirationTime":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.BeaconExpirationTime = int64(val * 1e9)
				}
			case "hoveringTimeout":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.HoveringTimeout = int64(val * 1e9)
				}
			case "defaultFlightModeResumeDelay":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.DefaultFlightModeResumeDelay = int64(val * 1e9)
				}
			case "checkRiskSameUAVDelay":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.CheckRiskSameUAVDelay = int64(val * 1e3) // ms
				}
			case "overtakeDelayTimeout":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.OvertakeDelayTimeout = int64(val * 1e9)
				}
			case "deadlockBaseTimeout":
				if val, err := strconv.ParseInt(value, 10, 64); err == nil {
					params.DeadlockBaseTimeout = val
				}
			case "minAdvertismentSpeed":
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					params.MinAdvertismentSpeed = val
				}
			}
		}
	}
	return params, scanner.Err()
}
