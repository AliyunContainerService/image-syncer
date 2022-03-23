package sync

import "strings"

type ctxKey struct {
	string
}

const Oauth2User = "_oauth2_"

// isPermanentServiceAccountToken returns true if user is a Google permanent service account token
func isPermanentServiceAccountToken(registry string, username string) bool {
	return strings.Contains(registry, ".gcr.io") && strings.Compare(username, Oauth2User) == 0
}
