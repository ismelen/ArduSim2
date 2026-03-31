package tests

import (
	"safe_takeoff/domain"
	"safe_takeoff/usecase"
	"testing"
)

func TestGenerateGridFormation(t *testing.T) {
	center := domain.Location{Lat: 40.0, Lon: -3.0, Alt: 0}
	
	targets := usecase.GenerateGridFormation(center, 4, 10.0, 15.0)
	
	if len(targets) != 4 {
		t.Errorf("Expected 4 targets, got %d", len(targets))
	}
	for _, tgt := range targets {
		if tgt.Alt != 15.0 {
			t.Errorf("Expected altitude 15.0, got %f", tgt.Alt)
		}
	}
}

func TestMatchUAVsToTargets(t *testing.T) {
	ground := map[int]domain.Location{
		1: {Lat: 40.0, Lon: -3.0},
		2: {Lat: 40.001, Lon: -3.001},
	}
	
	targets := []domain.Location{
		{Lat: 40.0, Lon: -3.0},
		{Lat: 40.001, Lon: -3.001},
	}
	
	matches := usecase.MatchUAVsToTargets(ground, targets)
	
	if len(matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(matches))
	}
	
	// Because of our distanceSq calculation, identical locations should have exactly 0 error.
	for _, m := range matches {
		if m.DistanceError > 0.001 {
			t.Errorf("Expected near-zero distance for exact targets, got %f", m.DistanceError)
		}
	}
}
