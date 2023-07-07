package utils

import (
	"fmt"
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
		"127.0.0.1:300/library/nginx:v1,v2",
		"127.0.0.1:300/library/nginx",
		"registry.cn-beijing.aliyuncs.com/hhyasdf/hybridnet@sha256:df2ef9e979fc063645dcbed51374233c6bcf4ab49308c0478702565e96b9bc9e",
	}

	var repoURLs []*RepoURL
	for _, url := range urls {
		tmpUrls, err := GenerateRepoURLs(url, func(registry, repository string) (tags []string, err error) {
			return []string{"latest"}, nil
		})
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		repoURLs = append(repoURLs, tmpUrls...)
	}

	assert.Equal(t, "gcr.io", repoURLs[0].GetRegistry())
	assert.Equal(t, "gcr.io/knative-releases/github.com/knative/build/cmd/creds-init:v1", repoURLs[0].String())
	assert.Equal(t, "knative-releases/github.com/knative/build/cmd/creds-init", repoURLs[0].GetRepo())
	assert.Equal(t, "v1", repoURLs[0].GetTag())
	assert.Equal(t, "latest", repoURLs[1].GetTag())
	assert.Equal(t, "registry.hub.docker.io", repoURLs[1].GetRegistry())
	assert.Equal(t, "library/nginx", repoURLs[1].GetRepo())
	assert.Equal(t, DockerHubURL, repoURLs[2].GetRegistry())
	assert.Equal(t, "v1", repoURLs[2].GetTag())
	assert.Equal(t, "library/nginx", repoURLs[2].GetRepo())
	assert.Equal(t, "127.0.0.1:300", repoURLs[3].GetRegistry())
	assert.Equal(t, "v1", repoURLs[3].GetTag())
	assert.Equal(t, "library/nginx:v1", repoURLs[3].GetRepoWithTag())
	assert.Equal(t, "127.0.0.1:300/library/nginx", repoURLs[4].GetURLWithoutTag())
	assert.Equal(t, "127.0.0.1:300", repoURLs[4].GetRegistry())
	assert.Equal(t, "library/nginx:latest", repoURLs[4].GetRepoWithTag())
	assert.Equal(t, "v2", repoURLs[6].GetTag())
}
