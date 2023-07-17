package client

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"gopkg.in/yaml.v2"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string]utils.Auth `json:"auth" yaml:"auth"`

	// a <source_repo>:<dest_repo> map
	ImageList map[string]string `json:"images" yaml:"images"`

	// only images with selected os can be sync
	osFilterList []string
	// only images with selected architecture can be sync
	archFilterList []string

	// If the destination registry and namespace is not provided,
	// the source image will be synchronized to defaultDestRegistry with origin repo name and tag.
	defaultDestRegistry string
}

// NewSyncConfig creates a Config struct
func NewSyncConfig(configFile, authFilePath, imageFilePath, defaultDestRegistry string,
	osFilterList, archFilterList []string) (*Config, error) {
	if len(configFile) == 0 && len(imageFilePath) == 0 {
		return nil, fmt.Errorf("neither config.json nor images.json is provided")
	}

	if len(configFile) == 0 && len(authFilePath) == 0 {
		log.Println("[Warning] No authentication information found because neither config.json nor auth.json provided, this may not work.")
	}

	var config Config

	if len(configFile) != 0 {
		if err := openAndDecode(configFile, &config); err != nil {
			return nil, fmt.Errorf("decode config file %v failed, error %v", configFile, err)
		}
	} else {
		if len(authFilePath) != 0 {
			if err := openAndDecode(authFilePath, &config.AuthList); err != nil {
				return nil, fmt.Errorf("decode auth file %v error: %v", authFilePath, err)
			}
		}
		config.AuthList = expandEnv(config.AuthList)

		if err := openAndDecode(imageFilePath, &config.ImageList); err != nil {
			return nil, fmt.Errorf("decode image file %v error: %v", imageFilePath, err)
		}
	}

	config.defaultDestRegistry = defaultDestRegistry
	config.osFilterList = osFilterList
	config.archFilterList = archFilterList

	return &config, nil
}

// Open json file and decode into target interface
func openAndDecode(filePath string, target interface{}) error {
	if !strings.HasSuffix(filePath, ".yaml") &&
		!strings.HasSuffix(filePath, ".yml") &&
		!strings.HasSuffix(filePath, ".json") {
		return fmt.Errorf("only one of yaml/yml/json format is supported")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %v not exist: %v", filePath, err)
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("open file %v error: %v", filePath, err)
	}

	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(target); err != nil {
			return fmt.Errorf("unmarshal config error: %v", err)
		}
	} else {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(target); err != nil {
			return fmt.Errorf("unmarshal config error: %v", err)
		}
	}

	return nil
}

// GetAuth gets the authentication information in Config
func (c *Config) GetAuth(repository string) (utils.Auth, bool) {
	auth := utils.Auth{}
	prefixLen := 0
	exist := false

	for key, value := range c.AuthList {
		if matched := utils.RepoMathPrefix(repository, key); matched {
			if len(key) > prefixLen {
				auth = value
				exist = true
			}
		}
	}

	return auth, exist
}

// GetImageList gets the ImageList map in Config
func (c *Config) GetImageList() map[string]string {
	return c.ImageList
}

func expandEnv(authMap map[string]utils.Auth) map[string]utils.Auth {
	result := make(map[string]utils.Auth)

	for registry, auth := range authMap {
		pwd := os.ExpandEnv(auth.Password)
		name := os.ExpandEnv(auth.Username)
		newAuth := utils.Auth{
			Username: name,
			Password: pwd,
			Insecure: auth.Insecure,
		}
		result[registry] = newAuth
	}

	return result
}
