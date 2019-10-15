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
	// get tag in the url
	var tag string
	var urlExceptTag string = url

	s := strings.Split(url, ":")
	if len(s) > 3 {
		return nil, fmt.Errorf("invalid repository url: %v", url)
	} else if len(s) == 2 {
		tag = s[1]
		urlExceptTag = s[0]
	} else if len(s) == 3 {
		// "ip:port" format of registry url
		tag = s[2]
		urlExceptTag = s[0] + ":" + s[1]
	}

	// split to registry/namespace/repo
	slice := strings.SplitN(urlExceptTag, "/", 3)
	if len(slice) == 3 {
		return &RepoUrl{
			url:       url,
			registry:  slice[0],
			namespace: slice[1],
			repo:      slice[2],
			tag:       tag,
		}, nil
	} else if len(slice) == 2 {
		return &RepoUrl{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: slice[0],
			repo:      slice[1],
			tag:       tag,
		}, nil
	} else {
		return &RepoUrl{
			url:       url,
			registry:  "registry.hub.docker.com",
			namespace: "library",
			repo:      slice[0],
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
