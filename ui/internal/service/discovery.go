package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	"ui/internal/simulation"
)

// Discoverer scans an algorithms directory for services that expose a schema.json.
type Discoverer struct {
	algorithmsDir string
}

// NewDiscoverer creates a Discoverer rooted at the given algorithms directory.
func NewDiscoverer(algorithmsDir string) *Discoverer {
	return &Discoverer{algorithmsDir: algorithmsDir}
}

// GetAvailableServices returns one ServiceType per algorithm sub-directory
// that contains a valid schema.json with a "title" field.
// Directories without a readable schema are silently skipped.
func (d *Discoverer) GetAvailableServices() []simulation.ServiceType {
	entries, err := os.ReadDir(d.algorithmsDir)
	if err != nil {
		return nil
	}

	services := make([]simulation.ServiceType, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		svc, ok := d.loadServiceType(entry.Name())
		if ok {
			services = append(services, svc)
		}
	}
	return services
}

// loadServiceType reads and parses the schema.json for a single algorithm directory.
// Returns (ServiceType, true) on success, or (zero, false) if the schema is missing
// or malformed.
func (d *Discoverer) loadServiceType(dirName string) (simulation.ServiceType, bool) {
	schemaPath := filepath.Join(d.algorithmsDir, dirName, "schema.json")
	rawData, err := os.ReadFile(schemaPath)
	if err != nil {
		return simulation.ServiceType{}, false
	}

	var schema struct {
		ServiceID string `json:"service_id"`
		Title     string `json:"title"`
	}
	if err := json.Unmarshal(rawData, &schema); err != nil {
		return simulation.ServiceType{}, false
	}

	svcID := schema.ServiceID
	if svcID == "" {
		svcID = dirName
	}

	return simulation.ServiceType{
		ID:         svcID,
		FolderName: dirName,
		Title:      schema.Title,
		SchemaRaw:  string(rawData),
	}, true
}
