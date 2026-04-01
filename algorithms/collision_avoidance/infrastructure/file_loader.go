package infrastructure

import (
	"encoding/json"
	"os"

	"collision_avoidance/domain"
)

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (f *FileLoader) LoadAppConfig(filePath string) (*domain.AppConfig, *domain.MBCAPParam, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &domain.AppConfig{}
	
	// Default hardcoded values for algorithmic parameters if they are missing
	config.CollisionWarningDistance = 25.0
	config.CollisionWarningAltitudeOffset = 60.0
	config.CollisionWarningTimeOffset = 1.5
	config.RiskCheckPeriod = 1.0
	config.BeaconExpirationTime = 10.0
	config.HoveringTimeout = 6.0
	config.DefaultFlightModeResumeDelay = 4.0
	config.CheckRiskSameUAVDelay = 5.0
	config.OvertakeDelayTimeout = 5.5
	config.DeadlockBaseTimeout = 120
	// Hardcoded ArduSim geometry values not usually parsed from schema
	config.SafePlaceDistance = 10.0
	config.SafetyDistanceRange = 2.0
	config.HopTimeNS = 1000000000
	config.MinAdvertismentSpeed = 1.0

	// Overwrite with actual JSON fields
	if err := decoder.Decode(config); err != nil {
		return nil, nil, err
	}

	// Prepare algorithm params struct properly converted to correct nanoseconds / scales
	params := &domain.MBCAPParam{
		CollisionWarningDistance:       config.CollisionWarningDistance,
		CollisionWarningAltitudeOffset: config.CollisionWarningAltitudeOffset,
		CollisionWarningTimeOffset:     int64(config.CollisionWarningTimeOffset * 1e9),
		RiskCheckPeriod:                int64(config.RiskCheckPeriod * 1e9),
		BeaconExpirationTime:           int64(config.BeaconExpirationTime * 1e9),
		HoveringTimeout:                int64(config.HoveringTimeout * 1e9),
		DefaultFlightModeResumeDelay:   int64(config.DefaultFlightModeResumeDelay * 1e9),
		CheckRiskSameUAVDelay:          int64(config.CheckRiskSameUAVDelay * 1e3), // ms
		OvertakeDelayTimeout:           int64(config.OvertakeDelayTimeout * 1e9),
		DeadlockBaseTimeout:            int64(config.DeadlockBaseTimeout),
		SafePlaceDistance:              config.SafePlaceDistance,
		SafetyDistanceRange:            config.SafetyDistanceRange,
		HopTimeNS:                      config.HopTimeNS,
		MinAdvertismentSpeed:           config.MinAdvertismentSpeed,
	}

	return config, params, nil
}
