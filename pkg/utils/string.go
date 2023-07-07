package utils

import "strings"

func RepoMathPrefix(repo, prefix string) bool {
	if len(prefix) == 0 {
		return false
	}

	s := strings.TrimPrefix(repo, prefix)
	if s == repo {
		return false
	}

	return string(s[0]) == "/" || string(prefix[len(prefix)-1]) == "/"
}
