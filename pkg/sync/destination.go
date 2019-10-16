package sync

import (
	"context"
	"fmt"
	"io"

	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/containers/image/docker"
	"github.com/containers/image/types"
)

// A PushTask reference to a remote image we will push to
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

// Generate a PullTask by repository, the repository string must include "tag",
// if username or password is empty, access to repository will be anonymous.
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

	ctx := context.WithValue(context.Background(), "ImageDestination", repository)
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
		registry:       registry,
		repository:     repository,
		tag:            tag,
	}, nil
}

// Push a manifest file to destinate image
func (i *ImageDestination) PushManifest(manifestByte []byte) error {
	return i.destination.PutManifest(i.ctx, manifestByte)
}

// Push a blob to destinate image
func (i *ImageDestination) PutABlob(blob io.ReadCloser, blobInfo types.BlobInfo) error {
	_, err := i.destination.PutBlob(i.ctx, blob, types.BlobInfo{
		Digest: blobInfo.Digest,
		Size:   blobInfo.Size,
	}, NoCache, true)

	// io.ReadCloser need to be close
	defer blob.Close()

	return err
}

func (i *ImageDestination) Close() error {
	return i.destination.Close()
}

func (i *ImageDestination) GetRegistry() string {
	return i.registry
}

func (i *ImageDestination) GetRepository() string {
	return i.repository
}

func (i *ImageDestination) GetTag() string {
	return i.tag
}
