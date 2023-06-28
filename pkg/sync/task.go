package sync

import (
	"fmt"

	"github.com/containers/image/v5/types"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
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

	osFilterList   []string
	archFilterList []string

	logger *logrus.Logger
}

// NewTask creates a sync task
func NewTask(source *ImageSource, destination *ImageDestination,
	osFilterList, archFilterList []string, logger *logrus.Logger) *Task {
	if logger == nil {
		logger = logrus.New()
	}

	return &Task{
		source:      source,
		destination: destination,
		logger:      logger,

		osFilterList:   osFilterList,
		archFilterList: archFilterList,
	}
}

// Run is the main function of a sync task
func (t *Task) Run() error {
	// get manifest from source
	manifestBytes, manifestType, err := t.source.GetManifest()
	if err != nil {
		return t.Errorf("Failed to get manifest from %s/%s:%s error: %v",
			t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
	}
	t.Infof("Get manifest from %s/%s:%s", t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag())

	destManifestInfo, destManifestBytes, subManifestInfoSlice, err := GenerateManifestObj(manifestBytes, manifestType,
		t.osFilterList, t.archFilterList, t.source, nil)
	if err != nil {
		return t.Errorf("Get manifest info from %s/%s:%s error: %v",
			t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
	}

	if changed := t.destination.CheckManifestChanged(destManifestBytes); !changed {
		// do nothing if manifest is not changed
		t.Infof("Dest manifest %s/%s:%s is not changed, will do nothing",
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		return nil
	}

	if destManifestInfo == nil {
		t.Infof("Skip synchronization from %s/%s:%s to %s/%s:%s, mismatch of os or architecture",
			t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(),
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		return nil
	}

	var blobInfos []types.BlobInfo
	if len(subManifestInfoSlice) == 0 {
		// non-list type image
		blobInfos, err = t.source.GetBlobInfos(destManifestInfo.(manifest.Manifest))
		if err != nil {
			return t.Errorf("Get blob infos from %s/%s:%s error: %v",
				t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
		}
	} else {
		// list type image
		blobInfos, err = t.source.GetBlobInfos(subManifestInfoSlice...)
		if err != nil {
			return t.Errorf("Get blob infos from %s/%s:%s error: %v",
				t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
		}
	}

	if err := t.SyncBlobs(blobInfos...); err != nil {
		return fmt.Errorf("sync blob infos error: %v", err)
	}

	// Push manifest list
	if manifestType == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo := destManifestInfo.(*manifest.Schema2List)
		var subManifestByte []byte

		// push manifest to destination
		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {
			t.Infof("Handle manifest OS:%s Architecture:%s ",
				manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)

			subManifestByte, _, err = t.source.source.GetManifest(t.source.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return t.Errorf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v",
					manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS,
					manifestDescriptorElem.Platform.Architecture, err)
			}

			if err := t.destination.PushManifest(subManifestByte); err != nil {
				return t.Errorf("Put manifest to %s/%s:%s error: %v",
					t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
			}

			t.Infof("Put manifest to %s/%s:%s os:%s arch:%s",
				t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(),
				manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)
		}
	} else if manifestType == specsv1.MediaTypeImageIndex {
		ociIndex := destManifestInfo.(*manifest.OCI1Index)
		var subManifestByte []byte

		// push manifest to destination
		for _, manifestDescriptorElem := range ociIndex.Manifests {
			t.Infof("Handle manifest OS:%s Architecture:%s ",
				manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)

			subManifestByte, _, err = t.source.source.GetManifest(t.source.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return t.Errorf("Get manifest %v of OS:%s Architecture:%s for manifest list error: %v",
					manifestDescriptorElem.Digest, manifestDescriptorElem.Platform.OS,
					manifestDescriptorElem.Platform.Architecture, err)
			}

			if err := t.destination.PushManifest(subManifestByte); err != nil {
				return t.Errorf("Put manifest to %s/%s:%s error: %v",
					t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
			}

			t.Infof("Put manifest to %s/%s:%s os:%s arch:%s",
				t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(),
				manifestDescriptorElem.Platform.OS, manifestDescriptorElem.Platform.Architecture)
		}
	}

	if err := t.destination.PushManifest(destManifestBytes); err != nil {
		return t.Errorf("Put manifest to %s/%s:%s error: %v",
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
	}

	t.Infof("Synchronization successfully from %s/%s:%s to %s/%s:%s",
		t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(),
		t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())

	return nil
}

func (t *Task) SyncBlobs(blobInfos ...types.BlobInfo) error {
	for _, b := range blobInfos {
		blobExist, err := t.destination.CheckBlobExist(b)
		if err != nil {
			return t.Errorf("Check blob %s(%v) to %s/%s:%s exist error: %v",
				b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
		}

		if !blobExist {
			// pull a blob from source
			blob, size, err := t.source.GetABlob(b)
			if err != nil {
				return t.Errorf("Get blob %s(%v) from %s/%s:%s failed: %v",
					b.Digest, size, t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
			}
			t.Infof("Get a blob %s(%v) from %s/%s:%s success",
				b.Digest, size, t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag())

			b.Size = size
			// push a blob to destination
			if err := t.destination.PutABlob(blob, b); err != nil {
				return t.Errorf("Put blob %s(%v) to %s/%s:%s failed: %v",
					b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
			}
			t.Infof("Put blob %s(%v) to %s/%s:%s success",
				b.Digest, b.Size, t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		} else {
			// print the log of ignored blob
			t.Infof("Blob %s(%v) has been pushed to %s, will not be pushed",
				b.Digest, b.Size, t.destination.GetRegistry()+"/"+t.destination.GetRepository())
		}
	}

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
