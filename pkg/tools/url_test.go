package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrl(t *testing.T) {
	urls := []string{
		"gcr.io/knative-releases/github.com/knative/build/cmd/creds-init:v1",
		"registry.hub.docker.io/library/nginx",
		"nginx:v1",
		"127.0.0.1:300/library/nginx:v1",
	}

	var repoUrls []*RepoUrl
	for _, url := range urls {
		repoUrl, _ := NewRepoUrl(url)
		repoUrls = append(repoUrls, repoUrl)
	}

	assert.Equal(t, "gcr.io", repoUrls[0].GetRegistry())
	assert.Equal(t, "knative-releases", repoUrls[0].GetNamespace())
	assert.Equal(t, "github.com/knative/build/cmd/creds-init", repoUrls[0].GetRepo())
	assert.Equal(t, "knative-releases/github.com/knative/build/cmd/creds-init", repoUrls[0].GetRepoWithNamespace())
	assert.Equal(t, "v1", repoUrls[0].GetTag())
	assert.Equal(t, "", repoUrls[1].GetTag())
	assert.Equal(t, "registry.hub.docker.io", repoUrls[1].GetRegistry())
	assert.Equal(t, "library", repoUrls[1].GetNamespace())
	assert.Equal(t, "nginx", repoUrls[1].GetRepo())
	assert.Equal(t, "library/nginx", repoUrls[1].GetRepoWithNamespace())
	assert.Equal(t, "library", repoUrls[2].GetNamespace())
	assert.Equal(t, "registry.hub.docker.com", repoUrls[2].GetRegistry())
	assert.Equal(t, "v1", repoUrls[2].GetTag())
	assert.Equal(t, "127.0.0.1:300", repoUrls[3].GetRegistry())
	assert.Equal(t, "127.0.0.1:300/library/nginx", repoUrls[3].GetUrlWithoutTag())
	assert.Equal(t, "v1", repoUrls[3].GetTag())
	assert.Equal(t, "nginx:v1", repoUrls[3].GetRepoWithTag())
}
