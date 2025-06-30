package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdvancedDirectoryOperations(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "fsx_adv_dir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("CopyDirectory", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "copy_src")
		dstDir := filepath.Join(tmpDir, "copy_dst")

		// Create source structure
		if err := CreateDirectories(filepath.Join(srcDir, "subdir1", "subdir2")); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file1.txt"), []byte("content1")); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "subdir1", "file2.txt"), []byte("content2")); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "subdir1", "subdir2", "file3.txt"), []byte("content3")); err != nil {
			t.Fatalf("Failed to create file3: %v", err)
		}

		// Copy directory
		if err := CopyDirectory(srcDir, dstDir); err != nil {
			t.Fatalf("Failed to copy directory: %v", err)
		}

		// Verify structure
		if !FileExist(filepath.Join(dstDir, "file1.txt")) {
			t.Error("file1.txt should exist in destination")
		}
		if !FileExist(filepath.Join(dstDir, "subdir1", "file2.txt")) {
			t.Error("file2.txt should exist in destination")
		}
		if !FileExist(filepath.Join(dstDir, "subdir1", "subdir2", "file3.txt")) {
			t.Error("file3.txt should exist in destination")
		}

		// Verify content
		content, err := ReadFileString(filepath.Join(dstDir, "file1.txt"))
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}
		if content != "content1" {
			t.Error("Content mismatch in copied file")
		}
	})

	t.Run("CopyDirectoryWithFilter", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "filter_src")
		dstDir := filepath.Join(tmpDir, "filter_dst")

		// Create source with various files
		if err := CreateDirectory(srcDir); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "keep.txt"), []byte("keep")); err != nil {
			t.Fatalf("Failed to create keep.txt: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "skip.log"), []byte("skip")); err != nil {
			t.Fatalf("Failed to create skip.log: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "keep.go"), []byte("keep")); err != nil {
			t.Fatalf("Failed to create keep.go: %v", err)
		}

		// Copy with filter (skip .log files)
		filter := func(path string, info os.FileInfo) bool {
			return !strings.HasSuffix(path, ".log")
		}

		if err := CopyDirectory(srcDir, dstDir, WithFilter(filter)); err != nil {
			t.Fatalf("Failed to copy directory with filter: %v", err)
		}

		// Verify filtered copy
		if !FileExist(filepath.Join(dstDir, "keep.txt")) {
			t.Error("keep.txt should exist")
		}
		if !FileExist(filepath.Join(dstDir, "keep.go")) {
			t.Error("keep.go should exist")
		}
		if FileExist(filepath.Join(dstDir, "skip.log")) {
			t.Error("skip.log should not exist (filtered out)")
		}
	})

	t.Run("CopyDirectoryWithProgress", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "progress_src")
		dstDir := filepath.Join(tmpDir, "progress_dst")

		// Create source
		if err := CreateDirectory(srcDir); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file1.txt"), []byte("12345")); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file2.txt"), []byte("1234567890")); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Track progress
		var lastProgress int64
		progressHandler := func(current, total int64, currentFile string) {
			if current > lastProgress {
				lastProgress = current
			}
		}

		if err := CopyDirectory(srcDir, dstDir, WithProgress(progressHandler)); err != nil {
			t.Fatalf("Failed to copy directory with progress: %v", err)
		}

		// Verify progress was tracked
		if lastProgress != 15 {
			t.Errorf("Expected total progress 15, got %d", lastProgress)
		}
	})

	t.Run("SyncDirectories", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "sync_src")
		dstDir := filepath.Join(tmpDir, "sync_dst")

		// Create source
		if err := CreateDirectory(srcDir); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file1.txt"), []byte("sync1")); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file2.txt"), []byte("sync2")); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Create destination with extra file
		if err := CreateDirectory(dstDir); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}
		if err := CreateFile(filepath.Join(dstDir, "file1.txt"), []byte("old")); err != nil {
			t.Fatalf("Failed to create old file1: %v", err)
		}
		if err := CreateFile(filepath.Join(dstDir, "extra.txt"), []byte("remove")); err != nil {
			t.Fatalf("Failed to create extra file: %v", err)
		}

		// Sync directories
		if err := SyncDirectories(srcDir, dstDir); err != nil {
			t.Fatalf("Failed to sync directories: %v", err)
		}

		// Verify sync
		content, err := ReadFileString(filepath.Join(dstDir, "file1.txt"))
		if err != nil {
			t.Fatalf("Failed to read synced file: %v", err)
		}
		if content != "sync1" {
			t.Error("file1.txt should be updated")
		}
		if !FileExist(filepath.Join(dstDir, "file2.txt")) {
			t.Error("file2.txt should exist")
		}
		if FileExist(filepath.Join(dstDir, "extra.txt")) {
			t.Error("extra.txt should be removed")
		}
	})

	t.Run("CompareDirectories", func(t *testing.T) {
		leftDir := filepath.Join(tmpDir, "compare_left")
		rightDir := filepath.Join(tmpDir, "compare_right")

		// Create left directory
		if err := CreateDirectory(leftDir); err != nil {
			t.Fatalf("Failed to create left directory: %v", err)
		}
		if err := CreateFile(filepath.Join(leftDir, "same.txt"), []byte("same content")); err != nil {
			t.Fatalf("Failed to create same.txt in left: %v", err)
		}
		if err := CreateFile(filepath.Join(leftDir, "modified.txt"), []byte("left content")); err != nil {
			t.Fatalf("Failed to create modified.txt in left: %v", err)
		}
		if err := CreateFile(filepath.Join(leftDir, "removed.txt"), []byte("only in left")); err != nil {
			t.Fatalf("Failed to create removed.txt: %v", err)
		}

		// Create right directory
		if err := CreateDirectory(rightDir); err != nil {
			t.Fatalf("Failed to create right directory: %v", err)
		}
		if err := CreateFile(filepath.Join(rightDir, "same.txt"), []byte("same content")); err != nil {
			t.Fatalf("Failed to create same.txt in right: %v", err)
		}
		if err := CreateFile(filepath.Join(rightDir, "modified.txt"), []byte("right content")); err != nil {
			t.Fatalf("Failed to create modified.txt in right: %v", err)
		}
		if err := CreateFile(filepath.Join(rightDir, "added.txt"), []byte("only in right")); err != nil {
			t.Fatalf("Failed to create added.txt: %v", err)
		}

		// Compare directories
		differences, err := CompareDirectories(leftDir, rightDir)
		if err != nil {
			t.Fatalf("Failed to compare directories: %v", err)
		}

		// Analyze differences
		var added, removed, modified, same int
		for _, diff := range differences {
			switch diff.Type {
			case DiffAdded:
				added++
			case DiffRemoved:
				removed++
			case DiffModified:
				modified++
			case DiffSame:
				same++
			}
		}

		if added != 1 {
			t.Errorf("Expected 1 added file, got %d", added)
		}
		if removed != 1 {
			t.Errorf("Expected 1 removed file, got %d", removed)
		}
		if modified != 1 {
			t.Errorf("Expected 1 modified file, got %d", modified)
		}
		if same < 1 {
			t.Errorf("Expected at least 1 same entry, got %d", same)
		}
	})

	t.Run("WalkDirectory", func(t *testing.T) {
		walkDir := filepath.Join(tmpDir, "walk_test")

		// Create structure
		if err := CreateDirectories(filepath.Join(walkDir, "sub")); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}
		if err := CreateFile(filepath.Join(walkDir, "file1.txt"), []byte("content")); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := CreateFile(filepath.Join(walkDir, "sub", "file2.txt"), []byte("content")); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Walk and count
		var fileCount, dirCount int
		walkFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				dirCount++
			} else {
				fileCount++
			}
			return nil
		}

		if err := WalkDirectory(walkDir, walkFn); err != nil {
			t.Fatalf("Failed to walk directory: %v", err)
		}

		if fileCount != 2 {
			t.Errorf("Expected 2 files, got %d", fileCount)
		}
		if dirCount != 2 {
			t.Errorf("Expected 2 directories, got %d", dirCount)
		}
	})

	t.Run("CalculateDirectorySize", func(t *testing.T) {
		sizeDir := filepath.Join(tmpDir, "size_test")

		// Create files with known sizes
		if err := CreateDirectory(sizeDir); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(sizeDir, "file1.txt"), []byte("12345")); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := CreateFile(filepath.Join(sizeDir, "file2.txt"), []byte("1234567890")); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}
		if err := CreateDirectory(filepath.Join(sizeDir, "sub")); err != nil {
			t.Fatalf("Failed to create sub directory: %v", err)
		}
		if err := CreateFile(filepath.Join(sizeDir, "sub", "file3.txt"), []byte("12345")); err != nil {
			t.Fatalf("Failed to create file3: %v", err)
		}

		// Calculate size
		size, err := CalculateDirectorySize(sizeDir)
		if err != nil {
			t.Fatalf("Failed to calculate directory size: %v", err)
		}

		if size != 20 {
			t.Errorf("Expected total size 20, got %d", size)
		}
	})

	t.Run("DirectoryChecksum", func(t *testing.T) {
		checksumDir1 := filepath.Join(tmpDir, "checksum1")
		checksumDir2 := filepath.Join(tmpDir, "checksum2")
		checksumDir3 := filepath.Join(tmpDir, "checksum3")

		// Create identical directories
		for _, dir := range []string{checksumDir1, checksumDir2} {
			if err := CreateDirectory(dir); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
			if err := CreateFile(filepath.Join(dir, "file.txt"), []byte("same content")); err != nil {
				t.Fatalf("Failed to create file in %s: %v", dir, err)
			}
		}

		// Create different directory
		if err := CreateDirectory(checksumDir3); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(checksumDir3, "file.txt"), []byte("different content")); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Calculate checksums
		checksum1, err := DirectoryChecksum(checksumDir1)
		if err != nil {
			t.Fatalf("Failed to calculate checksum1: %v", err)
		}

		checksum2, err := DirectoryChecksum(checksumDir2)
		if err != nil {
			t.Fatalf("Failed to calculate checksum2: %v", err)
		}

		checksum3, err := DirectoryChecksum(checksumDir3)
		if err != nil {
			t.Fatalf("Failed to calculate checksum3: %v", err)
		}

		// Verify checksums
		if checksum1 != checksum2 {
			t.Error("Identical directories should have same checksum")
		}
		if checksum1 == checksum3 {
			t.Error("Different directories should have different checksums")
		}
	})

	t.Run("FindDuplicateFiles", func(t *testing.T) {
		dupDir := filepath.Join(tmpDir, "duplicates")

		// Create files with duplicates
		if err := CreateDirectory(dupDir); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(dupDir, "unique1.txt"), []byte("unique content 1")); err != nil {
			t.Fatalf("Failed to create unique1: %v", err)
		}
		if err := CreateFile(filepath.Join(dupDir, "unique2.txt"), []byte("unique content 2")); err != nil {
			t.Fatalf("Failed to create unique2: %v", err)
		}
		if err := CreateFile(filepath.Join(dupDir, "dup1.txt"), []byte("duplicate content")); err != nil {
			t.Fatalf("Failed to create dup1: %v", err)
		}
		if err := CreateFile(filepath.Join(dupDir, "dup2.txt"), []byte("duplicate content")); err != nil {
			t.Fatalf("Failed to create dup2: %v", err)
		}
		if err := CreateDirectory(filepath.Join(dupDir, "sub")); err != nil {
			t.Fatalf("Failed to create sub directory: %v", err)
		}
		if err := CreateFile(filepath.Join(dupDir, "sub", "dup3.txt"), []byte("duplicate content")); err != nil {
			t.Fatalf("Failed to create dup3: %v", err)
		}

		// Find duplicates
		duplicates, err := FindDuplicateFiles(dupDir)
		if err != nil {
			t.Fatalf("Failed to find duplicate files: %v", err)
		}

		// Should have one group of duplicates with 3 files
		if len(duplicates) != 1 {
			t.Errorf("Expected 1 duplicate group, got %d", len(duplicates))
		}

		// Check the duplicate group
		for _, files := range duplicates {
			if len(files) != 3 {
				t.Errorf("Expected 3 duplicate files, got %d", len(files))
			}
		}
	})

	t.Run("CleanEmptyDirectories", func(t *testing.T) {
		cleanDir := filepath.Join(tmpDir, "clean_test")

		// Create structure with empty directories
		if err := CreateDirectories(filepath.Join(cleanDir, "empty1", "empty2")); err != nil {
			t.Fatalf("Failed to create empty directories: %v", err)
		}
		if err := CreateDirectories(filepath.Join(cleanDir, "nonempty", "empty3")); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}
		if err := CreateFile(filepath.Join(cleanDir, "nonempty", "file.txt"), []byte("content")); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := CreateDirectories(filepath.Join(cleanDir, "empty4")); err != nil {
			t.Fatalf("Failed to create empty4: %v", err)
		}

		// Clean empty directories
		if err := CleanEmptyDirectories(cleanDir); err != nil {
			t.Fatalf("Failed to clean empty directories: %v", err)
		}

		// Verify empty directories are removed
		if DirectoryExist(filepath.Join(cleanDir, "empty1")) {
			t.Error("empty1 should be removed")
		}
		if DirectoryExist(filepath.Join(cleanDir, "empty4")) {
			t.Error("empty4 should be removed")
		}
		if DirectoryExist(filepath.Join(cleanDir, "nonempty", "empty3")) {
			t.Error("empty3 should be removed")
		}

		// Verify non-empty directories remain
		if !DirectoryExist(filepath.Join(cleanDir, "nonempty")) {
			t.Error("nonempty should still exist")
		}
		if !FileExist(filepath.Join(cleanDir, "nonempty", "file.txt")) {
			t.Error("file.txt should still exist")
		}
	})

	t.Run("CopyDirectoryOverwrite", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "overwrite_src")
		dstDir := filepath.Join(tmpDir, "overwrite_dst")

		// Create source
		if err := CreateDirectory(srcDir); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "file.txt"), []byte("new content")); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Create destination with existing file
		if err := CreateDirectory(dstDir); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}
		if err := CreateFile(filepath.Join(dstDir, "file.txt"), []byte("old content")); err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}

		// Try copy without overwrite (should fail)
		err := CopyDirectory(srcDir, dstDir)
		if err == nil {
			t.Error("Copy without overwrite should fail when destination exists")
		}

		// Copy with overwrite
		if err := CopyDirectory(srcDir, dstDir, WithOverwrite()); err != nil {
			t.Fatalf("Failed to copy with overwrite: %v", err)
		}

		// Verify content was overwritten
		content, err := ReadFileString(filepath.Join(dstDir, "file.txt"))
		if err != nil {
			t.Fatalf("Failed to read overwritten file: %v", err)
		}
		if content != "new content" {
			t.Error("File should be overwritten")
		}
	})

	t.Run("CopyDirectoryPreserveAttributes", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "preserve_src")
		dstDir := filepath.Join(tmpDir, "preserve_dst")

		// Create source with specific permissions
		if err := CreateDirectory(srcDir, WithDirPermissions(0700)); err != nil {
			t.Fatalf("Failed to create directory with permissions: %v", err)
		}
		srcFile := filepath.Join(srcDir, "file.txt")
		if err := CreateFile(srcFile, []byte("content"), WithPermissions(0600)); err != nil {
			t.Fatalf("Failed to create file with permissions: %v", err)
		}

		// Set specific modification time
		oldTime := time.Now().Add(-24 * time.Hour)
		if err := os.Chtimes(srcFile, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to set modification time: %v", err)
		}

		// Copy with preserve options
		if err := CopyDirectory(srcDir, dstDir,
			WithPreservePermissions(true),
			WithPreserveTimes(true)); err != nil {
			t.Fatalf("Failed to copy with preserve: %v", err)
		}

		// Verify permissions preserved
		srcInfo, err := os.Stat(srcFile)
		if err != nil {
			t.Fatalf("Failed to stat source file: %v", err)
		}
		dstInfo, err := os.Stat(filepath.Join(dstDir, "file.txt"))
		if err != nil {
			t.Fatalf("Failed to stat destination file: %v", err)
		}

		if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
			t.Error("Permissions not preserved")
		}

		// Verify times preserved (within 1 second tolerance)
		timeDiff := srcInfo.ModTime().Sub(dstInfo.ModTime())
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Error("Modification time not preserved")
		}
	})

	t.Run("CopyDirectorySkipErrors", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "skip_errors_src")
		dstDir := filepath.Join(tmpDir, "skip_errors_dst")

		// Create source
		if err := CreateDirectory(srcDir); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "good1.txt"), []byte("content1")); err != nil {
			t.Fatalf("Failed to create good1: %v", err)
		}
		if err := CreateFile(filepath.Join(srcDir, "good2.txt"), []byte("content2")); err != nil {
			t.Fatalf("Failed to create good2: %v", err)
		}

		// Create a file that will cause an error when read
		badFile := filepath.Join(srcDir, "bad.txt")
		if err := CreateFile(badFile, []byte("content")); err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}
		if err := os.Chmod(badFile, 0000); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}

		// Copy with skip errors
		if err := CopyDirectory(srcDir, dstDir, WithSkipErrors()); err != nil {
			t.Fatalf("Copy with skip errors should not fail: %v", err)
		}

		// Verify good files were copied
		if !FileExist(filepath.Join(dstDir, "good1.txt")) {
			t.Error("good1.txt should be copied")
		}
		if !FileExist(filepath.Join(dstDir, "good2.txt")) {
			t.Error("good2.txt should be copied")
		}

		// Clean up permissions
		os.Chmod(badFile, 0644)
	})
}
