package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	urls := []string{
		"gcr.io/knative-releases/github.com/knative/build/cmd/creds-init:v1",
		"registry.hub.docker.io/library/nginx",
		"nginx:v1",
		"127.0.0.1:300/library/nginx:v1",
		"127.0.0.1:300/library/nginx",
	}

	var repoURLs []*RepoURL
	for _, url := range urls {
		repoURL, _ := NewRepoURL(url)
		repoURLs = append(repoURLs, repoURL)
	}

	assert.Equal(t, "gcr.io", repoURLs[0].GetRegistry())
	assert.Equal(t, "knative-releases", repoURLs[0].GetNamespace())
	assert.Equal(t, "github.com/knative/build/cmd/creds-init", repoURLs[0].GetRepo())
	assert.Equal(t, "knative-releases/github.com/knative/build/cmd/creds-init", repoURLs[0].GetRepoWithNamespace())
	assert.Equal(t, "v1", repoURLs[0].GetTag())
	assert.Equal(t, "", repoURLs[1].GetTag())
	assert.Equal(t, "registry.hub.docker.io", repoURLs[1].GetRegistry())
	assert.Equal(t, "library", repoURLs[1].GetNamespace())
	assert.Equal(t, "nginx", repoURLs[1].GetRepo())
	assert.Equal(t, "library/nginx", repoURLs[1].GetRepoWithNamespace())
	assert.Equal(t, "library", repoURLs[2].GetNamespace())
	assert.Equal(t, "registry.hub.docker.com", repoURLs[2].GetRegistry())
	assert.Equal(t, "v1", repoURLs[2].GetTag())
	assert.Equal(t, "127.0.0.1:300", repoURLs[3].GetRegistry())
	assert.Equal(t, "127.0.0.1:300/library/nginx", repoURLs[3].GetURLWithoutTag())
	assert.Equal(t, "v1", repoURLs[3].GetTag())
	assert.Equal(t, "nginx:v1", repoURLs[3].GetRepoWithTag())
	assert.Equal(t, "127.0.0.1:300/library/nginx", repoURLs[4].GetURLWithoutTag())
	assert.Equal(t, "127.0.0.1:300", repoURLs[4].GetRegistry())
	assert.Equal(t, "nginx", repoURLs[4].GetRepoWithTag())
}
