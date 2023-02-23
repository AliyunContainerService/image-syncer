package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string][]Auth `json:"auth" yaml:"auth"`

	// a <source_repo>:<dest_repo> map
	ImageList map[string]string `json:"images" yaml:"images"`

	// only images with selected os can be sync
	osFilterList []string
	// only images with selected architecture can be sync
	archFilterList []string

	// If the destination registry and namespace is not provided,
	// the source image will be synchronized to defaultDestRegistry
	// and defaultDestNamespace with origin repo name and tag.
	defaultDestRegistry  string
	defaultDestNamespace string
}

// Auth describes the authentication information of a registry
type Auth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Insecure bool   `json:"insecure" yaml:"insecure"`
}

// NewSyncConfig creates a Config struct
func NewSyncConfig(configFile, authFilePath, imageFilePath, defaultDestRegistry, defaultDestNamespace string,
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

	config.defaultDestNamespace = defaultDestNamespace
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
func (c *Config) GetAuth(registry, namespace, repository string) (Auth, bool) {
	// key of each AuthList item can be "registry/namespace" or "registry" only
	registryAndNamespace := registry + "/" + namespace

	_, ok1 := c.AuthList[registryAndNamespace]
	_, ok2 := c.AuthList[registry]
	if !ok1 && !ok2 {
		return Auth{}, false
	}

	auth, ok := c.selectAuth(registry, c.AuthList[registryAndNamespace])
	if ok {
		return auth, ok
	}

	return c.selectAuth(registry, c.AuthList[registry])
}

func (c *Config) selectAuth(registry string, auths []Auth) (Auth, bool) {
	if len(auths) == 0 {
		return Auth{}, false
	}

	if len(auths) == 1 {
		return auths[0], true
	}

	if registry != "registry.hub.docker.com" {
		index := rand.Intn(len(auths))
		return auths[index], true
	}

	// unlimited images
	rate := checkDockerPullRateLimits(registry, "", "")
	if rate.limit == 0 {
		index := rand.Intn(len(auths))
		return auths[index], true
	}

	// when remain gt 0, random select auth
	remains := make([]Auth, 0, len(auths))
	for _, auth := range auths {
		rate := checkDockerPullRateLimits(registry, auth.Username, auth.Password)
		if rate.remain > 0 {
			remains = append(remains, auth)
		}
	}
	index := rand.Intn(len(auths))
	return auths[index], true
}

// GetImageList gets the ImageList map in Config
func (c *Config) GetImageList() map[string]string {
	return c.ImageList
}

func expandEnv(authMap map[string][]Auth) map[string][]Auth {

	result := make(map[string][]Auth, len(authMap))

	for registry, auths := range authMap {
		result[registry] = make([]Auth, 0, len(auths))
		for _, auth := range auths {
			pwd := os.ExpandEnv(auth.Password)
			name := os.ExpandEnv(auth.Username)
			newAuth := Auth{
				Username: name,
				Password: pwd,
				Insecure: auth.Insecure,
			}
			result[registry] = append(result[registry], newAuth)
		}
	}

	return result
}
