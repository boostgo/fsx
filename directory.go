package fsx

import (
	"os"
	"path/filepath"
	"sort"
)

// DirectoryOption represents optional parameters for directory operations
type DirectoryOption func(*directoryOptions)

type directoryOptions struct {
	perm      os.FileMode
	recursive bool
	force     bool
}

// defaultDirectoryOptions returns default options for directory operations
func defaultDirectoryOptions() *directoryOptions {
	return &directoryOptions{
		perm:      0755,
		recursive: false,
		force:     false,
	}
}

// WithDirPermissions sets custom directory permissions
func WithDirPermissions(perm os.FileMode) DirectoryOption {
	return func(opts *directoryOptions) {
		opts.perm = perm
	}
}

// WithRecursive enables recursive operations
func WithRecursive() DirectoryOption {
	return func(opts *directoryOptions) {
		opts.recursive = true
	}
}

// WithForce forces operation (e.g., delete non-empty directories)
func WithForce() DirectoryOption {
	return func(opts *directoryOptions) {
		opts.force = true
	}
}

func DirectoryExist(path string) bool {
	stat, _ := os.Stat(path)
	if stat == nil {
		return false
	}

	return stat.IsDir()
}

// CreateDirectory creates a single directory
func CreateDirectory(path string, options ...DirectoryOption) error {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	if err := os.Mkdir(path, opts.perm); err != nil {
		if os.IsExist(err) {
			return nil // Already exists
		}
		return ErrCreateDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return nil
}

// CreateDirectories creates directory tree (like mkdir -p)
func CreateDirectories(path string, options ...DirectoryOption) error {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	if err := os.MkdirAll(path, opts.perm); err != nil {
		return ErrCreateDirectories.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return nil
}

// DeleteDirectory removes a directory
func DeleteDirectory(path string, options ...DirectoryOption) error {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	if !DirectoryExist(path) {
		return nil // Already doesn't exist
	}

	if opts.recursive || opts.force {
		// Remove directory and all contents
		if err := os.RemoveAll(path); err != nil {
			return ErrDeleteDirectory.
				SetError(err).
				SetData(pathErrorContext{
					Path:  path,
					Error: err,
				})
		}
	} else {
		// Remove only if empty
		if err := os.Remove(path); err != nil {
			if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == os.ErrNotExist {
				return nil
			}
			// Check if directory is not empty
			entries, _ := os.ReadDir(path)
			if len(entries) > 0 {
				return ErrDeleteDirectoryNotEmpty.
					SetData(pathErrorContext{
						Path:  path,
						Error: err,
					})
			}
			return ErrDeleteDirectory.
				SetError(err).
				SetData(pathErrorContext{
					Path:  path,
					Error: err,
				})
		}
	}

	return nil
}

// RenameDirectory renames/moves a directory
func RenameDirectory(oldPath, newPath string, options ...DirectoryOption) error {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Check if source exists and is a directory
	if !DirectoryExist(oldPath) {
		return ErrDirectoryNotExist.
			SetData(pathErrorContext{
				Path:  oldPath,
				Error: os.ErrNotExist,
			})
	}

	// Create parent directory if needed
	if opts.recursive {
		parentDir := filepath.Dir(newPath)
		if err := CreateDirectories(parentDir); err != nil {
			return err
		}
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return ErrRenameDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      oldPath,
				Destination: newPath,
				Error:       err,
			})
	}

	return nil
}

// DirectoryEntry represents a file or subdirectory in a directory
type DirectoryEntry struct {
	Name    string
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime string
	IsDir   bool
}

// ListDirectory returns entries in a directory
func ListDirectory(path string, options ...DirectoryOption) ([]DirectoryEntry, error) {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	if !DirectoryExist(path) {
		return nil, ErrDirectoryNotExist.
			SetData(pathErrorContext{
				Path:  path,
				Error: os.ErrNotExist,
			})
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, ErrReadDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	var result []DirectoryEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		dirEntry := DirectoryEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
		}

		result = append(result, dirEntry)

		// If recursive and it's a directory, list its contents
		if opts.recursive && entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			subEntries, err := ListDirectory(subPath, options...)
			if err == nil {
				result = append(result, subEntries...)
			}
		}
	}

	return result, nil
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

// GetDirectoryInfo returns detailed directory information
func GetDirectoryInfo(path string) (*DirectoryInfo, error) {
	if !DirectoryExist(path) {
		return nil, ErrDirectoryNotExist.
			SetData(pathErrorContext{
				Path:  path,
				Error: os.ErrNotExist,
			})
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, ErrStatDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	if !info.IsDir() {
		return nil, ErrNotDirectory.
			SetData(pathErrorContext{
				Path:  path,
				Error: nil,
			})
	}

	dirInfo := &DirectoryInfo{
		Path:    path,
		Mode:    info.Mode(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}

	// Calculate size and count files/dirs
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			if p != path { // Don't count the root directory itself
				dirInfo.DirCount++
			}
		} else {
			dirInfo.FileCount++
			dirInfo.TotalSize += info.Size()
		}

		return nil
	})

	return dirInfo, nil
}

// ChangeDirectoryPermissions changes directory permissions
func ChangeDirectoryPermissions(path string, mode os.FileMode, options ...DirectoryOption) error {
	opts := defaultDirectoryOptions()
	for _, opt := range options {
		opt(opts)
	}

	if !DirectoryExist(path) {
		return ErrDirectoryNotExist.
			SetData(pathErrorContext{
				Path:  path,
				Error: os.ErrNotExist,
			})
	}

	if opts.recursive {
		// Change permissions recursively
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return os.Chmod(p, mode)
			}
			return nil
		})

		if err != nil {
			return ErrChangeDirectoryPermissions.
				SetError(err).
				SetData(pathErrorContext{
					Path:  path,
					Error: err,
				})
		}
	} else {
		// Change only the specified directory
		if err := os.Chmod(path, mode); err != nil {
			return ErrChangeDirectoryPermissions.
				SetError(err).
				SetData(pathErrorContext{
					Path:  path,
					Error: err,
				})
		}
	}

	return nil
}

// IsEmptyDirectory checks if directory is empty
func IsEmptyDirectory(path string) (bool, error) {
	if !DirectoryExist(path) {
		return false, ErrDirectoryNotExist.
			SetData(pathErrorContext{
				Path:  path,
				Error: os.ErrNotExist,
			})
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, ErrReadDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return len(entries) == 0, nil
}

// ListDirectoryByName returns directory entries sorted by name
func ListDirectoryByName(path string, ascending bool) ([]DirectoryEntry, error) {
	entries, err := ListDirectory(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if ascending {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Name > entries[j].Name
	})

	return entries, nil
}

// ListDirectoryBySize returns directory entries sorted by size
func ListDirectoryBySize(path string, ascending bool) ([]DirectoryEntry, error) {
	entries, err := ListDirectory(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if ascending {
			return entries[i].Size < entries[j].Size
		}
		return entries[i].Size > entries[j].Size
	})

	return entries, nil
}

// ListDirectoryByModTime returns directory entries sorted by modification time
func ListDirectoryByModTime(path string, ascending bool) ([]DirectoryEntry, error) {
	entries, err := ListDirectory(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if ascending {
			return entries[i].ModTime < entries[j].ModTime
		}
		return entries[i].ModTime > entries[j].ModTime
	})

	return entries, nil
}
