package config

import (
	"encoding/json"
	"io/ioutil"
	"safe_takeoff/domain"
)

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (fl *FileLoader) LoadAppConfig(path string) (*domain.AppConfig, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config domain.AppConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
