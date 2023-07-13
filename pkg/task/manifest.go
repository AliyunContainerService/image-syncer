package task

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"

	"github.com/AliyunContainerService/image-syncer/pkg/concurrent"
	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/opencontainers/go-digest"
)

type ManifestTask struct {
	source      *sync.ImageSource
	destination *sync.ImageDestination

	// for manifest, this refers to a manifest list
	primary Task

	counter *concurrent.Counter

	bytes  []byte
	digest *digest.Digest
}

func (m *ManifestTask) Run() (Task, string, error) {
	var resultMsg string

	if err := m.destination.PushManifest(m.bytes, m.digest); err != nil {
		return nil, resultMsg, fmt.Errorf("failed to put manifest: %v", err)
	}

	if m.primary == nil {
		return nil, resultMsg, nil
	}

	if m.primary.ReleaseOnce() {
		resultMsg = "start to sync manifest list"
		return m.primary, resultMsg, nil
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
	var tagOrDigest string
	if m.primary == nil {
		tagOrDigest = m.GetDestination().GetTagOrDigest()
	} else {
		tagOrDigest = m.digest.String()
	}

	return fmt.Sprintf("sync manifest from %s/%s%s to %s/%s%s",
		m.GetSource().GetRegistry(), m.GetSource().GetRepository(), utils.AttachConnectorToTagOrDigest(tagOrDigest),
		m.GetDestination().GetRegistry(), m.GetDestination().GetRepository(), utils.AttachConnectorToTagOrDigest(tagOrDigest))
}
