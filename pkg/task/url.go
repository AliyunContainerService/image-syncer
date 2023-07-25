package task

import (
	"fmt"
	"strings"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"
	"github.com/containers/image/v5/manifest"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
)

// URLTask converts an image RepoURL pair (specific tag) to BlobTask(s) and ManifestTask(s).
type URLTask struct {
	source      *utils.RepoURL
	destination *utils.RepoURL

	sourceAuth      utils.Auth
	destinationAuth utils.Auth

	osFilterList, archFilterList []string

	forceUpdate bool
}

func NewURLTask(source, destination *utils.RepoURL, sourceAuth, destinationAuth utils.Auth, forceUpdate bool) Task {
	return &URLTask{
		source:          source,
		destination:     destination,
		sourceAuth:      sourceAuth,
		destinationAuth: destinationAuth,
		forceUpdate:     forceUpdate,
	}
}

func (u *URLTask) Run() ([]Task, string, error) {
	imageSource, err := sync.NewImageSource(u.source.GetRegistry(), u.source.GetRepo(), u.source.GetTagOrDigest(),
		u.sourceAuth.Username, u.sourceAuth.Password, u.sourceAuth.IdentityToken, u.sourceAuth.Insecure)
	if err != nil {
		return nil, "", fmt.Errorf("generate %s image source error: %v", u.source.String(), err)
	}

	imageDestination, err := sync.NewImageDestination(u.destination.GetRegistry(), u.destination.GetRepo(),
		u.destination.GetTagOrDigest(), u.destinationAuth.Username, u.destinationAuth.Password, u.destinationAuth.Insecure)
	if err != nil {
		return nil, "", fmt.Errorf("generate %s image destination error: %v", u.destination.String(), err)
	}

	tasks, msg, err := u.generateSyncTasks(imageSource, imageDestination, u.osFilterList, u.archFilterList)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate manifest/blob tasks: %v", err)
	}

	return tasks, msg, nil
}

func (u *URLTask) GetPrimary() Task {
	return nil
}

func (u *URLTask) Runnable() bool {
	// always runnable
	return true
}

func (u *URLTask) ReleaseOnce() bool {
	// do nothing
	return true
}

func (u *URLTask) GetSource() *sync.ImageSource {
	return nil
}

func (u *URLTask) GetDestination() *sync.ImageDestination {
	return nil
}

func (u *URLTask) String() string {
	return fmt.Sprintf("generating sync tasks from %s to %s", u.source, u.destination)
}

// generateSyncTasks generates blob/manifest tasks.
func (u *URLTask) generateSyncTasks(source *sync.ImageSource, destination *sync.ImageDestination,
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

	if changed := destination.CheckManifestChanged(destManifestBytes, nil); !u.forceUpdate && !changed {
		// do nothing if image is unchanged
		resultMsg = "skip synchronization because destination image exists"
		return nil, resultMsg, nil
	}

	destManifestTask := NewManifestTask(nil,
		source, destination, nil, destManifestBytes, nil)

	if len(subManifestInfoSlice) == 0 {
		// non-list type image
		blobInfos, err := source.GetBlobInfos(destManifestObj.(manifest.Manifest))
		if err != nil {
			return nil, resultMsg, fmt.Errorf("failed to get blob infos: %v", err)
		}

		destManifestTask.counter = concurrent.NewCounter(len(blobInfos), len(blobInfos))

		for _, info := range blobInfos {
			// only append blob tasks
			results = append(results, NewBlobTask(destManifestTask, info))
		}
	} else {
		// list type image
		var noExistSubManifestCounter int
		var ignoredManifestDigests []string

		for _, mfstInfo := range subManifestInfoSlice {
			if changed := destination.CheckManifestChanged(mfstInfo.Bytes, mfstInfo.Digest); !u.forceUpdate && !changed {
				// do nothing if manifest is unchanged
				ignoredManifestDigests = append(ignoredManifestDigests, mfstInfo.Digest.String())
				continue
			}

			noExistSubManifestCounter++

			blobInfos, err := source.GetBlobInfos(mfstInfo.Obj)
			if err != nil {
				return nil, resultMsg, fmt.Errorf("failed to get blob infos for manifest %s: %v", mfstInfo.Digest, err)
			}

			subManifestTask := NewManifestTask(destManifestTask, source, destination,
				concurrent.NewCounter(len(blobInfos), len(blobInfos)), mfstInfo.Bytes, mfstInfo.Digest)

			for _, info := range blobInfos {
				// only append blob tasks
				results = append(results, NewBlobTask(subManifestTask, info))
			}
		}
		destManifestTask.counter = concurrent.NewCounter(noExistSubManifestCounter, noExistSubManifestCounter)

		if noExistSubManifestCounter == 0 {
			// all the sub manifests are exist in destination
			results = append(results, destManifestTask)
		}

		if len(ignoredManifestDigests) != 0 {
			resultMsg = fmt.Sprintf("%v sub manifests in the list are ignored: %v", len(ignoredManifestDigests),
				strings.Join(ignoredManifestDigests, ", "))
		}
	}

	return results, resultMsg, nil
}
