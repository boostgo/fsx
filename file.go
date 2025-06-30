package fsx

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func FileExist(path string) bool {
	stat, _ := os.Stat(path)
	if stat == nil {
		return false
	}

	return !stat.IsDir()
}

func AnyFileExist(paths ...string) bool {
	if len(paths) == 0 {
		return false
	}

	for _, path := range paths {
		if FileExist(path) {
			return true
		}
	}

	return false
}

// FileOption represents optional parameters for file operations
type FileOption func(*fileOptions)

type fileOptions struct {
	perm       os.FileMode
	createDirs bool
	backup     bool
	bufferSize int
}

// defaultFileOptions returns default options for file operations
func defaultFileOptions() *fileOptions {
	return &fileOptions{
		perm:       0644,
		createDirs: false,
		backup:     false,
		bufferSize: 32 * 1024, // 32KB
	}
}

// WithPermissions sets custom file permissions
func WithPermissions(perm os.FileMode) FileOption {
	return func(opts *fileOptions) {
		opts.perm = perm
	}
}

// WithCreateDirs creates parent directories if they don't exist
func WithCreateDirs() FileOption {
	return func(opts *fileOptions) {
		opts.createDirs = true
	}
}

// WithBackup creates a backup before overwriting
func WithBackup() FileOption {
	return func(opts *fileOptions) {
		opts.backup = true
	}
}

// WithBufferSize sets custom buffer size for operations
func WithBufferSize(size int) FileOption {
	return func(opts *fileOptions) {
		opts.bufferSize = size
	}
}

// CreateFile creates a new file with optional content
func CreateFile(path string, content []byte, options ...FileOption) error {
	opts := defaultFileOptions()
	for _, opt := range options {
		opt(opts)
	}

	if opts.createDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return newCreateFileDirectoriesError(path, err)
		}
	}

	return os.WriteFile(path, content, opts.perm)
}

// ReadFile reads entire file content as bytes
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, newReadFileError(path, err)
	}

	return data, nil
}

// ReadFileString reads entire file content as string
func ReadFileString(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ReadFileLines reads file content as slice of lines
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, newOpenFileError(path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, newReadFileLinesError(path, err)
	}

	return lines, nil
}

// WriteFile writes data to file (overwrites if exists)
func WriteFile(path string, data []byte, options ...FileOption) error {
	opts := defaultFileOptions()
	for _, opt := range options {
		opt(opts)
	}

	if opts.backup && FileExist(path) {
		backupPath := path + ".backup"
		if err := CopyFile(path, backupPath); err != nil {
			return newCreateBackupFileError(path, err)
		}
	}

	if opts.createDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return newCreateDirectories(path, err)
		}
	}

	return os.WriteFile(path, data, opts.perm)
}

// WriteFileString writes string content to file
func WriteFileString(path string, content string, options ...FileOption) error {
	return WriteFile(path, []byte(content), options...)
}

// WriteFileLines writes lines to file
func WriteFileLines(path string, lines []string, options ...FileOption) error {
	content := strings.Join(lines, "\n")
	return WriteFileString(path, content, options...)
}

// AppendFile appends data to existing file
func AppendFile(path string, data []byte, options ...FileOption) error {
	opts := defaultFileOptions()
	for _, opt := range options {
		opt(opts)
	}

	if opts.createDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return newCreateDirectories(path, err)
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, opts.perm)
	if err != nil {
		return newOpenFileError(path, err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return newAppendFile(path, err)
	}

	return nil
}

// AppendFileString appends string to file
func AppendFileString(path string, content string, options ...FileOption) error {
	return AppendFile(path, []byte(content), options...)
}

// DeleteFile removes a file
func DeleteFile(path string) error {
	if !FileExist(path) {
		return nil // Already doesn't exist
	}

	if err := os.Remove(path); err != nil {
		return newDeleteFile(path, err)
	}

	return nil
}

// MoveFile moves/renames a file
func MoveFile(src, dst string, options ...FileOption) error {
	opts := defaultFileOptions()
	for _, opt := range options {
		opt(opts)
	}

	if opts.createDirs {
		dir := filepath.Dir(dst)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return newCreateDirectories(dst, err)
		}
	}

	if opts.backup && FileExist(dst) {
		backupPath := dst + ".backup"
		if err := CopyFile(dst, backupPath); err != nil {
			return newCreateBackupFileError(dst, err)
		}
	}

	if err := os.Rename(src, dst); err != nil {
		// If rename fails (e.g., across filesystems), try copy and delete
		if err := CopyFile(src, dst, options...); err != nil {
			return err
		}

		if err := DeleteFile(src); err != nil {
			return err
		}
	}

	return nil
}

// CopyFile copies file from source to destination
func CopyFile(src, dst string, options ...FileOption) error {
	opts := defaultFileOptions()
	for _, opt := range options {
		opt(opts)
	}

	if opts.createDirs {
		dir := filepath.Dir(dst)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return newCreateDirectories(dst, err)
		}
	}

	if opts.backup && FileExist(dst) {
		backupPath := dst + ".backup"
		if err := CopyFile(dst, backupPath); err != nil {
			return newCreateBackupFileError(dst, err)
		}
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return newOpenFileError(src, err)
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return newStatFile(src, err)
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return newOpenFileError(dst, err)
	}
	defer destFile.Close()

	// Copy with buffer
	buf := make([]byte, opts.bufferSize)
	if _, err := io.CopyBuffer(destFile, sourceFile, buf); err != nil {
		return newCopyFile(dst, err)
	}

	return nil
}

// FileInfo represents file information
type FileInfo struct {
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime string
	IsDir   bool
}

// GetFileInfo returns detailed file information
func GetFileInfo(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, newStatFile(path, err)
	}

	return &FileInfo{
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		IsDir:   info.IsDir(),
	}, nil
}

// ChangeFilePermissions changes file permissions
func ChangeFilePermissions(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return newFailedChangePermissionsError(path, mode, err)
	}

	return nil
}

// TouchFile creates an empty file or updates its modification time
func TouchFile(path string, options ...FileOption) error {
	if FileExist(path) {
		// Update modification time
		currentTime := time.Now()
		return os.Chtimes(path, currentTime, currentTime)
	}

	// Create empty file
	return CreateFile(path, []byte{}, options...)
}
