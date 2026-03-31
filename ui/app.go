package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

type ServiceType struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SchemaRaw string `json:"schemaRaw"`
}

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAvailableServices() []ServiceType {
	baseDir := "/home/isma/dev/ArduSim2/algorithms"
	var services []ServiceType

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return services
	}

	for _, entry := range entries {
		if entry.IsDir() {
			schemaPath := filepath.Join(baseDir, entry.Name(), "schema.json")
			data, err := os.ReadFile(schemaPath)
			if err == nil {
				var schema struct {
					Title string `json:"title"`
				}
				if err := json.Unmarshal(data, &schema); err == nil {
					services = append(services, ServiceType{
						ID:        entry.Name(),
						Title:     schema.Title,
						SchemaRaw: string(data),
					})
				}
			}
		}
	}
	return services
}
