package sync

import (
	"fmt"
	"io"
	"strings"

	"github.com/containers/image/v5/manifest"
	"github.com/tidwall/gjson"
)

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(manifestBytes []byte, manifestType string, osFilterList, archFilterList []string,
	i *ImageSource, parent *manifest.Schema2List) ([]manifest.Manifest, interface{}, error) {
	var manifestInfoSlice []manifest.Manifest

	if manifestType == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(manifestBytes)
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
			bytes, err := io.ReadAll(blob)
			if err != nil {
				return nil, nil, err
			}
			results := gjson.GetManyBytes(bytes, "architecture", "os")

			if !platformValidate(osFilterList, archFilterList,
				&manifest.Schema2PlatformSpec{Architecture: results[0].String(), OS: results[1].String()}) {
				return manifestInfoSlice, manifestInfo, nil
			}
		}

		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil, nil
	} else if manifestType == manifest.DockerV2Schema1MediaType || manifestType == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}

		// v1 only support architecture and this field is for information purposes and not currently used by the engine.
		if parent == nil && !platformValidate(osFilterList, archFilterList,
			&manifest.Schema2PlatformSpec{Architecture: manifestInfo.Architecture}) {
			return manifestInfoSlice, manifestInfo, nil
		}

		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, nil, nil
	} else if manifestType == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}

		var nm []manifest.Schema2ManifestDescriptor

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			// select os and arch
			if !platformValidate(osFilterList, archFilterList, &manifestDescriptorElem.Platform) {
				continue
			}

			nm = append(nm, manifestDescriptorElem)
			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, err
			}

			platformSpecManifest, _, err := ManifestHandler(manifestByte, manifestType,
				archFilterList, osFilterList, i, manifestSchemaListInfo)
			if err != nil {
				return nil, nil, err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}

		// return a new Schema2List
		if len(nm) != len(manifestSchemaListInfo.Manifests) {
			manifestSchemaListInfo.Manifests = nm
			return manifestInfoSlice, manifestSchemaListInfo, nil
		}

		return manifestInfoSlice, nil, nil
	}

	return nil, nil, fmt.Errorf("unsupported manifest type: %v", manifestType)
}

// compare first:second to pat, second is optional
func colonMatch(pat string, first string, second string) bool {
	if strings.Index(pat, first) != 0 {
		return false
	}

	return len(first) == len(pat) || (pat[len(first)] == ':' && pat[len(first)+1:] == second)
}

// Match platform selector according to the source image and its platform.
// If platform.OS is not specified, the manifest will never be filtered, the same with platform.Architecture.
func platformValidate(osFilterList, archFilterList []string, platform *manifest.Schema2PlatformSpec) bool {
	osMatched := true
	archMatched := true

	if len(osFilterList) != 0 && platform.OS != "" {
		osMatched = false
		for _, o := range osFilterList {
			// match os:osversion
			if colonMatch(o, platform.OS, platform.OSVersion) {
				osMatched = true
			}
		}
	}

	if len(archFilterList) != 0 && platform.Architecture != "" {
		archMatched = false
		for _, a := range archFilterList {
			// match architecture:variant
			if colonMatch(a, platform.Architecture, platform.Variant) {
				archMatched = true
			}
		}
	}

	return osMatched && archMatched
}
