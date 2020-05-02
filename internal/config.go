package internal

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config struct
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Port     int    `yaml:"port"`
	} `yaml:"database"`
	Email struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Hostname string `yaml:"hostname"`
		Port     int    `yaml:"port"`
	} `yaml:"email"`
}

// NewConfig - initialize config structure
func NewConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)

	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
