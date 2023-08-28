package task

import (
	"github.com/AliyunContainerService/image-syncer/pkg/sync"
)

type Type string

const (
	URLType      = Type("URL")
	ManifestType = Type("Manifest")
	RuleType     = Type("Rule")
	BlobType     = Type("Blob")
)

type Task interface {
	// Run returns primary task and result message if success while primary task is not nil and can run immediately.
	Run() ([]Task, string, error)

	// GetPrimary returns primary task, manifest for a blob, or manifest list for a manifest
	GetPrimary() Task

	// Runnable returns if the task can be executed immediately
	Runnable() bool

	// ReleaseOnce try to release once and return if the task is runnable after being released.
	ReleaseOnce() bool

	// GetSource return a source refers to the source images.
	GetSource() *sync.ImageSource

	// GetDestination return a source refers to the destination images
	GetDestination() *sync.ImageDestination

	String() string

	Type() Type
}
