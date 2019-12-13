package client

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string]Auth `json:"auth"`

	// a <source_repo>:<dest_repo> map
	ImageList map[string]string `json:"images"`

	// If the destinate registry and namespace is not provided,
	// the source image will be synchronized to defaultDestRegistry
	// and defaultDestNamespace with origin repo name and tag.
	defaultDestRegistry  string
	defaultDestNamespace string
}

// Auth describes the authentication information of a registry
type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// NewSyncConfig creates a Config struct
func NewSyncConfig(configFilePath, defaultDestRegistry, defaultDestNamespace string) (*Config, error) {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %v not exist: %v", configFilePath, err)
	}

	file, err := os.OpenFile(configFilePath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("open config file %v error: %v", configFilePath, err)
	}

	decoder := json.NewDecoder(file)
	config := Config{
		defaultDestNamespace: defaultDestNamespace,
		defaultDestRegistry:  defaultDestRegistry,
	}

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config error: %v", err)
	}
	return &config, nil
}

// GetAuth gets the authentication information in Config
func (c *Config) GetAuth(registry string, namespace string) (Auth, bool) {
	// key of each AuthList item can be "registry/namespace" or "registry" only
	registryAndNamespace := registry + "/" + namespace

	if moreSpecificAuth, exist := c.AuthList[registryAndNamespace]; exist {
		return moreSpecificAuth, exist
	}

	auth, exist := c.AuthList[registry]
	return auth, exist
}

// GetImageList gets the ImageList map in Config
func (c *Config) GetImageList() map[string]string {
	return c.ImageList
}
