package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	registry   string
	repository string
	tag        string
}

// NewImageDestination generates an ImageDestination by repository, the repository string must include "tag".
// If username or password is empty, access to repository will be anonymous.
func NewImageDestination(registry, repository, tag, username, password string, insecure bool) (*ImageDestination, error) {
	if utils.CheckIfIncludeTag(repository) {
		return nil, fmt.Errorf("repository string should not include tag")
	}

	// tag may be empty
	tagWithColon := ""
	if tag != "" {
		tagWithColon = ":" + tag
	}

	// if tag is empty, will attach to the "latest" tag
	destRef, err := docker.ParseReference("//" + registry + "/" + repository + tagWithColon)
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
		if utils.IsGCRPermanentServiceAccountToken(registry, username) {
			fmt.Printf("Getting oauth2 token for %s...\n", username)
			token, expiry, err := utils.GCPTokenFromCreds(password)
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
		tag:         tag,
	}, nil
}

// PushManifest push a manifest file to destination image.
// If instanceDigest is not nil, it contains a digest of the specific manifest instance to write the manifest for
// (when the primary manifest is a manifest list); this should always be nil if the primary manifest is not a manifest list.
func (i *ImageDestination) PushManifest(manifestByte []byte, instanceDigest *digest.Digest) error {
	return i.destination.PutManifest(i.ctx, manifestByte, instanceDigest)
}

// CheckManifestChanged checks if manifest of destination (tag) has changed.
func (i *ImageDestination) CheckManifestChanged(destManifestBytes []byte, tagOrDigest string) bool {
	// just use tag to get manifest
	existManifestBytes := i.GetManifest(tagOrDigest)
	return !manifestEqual(existManifestBytes, destManifestBytes)
}

func (i *ImageDestination) GetManifest(tagOrDigest string) []byte {
	var srcRef types.ImageReference
	var convertDigest *digest.Digest

	if len(tagOrDigest) != 0 {
		_, err := digest.Parse(tagOrDigest)
		manifestURL := i.registry + "/" + i.repository + ":" + tagOrDigest
		if err == nil {
			// has digest
			manifestURL = i.registry + "/" + i.repository + "@" + tagOrDigest
		}

		// create source to check manifest
		srcRef, err = docker.ParseReference("//" + manifestURL)
		if err != nil {
			return nil
		}

		tmp := digest.Digest(tagOrDigest)
		convertDigest = &tmp
	} else {
		srcRef = i.ref
	}

	source, err := srcRef.NewImageSource(i.ctx, i.sysctx)
	if err != nil {
		// if the source cannot be created, manifest not exist
		return nil
	}

	// if tagOrDigest is empty, convertDigest will be nil
	tManifestByte, _, err := source.GetManifest(i.ctx, convertDigest)
	if err != nil {
		// if error happens, it's considered that the manifest not exist
		return nil
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

// GetTag return the tag of a ImageDestination
func (i *ImageDestination) GetTag() string {
	return i.tag
}

func manifestEqual(m1, m2 []byte) bool {
	var a bytes.Buffer
	_ = json.Compact(&a, m1)

	var b bytes.Buffer
	_ = json.Compact(&b, m2)

	return a.String() == b.String()
}
