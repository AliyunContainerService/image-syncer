package sync

import (
	"context"
	"fmt"
	"io"

	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/containers/image/docker"
	"github.com/containers/image/types"
)

// ImageDestination is a reference of a remote image we will push to
type ImageDestination struct {
	destinationRef types.ImageReference
	destination    types.ImageDestination
	ctx            context.Context
	sysctx         *types.SystemContext

	// destinate image description
	registry   string
	repository string
	tag        string
}

// NewImageDestination generates a ImageDestination by repository, the repository string must include "tag".
// If username or password is empty, access to repository will be anonymous.
func NewImageDestination(registry, repository, tag, username, password string, insecure bool) (*ImageDestination, error) {
	if tools.CheckIfIncludeTag(repository) {
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
		// destinatoin registry is http service
		sysctx = &types.SystemContext{
			DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		}
	} else {
		sysctx = &types.SystemContext{}
	}

	ctx := context.WithValue(context.Background(), interface{}("ImageDestination"), repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	rawDestination, err := destRef.NewImageDestination(ctx, sysctx)
	if err != nil {
		return nil, err
	}

	return &ImageDestination{
		destinationRef: destRef,
		destination:    rawDestination,
		ctx:            ctx,
		sysctx:         sysctx,
		registry:       registry,
		repository:     repository,
		tag:            tag,
	}, nil
}

// PushManifest push a manifest file to destinate image
func (i *ImageDestination) PushManifest(manifestByte []byte) error {
	return i.destination.PutManifest(i.ctx, manifestByte)
}

// PutABlob push a blob to destinate image
func (i *ImageDestination) PutABlob(blob io.ReadCloser, blobInfo types.BlobInfo) error {
	_, err := i.destination.PutBlob(i.ctx, blob, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, true)

	// io.ReadCloser need to be close
	defer blob.Close()

	return err
}

// CheckBlobExist checks if a blob exist for destination
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
