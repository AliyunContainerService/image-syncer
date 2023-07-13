package task

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"

	"github.com/containers/image/v5/manifest"
)

type Task interface {
	// Run returns primary task and result message if success while primary task is not nil and can run immediately.
	Run() (Task, string, error)

	// GetPrimary returns primary task, manifest for a blob, or manifest list for a manifest
	GetPrimary() Task

	// Runnable returns if the task can be executed immediately
	Runnable() bool

	// ReleaseOnce try to release once and return if the task is runnable after being released.
	ReleaseOnce() bool

	// GetSource return a source refers to the source images.
	GetSource() *sync.ImageSource

	// GetDestination return a source refers to the destination images
	GetDestination() *sync.ImageDestination

	String() string
}

func GenerateTasks(source *sync.ImageSource, destination *sync.ImageDestination,
	osFilterList, archFilterList []string) ([]Task, string, error) {
	var results []Task
	var resultMsg string

	// get manifest from source
	manifestBytes, manifestType, err := source.GetManifest()
	if err != nil {
		return nil, resultMsg, fmt.Errorf("failed to get manifest: %v", err)
	}

	destManifestObj, destManifestBytes, subManifestInfoSlice, err := sync.GenerateManifestObj(manifestBytes,
		manifestType, osFilterList, archFilterList, source, nil)
	if err != nil {
		return nil, resultMsg, fmt.Errorf(" failed to get manifest info: %v", err)
	}

	if destManifestObj == nil {
		resultMsg = "skip synchronization because no manifest fits platform filters"
		return nil, resultMsg, nil
	}

	if changed := destination.CheckManifestChanged(destManifestBytes, nil); !changed {
		// do nothing if image is unchanged
		resultMsg = "skip synchronization because destination image exists"
		return nil, resultMsg, nil
	}

	destManifestTask := &ManifestTask{
		source:      source,
		destination: destination,
		primary:     nil,
		bytes:       destManifestBytes,
		digest:      nil,
	}

	if len(subManifestInfoSlice) == 0 {
		// non-list type image
		blobInfos, err := source.GetBlobInfos(destManifestObj.(manifest.Manifest))
		if err != nil {
			return nil, resultMsg, fmt.Errorf("failed to get blob infos: %v", err)
		}

		destManifestTask.counter = concurrent.NewCounter(len(blobInfos), len(blobInfos))

		for _, info := range blobInfos {
			// only append blob tasks
			results = append(results, &BlobTask{
				primary: destManifestTask,
				info:    info,
			})
		}
	} else {
		// list type image
		var noExistSubManifestCounter int
		for _, mfstInfo := range subManifestInfoSlice {
			if changed := destination.CheckManifestChanged(mfstInfo.Bytes, mfstInfo.Digest); !changed {
				// do nothing if manifest is unchanged
				continue
			}

			noExistSubManifestCounter++

			blobInfos, err := source.GetBlobInfos(mfstInfo.Obj)
			if err != nil {
				return nil, resultMsg, fmt.Errorf("failed to get blob infos for manifest %s: %v", mfstInfo.Digest, err)
			}

			subManifestTask := &ManifestTask{
				source:      source,
				destination: destination,
				primary:     destManifestTask,
				counter:     concurrent.NewCounter(len(blobInfos), len(blobInfos)),
				bytes:       mfstInfo.Bytes,
				digest:      mfstInfo.Digest,
			}

			for _, info := range blobInfos {
				// only append blob tasks
				results = append(results, &BlobTask{
					primary: subManifestTask,
					info:    info,
				})
			}
		}
		destManifestTask.counter = concurrent.NewCounter(noExistSubManifestCounter, noExistSubManifestCounter)

		if noExistSubManifestCounter == 0 {
			// all the sub manifests are exist in destination
			results = append(results, destManifestTask)
		}
	}

	return results, resultMsg, nil
}
