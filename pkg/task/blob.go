package task

import (
	"fmt"

	"github.com/AliyunContainerService/image-syncer/pkg/sync"
	"github.com/containers/image/v5/types"
)

type BlobTask struct {
	primary Task

	info types.BlobInfo
}

func (b *BlobTask) Run() (Task, string, error) {
	var resultMsg string

	dst := b.primary.GetDestination()
	src := b.primary.GetSource()

	blobExist, err := dst.CheckBlobExist(b.info)
	if err != nil {
		return nil, resultMsg, fmt.Errorf("failed to check blob %s(%v) exist: %v", b.info.Digest, b.info.Size, err)
	}

	// ignore exist blob
	if !blobExist {
		// pull a blob from source
		blob, size, err := src.GetABlob(b.info)
		if err != nil {
			return nil, resultMsg, fmt.Errorf("failed to get blob %s(%v): %v", b.info.Digest, size, err)
		}

		b.info.Size = size
		// push a blob to destination
		if err = dst.PutABlob(blob, b.info); err != nil {
			return nil, resultMsg, fmt.Errorf("failed to put blob %s(%v): %v", b.info.Digest, b.info.Size, err)
		}
	} else {
		resultMsg = fmt.Sprintf("ignore exist blob")
	}

	b.primary.FreeOnce()
	if b.primary.Runnable() {
		resultMsg = fmt.Sprintf("start to sync manifest")
		return b.primary, resultMsg, nil
	}
	return nil, resultMsg, nil
}

func (b *BlobTask) GetPrimary() Task {
	return b.primary
}

func (b *BlobTask) Runnable() bool {
	// always runnable
	return true
}

func (b *BlobTask) FreeOnce() {
	// do nothing
}

func (b *BlobTask) GetSource() *sync.ImageSource {
	return b.primary.GetSource()
}

func (b *BlobTask) GetDestination() *sync.ImageDestination {
	return b.primary.GetDestination()
}

func (b *BlobTask) String() string {
	return fmt.Sprintf("sync blob %s(%v) from %s to %s",
		b.info.Digest, b.info.Size, b.GetSource().String(), b.GetDestination().String())
}
