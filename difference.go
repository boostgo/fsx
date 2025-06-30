package fsx

import "os"

// DifferenceType represents the type of difference between files/directories
type DifferenceType string

const (
	DiffAdded    DifferenceType = "added"
	DiffRemoved  DifferenceType = "removed"
	DiffModified DifferenceType = "modified"
	DiffSame     DifferenceType = "same"
)

// Difference represents a difference between directories
type Difference struct {
	Path      string
	Type      DifferenceType
	LeftInfo  os.FileInfo
	RightInfo os.FileInfo
}
