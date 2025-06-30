package fsx

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilterFunc is used to filter files/directories during operations
type FilterFunc func(path string, info os.FileInfo) bool

// ProgressFunc is called to report progress during operations
type ProgressFunc func(current, total int64, currentFile string)

// WalkFunc is called for each file/directory during tree walk
type WalkFunc func(path string, info os.FileInfo, err error) error

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

// CopyDirectory copies entire directory tree from source to destination
func CopyDirectory(src, dst string, options ...CopyOption) error {
	opts := defaultCopyOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Validate source
	srcInfo, err := os.Stat(src)
	if err != nil {
		return ErrCopyDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	if !srcInfo.IsDir() {
		return ErrSourceNotDirectory.
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       nil,
			})
	}

	// Check destination
	if !opts.overwrite && DirectoryExist(dst) {
		return ErrDestinationExists.
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       nil,
			})
	}

	// Calculate total size for progress
	var totalSize, copiedSize int64
	if opts.progressHandler != nil {
		totalSize, _ = CalculateDirectorySize(src)
	}

	// Create destination directory
	if err := CreateDirectories(dst); err != nil {
		return err
	}

	// Copy directory attributes
	if opts.preservePerms {
		_ = os.Chmod(dst, srcInfo.Mode())
	}

	// Walk through source directory
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if opts.skipErrors {
				return nil
			}
			return err
		}

		// Apply filter if provided
		if opts.filter != nil && !opts.filter(path, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			if !opts.followSymlinks {
				// Copy symlink as-is
				link, err := os.Readlink(path)
				if err != nil {
					if opts.skipErrors {
						return nil
					}
					return err
				}
				return os.Symlink(link, dstPath)
			}
			// If following symlinks, continue to copy the target
		}

		// Copy based on type
		if info.IsDir() {
			// Create directory
			if err := CreateDirectory(dstPath); err != nil {
				if opts.skipErrors {
					return nil
				}
				return err
			}

			// Preserve directory attributes
			if opts.preservePerms {
				os.Chmod(dstPath, info.Mode())
			}
			if opts.preserveTimes {
				os.Chtimes(dstPath, info.ModTime(), info.ModTime())
			}
		} else {
			// Copy file
			if err := copyFileWithOptions(path, dstPath, info, opts); err != nil {
				if opts.skipErrors {
					return nil
				}
				return err
			}

			// Update progress
			if opts.progressHandler != nil {
				copiedSize += info.Size()
				opts.progressHandler(copiedSize, totalSize, path)
			}
		}

		return nil
	})

	if err != nil {
		return ErrCopyDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	return nil
}

// copyFileWithOptions is a helper to copy files with options
func copyFileWithOptions(src, dst string, srcInfo os.FileInfo, opts *copyOptions) error {
	// Check if destination exists
	if !opts.overwrite && FileExist(dst) {
		return nil
	}

	// Open source
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve attributes
	if opts.preservePerms {
		os.Chmod(dst, srcInfo.Mode())
	}
	if opts.preserveTimes {
		os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
	}

	return nil
}

// SyncDirectories synchronizes source directory to destination
func SyncDirectories(src, dst string, options ...CopyOption) error {
	// Create options with overwrite enabled by default for sync
	syncOptions := append([]CopyOption{WithOverwrite()}, options...)

	// First, copy all from source to destination
	if err := CopyDirectory(src, dst, syncOptions...); err != nil {
		return ErrSyncDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	// Then, remove files from destination that don't exist in source
	srcFiles := make(map[string]bool)

	// Collect all source files
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		srcFiles[relPath] = true
		return nil
	})

	if err != nil {
		return ErrSyncDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	// Remove extra files from destination
	err = filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dst, path)
		if err != nil {
			return err
		}

		if !srcFiles[relPath] {
			// File doesn't exist in source, remove it
			if info.IsDir() {
				return DeleteDirectory(path, WithForce())
			}
			return DeleteFile(path)
		}

		return nil
	})

	if err != nil {
		return ErrSyncDirectory.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	return nil
}

// CompareDirectories compares two directories and returns differences
func CompareDirectories(left, right string) ([]Difference, error) {
	if !DirectoryExist(left) || !DirectoryExist(right) {
		return nil, ErrCompareDirectory.
			SetData(struct {
				Left  string `json:"left"`
				Right string `json:"right"`
			}{
				Left:  left,
				Right: right,
			})
	}

	leftFiles := make(map[string]os.FileInfo)
	rightFiles := make(map[string]os.FileInfo)
	var differences []Difference

	// Collect files from left directory
	err := filepath.Walk(left, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(left, path)
		if err != nil {
			return err
		}

		leftFiles[relPath] = info
		return nil
	})

	if err != nil {
		return nil, ErrCompareDirectory.SetError(err)
	}

	// Collect files from right directory
	err = filepath.Walk(right, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(right, path)
		if err != nil {
			return err
		}

		rightFiles[relPath] = info
		return nil
	})

	if err != nil {
		return nil, ErrCompareDirectory.SetError(err)
	}

	// Compare files
	for path, leftInfo := range leftFiles {
		if rightInfo, exists := rightFiles[path]; exists {
			// File exists in both, check if modified
			if leftInfo.IsDir() == rightInfo.IsDir() {
				if !leftInfo.IsDir() {
					// Compare file content by size and modification time
					// For more accuracy, could compare checksums
					if leftInfo.Size() != rightInfo.Size() ||
						leftInfo.ModTime().Unix() != rightInfo.ModTime().Unix() {
						differences = append(differences, Difference{
							Path:      path,
							Type:      DiffModified,
							LeftInfo:  leftInfo,
							RightInfo: rightInfo,
						})
					} else {
						differences = append(differences, Difference{
							Path:      path,
							Type:      DiffSame,
							LeftInfo:  leftInfo,
							RightInfo: rightInfo,
						})
					}
				}
			} else {
				// Type changed (file <-> directory)
				differences = append(differences, Difference{
					Path:      path,
					Type:      DiffModified,
					LeftInfo:  leftInfo,
					RightInfo: rightInfo,
				})
			}
		} else {
			// File only in left (removed from right)
			differences = append(differences, Difference{
				Path:     path,
				Type:     DiffRemoved,
				LeftInfo: leftInfo,
			})
		}
	}

	// Check for files only in right (added)
	for path, rightInfo := range rightFiles {
		if _, exists := leftFiles[path]; !exists {
			differences = append(differences, Difference{
				Path:      path,
				Type:      DiffAdded,
				RightInfo: rightInfo,
			})
		}
	}

	return differences, nil
}

// WalkDirectory walks through directory tree with custom function
func WalkDirectory(root string, walkFn WalkFunc) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return walkFn(path, info, err)
	})

	if err != nil {
		return ErrWalkDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return nil
}

// CalculateDirectorySize calculates total size of directory
func CalculateDirectorySize(path string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, ErrCalculateSize.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return totalSize, nil
}

// DirectoryChecksum calculates checksum of all files in directory
func DirectoryChecksum(path string) (string, error) {
	hash := md5.New()

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Include file path in hash
		relPath, _ := filepath.Rel(path, filePath)
		hash.Write([]byte(relPath))

		if !info.IsDir() {
			// Include file content in hash
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(hash, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", ErrWalkDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// FindDuplicateFiles finds duplicate files in directory based on content
func FindDuplicateFiles(root string) (map[string][]string, error) {
	fileHashes := make(map[string][]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Calculate file hash
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			hash := md5.New()
			if _, err := io.Copy(hash, file); err != nil {
				return err
			}

			hashStr := hex.EncodeToString(hash.Sum(nil))
			fileHashes[hashStr] = append(fileHashes[hashStr], path)
		}

		return nil
	})

	if err != nil {
		return nil, ErrWalkDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	// Filter out unique files
	duplicates := make(map[string][]string)
	for hash, files := range fileHashes {
		if len(files) > 1 {
			duplicates[hash] = files
		}
	}

	return duplicates, nil
}

// CleanEmptyDirectories removes all empty directories recursively
func CleanEmptyDirectories(root string) error {
	// First pass: collect all directories
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != root {
			dirs = append(dirs, path)
		}

		return nil
	})

	if err != nil {
		return ErrWalkDirectory.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	// Sort dirs by depth (deepest first)
	for i := 0; i < len(dirs)-1; i++ {
		for j := i + 1; j < len(dirs); j++ {
			if strings.Count(dirs[i], string(os.PathSeparator)) < strings.Count(dirs[j], string(os.PathSeparator)) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	// Remove empty directories
	for _, dir := range dirs {
		empty, err := IsEmptyDirectory(dir)
		if err != nil {
			continue
		}

		if empty {
			DeleteDirectory(dir)
		}
	}

	return nil
}
