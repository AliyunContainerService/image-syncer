package sync

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/containers/image/v5/types"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"github.com/sirupsen/logrus"
)

var (
	// NoCache used to disable a blobinfocache
	NoCache = none.NoCache
)

// Task act as a sync action, it will pull images from source to destination
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

	destManifestObj, destManifestBytes, subManifestInfoSlice, err := GenerateManifestObj(manifestBytes,
		manifestType, t.osFilterList, t.archFilterList, t.source, nil)
	if err != nil {
		return t.Errorf("Get manifest info from %s/%s:%s error: %v",
			t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(), err)
	}

	if destManifestObj == nil {
		t.Infof("Skip synchronization from %s/%s:%s to %s/%s:%s, mismatch of os or architecture",
			t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(),
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		return nil
	}

	if len(subManifestInfoSlice) == 0 {
		// non-list type image
		if err = t.SyncNonListTypeImageByManifest(destManifestObj, destManifestBytes,
			t.source.tag, t.destination.tag, false); err != nil {
			return err
		}
	} else {
		// list type image
		if err = t.SyncListTypeImageByManifest(destManifestBytes, subManifestInfoSlice); err != nil {
			return err
		}
	}

	t.Infof("Synchronization successfully from %s/%s:%s to %s/%s:%s",
		t.source.GetRegistry(), t.source.GetRepository(), t.source.GetTag(),
		t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())

	return nil
}

func (t *Task) SyncListTypeImageByManifest(manifestBytes []byte, subManifestInfoSlice []*ManifestInfo) error {
	if changed := t.destination.CheckManifestChanged(manifestBytes, ""); !changed {
		// do nothing if manifest is unchanged
		t.Infof("Dest manifest list %s/%s:%s is unchanged, will do nothing",
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag())
		return nil
	}

	for _, mfstInfo := range subManifestInfoSlice {
		if err := t.SyncNonListTypeImageByManifest(mfstInfo.obj, mfstInfo.bytes,
			mfstInfo.digest.String(), mfstInfo.digest.String(), true); err != nil {
			return err
		}
	}

	if err := t.destination.PushManifest(manifestBytes, nil); err != nil {
		return t.Errorf("Put list-type manifest to %s/%s:%s error: %v",
			t.destination.GetRegistry(), t.destination.GetRepository(), t.destination.GetTag(), err)
	}

	return nil
}

func (t *Task) SyncNonListTypeImageByManifest(manifestObj interface{}, manifestBytes []byte,
	srcTagOrDigest, dstTagOrDigest string, belongsToList bool) error {

	if changed := t.destination.CheckManifestChanged(manifestBytes, dstTagOrDigest); !changed {
		// do nothing if manifest is unchanged
		t.Infof("Dest manifest %s/%s:%s is unchanged, will do nothing",
			t.destination.GetRegistry(), t.destination.GetRepository(), dstTagOrDigest)
		return nil
	}

	blobInfos, err := t.source.GetBlobInfos(manifestObj.(manifest.Manifest))
	if err != nil {
		return t.Errorf("Get blob infos from %s/%s:%s error: %v",
			t.source.GetRegistry(), t.source.GetRepository(), srcTagOrDigest, err)
	}

	if err := t.SyncBlobs(blobInfos...); err != nil {
		return fmt.Errorf("sync blob infos error: %v", err)
	}

	var instanceDigest *digest.Digest
	if belongsToList {
		tmp := digest.Digest(dstTagOrDigest)
		instanceDigest = &tmp
	}

	if err := t.destination.PushManifest(manifestBytes, instanceDigest); err != nil {
		return t.Errorf("Put manifest to %s/%s:%s error: %v",
			t.destination.GetRegistry(), t.destination.GetRepository(), dstTagOrDigest, err)
	}

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
