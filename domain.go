package fsx

import "os"

// DirectoryEntry represents a file or subdirectory in a directory
type DirectoryEntry struct {
	Name    string
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime string
	IsDir   bool
}

// DirectoryInfo represents directory information
type DirectoryInfo struct {
	Path      string
	TotalSize int64
	FileCount int
	DirCount  int
	Mode      os.FileMode
	ModTime   string
}
