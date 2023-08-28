package concurrent

import (
	"github.com/AliyunContainerService/image-syncer/pkg/utils/types"
)

type ImageList struct {
	c       chan struct{}
	content types.ImageList
}

func NewImageList() *ImageList {
	return &ImageList{
		c:       make(chan struct{}, 1),
		content: types.ImageList{},
	}
}

func (i *ImageList) Add(src, dst string) {
	i.c <- struct{}{}
	defer func() {
		<-i.c
	}()

	i.content.Add(src, dst)
}

func (i *ImageList) Query(src, dst string) bool {
	i.c <- struct{}{}
	defer func() {
		<-i.c
	}()

	return i.content.Query(src, dst)
}

func (i *ImageList) Delete(key string) {
	i.c <- struct{}{}
	defer func() {
		<-i.c
	}()

	delete(i.content, key)
}

func (i *ImageList) Rest() {
	i.c <- struct{}{}
	defer func() {
		<-i.c
	}()

	i.content = types.ImageList{}
}

func (i *ImageList) Content() types.ImageList {
	i.c <- struct{}{}
	defer func() {
		<-i.c
	}()

	return i.content
}
