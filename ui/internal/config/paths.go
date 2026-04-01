// Package config resolves all project-level paths relative to the working
// directory so that no absolute paths are hardcoded anywhere in the backend.
package config

import (
	"path/filepath"
)

// Paths holds all project-level directory paths resolved at startup.
type Paths struct {
	// AlgorithmsDir is where algorithm sub-directories (each with a schema.json) live.
	AlgorithmsDir string
	// SimulationsDir is where per-run simulation directories are created.
	SimulationsDir string
	// ResourcesDir is where shared base config templates reside.
	ResourcesDir string
}

// NewPaths resolves all paths relative to projectRoot (the working directory).
func NewPaths(projectRoot string) Paths {
	base := filepath.Clean(projectRoot)
	return Paths{
		AlgorithmsDir:  filepath.Join(base, "..", "algorithms"),
		SimulationsDir: filepath.Join(base, "..", "simulations"),
		ResourcesDir:   filepath.Join(base, "..", "resources"),
	}
}
