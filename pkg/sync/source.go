package sync

import (
	"context"
	"fmt"
	"io"

	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/containers/image/docker"
	"github.com/containers/image/types"
)

// A ImageSource reference to a remote image need to be pulled.
type ImageSource struct {
	sourceRef types.ImageReference
	source    types.ImageSource
	ctx       context.Context
	sysctx    *types.SystemContext

	// source image description
	registry   string
	repository string
	tag        string
}

// Generate a PullTask by repository, the repository string must include "tag",
// if username or password is empty, access to repository will be anonymous.
// a repository string is the rest part of the images url except "tag" and "registry"
func NewImageSource(registry, repository, tag, username, password string) (*ImageSource, error) {
	if tools.CheckIfIncludeTag(repository) {
		return nil, fmt.Errorf("repository string should not include tag")
	}

	// tag may be empty
	tagWithColon := ""
	if tag != "" {
		tagWithColon = ":" + tag
	}

	srcRef, err := docker.ParseReference("//" + registry + "/" + repository + tagWithColon)
	if err != nil {
		return nil, err
	}

	sysctx := &types.SystemContext{}
	ctx := context.WithValue(context.Background(), "ImageSource", repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	var rawSource types.ImageSource = nil
	if tag != "" {
		// if tag is empty, will attach to the "latest" tag, and will get a error if "latest" is not exist
		rawSource, err = srcRef.NewImageSource(ctx, sysctx)
		if err != nil {
			return nil, err
		}
	}

	return &ImageSource{
		sourceRef:  srcRef,
		source:     rawSource,
		ctx:        ctx,
		registry:   registry,
		repository: repository,
		tag:        tag,
	}, nil
}

// Get manifest file from source image
func (i *ImageSource) GetManifest() ([]byte, string, error) {
	if i.source == nil {
		return nil, "", fmt.Errorf("cannot get manifest file without specfied a tag")
	}
	return i.source.GetManifest(i.ctx, nil)
}

// Get blobs from source image.
func (i *ImageSource) GetBlobInfos(manifestByte []byte, manifestType string) ([]types.BlobInfo, error) {
	if i.source == nil {
		return nil, fmt.Errorf("cannot get blobs without specfied a tag")
	}

	manifestInfo, digests, err := ManifestHandler(manifestByte, manifestType)
	if err != nil {
		return nil, err
	}

	if digests != nil {
		// TODO: manifest list support
		return nil, fmt.Errorf("Manifest list is not supported right now!")
	}
	// Recieved a manifest

	blobInfos := manifestInfo.LayerInfos()
	srcBlobs := []types.BlobInfo{}

	for _, l := range blobInfos {
		srcBlobs = append(srcBlobs, l.BlobInfo)
	}

	// append config blob info
	configBlob := manifestInfo.ConfigInfo()
	if configBlob.Digest != "" {
		srcBlobs = append(srcBlobs, configBlob)
	}

	return srcBlobs, nil
}

func (i *ImageSource) GetABlob(blobInfo types.BlobInfo) (io.ReadCloser, int64, error) {
	return i.source.GetBlob(i.ctx, types.BlobInfo{Digest: blobInfo.Digest, Size: -1}, NoCache)
}

func (i *ImageSource) Close() error {
	return i.source.Close()
}

func (i *ImageSource) GetRegistry() string {
	return i.registry
}

func (i *ImageSource) GetRepository() string {
	return i.repository
}

func (i *ImageSource) GetTag() string {
	return i.tag
}

func (i *ImageSource) GetSourceRepoTags() ([]string, error) {
	return docker.GetRepositoryTags(i.ctx, i.sysctx, i.sourceRef)
}
