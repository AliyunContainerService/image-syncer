package sync

import (
	"fmt"
	"io"
	"strings"

	"github.com/containers/image/v5/manifest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tidwall/gjson"
)

// GenerateManifestObj returns a manifest object and a sub manifest object array for list-type manifests.
// For list type manifest, the origin manifest info might be modified because of platform filters, and a nil manifest
// object will be returned if no sub manifest need to transport.
// For non-list type manifests, which doesn't match the filters, a nil manifest object will be returned.
func GenerateManifestObj(manifestBytes []byte, manifestType string, osFilterList, archFilterList []string,
	i *ImageSource, parent *manifest.Schema2List) (interface{}, []manifest.Manifest, error) {

	switch manifestType {
	case manifest.DockerV2Schema2MediaType:
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
				return nil, nil, nil
			}
		}

		return manifestInfo, nil, nil
	case manifest.DockerV2Schema1MediaType, manifest.DockerV2Schema1SignedMediaType:
		manifestInfo, err := manifest.Schema1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}

		// v1 only support architecture and this field is for information purposes and not currently used by the engine.
		if parent == nil && !platformValidate(osFilterList, archFilterList,
			&manifest.Schema2PlatformSpec{Architecture: manifestInfo.Architecture}) {
			return nil, nil, nil
		}

		return manifestInfo, nil, nil
	case specsv1.MediaTypeImageManifest:
		//TODO: platform filter?
		ociImage, err := manifest.OCI1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}
		return ociImage, nil, nil
	case manifest.DockerV2ListMediaType:
		var subManifestInfoSlice []manifest.Manifest

		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}

		var filteredDescriptors []manifest.Schema2ManifestDescriptor

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			// select os and arch
			if !platformValidate(osFilterList, archFilterList, &manifestDescriptorElem.Platform) {
				continue
			}

			filteredDescriptors = append(filteredDescriptors, manifestDescriptorElem)
			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, err
			}

			//TODO: will the sub manifest be list-type?
			subManifest, _, err := GenerateManifestObj(manifestByte, manifestType,
				archFilterList, osFilterList, i, manifestSchemaListInfo)
			if err != nil {
				return nil, nil, err
			}

			if subManifest != nil {
				subManifestInfoSlice = append(subManifestInfoSlice, subManifest.(manifest.Manifest))
			}
		}

		// no sub manifests need to transport
		if len(filteredDescriptors) == 0 {
			return nil, nil, nil
		}

		// return a new Schema2List
		if len(filteredDescriptors) != len(manifestSchemaListInfo.Manifests) {
			manifestSchemaListInfo.Manifests = filteredDescriptors
		}

		return manifestSchemaListInfo, subManifestInfoSlice, nil
	case specsv1.MediaTypeImageIndex:
		var subManifestInfoSlice []manifest.Manifest

		ociIndexes, err := manifest.OCI1IndexFromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}

		var filteredDescriptors []specsv1.Descriptor

		for _, descriptor := range ociIndexes.Manifests {
			// select os and arch
			if !platformValidate(osFilterList, archFilterList, &manifest.Schema2PlatformSpec{
				Architecture: descriptor.Platform.Architecture,
				OS:           descriptor.Platform.OS,
			}) {
				continue
			}

			filteredDescriptors = append(filteredDescriptors, descriptor)

			manifestByte, manifestType, innerErr := i.source.GetManifest(i.ctx, &descriptor.Digest)
			if innerErr != nil {
				return nil, nil, innerErr
			}

			//TODO: will the sub manifest be list-type?
			subManifest, _, innerErr := GenerateManifestObj(manifestByte, manifestType,
				archFilterList, osFilterList, i, nil)
			if innerErr != nil {
				return nil, nil, err
			}

			subManifestInfoSlice = append(subManifestInfoSlice, subManifest.(manifest.Manifest))
		}

		// no sub manifests need to transport
		if len(filteredDescriptors) == 0 {
			return nil, nil, nil
		}

		// return a new Schema2List
		if len(filteredDescriptors) != len(ociIndexes.Manifests) {
			ociIndexes.Manifests = filteredDescriptors
		}

		return ociIndexes, subManifestInfoSlice, nil
	default:
		return nil, nil, fmt.Errorf("unsupported manifest type: %v", manifestType)
	}
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
