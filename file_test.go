package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileOperations(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "fsx_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("CreateAndReadFile", func(t *testing.T) {
		path := filepath.Join(tmpDir, "test.txt")
		content := []byte("Hello, FSX!")

		// Create file
		if err := CreateFile(path, content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Read file
		readContent, err := ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(readContent) != string(content) {
			t.Errorf("Content mismatch: got %s, want %s", readContent, content)
		}
	})

	t.Run("CreateFileWithDirs", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nested", "dirs", "file.txt")
		content := []byte("Nested content")

		// Create file with directories
		if err := CreateFile(path, content, WithCreateDirs()); err != nil {
			t.Fatalf("Failed to create file with dirs: %v", err)
		}

		// Verify file exists
		if !FileExist(path) {
			t.Error("File should exist")
		}
	})

	t.Run("ReadFileLines", func(t *testing.T) {
		path := filepath.Join(tmpDir, "lines.txt")
		lines := []string{"Line 1", "Line 2", "Line 3"}

		// Write lines
		if err := WriteFileLines(path, lines); err != nil {
			t.Fatalf("Failed to write lines: %v", err)
		}

		// Read lines
		readLines, err := ReadFileLines(path)
		if err != nil {
			t.Fatalf("Failed to read lines: %v", err)
		}

		if len(readLines) != len(lines) {
			t.Errorf("Lines count mismatch: got %d, want %d", len(readLines), len(lines))
		}

		for i, line := range readLines {
			if line != lines[i] {
				t.Errorf("Line %d mismatch: got %s, want %s", i, line, lines[i])
			}
		}
	})

	t.Run("AppendFile", func(t *testing.T) {
		path := filepath.Join(tmpDir, "append.txt")

		// Create initial file
		if err := WriteFileString(path, "Initial content\n"); err != nil {
			t.Fatalf("Failed to write initial content: %v", err)
		}

		// Append content
		if err := AppendFileString(path, "Appended content\n"); err != nil {
			t.Fatalf("Failed to append content: %v", err)
		}

		// Read and verify
		content, err := ReadFileString(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		expected := "Initial content\nAppended content\n"
		if content != expected {
			t.Errorf("Content mismatch: got %s, want %s", content, expected)
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source.txt")
		dst := filepath.Join(tmpDir, "destination.txt")
		content := []byte("Content to copy")

		// Create source file
		if err := CreateFile(src, content); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy file
		if err := CopyFile(src, dst); err != nil {
			t.Fatalf("Failed to copy file: %v", err)
		}

		// Verify both files exist and have same content
		srcContent, _ := ReadFile(src)
		dstContent, _ := ReadFile(dst)

		if string(srcContent) != string(dstContent) {
			t.Error("Copy content mismatch")
		}
	})

	t.Run("MoveFile", func(t *testing.T) {
		src := filepath.Join(tmpDir, "move_source.txt")
		dst := filepath.Join(tmpDir, "move_dest.txt")
		content := []byte("Content to move")

		// Create source file
		if err := CreateFile(src, content); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Move file
		if err := MoveFile(src, dst); err != nil {
			t.Fatalf("Failed to move file: %v", err)
		}

		// Verify source doesn't exist and destination does
		if FileExist(src) {
			t.Error("Source file should not exist after move")
		}

		if !FileExist(dst) {
			t.Error("Destination file should exist after move")
		}

		// Verify content
		movedContent, _ := ReadFile(dst)
		if string(movedContent) != string(content) {
			t.Error("Moved content mismatch")
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		path := filepath.Join(tmpDir, "delete_me.txt")

		// Create file
		if err := CreateFile(path, []byte("Delete me")); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Delete file
		if err := DeleteFile(path); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// Verify file doesn't exist
		if FileExist(path) {
			t.Error("File should not exist after deletion")
		}
	})

	t.Run("FileInfo", func(t *testing.T) {
		path := filepath.Join(tmpDir, "info.txt")
		content := []byte("File for info test")

		// Create file
		if err := CreateFile(path, content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Get file info
		info, err := GetFileInfo(path)
		if err != nil {
			t.Fatalf("Failed to get file info: %v", err)
		}

		if info.Size != int64(len(content)) {
			t.Errorf("Size mismatch: got %d, want %d", info.Size, len(content))
		}

		if info.IsDir {
			t.Error("File should not be directory")
		}
	})

	t.Run("ChangePermissions", func(t *testing.T) {
		path := filepath.Join(tmpDir, "perms.txt")

		// Create file
		if err := CreateFile(path, []byte("Permission test")); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Change permissions
		newMode := os.FileMode(0600)
		if err := ChangeFilePermissions(path, newMode); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}

		// Verify permissions
		info, _ := GetFileInfo(path)
		if info.Mode.Perm() != newMode {
			t.Errorf("Permission mismatch: got %v, want %v", info.Mode.Perm(), newMode)
		}
	})
}
