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
