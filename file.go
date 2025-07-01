package fsx

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Global lock manager to track locks
var (
	lockManager = make(map[string]*FileLock)
	lockMu      sync.RWMutex
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

// AtomicWriteFile writes data to a file atomically
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	tmpPath := tmpFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if FileExist(tmpPath) {
			os.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	// Close temp file
	if err := tmpFile.Close(); err != nil {
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	// Set permissions
	if err := os.Chmod(tmpPath, perm); err != nil {
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return ErrAtomicOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return nil
}

// AtomicWriteFileString writes string data atomically
func AtomicWriteFileString(path string, content string, perm os.FileMode) error {
	return AtomicWriteFile(path, []byte(content), perm)
}

// CreateTempFile creates a temporary file with optional prefix/suffix
func CreateTempFile(dir, pattern string, content []byte) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", ErrTempFile.
			SetError(err).
			SetData(struct {
				Dir     string `json:"dir"`
				Pattern string `json:"pattern"`
				Error   error  `json:"error"`
			}{
				Dir:     dir,
				Pattern: pattern,
				Error:   err,
			})
	}

	path := file.Name()

	if len(content) > 0 {
		if _, err := file.Write(content); err != nil {
			file.Close()
			os.Remove(path)
			return "", ErrTempFile.
				SetError(err).
				SetData(pathErrorContext{
					Path:  path,
					Error: err,
				})
		}
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", ErrTempFile.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return path, nil
}

// CreateTempDirectory creates a temporary directory
func CreateTempDirectory(dir, pattern string) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", ErrTempFile.
			SetError(err).
			SetData(struct {
				Dir     string `json:"dir"`
				Pattern string `json:"pattern"`
				Error   error  `json:"error"`
			}{
				Dir:     dir,
				Pattern: pattern,
				Error:   err,
			})
	}

	return path, nil
}

// LockFile creates an exclusive lock on a file
func LockFile(path string) (*FileLock, error) {
	lockMu.Lock()
	defer lockMu.Unlock()

	// Check if already locked
	if existingLock, exists := lockManager[path]; exists && existingLock.isLocked {
		return nil, ErrFileAlreadyLocked.
			SetData(pathErrorContext{
				Path:  path,
				Error: nil,
			})
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, ErrFileLock.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	// Open file for exclusive access
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, ErrFileLock.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	lock := &FileLock{
		path:     path,
		file:     file,
		isLocked: true,
	}

	lockManager[path] = lock
	return lock, nil
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if !fl.isLocked {
		return ErrFileNotLocked.
			SetData(pathErrorContext{
				Path:  fl.path,
				Error: nil,
			})
	}

	if err := fl.file.Close(); err != nil {
		return ErrFileLock.
			SetError(err).
			SetData(pathErrorContext{
				Path:  fl.path,
				Error: err,
			})
	}

	lockMu.Lock()
	delete(lockManager, fl.path)
	lockMu.Unlock()

	fl.isLocked = false
	return nil
}

// Write writes data to a locked file
func (fl *FileLock) Write(data []byte) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if !fl.isLocked {
		return ErrFileNotLocked.
			SetData(pathErrorContext{
				Path:  fl.path,
				Error: nil,
			})
	}

	// Truncate file before writing
	if err := fl.file.Truncate(0); err != nil {
		return err
	}

	if _, err := fl.file.Seek(0, 0); err != nil {
		return err
	}

	if _, err := fl.file.Write(data); err != nil {
		return err
	}

	return fl.file.Sync()
}

// StreamProcessFunc is a function that processes file content line by line
type StreamProcessFunc func(line string, lineNum int) error

// StreamProcessFile processes a file line by line
func StreamProcessFile(path string, processor StreamProcessFunc) error {
	file, err := os.Open(path)
	if err != nil {
		return ErrStreamOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if err := processor(scanner.Text(), lineNum); err != nil {
			return ErrStreamOperation.
				SetError(err).
				SetData(struct {
					Path    string `json:"path"`
					LineNum int    `json:"line_num"`
					Error   error  `json:"error"`
				}{
					Path:    path,
					LineNum: lineNum,
					Error:   err,
				})
		}
	}

	if err := scanner.Err(); err != nil {
		return ErrStreamOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return nil
}

// StreamCopyWithBuffer copies file with custom buffer and optional processing
func StreamCopyWithBuffer(src, dst string, bufferSize int, processor func([]byte) []byte) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return ErrStreamOperation.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return ErrStreamOperation.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}
	defer dstFile.Close()

	buffer := make([]byte, bufferSize)

	for {
		n, err := srcFile.Read(buffer)
		if err != nil && err != io.EOF {
			return ErrStreamOperation.
				SetError(err).
				SetData(moveErrorContext{
					Source:      src,
					Destination: dst,
					Error:       err,
				})
		}

		if n == 0 {
			break
		}

		data := buffer[:n]
		if processor != nil {
			data = processor(data)
		}

		if _, err := dstFile.Write(data); err != nil {
			return ErrStreamOperation.
				SetError(err).
				SetData(moveErrorContext{
					Source:      src,
					Destination: dst,
					Error:       err,
				})
		}
	}

	return dstFile.Sync()
}

// CompressFile compresses a file using gzip
func CompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  src,
				Error: err,
			})
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  dst,
				Error: err,
			})
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// Set the original filename in gzip header
	gzWriter.Name = filepath.Base(src)

	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		return ErrCompress.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	return nil
}

// DecompressFile decompresses a gzip file
func DecompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  src,
				Error: err,
			})
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  src,
				Error: err,
			})
	}
	defer gzReader.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  dst,
				Error: err,
			})
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, gzReader); err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(moveErrorContext{
				Source:      src,
				Destination: dst,
				Error:       err,
			})
	}

	return nil
}

// CalculateFileChecksum calculates checksum of a file
func CalculateFileChecksum(path string, hashType HashType) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", ErrChecksum.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}
	defer file.Close()

	var h hash.Hash
	switch hashType {
	case HashMD5:
		h = md5.New()
	case HashSHA1:
		h = sha1.New()
	case HashSHA256:
		h = sha256.New()
	default:
		return "", ErrChecksum.
			SetData(struct {
				Path     string   `json:"path"`
				HashType HashType `json:"hash_type"`
			}{
				Path:     path,
				HashType: hashType,
			})
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", ErrChecksum.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyFileChecksum verifies if a file matches the given checksum
func VerifyFileChecksum(path string, expectedChecksum string, hashType HashType) (bool, error) {
	actualChecksum, err := CalculateFileChecksum(path, hashType)
	if err != nil {
		return false, err
	}

	return actualChecksum == expectedChecksum, nil
}

// CreateZipArchive creates a zip archive from files
func CreateZipArchive(zipPath string, files []string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  zipPath,
				Error: err,
			})
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if err := addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}

	return nil
}

// addFileToZip is a helper to add files to zip archive
func addFileToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  filename,
				Error: err,
			})
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  filename,
				Error: err,
			})
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  filename,
				Error: err,
			})
	}

	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return ErrCompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  filename,
				Error: err,
			})
	}

	_, err = io.Copy(writer, file)
	return err
}

// ExtractZipArchive extracts a zip archive
func ExtractZipArchive(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  zipPath,
				Error: err,
			})
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := extractZipFile(file, path); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile is a helper to extract individual files from zip
func extractZipFile(file *zip.File, destPath string) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  destPath,
				Error: err,
			})
	}

	fileReader, err := file.Open()
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  file.Name,
				Error: err,
			})
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return ErrDecompress.
			SetError(err).
			SetData(pathErrorContext{
				Path:  destPath,
				Error: err,
			})
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, fileReader)
	return err
}

// SplitFile splits a large file into smaller chunks
func SplitFile(path string, chunkSize int64) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrStreamOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, ErrStreamOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  path,
				Error: err,
			})
	}

	var chunks []string
	buffer := make([]byte, chunkSize)

	for i := 0; ; i++ {
		chunkPath := fmt.Sprintf("%s.part%d", path, i)
		chunkFile, err := os.Create(chunkPath)
		if err != nil {
			// Clean up created chunks on error
			for _, chunk := range chunks {
				os.Remove(chunk)
			}
			return nil, ErrStreamOperation.
				SetError(err).
				SetData(pathErrorContext{
					Path:  chunkPath,
					Error: err,
				})
		}

		written := int64(0)
		for written < chunkSize {
			toRead := chunkSize - written
			if toRead > int64(len(buffer)) {
				toRead = int64(len(buffer))
			}

			n, err := file.Read(buffer[:toRead])
			if err == io.EOF {
				if written == 0 {
					chunkFile.Close()
					os.Remove(chunkPath)
					return chunks, nil
				}
				break
			}
			if err != nil {
				chunkFile.Close()
				// Clean up
				for _, chunk := range chunks {
					os.Remove(chunk)
				}
				return nil, ErrStreamOperation.
					SetError(err).
					SetData(pathErrorContext{
						Path:  path,
						Error: err,
					})
			}

			if _, err := chunkFile.Write(buffer[:n]); err != nil {
				chunkFile.Close()
				// Clean up
				for _, chunk := range chunks {
					os.Remove(chunk)
				}
				return nil, ErrStreamOperation.
					SetError(err).
					SetData(pathErrorContext{
						Path:  chunkPath,
						Error: err,
					})
			}

			written += int64(n)
		}

		chunkFile.Close()
		chunks = append(chunks, chunkPath)

		// Check if we've read the entire file
		if file, _ := file.Seek(0, 1); file >= fileInfo.Size() {
			break
		}
	}

	return chunks, nil
}

// MergeFiles merges multiple files into one
func MergeFiles(files []string, destPath string) error {
	destFile, err := os.Create(destPath)
	if err != nil {
		return ErrStreamOperation.
			SetError(err).
			SetData(pathErrorContext{
				Path:  destPath,
				Error: err,
			})
	}
	defer destFile.Close()

	for _, file := range files {
		srcFile, err := os.Open(file)
		if err != nil {
			return ErrStreamOperation.
				SetError(err).
				SetData(pathErrorContext{
					Path:  file,
					Error: err,
				})
		}

		if _, err := io.Copy(destFile, srcFile); err != nil {
			srcFile.Close()
			return ErrStreamOperation.
				SetError(err).
				SetData(moveErrorContext{
					Source:      file,
					Destination: destPath,
					Error:       err,
				})
		}

		srcFile.Close()
	}

	return destFile.Sync()
}
