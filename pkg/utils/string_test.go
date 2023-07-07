package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	cases := [][]string{
		{
			"gcr.io/knative-releases/github.com/knative/build/cmd/creds-init:v1", "gcr.io/knative-releases/github.com/knative/build/cmd",
		}, {
			"registry.hub.docker.io/library/nginx", "registry.hub.docker.io/library/",
		}, {
			"registry.hub.docker.io/library/nginx", "registry.hub.docker.io/libr",
		}, {
			"registry.hub.docker.io/library/nginx", "",
		},
	}

	var results []bool
	for _, c := range cases {
		result := RepoMathPrefix(c[0], c[1])
		results = append(results, result)
	}

	assert.Equal(t, true, results[0])
	assert.Equal(t, true, results[1])
	assert.Equal(t, false, results[2])
	assert.Equal(t, false, results[3])
}
