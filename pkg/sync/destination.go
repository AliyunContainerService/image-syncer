package sync

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"strings"
	"time"

	"github.com/AliyunContainerService/image-syncer/pkg/tools"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
)

// ImageDestination is a reference of a remote image we will push to
type ImageDestination struct {
	destinationRef types.ImageReference
	destination    types.ImageDestination
	ctx            context.Context
	sysctx         *types.SystemContext

	// destination image description
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
		// destination registry is http service
		sysctx = &types.SystemContext{
			DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		}
	} else {
		sysctx = &types.SystemContext{}
	}

	ctx := context.WithValue(context.Background(), ctxKey{"ImageDestination"}, repository)
	if username != "" && password != "" {
		fmt.Println("Credentials defined...processing")
		if strings.Contains(repository, "eu.gcr.io") {
			fmt.Println("GCR Credentials defined...processing")
			tempToken, _, _ := gcpTokenFromCreds(password)
			fmt.Printf("Temp Token: %s\n", tempToken)
		} else {
			fmt.Println("Not a GCR Repo... continuing")
		}
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
		//os.Exit(0)
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

// PushManifest push a manifest file to destination image
func (i *ImageDestination) PushManifest(manifestByte []byte) error {
	return i.destination.PutManifest(i.ctx, manifestByte, nil)
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

func gcpTokenFromCreds(creds string) (string, time.Time, error) {
	// create byte array from string
	b := []byte(creds)

	conf, err := google.JWTConfigFromJSON(
		b, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return "", time.Time{}, err
	}

	token, err := conf.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return "", time.Time{}, err
	}

	return token.AccessToken, token.Expiry, nil
}
