package sync

import (
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/containers/image/v5/manifest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tidwall/gjson"
)

type ManifestInfo struct {
	obj    manifest.Manifest
	digest digest.Digest

	// (manifest.Manifest).Serialize() does not in general reproduce the original blob if this object was loaded from one,
	// even if no modifications were made! Non-list type image seems cannot use the result to push to dest
	// repo while it is a part of list-type image.
	bytes []byte
}

// GenerateManifestObj returns a new manifest object along with its byte serialization, and a sub manifest object array,
// and a digest array of sub manifests for list-type manifests.
// For list type manifest, the origin manifest info might be modified because of platform filters, and a nil manifest
// object will be returned if no sub manifest need to transport.
// For non-list type manifests, which doesn't match the filters, a nil manifest object will be returned.
func GenerateManifestObj(manifestBytes []byte, manifestType string, osFilterList, archFilterList []string,
	i *ImageSource, parent *manifest.Schema2List) (interface{}, []byte, []*ManifestInfo, error) {

	switch manifestType {
	case manifest.DockerV2Schema2MediaType:
		manifestObj, err := manifest.Schema2FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, nil, err
		}

		// platform info stored in config blob
		if parent == nil && manifestObj.ConfigInfo().Digest != "" {
			blob, _, err := i.GetABlob(manifestObj.ConfigInfo())
			if err != nil {
				return nil, nil, nil, err
			}
			defer blob.Close()
			bytes, err := io.ReadAll(blob)
			if err != nil {
				return nil, nil, nil, err
			}
			results := gjson.GetManyBytes(bytes, "architecture", "os")

			if !platformValidate(osFilterList, archFilterList,
				&manifest.Schema2PlatformSpec{Architecture: results[0].String(), OS: results[1].String()}) {
				return nil, nil, nil, nil
			}
		}

		return manifestObj, manifestBytes, nil, nil
	case manifest.DockerV2Schema1MediaType, manifest.DockerV2Schema1SignedMediaType:
		manifestObj, err := manifest.Schema1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, nil, err
		}

		// v1 only support architecture and this field is for information purposes and not currently used by the engine.
		if parent == nil && !platformValidate(osFilterList, archFilterList,
			&manifest.Schema2PlatformSpec{Architecture: manifestObj.Architecture}) {
			return nil, nil, nil, nil
		}

		return manifestObj, manifestBytes, nil, nil
	case specsv1.MediaTypeImageManifest:
		//TODO: platform filter?
		manifestObj, err := manifest.OCI1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, nil, err
		}
		return manifestObj, manifestBytes, nil, nil
	case manifest.DockerV2ListMediaType:
		var subManifestInfoSlice []*ManifestInfo

		manifestSchemaListObj, err := manifest.Schema2ListFromManifest(manifestBytes)
		if err != nil {
			return nil, nil, nil, err
		}

		var filteredDescriptors []manifest.Schema2ManifestDescriptor

		for _, manifestDescriptorElem := range manifestSchemaListObj.Manifests {
			// select os and arch
			if !platformValidate(osFilterList, archFilterList, &manifestDescriptorElem.Platform) {
				continue
			}

			filteredDescriptors = append(filteredDescriptors, manifestDescriptorElem)
			mfstBytes, mfstType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, nil, err
			}

			//TODO: will the sub manifest be list-type?
			subManifest, _, _, err := GenerateManifestObj(mfstBytes, mfstType,
				archFilterList, osFilterList, i, manifestSchemaListObj)
			if err != nil {
				return nil, nil, nil, err
			}

			if subManifest != nil {
				subManifestInfoSlice = append(subManifestInfoSlice, &ManifestInfo{
					obj:    subManifest.(manifest.Manifest),
					digest: manifestDescriptorElem.Digest,
					bytes:  mfstBytes,
				})
			}
		}

		// no sub manifests need to transport
		if len(filteredDescriptors) == 0 {
			return nil, nil, nil, nil
		}

		// return a new Schema2List
		if len(filteredDescriptors) != len(manifestSchemaListObj.Manifests) {
			manifestSchemaListObj.Manifests = filteredDescriptors
		}

		newManifestBytes, _ := manifestSchemaListObj.Serialize()

		return manifestSchemaListObj, newManifestBytes, subManifestInfoSlice, nil
	case specsv1.MediaTypeImageIndex:
		var subManifestInfoSlice []*ManifestInfo

		ociIndexesObj, err := manifest.OCI1IndexFromManifest(manifestBytes)
		if err != nil {
			return nil, nil, nil, err
		}

		var filteredDescriptors []specsv1.Descriptor

		for _, descriptor := range ociIndexesObj.Manifests {
			// select os and arch
			if !platformValidate(osFilterList, archFilterList, &manifest.Schema2PlatformSpec{
				Architecture: descriptor.Platform.Architecture,
				OS:           descriptor.Platform.OS,
			}) {
				continue
			}

			filteredDescriptors = append(filteredDescriptors, descriptor)

			mfstBytes, mfstType, innerErr := i.source.GetManifest(i.ctx, &descriptor.Digest)
			if innerErr != nil {
				return nil, nil, nil, innerErr
			}

			//TODO: will the sub manifest be list-type?
			subManifest, _, _, innerErr := GenerateManifestObj(mfstBytes, mfstType,
				archFilterList, osFilterList, i, nil)
			if innerErr != nil {
				return nil, nil, nil, err
			}

			if subManifest != nil {
				subManifestInfoSlice = append(subManifestInfoSlice, &ManifestInfo{
					obj:    subManifest.(manifest.Manifest),
					digest: descriptor.Digest,
					bytes:  mfstBytes,
				})
			}
		}

		// no sub manifests need to transport
		if len(filteredDescriptors) == 0 {
			return nil, nil, nil, nil
		}

		// return a new Schema2List
		if len(filteredDescriptors) != len(ociIndexesObj.Manifests) {
			ociIndexesObj.Manifests = filteredDescriptors
		}

		newManifestBytes, _ := ociIndexesObj.Serialize()

		return ociIndexesObj, newManifestBytes, subManifestInfoSlice, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported manifest type: %v", manifestType)
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
