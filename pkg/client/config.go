package client

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/containers/image/v5/manifest"
	"gopkg.in/yaml.v2"
)

// Config information of sync client
type Config struct {
	// the authentication information of each registry
	AuthList map[string]Auth `json:"auth" yaml:"auth"`

	// a <source_repo>:<dest_repo> map
	ImageList map[string]string `json:"images" yaml:"images"`

	// global platform selector
	Platform Platform `json:"platform" yaml:"platform"`

	// If the destinate registry and namespace is not provided,
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

// Platform selector of sync client
type Platform struct {
	// default os
	Os []string `json:"os" yaml:"os"`
	// default arch
	Arch []string `json:"arch" yaml:"arch"`

	// set include or exclude filters for source image, when both are present, exclude filters take precedence
	// filter string use unix glob syntax
	SourceFilter struct {
		// include filters
		Include []string `json:"include" yaml:"include"`

		// exclude filters
		Exclude []string `json:"exclude" yaml:"exclude"`
	} `json:"source" yaml:"source"`
}

//  NewSyncConfig creates a Config struct
// configFile
func NewSyncConfig(configFile, authFilePath, imageFilePath, platformFilePath, defaultDestRegistry, defaultDestNamespace string) (*Config, error) {
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

		if err := openAndDecode(imageFilePath, &config.ImageList); err != nil {
			return nil, fmt.Errorf("decode image file %v error: %v", imageFilePath, err)
		}

		if len(platformFilePath) != 0 {
			if err := openAndDecode(platformFilePath, &config.Platform); err != nil {
				return nil, fmt.Errorf("decode platform file %v error: %v", platformFilePath, err)
			}
		}
	}

	config.defaultDestNamespace = defaultDestNamespace
	config.defaultDestRegistry = defaultDestRegistry

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

// Match platform selector according to the source image and  its os and arch
func (c *Config) MatchPlatform(source string, platform manifest.Schema2PlatformSpec) (bool, error) {
	doSelect := true
	if len(c.Platform.SourceFilter.Exclude) != 0 {
		for _, p := range c.Platform.SourceFilter.Exclude {
			if matched, err := path.Match(p, source); err != nil {
				return false, err
			} else if matched {
				doSelect = false
				break
			}
		}

	} else if len(c.Platform.SourceFilter.Include) != 0 {
		doSelect = false
		for _, p := range c.Platform.SourceFilter.Include {
			if matched, err := path.Match(p, source); err != nil {
				return false, err
			} else if matched {
				doSelect = true
				break
			}
		}
	}

	if doSelect {
		osMatched := true
		archMatched := true
		if len(c.Platform.Os) != 0 {
			osMatched = false
			for _, o := range c.Platform.Os {
				// match os:osversion
				if o == os {
					osMatched = true
				}
			}
		}

		if len(c.Platform.Arch) != 0 {
			archMatched = false
			for _, a := range c.Platform.Arch {
				// match arch:variant
				if a == arch {
					archMatched = true
				}
			}
		}

		return osMatched && archMatched, nil
	}

	return true, nil
}
