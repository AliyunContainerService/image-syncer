package utils

// Auth describes the authentication information of a registry or a repository
type Auth struct {
	Username      string `json:"username" yaml:"username"`
	Password      string `json:"password" yaml:"password"`
	IdentityToken string `json:"identityToken" yaml:"identityToken"`
	Insecure      bool   `json:"insecure" yaml:"insecure"`
}
