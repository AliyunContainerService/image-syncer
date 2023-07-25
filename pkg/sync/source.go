package sync

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
)

// ImageSource is a reference to a remote image need to be pulled.
type ImageSource struct {
	ref    types.ImageReference
	source types.ImageSource
	ctx    context.Context
	sysctx *types.SystemContext

	// source image description
	registry    string
	repository  string
	tagOrDigest string
}

// NewImageSource generates a PullTask by repository, the repository string must include tag or digest, or it can only be used
// to list tags.
// If username or password is empty, access to repository will be anonymous.
// A repository string is the rest part of the images url except tag digest and registry
func NewImageSource(registry, repository, tagOrDigest, username, password, identityToken string, insecure bool) (*ImageSource, error) {
	if strings.Contains(repository, ":") {
		return nil, fmt.Errorf("repository string should not include ':'")
	}

	srcRef, err := docker.ParseReference("//" + registry + "/" + repository + utils.AttachConnectorToTagOrDigest(tagOrDigest))
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

	ctx := context.WithValue(context.Background(), utils.CTXKey("ImageSource"), repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	if identityToken != "" {
		if sysctx.DockerAuthConfig == nil {
			sysctx.DockerAuthConfig = &types.DockerAuthConfig{}
		}
		sysctx.DockerAuthConfig.IdentityToken = identityToken
	}

	var source types.ImageSource
	if tagOrDigest != "" {
		// if tagOrDigest is empty, will attach to the "latest" tag, and will get an error if "latest" is not exist
		source, err = srcRef.NewImageSource(ctx, sysctx)
		if err != nil {
			return nil, err
		}
	}

	return &ImageSource{
		ref:         srcRef,
		source:      source,
		ctx:         ctx,
		sysctx:      sysctx,
		registry:    registry,
		repository:  repository,
		tagOrDigest: tagOrDigest,
	}, nil
}

// GetManifest get manifest file from source image
func (i *ImageSource) GetManifest() ([]byte, string, error) {
	if i.source == nil {
		return nil, "", fmt.Errorf("cannot get manifest file without specified a tag or digest")
	}
	return i.source.GetManifest(i.ctx, nil)
}

// GetBlobInfos get blob infos from non-list type manifests.
func (i *ImageSource) GetBlobInfos(manifestObjSlice ...manifest.Manifest) ([]types.BlobInfo, error) {
	if i.source == nil {
		return nil, fmt.Errorf("cannot get blobs without specified a tag or digest")
	}

	var srcBlobs []types.BlobInfo
	for _, manifestObj := range manifestObjSlice {
		blobInfos := manifestObj.LayerInfos()
		for _, l := range blobInfos {
			srcBlobs = append(srcBlobs, l.BlobInfo)
		}
		// append config blob info
		configBlob := manifestObj.ConfigInfo()
		if configBlob.Digest != "" {
			srcBlobs = append(srcBlobs, configBlob)
		}
	}

	return srcBlobs, nil
}

// GetABlob gets a blob from remote image
func (i *ImageSource) GetABlob(blobInfo types.BlobInfo) (io.ReadCloser, int64, error) {
	return i.source.GetBlob(i.ctx, types.BlobInfo{Digest: blobInfo.Digest, URLs: blobInfo.URLs, Size: -1}, NoCache)
}

// Close an ImageSource
func (i *ImageSource) Close() error {
	return i.source.Close()
}

// GetRegistry returns the registry of a ImageSource
func (i *ImageSource) GetRegistry() string {
	return i.registry
}

// GetRepository returns the repository of a ImageSource
func (i *ImageSource) GetRepository() string {
	return i.repository
}

// GetTagOrDigest returns the tag or digest a ImageSource
func (i *ImageSource) GetTagOrDigest() string {
	return i.tagOrDigest
}

func (i *ImageSource) String() string {
	return i.registry + "/" + i.repository + utils.AttachConnectorToTagOrDigest(i.tagOrDigest)
}

// GetSourceRepoTags gets all the tags of a repository which ImageSource belongs to
func (i *ImageSource) GetSourceRepoTags() ([]string, error) {
	// this function still works out even the tagOrDigest is empty
	return docker.GetRepositoryTags(i.ctx, i.sysctx, i.ref)
}
