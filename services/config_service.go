package services

import (
	"encoding/json"
	"errors"

	"os"
	"path/filepath"
	"os/user"
)

type DotfileConfig struct {
	Path string `json:"path"`
}

type DotmanConfig struct {
	Dotfile DotfileConfig `json:"dotfile"`
}

type ConfigService struct {
	config DotmanConfig
	path   string
}

func getDefaultConfigPath() string {
	usr, err := user.Current()
	if err != nil {
		return ".dotman.json"
	}
	return filepath.Join(usr.HomeDir, ".dotman.json")
}

func NewConfigService() *ConfigService {
	return &ConfigService{
		config: DotmanConfig{},
		path:   getDefaultConfigPath(),
	}
}

func (c *ConfigService) Load() error {
	bytes, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			c.config = DotmanConfig{}
			return nil
		}
		return err
	}
	return json.Unmarshal(bytes, &c.config)
}

func (c *ConfigService) Save() error {
	bytes, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, bytes, 0644)
}

// Get returns a value using dot notation (supports nested fields)
func (c *ConfigService) Get(key string) (interface{}, error) {
	switch key {
	case "dotfile.path":
		return c.config.Dotfile.Path, nil
	case "dotfile":
		return c.config.Dotfile, nil
	default:
		return nil, errors.New("unsupported key")
	}
}

// Set sets a value using dot notation (supports nested fields)
func (c *ConfigService) Set(key string, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return errors.New("value must be a string")
	}
	switch key {
	case "dotfile.path":
		c.config.Dotfile.Path = strVal
		return nil
	default:
		return errors.New("unsupported key")
	}
}
