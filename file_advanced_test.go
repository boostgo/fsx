package fsx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAdvancedFileOperations(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "fsx_adv_file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("AtomicWriteFile", func(t *testing.T) {
		path := filepath.Join(tmpDir, "atomic.txt")
		content := []byte("atomic content")

		// Write atomically
		if err := AtomicWriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to write file atomically: %v", err)
		}

		// Verify content
		readContent, err := ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !bytes.Equal(readContent, content) {
			t.Error("Content mismatch after atomic write")
		}

		// Test atomic overwrite
		newContent := []byte("new atomic content")
		if err := AtomicWriteFile(path, newContent, 0644); err != nil {
			t.Fatalf("Failed to overwrite file atomically: %v", err)
		}

		readContent, _ = ReadFile(path)
		if !bytes.Equal(readContent, newContent) {
			t.Error("Content mismatch after atomic overwrite")
		}
	})

	t.Run("TempFile", func(t *testing.T) {
		// Create temp file with content
		content := []byte("temp content")
		tempPath, err := CreateTempFile(tmpDir, "test-*.tmp", content)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempPath)

		// Verify it exists and has correct content
		if !FileExist(tempPath) {
			t.Error("Temp file should exist")
		}

		readContent, _ := ReadFile(tempPath)
		if !bytes.Equal(readContent, content) {
			t.Error("Temp file content mismatch")
		}

		// Verify it's in the correct directory
		if filepath.Dir(tempPath) != tmpDir {
			t.Error("Temp file not in specified directory")
		}
	})

	t.Run("TempDirectory", func(t *testing.T) {
		// Create temp directory
		tempDir, err := CreateTempDirectory(tmpDir, "testdir-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Verify it exists
		if !DirectoryExist(tempDir) {
			t.Error("Temp directory should exist")
		}

		// Verify it's in the correct parent directory
		if filepath.Dir(tempDir) != tmpDir {
			t.Error("Temp directory not in specified parent")
		}
	})

	t.Run("FileLock", func(t *testing.T) {
		lockPath := filepath.Join(tmpDir, "locked.txt")

		// Create and lock file
		lock, err := LockFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to lock file: %v", err)
		}

		// Try to lock again (should fail)
		_, err = LockFile(lockPath)
		if err == nil {
			t.Error("Should not be able to lock already locked file")
		}

		// Write to locked file
		data := []byte("locked content")
		if err := lock.Write(data); err != nil {
			t.Fatalf("Failed to write to locked file: %v", err)
		}

		// Unlock
		if err := lock.Unlock(); err != nil {
			t.Fatalf("Failed to unlock file: %v", err)
		}

		// Verify content
		readContent, _ := ReadFile(lockPath)
		if !bytes.Equal(readContent, data) {
			t.Error("Locked file content mismatch")
		}

		// Should be able to lock again after unlock
		lock2, err := LockFile(lockPath)
		if err != nil {
			t.Error("Should be able to lock after unlock")
		}
		lock2.Unlock()
	})

	t.Run("StreamProcessFile", func(t *testing.T) {
		// Create file with multiple lines
		path := filepath.Join(tmpDir, "stream.txt")
		lines := []string{"line 1", "line 2", "line 3"}
		if err := WriteFileLines(path, lines); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Process lines
		processedLines := make([]string, 0)
		lineNumbers := make([]int, 0)

		processor := func(line string, lineNum int) error {
			processedLines = append(processedLines, line)
			lineNumbers = append(lineNumbers, lineNum)
			return nil
		}

		if err := StreamProcessFile(path, processor); err != nil {
			t.Fatalf("Failed to process file: %v", err)
		}

		// Verify processing
		if len(processedLines) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(processedLines))
		}

		for i, line := range processedLines {
			if line != lines[i] {
				t.Errorf("Line %d mismatch: got %s, want %s", i, line, lines[i])
			}
			if lineNumbers[i] != i+1 {
				t.Errorf("Line number mismatch: got %d, want %d", lineNumbers[i], i+1)
			}
		}
	})

	t.Run("StreamCopyWithBuffer", func(t *testing.T) {
		src := filepath.Join(tmpDir, "stream_src.txt")
		dst := filepath.Join(tmpDir, "stream_dst.txt")
		content := []byte("HELLO WORLD")

		// Create source file
		if err := CreateFile(src, content); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy with processing (convert to lowercase)
		processor := func(data []byte) []byte {
			return bytes.ToLower(data)
		}

		if err := StreamCopyWithBuffer(src, dst, 1024, processor); err != nil {
			t.Fatalf("Failed to stream copy: %v", err)
		}

		// Verify processed content
		result, _ := ReadFile(dst)
		expected := bytes.ToLower(content)
		if !bytes.Equal(result, expected) {
			t.Errorf("Processed content mismatch: got %s, want %s", result, expected)
		}
	})

	t.Run("CompressDecompress", func(t *testing.T) {
		original := filepath.Join(tmpDir, "original.txt")
		compressed := filepath.Join(tmpDir, "compressed.gz")
		decompressed := filepath.Join(tmpDir, "decompressed.txt")

		// Create original file with larger content that will compress well
		content := []byte(strings.Repeat("This is test content for compression. ", 100))
		if err := CreateFile(original, content); err != nil {
			t.Fatalf("Failed to create original file: %v", err)
		}

		// Compress
		if err := CompressFile(original, compressed); err != nil {
			t.Fatalf("Failed to compress file: %v", err)
		}

		// Verify compressed file exists
		if !FileExist(compressed) {
			t.Error("Compressed file should exist")
		}

		originalInfo, _ := GetFileInfo(original)
		compressedInfo, _ := GetFileInfo(compressed)

		// Log sizes for debugging
		t.Logf("Original size: %d bytes, Compressed size: %d bytes", originalInfo.Size, compressedInfo.Size)

		// For repetitive content, compressed should be smaller
		if compressedInfo.Size >= originalInfo.Size {
			t.Error("Compressed file should be smaller than original for repetitive content")
		}

		// Decompress
		if err := DecompressFile(compressed, decompressed); err != nil {
			t.Fatalf("Failed to decompress file: %v", err)
		}

		// Verify decompressed content matches original
		decompressedContent, _ := ReadFile(decompressed)
		if !bytes.Equal(decompressedContent, content) {
			t.Error("Decompressed content doesn't match original")
		}
	})

	t.Run("FileChecksum", func(t *testing.T) {
		path := filepath.Join(tmpDir, "checksum.txt")
		content := []byte("checksum test content")

		// Create file
		if err := CreateFile(path, content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Calculate different checksums
		md5sum, err := CalculateFileChecksum(path, HashMD5)
		if err != nil {
			t.Fatalf("Failed to calculate MD5: %v", err)
		}

		sha1sum, err := CalculateFileChecksum(path, HashSHA1)
		if err != nil {
			t.Fatalf("Failed to calculate SHA1: %v", err)
		}

		sha256sum, err := CalculateFileChecksum(path, HashSHA256)
		if err != nil {
			t.Fatalf("Failed to calculate SHA256: %v", err)
		}

		// Verify checksums are different
		if md5sum == sha1sum || sha1sum == sha256sum || md5sum == sha256sum {
			t.Error("Different hash algorithms should produce different results")
		}

		// Verify checksum
		valid, err := VerifyFileChecksum(path, md5sum, HashMD5)
		if err != nil {
			t.Fatalf("Failed to verify checksum: %v", err)
		}
		if !valid {
			t.Error("Checksum verification should pass")
		}

		// Verify with wrong checksum
		valid, _ = VerifyFileChecksum(path, "wrongchecksum", HashMD5)
		if valid {
			t.Error("Wrong checksum should not verify")
		}
	})

	t.Run("ZipArchive", func(t *testing.T) {
		// Create test files
		file1 := filepath.Join(tmpDir, "zip1.txt")
		file2 := filepath.Join(tmpDir, "zip2.txt")
		if err := WriteFileString(file1, "content 1"); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := WriteFileString(file2, "content 2"); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Create zip archive
		zipPath := filepath.Join(tmpDir, "archive.zip")
		if err := CreateZipArchive(zipPath, []string{file1, file2}); err != nil {
			t.Fatalf("Failed to create zip archive: %v", err)
		}

		// Verify zip exists
		if !FileExist(zipPath) {
			t.Error("Zip archive should exist")
		}

		// Extract to new directory
		extractDir := filepath.Join(tmpDir, "extracted")
		if err := ExtractZipArchive(zipPath, extractDir); err != nil {
			t.Fatalf("Failed to extract zip archive: %v", err)
		}

		// Verify extracted files
		extracted1 := filepath.Join(extractDir, "zip1.txt")
		extracted2 := filepath.Join(extractDir, "zip2.txt")

		if !FileExist(extracted1) || !FileExist(extracted2) {
			t.Error("Extracted files should exist")
		}

		// Verify content
		content1, _ := ReadFileString(extracted1)
		content2, _ := ReadFileString(extracted2)

		if content1 != "content 1" || content2 != "content 2" {
			t.Error("Extracted content mismatch")
		}
	})

	t.Run("SplitMergeFiles", func(t *testing.T) {
		// Create a file to split
		originalPath := filepath.Join(tmpDir, "tosplit.txt")
		content := strings.Repeat("Hello World! ", 100) // Create larger content
		if err := WriteFileString(originalPath, content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Split file into 100-byte chunks
		chunks, err := SplitFile(originalPath, 100)
		if err != nil {
			t.Fatalf("Failed to split file: %v", err)
		}
		defer func() {
			// Clean up chunks
			for _, chunk := range chunks {
				os.Remove(chunk)
			}
		}()

		// Verify chunks were created
		if len(chunks) == 0 {
			t.Error("Should have created at least one chunk")
		}

		// Verify each chunk exists
		for _, chunk := range chunks {
			if !FileExist(chunk) {
				t.Errorf("Chunk %s should exist", chunk)
			}
		}

		// Merge chunks back
		mergedPath := filepath.Join(tmpDir, "merged.txt")
		if err := MergeFiles(chunks, mergedPath); err != nil {
			t.Fatalf("Failed to merge files: %v", err)
		}

		// Verify merged content matches original
		mergedContent, _ := ReadFileString(mergedPath)
		if mergedContent != content {
			t.Error("Merged content doesn't match original")
		}
	})

	t.Run("ConcurrentFileLocks", func(t *testing.T) {
		lockPath := filepath.Join(tmpDir, "concurrent.txt")

		// Test concurrent lock attempts
		var wg sync.WaitGroup
		errors := make([]error, 10)
		locks := make([]*FileLock, 10)

		// Try to acquire 10 locks concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				lock, err := LockFile(lockPath)
				errors[index] = err
				locks[index] = lock
			}(i)
		}

		wg.Wait()

		// Only one should succeed
		successCount := 0
		var successfulLock *FileLock
		for i, err := range errors {
			if err == nil {
				successCount++
				successfulLock = locks[i]
			}
		}

		if successCount != 1 {
			t.Errorf("Expected exactly 1 successful lock, got %d", successCount)
		}

		// Unlock the successful lock
		if successfulLock != nil {
			successfulLock.Unlock()
		}
	})

	t.Run("LargeFileStream", func(t *testing.T) {
		// Create a large file
		largePath := filepath.Join(tmpDir, "large.txt")
		var content strings.Builder
		for i := 0; i < 10000; i++ {
			content.WriteString(fmt.Sprintf("Line %d: This is a test line for streaming operations\n", i))
		}

		if err := WriteFileString(largePath, content.String()); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		// Process with streaming
		lineCount := 0
		processor := func(line string, lineNum int) error {
			lineCount++
			// Verify line format
			if !strings.HasPrefix(line, "Line ") {
				return fmt.Errorf("unexpected line format at %d", lineNum)
			}
			return nil
		}

		if err := StreamProcessFile(largePath, processor); err != nil {
			t.Fatalf("Failed to stream process large file: %v", err)
		}

		if lineCount != 10000 {
			t.Errorf("Expected 10000 lines, got %d", lineCount)
		}
	})
}
