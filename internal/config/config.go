package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name           string `yaml:"name"`
		Env            string `yaml:"env"`
		DefaultRobotID string `yaml:"default_robot_id"`
	} `yaml:"app"`

	HTTP struct {
		Port int `yaml:"port"`
	} `yaml:"http"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`

	Rosbridge struct {
		URL    string   `yaml:"url"`
		Topics []string `yaml:"topics"`
	} `yaml:"rosbridge"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
