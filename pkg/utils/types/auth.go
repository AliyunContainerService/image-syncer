package types

// Auth describes the authentication information of a registry or a repository
type Auth struct {
	Username         string `json:"username" yaml:"username"`
	Password         string `json:"password" yaml:"password"`
	Insecure         bool   `json:"insecure" yaml:"insecure"`
	DisableExpandEnv bool   `json:"disableexpandenv" yaml:"disableexpandenv"`
}
