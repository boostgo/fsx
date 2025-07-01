package fsx

import (
	"os"
	"sync"
)

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

// SearchResult represents a search result
type SearchResult struct {
	Path       string
	Info       os.FileInfo
	MatchedBy  string // What caused the match (name, content, size, etc.)
	LineNumber int    // For content searches
	Line       string // For content searches
}

// FileLock represents a file lock
type FileLock struct {
	path     string
	file     *os.File
	mu       sync.Mutex
	isLocked bool
}
