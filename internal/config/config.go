package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type PathConfig struct {
	Path        string        `yaml:"path"`
	Method      string        `yaml:"method"`
	ForwardPort int           `yaml:"forwardPort"`
	MaxRetry    int           `yaml:"maxRetry"`
	Timeout     time.Duration `yaml:"timeout"`
}

type ProxyConfig struct {
	Port    int `yaml:"port"`
	Metrics struct {
		Path string `yaml:"path"`
		Port int    `yaml:"port"`
	} `yaml:"metrics"`
	Paths []PathConfig `yaml:"paths"`
}

func LoadConfig(path string) (ProxyConfig, error) {
	var cfg ProxyConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}
