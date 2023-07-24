package task

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/AliyunContainerService/image-syncer/pkg/utils"
)

// RuleTask analyze an image config rule ("xxx:xxx") and generates URLTask(s).
type RuleTask struct {
	source      string
	destination string

	getAuthFunc func(repository string) utils.Auth

	forceUpdate bool
}

func NewRuleTask(source, destination string,
	getAuthFunc func(repository string) utils.Auth, forceUpdate bool) (*RuleTask, error) {
	if source == "" {
		return nil, fmt.Errorf("source url should not be empty")
	}

	if destination == "" {
		return nil, fmt.Errorf("destination url should not be empty")
	}

	return &RuleTask{
		source:      source,
		destination: destination,
		getAuthFunc: getAuthFunc,
		forceUpdate: forceUpdate,
	}, nil
}

func (r *RuleTask) Run() ([]Task, string, error) {
	// if source tag is not specific, get all tags of this source repo
	sourceURLs, err := utils.GenerateRepoURLs(r.source, r.listAllTags)
	if err != nil {
		return nil, "", fmt.Errorf("source url %s format error: %v", r.source, err)
	}

	// if destination tags or digest is not specific, reuse tags or digest of sourceURLs
	destinationURLs, err := utils.GenerateRepoURLs(r.destination, func(registry, repository string) ([]string, error) {
		var result []string
		for _, item := range sourceURLs {
			result = append(result, item.GetTagOrDigest())
		}
		return result, nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("source url %s format error: %v", r.source, err)
	}

	// TODO: remove duplicated sourceURL and destinationURL pair?
	if err = checkSourceAndDestinationURLs(sourceURLs, destinationURLs); err != nil {
		return nil, "", fmt.Errorf("failed to check source and destination urls for %s:%s: %v",
			r.source, r.destination, err)
	}

	var results []Task
	for index, s := range sourceURLs {
		results = append(results, NewURLTask(s, destinationURLs[index], r.getAuthFunc(s.GetURLWithoutTagOrDigest()),
			r.getAuthFunc(destinationURLs[index].GetURLWithoutTagOrDigest()), r.forceUpdate))
	}

	return results, "", nil
}

func (r *RuleTask) GetPrimary() Task {
	return nil
}

func (r *RuleTask) Runnable() bool {
	// always runnable
	return true
}

func (r *RuleTask) ReleaseOnce() bool {
	// do nothing
	return true
}

func (r *RuleTask) GetSource() *sync.ImageSource {
	return nil
}

func (r *RuleTask) GetDestination() *sync.ImageDestination {
	return nil
}

func (r *RuleTask) String() string {
	return fmt.Sprintf("analyzing image rule for %s -> %s", r.source, r.destination)
}

func (r *RuleTask) listAllTags(sourceRegistry, sourceRepository string) ([]string, error) {
	auth := r.getAuthFunc(sourceRegistry + "/" + sourceRepository)

	imageSource, err := sync.NewImageSource(sourceRegistry, sourceRepository, "",
		auth.Username, auth.Password, auth.Insecure)
	if err != nil {
		return nil, fmt.Errorf("generate %s image source error: %v", sourceRegistry+"/"+sourceRepository, err)
	}

	return imageSource.GetSourceRepoTags()
}

func checkSourceAndDestinationURLs(sourceURLs, destinationURLs []*utils.RepoURL) error {
	if len(sourceURLs) != len(destinationURLs) {
		return fmt.Errorf("the number of tags of source and destination is not matched")
	}

	// digest must be the same
	if len(sourceURLs) == 1 && sourceURLs[0].HasDigest() && destinationURLs[0].HasDigest() {
		if sourceURLs[0].GetTagOrDigest() != destinationURLs[0].GetTagOrDigest() {
			return fmt.Errorf("the digest of source and destination must match")
		}
	}

	return nil
}
