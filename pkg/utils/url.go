package utils

import (
	"fmt"
	"strings"

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
// If url has no tag(s), tags should be provided by externalTags func, and will return empty slice if no tag(s) provided.
func GenerateRepoURLs(url string, externalTags func(registry, repository string) (tags []string, err error)) ([]*RepoURL, error) {
	var result []*RepoURL
	ref, err := reference.ParseNormalizedNamed(url)

	var tagsOrDigest []string
	var urlWithoutTagOrDigest string
	var hasDigest bool

	if canonicalRef, ok := ref.(reference.Canonical); ok {
		// url has digest
		tagsOrDigest = append(tagsOrDigest, canonicalRef.Digest().String())
		urlWithoutTagOrDigest = canonicalRef.Name()
		hasDigest = true
	} else if taggedRef, ok := ref.(reference.NamedTagged); ok {
		// url has one normal tag
		tagsOrDigest = append(tagsOrDigest, taggedRef.Tag())
		urlWithoutTagOrDigest = taggedRef.Name()
	} else if err == nil {
		// url has no specified digest or tag
		slice := strings.SplitN(url, "/", 2)
		allTags, err := externalTags(slice[0], slice[1])
		if err != nil {
			return nil, fmt.Errorf("failed to get external tags: %v", err)
		}

		urlWithoutTagOrDigest = url
		tagsOrDigest = append(tagsOrDigest, allTags...)
	} else {
		// url might have special tag(s)
		// cannot split url by ":" at the first place, because url might have digest

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

	var registry, repo string
	// split to registry/repo
	slice := strings.SplitN(urlWithoutTagOrDigest, "/", 2)
	if len(slice) == 0 {
		registry = DockerHubURL
		repo = slice[0]
	} else {
		registry = slice[0]
		repo = slice[1]
	}

	// if no tag(s) or digest provided, an empty slice will be returned
	for _, item := range tagsOrDigest {
		if hasDigest {
			newURL := registry + "/" + repo + "@" + item
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
		} else {
			newURL := registry + "/" + repo + ":" + item
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

// GetTag returns the tag in a url
func (r *RepoURL) GetTag() string {
	return r.tagOrDigest
}

// GetRepoWithTag returns repository:tag in a url
func (r *RepoURL) GetRepoWithTag() string {
	if r.tagOrDigest == "" {
		return r.repo
	}

	if _, ok := r.ref.(reference.NamedTagged); ok {
		return r.repo + ":" + r.tagOrDigest
	} else if _, ok = r.ref.(reference.Canonical); ok {
		return r.repo + r.tagOrDigest
	}

	return ""
}

func (r *RepoURL) IsCanonical() bool {
	_, result := r.ref.(reference.Canonical)
	return result
}

func (r *RepoURL) GetURLWithoutTag() string {
	return r.registry + "/" + r.repo
}

// CheckIfIncludeTag checks if a repository string includes tag
func CheckIfIncludeTag(repository string) bool {
	return strings.Contains(repository, ":")
}
