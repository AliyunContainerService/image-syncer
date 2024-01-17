package concurrent

import (
	"sync"

	"github.com/AliyunContainerService/image-syncer/pkg/utils/types"
)

type ImageList struct {
	sync.Mutex
	content types.ImageList
}

func NewImageList() *ImageList {
	return &ImageList{
		content: types.ImageList{},
	}
}

func (i *ImageList) Add(src, dst string) {
	i.Lock()
	defer i.Unlock()

	i.content.Add(src, dst)
}

func (i *ImageList) Query(src, dst string) bool {
	i.Lock()
	defer i.Unlock()

	return i.content.Query(src, dst)
}

func (i *ImageList) Delete(key string) {
	i.Lock()
	defer i.Unlock()

	delete(i.content, key)
}

func (i *ImageList) Rest() {
	i.Lock()
	defer i.Unlock()

	i.content = types.ImageList{}
}

func (i *ImageList) Content() types.ImageList {
	i.Lock()
	defer i.Unlock()

	return i.content
}
