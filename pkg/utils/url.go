package utils

import (
	"fmt"
	"strings"
)

const (
	DockerHubURL = "docker.io"
)

// The RepoURL will divide an images url to <registry>/<namespace>/<repo>:<tag>
type RepoURL struct {
	// origin url
	url string

	registry  string
	namespace string
	repo      string
	tag       string
}

// NewRepoURL creates a RepoURL
func NewRepoURL(url string) (*RepoURL, error) {
	// split to registry/namespace/repoAndTag
	slice := strings.SplitN(url, "/", 3)

	var tag, repo string
	repoAndTag := slice[len(slice)-1]
	s := strings.Split(repoAndTag, ":")
	if len(s) > 2 {
		return nil, fmt.Errorf("invalid repository url: %v", url)
	} else if len(s) == 2 {
		repo = s[0]
		tag = s[1]
	} else {
		repo = s[0]
		tag = ""
	}

	if len(slice) == 3 {
		return &RepoURL{
			url:       url,
			registry:  slice[0],
			namespace: slice[1],
			repo:      repo,
			tag:       tag,
		}, nil
	} else if len(slice) == 2 {
		// if first string is a domain
		if strings.Contains(slice[0], ".") {
			return &RepoURL{
				url:       url,
				registry:  slice[0],
				namespace: "",
				repo:      repo,
				tag:       tag,
			}, nil
		}

		return &RepoURL{
			url:       url,
			registry:  DockerHubURL,
			namespace: slice[0],
			repo:      repo,
			tag:       tag,
		}, nil
	} else {
		return &RepoURL{
			url:       url,
			registry:  DockerHubURL,
			namespace: "library",
			repo:      repo,
			tag:       tag,
		}, nil
	}
}

// GetURL returns the whole url
func (r *RepoURL) GetURL() string {
	url := r.GetURLWithoutTag()
	if r.tag != "" {
		url = url + ":" + r.tag
	}
	return url
}

// GetOriginURL returns the whole url
func (r *RepoURL) GetOriginURL() string {
	return r.url
}

// GetRegistry returns the registry in a url
func (r *RepoURL) GetRegistry() string {
	return r.registry
}

// GetNamespace returns the namespace in a url
func (r *RepoURL) GetNamespace() string {
	return r.namespace
}

// GetRepo returns the repository in a url
func (r *RepoURL) GetRepo() string {
	return r.repo
}

// GetTag returns the tag in a url
func (r *RepoURL) GetTag() string {
	return r.tag
}

// GetRepoWithNamespace returns namespace/repository in a url
func (r *RepoURL) GetRepoWithNamespace() string {
	if r.namespace == "" {
		return r.repo
	}
	return r.namespace + "/" + r.repo
}

// GetRepoWithTag returns repository:tag in a url
func (r *RepoURL) GetRepoWithTag() string {
	if r.tag == "" {
		return r.repo
	}
	return r.repo + ":" + r.tag
}

// GetURLWithoutTag returns registry/namespace/repository in a url
func (r *RepoURL) GetURLWithoutTag() string {
	if r.namespace == "" {
		return r.registry + "/" + r.repo
	}
	return r.registry + "/" + r.namespace + "/" + r.repo
}

// CheckIfIncludeTag checks if a repository string includes tag
func CheckIfIncludeTag(repository string) bool {
	return strings.Contains(repository, ":")
}
