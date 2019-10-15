package sync

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/containers/image/manifest"
	"github.com/opencontainers/go-digest"
)

type ManifestSchemaV2List struct {
	Manifests []ManifestSchemaV2Info `json:"manifests"`
}

type ManifestSchemaV2Info struct {
	Digest string `json:"digest"`
}

// Expend the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string) (manifest.Manifest, []*digest.Digest, error) {
	if t == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(m)
		if err != nil {
			return nil, nil, err
		}
		return manifestInfo, nil, nil
	} else if strings.Contains(t, "application/vnd.docker.distribution.manifest.v1") {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, nil, err
		}
		return manifestInfo, nil, nil
	} else if t == manifest.DockerV2ListMediaType {
		// get the digest of manifests in the manifest list
		ml := ManifestSchemaV2List{}
		if err := json.Unmarshal(m, &ml); err != nil {
			return nil, nil, err
		}

		manifestDigests := []*digest.Digest{}
		for _, item := range ml.Manifests {
			if d, err := digest.Parse(item.Digest); err != nil {
				return nil, nil, err
			} else {
				manifestDigests = append(manifestDigests, &d)
			}
		}
		return nil, manifestDigests, nil
	}
	return nil, nil, fmt.Errorf("unsupported manifest type: %v", t)
}
