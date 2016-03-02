package config

import (
	"encoding/json"
	"github.com/bingosummer/azure-meta-service-broker/utils"
)

type Config struct {
	Port string `json:"port"`
}

var (
	currentConfiguration Config
)

func LoadConfig(path string) (*Config, error) {
	bytes, err := utils.ReadFile(path)
	if err != nil {
		return &currentConfiguration, err
	}

	err = json.Unmarshal(bytes, &currentConfiguration)
	if err != nil {
		return &currentConfiguration, err
	}
	return &currentConfiguration, nil
}

func GetConfig() *Config {
	return &currentConfiguration
}
