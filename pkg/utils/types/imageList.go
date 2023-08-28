package types

import (
	"fmt"
	"os"

	"github.com/AliyunContainerService/image-syncer/pkg/utils"
)

type ImageList map[string][]string

func NewImageList(origin map[string]interface{}) (ImageList, error) {
	result := map[string][]string{}

	for source, dest := range origin {
		convertErr := fmt.Errorf("invalid destination %v for source \"%v\", "+
			"destination should only be string or []string", dest, source)

		emptyDestErr := fmt.Errorf("empty destination is not supported for source: %v", source)

		if destList, ok := dest.([]interface{}); ok {
			// check if is destination is a []string
			for _, d := range destList {
				destStr, ok := d.(string)
				if !ok {
					return nil, convertErr
				}

				if len(destStr) == 0 {
					return nil, emptyDestErr
				}
				result[source] = append(result[source], os.ExpandEnv(destStr))
			}

			// empty slice is the same with an empty string
			if len(destList) == 0 {
				return nil, emptyDestErr
			}

			result[source] = utils.RemoveDuplicateItems(result[source])
		} else if destStr, ok := dest.(string); ok {
			// check if is destination is a string
			if len(destStr) == 0 {
				return nil, emptyDestErr
			}
			result[source] = append(result[source], os.ExpandEnv(destStr))
		} else {
			return nil, convertErr
		}
	}

	return result, nil
}

func (i ImageList) Query(src, dst string) bool {
	destList, exist := i[src]
	if exist {
		// check if is destination is a []string
		for _, d := range destList {
			if d == dst {
				return true
			}
		}
	}

	return false
}

func (i ImageList) Add(src, dst string) {
	if exist := i.Query(src, dst); !exist {
		i[src] = append(i[src], dst)
	}
}
