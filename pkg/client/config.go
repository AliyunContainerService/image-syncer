package client

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"gopkg.in/yaml.v2"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string]utils.Auth `json:"auth" yaml:"auth"`

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
		config.AuthList = expandEnv(config.AuthList)

		if err := openAndDecode(imageFilePath, &config.ImageList); err != nil {
			return nil, fmt.Errorf("decode image file %v error: %v", imageFilePath, err)
		}
	}

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

// GetImageList gets the ImageList map in Config, and will transform ImageList to map[string]string
func (c *Config) GetImageList() (map[string][]string, error) {
	result := map[string][]string{}

	for source, dest := range c.ImageList {
		convertErr := fmt.Errorf("invalid destination %v for source \"%v\", "+
			"destination should only be string or []string", dest, source)

		emptyDestErr := fmt.Errorf("empty destination is not supported for source: %v", source)

		if destList, ok := dest.([]interface{}); ok {
			// check if is destination is a []string
			for _, d := range destList {
				destStr, ok := d.(string)
				if !ok {
					return nil, convertErr
				}

				if len(destStr) == 0 {
					return nil, emptyDestErr
				}
				result[source] = append(result[source], os.ExpandEnv(destStr))
			}

			// empty slice is the same with an empty string
			if len(destList) == 0 {
				return nil, emptyDestErr
			}
		} else if destStr, ok := dest.(string); ok {
			// check if is destination is a string
			if len(destStr) == 0 {
				return nil, emptyDestErr
			}
			result[source] = append(result[source], os.ExpandEnv(destStr))
		} else {
			return nil, convertErr
		}
	}

	return result, nil
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
