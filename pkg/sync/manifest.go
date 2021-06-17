package sync

import (
	"fmt"
	"io/ioutil"

	"github.com/containers/image/v5/manifest"
	"github.com/tidwall/gjson"
)

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string, i *ImageSource, parent *manifest.Schema2List) ([]manifest.Manifest, interface{}, error) {
	var manifestInfoSlice []manifest.Manifest

	if t == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(m)
		if err != nil {
			return nil, nil, err
		}

		// platform info stored in config blob
		if parent == nil && manifestInfo.ConfigInfo().Digest != "" {
			blob, _, err := i.GetABlob(manifestInfo.ConfigInfo())
			if err != nil {
				return nil, nil, err
			}
			defer blob.Close()
			bytes, err := ioutil.ReadAll(blob)
			if err != nil {
				return nil, nil, err
			}
			results := gjson.GetManyBytes(bytes, "architecture", "os")
			if i.platformMatcher != nil && !i.platformMatcher.Match(i.registry, i.repository, i.tag, &manifest.Schema2PlatformSpec{Architecture: results[0].String(), OS: results[1].String()}) {
				return manifestInfoSlice, manifestInfo, nil
			}
		}

		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil, nil
	} else if t == manifest.DockerV2Schema1MediaType || t == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, nil, err
		}

		// v1 only support architecture
		if parent == nil && i.platformMatcher != nil && !i.platformMatcher.Match(i.registry, i.repository, i.tag, &manifest.Schema2PlatformSpec{Architecture: manifestInfo.Architecture}) {
			return manifestInfoSlice, manifestInfo, nil
		}

		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil, nil
	} else if t == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(m)
		if err != nil {
			return nil, nil, err
		}

		var nm []manifest.Schema2ManifestDescriptor

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			// select os arch as configed
			if i.platformMatcher != nil && !i.platformMatcher.Match(i.registry, i.repository, i.tag, &manifestDescriptorElem.Platform) {
				continue
			}

			nm = append(nm, manifestDescriptorElem)
			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, err
			}

			platformSpecManifest, _, err := ManifestHandler(manifestByte, manifestType, i, manifestSchemaListInfo)
			if err != nil {
				return nil, nil, err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}

		// return a new Schema2List
		if len(nm) != len(manifestSchemaListInfo.Manifests) {
			manifestSchemaListInfo.Manifests = nm
			return manifestInfoSlice, manifestSchemaListInfo, nil
		} else {
			return manifestInfoSlice, nil, nil
		}
	}

	return nil, nil, fmt.Errorf("unsupported manifest type: %v", t)
}
