package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/utils/auth"

	"github.com/containers/image/v5/manifest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// ImageDestination is a reference of a remote image we will push to
type ImageDestination struct {
	ref         types.ImageReference
	destination types.ImageDestination
	ctx         context.Context
	sysctx      *types.SystemContext

	// destination image description
	registry    string
	repository  string
	tagOrDigest string
}

// NewImageDestination generates an ImageDestination by repository, the repository string must include tag or digest.
// If username or password is empty, access to repository will be anonymous.
func NewImageDestination(registry, repository, tagOrDigest, username, password string, insecure bool) (*ImageDestination, error) {
	if strings.Contains(repository, ":") {
		return nil, fmt.Errorf("repository string should not include ':'")
	}

	// if tagOrDigest is empty, will attach to the "latest" tag
	destRef, err := docker.ParseReference("//" + registry + "/" + repository + utils.AttachConnectorToTagOrDigest(tagOrDigest))
	if err != nil {
		return nil, err
	}

	var sysctx *types.SystemContext
	if insecure {
		// destination registry is http service
		sysctx = &types.SystemContext{
			DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		}
	} else {
		sysctx = &types.SystemContext{}
	}

	ctx := context.WithValue(context.Background(), utils.CTXKey("ImageDestination"), repository)
	if username != "" && password != "" {
		//fmt.Printf("Credential processing for %s/%s ...\n", registry, repository)
		if auth.IsGCRPermanentServiceAccountToken(registry, username) {
			fmt.Printf("Getting oauth2 token for %s...\n", username)
			token, expiry, err := auth.GCPTokenFromCreds(password)
			if err != nil {
				return nil, err
			}

			fmt.Printf("oauth2 token expiry: %s\n", expiry)
			password = token
			username = "oauth2accesstoken"
		}
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	destination, err := destRef.NewImageDestination(ctx, sysctx)
	if err != nil {
		return nil, err
	}

	return &ImageDestination{
		ref:         destRef,
		destination: destination,
		ctx:         ctx,
		sysctx:      sysctx,
		registry:    registry,
		repository:  repository,
		tagOrDigest: tagOrDigest,
	}, nil
}

// PushManifest push a manifest file to destination image.
// If instanceDigest is not nil, it contains a digest of the specific manifest instance to write the manifest for
// (when the primary manifest is a manifest list); this should always be nil if the primary manifest is not a manifest list.
func (i *ImageDestination) PushManifest(manifestByte []byte, instanceDigest *digest.Digest) error {
	return i.destination.PutManifest(i.ctx, manifestByte, instanceDigest)
}

// CheckManifestChanged checks if manifest of specified tag or digest has changed.
func (i *ImageDestination) CheckManifestChanged(destManifestBytes []byte, instanceDigest *digest.Digest) bool {
	existManifestBytes := i.GetManifest(instanceDigest)
	return !manifestEqual(existManifestBytes, destManifestBytes)
}

func (i *ImageDestination) GetManifest(instanceDigest *digest.Digest) []byte {
	var err error
	var srcRef types.ImageReference

	if instanceDigest != nil {
		manifestURL := i.registry + "/" + i.repository + utils.AttachConnectorToTagOrDigest(instanceDigest.String())

		// create source to check manifest
		srcRef, err = docker.ParseReference("//" + manifestURL)
		if err != nil {
			return nil
		}
	} else {
		srcRef = i.ref
	}

	source, err := srcRef.NewImageSource(i.ctx, i.sysctx)
	if err != nil {
		// if the source cannot be created, manifest not exist
		return nil
	}

	tManifestByte, mineType, err := source.GetManifest(i.ctx, instanceDigest)
	if err != nil {
		// if error happens, it's considered that the manifest not exist
		return nil
	}

	// only for manifest list
	switch mineType {
	case manifest.DockerV2ListMediaType:
		manifestSchemaListObj, err := manifest.Schema2ListFromManifest(tManifestByte)
		if err != nil {
			return nil
		}

		for _, manifestDescriptorElem := range manifestSchemaListObj.Manifests {
			mfstBytes := i.GetManifest(&manifestDescriptorElem.Digest)
			if mfstBytes == nil {
				// cannot find sub manifest, manifest list not exist
				return nil
			}
		}

	case specsv1.MediaTypeImageIndex:
		ociIndexesObj, err := manifest.OCI1IndexFromManifest(tManifestByte)
		if err != nil {
			return nil
		}

		for _, manifestDescriptorElem := range ociIndexesObj.Manifests {
			mfstBytes := i.GetManifest(&manifestDescriptorElem.Digest)
			if mfstBytes == nil {
				// cannot find sub manifest, manifest list not exist
				return nil
			}
		}
	}

	return tManifestByte
}

// PutABlob push a blob to destination image
func (i *ImageDestination) PutABlob(blob io.ReadCloser, blobInfo types.BlobInfo) error {
	_, err := i.destination.PutBlob(i.ctx, blob, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, true)

	// io.ReadCloser need to be close
	defer blob.Close()

	return err
}

// CheckBlobExist checks if a blob exist for destination and reuse exist blobs
func (i *ImageDestination) CheckBlobExist(blobInfo types.BlobInfo) (bool, error) {
	exist, _, err := i.destination.TryReusingBlob(i.ctx, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, false)

	return exist, err
}

// Close a ImageDestination
func (i *ImageDestination) Close() error {
	return i.destination.Close()
}

// GetRegistry returns the registry of a ImageDestination
func (i *ImageDestination) GetRegistry() string {
	return i.registry
}

// GetRepository returns the repository of a ImageDestination
func (i *ImageDestination) GetRepository() string {
	return i.repository
}

// GetTagOrDigest return the tag or digest of a ImageDestination
func (i *ImageDestination) GetTagOrDigest() string {
	return i.tagOrDigest
}

func (i *ImageDestination) String() string {
	return i.registry + "/" + i.repository + utils.AttachConnectorToTagOrDigest(i.tagOrDigest)
}

func manifestEqual(m1, m2 []byte) bool {
	var a map[string]interface{}
	var b map[string]interface{}

	if err := json.Unmarshal(m1, &a); err != nil {
		//Received an unexpected manifest retrieval result, return false to trigger a fallback to the push task.
		return false
	}
	if err := json.Unmarshal(m2, &b); err != nil {
		return false
	}

	return reflect.DeepEqual(a, b)
}
