package client

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/utils/types"

	"github.com/sirupsen/logrus"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"gopkg.in/yaml.v2"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string]types.Auth `json:"auth" yaml:"auth"`

	// a <source_repo>:<dest_repo> map
	ImageList map[string]interface{} `json:"images" yaml:"images"`

	// only images with selected os can be sync
	osFilterList []string
	// only images with selected architecture can be sync
	archFilterList []string
}

// NewSyncConfig creates a Config struct
func NewSyncConfig(configFile, authFilePath, imageFilePath string,
	osFilterList, archFilterList []string, logger *logrus.Logger) (*Config, error) {
	if len(configFile) == 0 && len(imageFilePath) == 0 {
		return nil, fmt.Errorf("neither config.json nor images.json is provided")
	}

	if len(configFile) == 0 && len(authFilePath) == 0 {
		logger.Warnf("[Warning] No authentication information found because neither " +
			"config.json nor auth.json provided, image-syncer may not work fine.")
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
		if err := openAndDecode(imageFilePath, &config.ImageList); err != nil {
			return nil, fmt.Errorf("decode image file %v error: %v", imageFilePath, err)
		}
	}
	config.AuthList = expandEnv(config.AuthList)
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
func (c *Config) GetAuth(repository string) (types.Auth, bool) {
	auth := types.Auth{}
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

func expandEnv(authMap map[string]types.Auth) map[string]types.Auth {
	result := make(map[string]types.Auth)

	for registry, auth := range authMap {
		newAuth := auth
		if !auth.DisableExpandEnv {
			pwd := os.ExpandEnv(auth.Password)
			name := os.ExpandEnv(auth.Username)
			newAuth = types.Auth{
				Username:         name,
				Password:         pwd,
				Insecure:         auth.Insecure,
				DisableExpandEnv: auth.DisableExpandEnv,
			}
		}
		result[registry] = newAuth
	}

	return result
}
