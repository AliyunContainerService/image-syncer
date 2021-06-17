package tools

import (
	"strings"

	"github.com/containers/image/v5/manifest"
)

type RepoFilter struct {
	Registry   string
	Repository string
	Tag        string
}

// Platform selector of sync client
type Platform struct {
	// default os selectors, foramt: os[:version]
	OsList []string `json:"os" yaml:"os"`
	// default arch selectors, format: architecture[:variant]
	ArchList []string `json:"arch" yaml:"arch"`

	// set include or exclude filters for source image, when both are present, exclude filters take precedence
	// filter string use repo url format: registry/namespace/repo:tag, empty field means match any
	Source struct {
		// include filters
		Include []string `json:"include" yaml:"include"`

		// exclude filters
		Exclude []string `json:"exclude" yaml:"exclude"`

		// Filers is exclude or include
		IsExclude bool

		// parsed filter
		Filters []RepoFilter
	} `json:"source" yaml:"source"`
}

// compare first:second to pat, second is optional
func colonMatch(pat string, first string, second string) bool {
	if strings.Index(pat, first) != 0 {
		return false
	}

	return len(first) == len(pat) || (pat[len(first)] == ':' && pat[len(first)+1:] == second)
}

// Match platform selector according to the source image and its platform
func (p *Platform) Match(registry string, repo string, tag string, platform *manifest.Schema2PlatformSpec) bool {
	doSelect := p.Source.IsExclude

	for _, p := range p.Source.Filters {
		if (p.Registry == "" || p.Registry == registry) &&
			(p.Repository == "" || p.Repository == repo) &&
			(p.Tag == "" || p.Tag == tag) {
			doSelect = !doSelect
			break
		}
	}

	if doSelect {
		osMatched := true
		archMatched := true
		if len(p.OsList) != 0 {
			osMatched = false
			for _, o := range p.OsList {
				// match os:osversion
				if colonMatch(o, platform.OS, platform.OSVersion) {
					osMatched = true
				}
			}
		}

		if len(p.ArchList) != 0 {
			archMatched = false
			for _, a := range p.ArchList {
				// match architecture:variant
				if colonMatch(a, platform.Architecture, platform.Variant) {
					archMatched = true
				}
			}
		}

		return osMatched && archMatched
	}

	return true
}
