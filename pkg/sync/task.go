package sync

import (
	"fmt"

	"github.com/containers/image/v5/manifest"

	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"github.com/sirupsen/logrus"
)

var (
	// NoCache used to disable a blobinfocache
	NoCache = none.NoCache
)

// Task act as a sync action, it will pull a images from source to destination
type Task struct {
	source      *ImageSource
	destination *ImageDestination

	logger *logrus.Logger
}

// NewTask creates a sync task
func NewTask(source *ImageSource, destination *ImageDestination, logger *logrus.Logger) *Task {
	if logger == nil {
		logger = logrus.New()
	}

	return &Task{
		source:      source,
		destination: destination,
		logger:      logger,
	}
}

// Run is the main function of a sync task
func (t *Task) Run() error {

	// get manifest from source
	manifestByte, manifestType, err := t.source.GetManifest()
	if err != nil {
		return t.Errorf("Failed to get manifest from %s/%s:%s error: %v", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
	}
	t.Infof("Get manifest from %s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag())

	blobInfos, err := t.source.GetBlobInfos(manifestByte, manifestType)
	if err != nil {
		return t.Errorf("Get blob info from %s/%s:%s error: %v", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
	}

	// blob transformation
	for _, b := range blobInfos {
		blobExist, err := t.destination.CheckBlobExist(b)
		if err != nil {
			return t.Errorf("Check blob %s(%v) to %s/%s:%s exist error: %v", b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
		}

		if !blobExist {
			// pull a blob from source
			blob, size, err := t.source.GetABlob(b)
			if err != nil {
				return t.Errorf("Get blob %s(%v) from %s/%s:%s failed: %v", b.Digest, size, t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
			}
			t.Infof("Get a blob %s(%v) from %s/%s:%s success", b.Digest, size, t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag())

			b.Size = size
			// push a blob to destination
			if err := t.destination.PutABlob(blob, b); err != nil {
				return t.Errorf("Put blob %s(%v) to %s/%s:%s failed: %v", b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
			}
			t.Infof("Put blob %s(%v) to %s/%s:%s success", b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		} else {
			// print the log of ignored blob
			t.Infof("Blob %s(%v) has been pushed to %s, will not be pushed", b.Digest, b.Size, t.destination.GetRegistry()+"/"+t.destination.GetRepository())
		}

	}

	//Push manifest list
	if manifestType == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(manifestByte)
		if err != nil {
			return err
		}

		var subManifestByte []byte

		// push manifest to destination
		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {

			t.Infof("handle manifest OS:%s Architecture:%s ", manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)

			subManifestByte, _, err = t.source.source.GetManifest(t.source.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return t.Errorf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v", manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture, err)
			}

			if err := t.destination.PushManifest(subManifestByte); err != nil {
				return t.Errorf("Put manifest to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
			}

			t.Infof("Put manifest to %s/%s:%s os:%s arch:%s", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)

		}

		// push manifest list to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return t.Errorf("Put manifestList to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
		}

		t.Infof("Put manifestList to %s/%s:%s", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())

	} else {

		// push manifest to destination
		if err := t.destination.PushManifest(manifestByte); err != nil {
			return t.Errorf("Put manifest to %s/%s:%s error: %v", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
		}

		t.Infof("Put manifest to %s/%s:%s", t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
	}

	t.Infof("Synchronization successfully from %s/%s:%s to %s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())

	return nil
}

// Errorf logs error to logger
func (t *Task) Errorf(format string, args ...interface{}) error {
	t.logger.Errorf(format, args...)
	return fmt.Errorf(format, args...)
}

// Infof logs info to logger
func (t *Task) Infof(format string, args ...interface{}) {
	t.logger.Infof(format, args...)
}
