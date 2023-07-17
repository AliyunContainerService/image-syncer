package utils

import (
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/containers/image/v5/docker/reference"
)

const (
	DockerHubURL = "docker.io"
)

type RepoURL struct {
	// origin url
	ref reference.Reference

	// "namespace" is part of repo
	registry    string
	repo        string
	tagOrDigest string
}

// GenerateRepoURLs creates a RepoURL slice.
// If url has no tags or digest, tags or digest should be provided by externalTagsOrDigest func,
// and empty slice will be returned if no tags or digest is provided.
func GenerateRepoURLs(url string, externalTagsOrDigest func(registry, repository string,
) (tagsOrDigest []string, err error)) ([]*RepoURL, error) {

	var result []*RepoURL
	ref, err := reference.ParseNormalizedNamed(url)

	var tagsOrDigest []string
	var urlWithoutTagOrDigest string

	if canonicalRef, ok := ref.(reference.Canonical); ok {
		// url has digest
		tagsOrDigest = append(tagsOrDigest, canonicalRef.Digest().String())
		urlWithoutTagOrDigest = canonicalRef.Name()
	} else if taggedRef, ok := ref.(reference.NamedTagged); ok {
		// url has one normal tag
		tagsOrDigest = append(tagsOrDigest, taggedRef.Tag())
		urlWithoutTagOrDigest = taggedRef.Name()
	} else if err == nil {
		// url has no specified digest or tag
		registry, repo := getRegistryAndRepositoryFromURLWithoutTagOrDigest(url)
		allTags, err := externalTagsOrDigest(registry, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get external tags: %v", err)
		}

		urlWithoutTagOrDigest = url
		tagsOrDigest = append(tagsOrDigest, allTags...)
	} else {
		// url might have special tags

		// multiple tags exist
		slice := strings.SplitN(url, ",", -1)
		if len(slice) < 1 {
			return nil, fmt.Errorf("invalid repository url: %v", url)
		}

		ref, err = reference.ParseNormalizedNamed(slice[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse first tag with url %v: %v", slice[0], err)
		}

		urlWithoutTagOrDigest = ref.(reference.NamedTagged).Name()
		tagsOrDigest = append(tagsOrDigest, ref.(reference.NamedTagged).Tag())
		tagsOrDigest = append(tagsOrDigest, slice[1:]...)
	}

	registry, repo := getRegistryAndRepositoryFromURLWithoutTagOrDigest(urlWithoutTagOrDigest)

	// if no tags or digest provided, an empty slice will be returned
	for _, item := range tagsOrDigest {
		newURL := registry + "/" + repo + AttachConnectorToTagOrDigest(item)
		ref, err = reference.ParseNormalizedNamed(newURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parese canonical url: %v", newURL)
		}

		result = append(result, &RepoURL{
			ref:         ref,
			registry:    registry,
			repo:        repo,
			tagOrDigest: item,
		})
	}

	return result, nil
}

// GetURL returns the whole url
func (r *RepoURL) String() string {
	return r.ref.String()
}

// GetRegistry returns the registry in a url
func (r *RepoURL) GetRegistry() string {
	return r.registry
}

// GetRepo returns the repository in a url
func (r *RepoURL) GetRepo() string {
	return r.repo
}

// GetTagOrDigest returns the tag in a url
func (r *RepoURL) GetTagOrDigest() string {
	return r.tagOrDigest
}

// GetRepoWithTagOrDigest returns repository:tag in a url
func (r *RepoURL) GetRepoWithTagOrDigest() string {
	if r.tagOrDigest == "" {
		return r.repo
	}

	return r.repo + AttachConnectorToTagOrDigest(r.tagOrDigest)
}

func (r *RepoURL) HasDigest() bool {
	_, result := r.ref.(reference.Canonical)
	return result
}

func (r *RepoURL) GetURLWithoutTagOrDigest() string {
	return r.registry + "/" + r.repo
}

func AttachConnectorToTagOrDigest(tagOrDigest string) string {
	if len(tagOrDigest) == 0 {
		return ""
	}

	tmpDigest := digest.Digest(tagOrDigest)
	if err := tmpDigest.Validate(); err != nil {
		return ":" + tagOrDigest
	}
	return "@" + tagOrDigest
}

func getRegistryAndRepositoryFromURLWithoutTagOrDigest(urlWithoutTagOrDigest string) (registry string, repo string) {
	slice := strings.SplitN(urlWithoutTagOrDigest, "/", 2)
	if len(slice) == 1 {
		registry = DockerHubURL
		repo = slice[0]
	} else {
		registry = slice[0]
		repo = slice[1]
	}

	return
}
