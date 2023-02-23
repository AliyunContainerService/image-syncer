package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type rateLimit struct {
	limit  int
	remain int
}

type authenticationToken struct {
	Token string `json:"token"`
}

type AuthClaims struct {
	Access []AccessItem `json:"access"`
	jwt.RegisteredClaims
}

type AccessItem struct {
	Type       string    `json:"type,omitempty"`
	Name       string    `json:"name,omitempty"`
	Actions    []string  `json:"actions,omitempty"`
	Parameters Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	PullLimit         string `json:"pull_limit,omitempty"`
	PullLimitInterval string `json:"pull_limit_interval,omitempty"`
}

func checkDockerPullRateLimits(repository, username, password string) (rate rateLimit) {
	token, err := generateDockerHubAuthToken(repository, username, password)
	if err != nil || token == "" {
		// err? unlimited?
		return
	}

	url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/latest", repository)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", "Bearer "+token)
	cli := http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return
	}

	limit := resp.Header.Get("Ratelimit-Limit")
	remain := resp.Header.Get("Ratelimit-Remaining")
	rate.limit, _ = strconv.Atoi(strings.Split(limit, ";")[0])
	rate.remain, _ = strconv.Atoi(strings.Split(remain, ";")[0])
	return rate
}

func generateDockerHubAuthToken(repository, username, password string) (string, error) {
	url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repository)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	cli := http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var token authenticationToken
	json.Unmarshal(respBody, &token)
	if err != nil {
		return "", err
	}

	if !checkRepositoryRateLimit(token.Token) {
		return "", nil
	}

	return token.Token, nil
}

// checkRepositoryRateLimit return true if repository is limitied
func checkRepositoryRateLimit(tokenString string) bool {
	var claim AuthClaims
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, &claim)
	return err == nil && len(claim.Access) > 0
}
