package utils

func RemoveEmptyItems(slice []string) []string {
	var result []string
	for _, item := range slice {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func RemoveDuplicateItems(slice []string) []string {
	result := make([]string, 0, len(slice))
	temp := map[string]struct{}{}
	for _, item := range slice {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
