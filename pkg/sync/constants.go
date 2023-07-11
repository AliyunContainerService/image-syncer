package sync

import "github.com/containers/image/v5/pkg/blobinfocache/none"

var (
	// NoCache used to disable a blobinfocache
	NoCache = none.NoCache
)
