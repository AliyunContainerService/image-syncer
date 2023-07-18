package task

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"
	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/opencontainers/go-digest"
)

// ManifestTask sync a manifest from source to destination.
type ManifestTask struct {
	source      *sync.ImageSource
	destination *sync.ImageDestination

	// for manifest, this refers to a manifest list
	primary Task

	counter *concurrent.Counter

	bytes  []byte
	digest *digest.Digest
}

func NewManifestTask(manifestListTask Task, source *sync.ImageSource, destination *sync.ImageDestination,
	counter *concurrent.Counter, bytes []byte, digest *digest.Digest) *ManifestTask {
	return &ManifestTask{
		primary:     manifestListTask,
		source:      source,
		destination: destination,
		counter:     counter,
		bytes:       bytes,
		digest:      digest,
	}
}

func (m *ManifestTask) Run() ([]Task, string, error) {
	var resultMsg string

	//// random failure test
	//rand.Seed(time.Now().UnixNano())
	//if rand.Intn(100)%2 == 1 {
	//	return nil, resultMsg, fmt.Errorf("random failure")
	//}

	if err := m.destination.PushManifest(m.bytes, m.digest); err != nil {
		return nil, resultMsg, fmt.Errorf("failed to put manifest: %v", err)
	}

	if m.primary == nil {
		return nil, resultMsg, nil
	}

	if m.primary.ReleaseOnce() {
		resultMsg = "start to sync manifest list"
		return []Task{m.primary}, resultMsg, nil
	}
	return nil, resultMsg, nil
}

func (m *ManifestTask) GetPrimary() Task {
	return m.primary
}

func (m *ManifestTask) Runnable() bool {
	count, _ := m.counter.Value()
	return count == 0
}

func (m *ManifestTask) ReleaseOnce() bool {
	count, _ := m.counter.Decrease()
	return count == 0
}

func (m *ManifestTask) GetSource() *sync.ImageSource {
	return m.source
}

func (m *ManifestTask) GetDestination() *sync.ImageDestination {
	return m.destination
}

func (m *ManifestTask) String() string {
	var srcTagOrDigest, dstTagOrDigest string
	if m.primary == nil {
		srcTagOrDigest = m.GetSource().GetTagOrDigest()
		dstTagOrDigest = m.GetDestination().GetTagOrDigest()
	} else {
		srcTagOrDigest = m.digest.String()
		dstTagOrDigest = m.digest.String()
	}

	return fmt.Sprintf("synchronizing manifest from %s/%s%s to %s/%s%s",
		m.GetSource().GetRegistry(), m.GetSource().GetRepository(), utils.AttachConnectorToTagOrDigest(srcTagOrDigest),
		m.GetDestination().GetRegistry(), m.GetDestination().GetRepository(), utils.AttachConnectorToTagOrDigest(dstTagOrDigest))
}
