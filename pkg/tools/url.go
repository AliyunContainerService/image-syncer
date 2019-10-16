package tools

import (
	"fmt"
	"strings"
)

// The RepoUrl will divide a images url to <registry>/<namespace>/<repo>:<tag>
type RepoUrl struct {
	// origin url
	url string

	registry  string
	namespace string
	repo      string
	tag       string
}

func NewRepoUrl(url string) (*RepoUrl, error) {
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
		return &RepoUrl{
			url:       url,
			registry:  slice[0],
			namespace: slice[1],
			repo:      repo,
			tag:       tag,
		}, nil
	} else if len(slice) == 2 {
		return &RepoUrl{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: slice[0],
			repo:      repo,
			tag:       tag,
		}, nil
	} else {
		return &RepoUrl{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: "library",
			repo:      repo,
			tag:       tag,
		}, nil
	}
}

func (r *RepoUrl) GetUrl() string {
	return r.url
}

func (r *RepoUrl) GetRegistry() string {
	return r.registry
}

func (r *RepoUrl) GetNamespace() string {
	return r.namespace
}

func (r *RepoUrl) GetRepo() string {
	return r.repo
}

func (r *RepoUrl) GetTag() string {
	return r.tag
}

func (r *RepoUrl) GetRepoWithNamespace() string {
	return r.namespace + "/" + r.repo
}

func (r *RepoUrl) GetRepoWithTag() string {
	if r.tag == "" {
		return r.repo
	}
	return r.repo + ":" + r.tag
}

func (r *RepoUrl) GetUrlWithoutTag() string {
	return r.registry + "/" + r.namespace + "/" + r.repo
}

func CheckIfIncludeTag(repository string) bool {
	return strings.Contains(repository, ":")
}
