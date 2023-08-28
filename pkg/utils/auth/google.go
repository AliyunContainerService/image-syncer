package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

const Oauth2User = "_oauth2_"

// IsGCRPermanentServiceAccountToken returns true if user is a Google permanent service account token
func IsGCRPermanentServiceAccountToken(registry string, username string) bool {
	return strings.Contains(registry, ".gcr.io") && strings.Compare(username, Oauth2User) == 0
}

// GCPTokenFromCreds creates oauth2 token from permanent service account token
func GCPTokenFromCreds(creds string) (string, time.Time, error) {
	b, err := base64.StdEncoding.DecodeString(creds)
	if err != nil {
		return "", time.Time{}, err
	}
	conf, err := google.JWTConfigFromJSON(
		b, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return "", time.Time{}, err
	}

	token, err := conf.TokenSource(context.Background()).Token()
	if err != nil {
		return "", time.Time{}, err
	}

	return token.AccessToken, token.Expiry, nil
}
