package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/containerd/containerd/images"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"strings"
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
			bytes, err := ioutil.ReadAll(blob)
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
	} else if manifestType == ocispec.MediaTypeImageManifest {
		manifestInfo, err := manifest.OCI1FromManifest(manifestBytes)
		if err != nil {
			return nil, nil, err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)

		return manifestInfoSlice, nil, nil
	} else if manifestType == ocispec.MediaTypeImageIndex {
		newManifest, err := CreateDockerV2Manifest(context.TODO(), manifestBytes, i.source)
		if err != nil {
			return nil, nil, err
		}

		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(newManifest)
		if err != nil {
			return nil, nil, err
		}

		var nm []manifest.Schema2ManifestDescriptor

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			//过滤os跟arch，默认是所有os跟arch都同步
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

type Source struct {
	Desc          ocispec.Descriptor
	Ref           reference.Named
	ManifestBytes []byte
}

func CreateDockerV2Manifest(ctx context.Context, manifestBytes []byte, source types.ImageSource) ([]byte, error) {
	manifestDescription, err := manifest.OCI1IndexFromManifest(manifestBytes)
	if err != nil {
		return nil, err
	}
	ss := make([]*Source, 0)

	for _, md := range manifestDescription.Manifests {

		manifestBytes, _, err := source.GetManifest(ctx, &md.Digest)
		if err != nil {
			return nil, err
		}
		ss = append(ss, &Source{Desc: md, ManifestBytes: manifestBytes})
	}

	//构造docker v2 manifest
	dt, _, err := Combine(ctx, ss)
	if err != nil {
		return nil, err
	}

	return dt, nil
}

func Combine(ctx context.Context, srcs []*Source) ([]byte, ocispec.Descriptor, error) {
	eg, ctx := errgroup.WithContext(ctx)

	dts := make([][]byte, len(srcs))
	for i := range dts {
		func(i int) {
			eg.Go(func() error {

				if srcs[i].Desc.MediaType != "" {
					mt, err := detectMediaType(srcs[i].ManifestBytes)
					if err != nil {
						return err
					}
					srcs[i].Desc.MediaType = mt
				}

				mt := srcs[i].Desc.MediaType

				switch mt {
				case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
					p := srcs[i].Desc.Platform
					if srcs[i].Desc.Platform == nil {
						p = &ocispec.Platform{}
					}
					srcs[i].Desc.Platform = p
				case images.MediaTypeDockerSchema1Manifest:
					return errors.Errorf("schema1 manifests are not allowed in manifest lists")
				}

				return nil
			})
		}(i)
	}

	if err := eg.Wait(); err != nil {
		return nil, ocispec.Descriptor{}, err
	}

	// on single source, return original bytes
	if len(srcs) == 1 {
		if mt := srcs[0].Desc.MediaType; mt == images.MediaTypeDockerSchema2ManifestList || mt == ocispec.MediaTypeImageIndex {
			return dts[0], srcs[0].Desc, nil
		}
	}

	m := map[digest.Digest]int{}
	newDescs := make([]ocispec.Descriptor, 0, len(srcs))

	addDesc := func(d ocispec.Descriptor) {
		idx, ok := m[d.Digest]
		if ok {
			old := newDescs[idx]
			if old.MediaType == "" {
				old.MediaType = d.MediaType
			}
			if d.Platform != nil {
				old.Platform = d.Platform
			}
			if old.Annotations == nil {
				old.Annotations = map[string]string{}
			}
			for k, v := range d.Annotations {
				old.Annotations[k] = v
			}
			newDescs[idx] = old
		} else {
			m[d.Digest] = len(newDescs)
			newDescs = append(newDescs, d)
		}
	}

	for i, src := range srcs {
		switch src.Desc.MediaType {
		case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
			var mfst ocispec.Index
			if err := json.Unmarshal(dts[i], &mfst); err != nil {
				return nil, ocispec.Descriptor{}, errors.WithStack(err)
			}
			for _, d := range mfst.Manifests {
				addDesc(d)
			}
		default:
			addDesc(src.Desc)
		}
	}

	mt := images.MediaTypeDockerSchema2ManifestList //ocispec.MediaTypeImageIndex
	idx := struct {
		// MediaType is reserved in the OCI spec but
		// excluded from go types.
		MediaType string `json:"mediaType,omitempty"`

		ocispec.Index
	}{
		MediaType: mt,
		Index: ocispec.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
			Manifests: newDescs,
		},
	}

	idxBytes, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return nil, ocispec.Descriptor{}, errors.Wrap(err, "failed to marshal index")
	}

	return idxBytes, ocispec.Descriptor{
		MediaType: mt,
		Size:      int64(len(idxBytes)),
		Digest:    digest.FromBytes(idxBytes),
	}, nil
}

func detectMediaType(manifestByte []byte) (string, error) {
	manifestInfo, err := manifest.OCI1FromManifest(manifestByte)
	if err != nil {
		return "", err
	}

	if &manifestInfo.Config != nil {
		return images.MediaTypeDockerSchema2Manifest, nil
	}
	return images.MediaTypeDockerSchema2ManifestList, nil
}
