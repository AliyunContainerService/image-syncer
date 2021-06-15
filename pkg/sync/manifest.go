package sync

import (
	"fmt"

	"github.com/containers/image/v5/manifest"
)

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string, i *ImageSource) ([]manifest.Manifest, error) {

	var manifestInfoSlice []manifest.Manifest

	if t == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2Schema1MediaType || t == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil
	} else if t == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(m)
		if err != nil {
			return nil, err
		}

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			// select os arch as configed
			if i.platformMatcher != nil && i.platformMatcher.Match(i.registry, i.repository, i.tag, &manifestDescriptorElem.Platform) {
				continue
			}

			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, err
			}

			platformSpecManifest, err := ManifestHandler(manifestByte, manifestType, i)
			if err != nil {
				return nil, err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}
		return manifestInfoSlice, nil
	}

	return nil, fmt.Errorf("unsupported manifest type: %v", t)
}
