package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFilename string = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func (c *Config) write() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Unable to access config file for write: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("Unable to serialize config: %w", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("Unable to write config to file: %w", err)
	}

	file.Sync()

	return nil
}

func (c *Config) SetUser(name string) error {
	c.CurrentUserName = name
	err := c.write()
	if err != nil {
		return err
	}
	return nil
}

func Read() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return &Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{}, fmt.Errorf("Unable to read file at %s", path)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return &config, fmt.Errorf("Unable to parse config file: %w", err)
	}

	return &config, nil
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Error getting user home dir: %w", err)
	}
	path := filepath.Join(homeDir, configFilename)
	return path, nil
}
