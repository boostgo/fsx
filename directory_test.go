package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryOperations(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "fsx_dir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("CreateDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "testdir")

		// Create directory
		if err := CreateDirectory(dirPath); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Verify it exists
		if !DirectoryExist(dirPath) {
			t.Error("Directory should exist")
		}

		// Try to create again (should not error)
		if err := CreateDirectory(dirPath); err != nil {
			t.Error("Creating existing directory should not error")
		}
	})

	t.Run("CreateDirectories", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "level1", "level2", "level3")

		// Create nested directories
		if err := CreateDirectories(nestedPath); err != nil {
			t.Fatalf("Failed to create nested directories: %v", err)
		}

		// Verify all levels exist
		if !DirectoryExist(filepath.Join(tmpDir, "level1")) {
			t.Error("Level 1 directory should exist")
		}
		if !DirectoryExist(filepath.Join(tmpDir, "level1", "level2")) {
			t.Error("Level 2 directory should exist")
		}
		if !DirectoryExist(nestedPath) {
			t.Error("Level 3 directory should exist")
		}
	})

	t.Run("CreateDirectoryWithPermissions", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "customperms")
		customMode := os.FileMode(0700)

		// Create with custom permissions
		if err := CreateDirectory(dirPath, WithDirPermissions(customMode)); err != nil {
			t.Fatalf("Failed to create directory with custom permissions: %v", err)
		}

		// Verify permissions
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Failed to stat directory: %v", err)
		}

		if info.Mode().Perm() != customMode {
			t.Errorf("Permission mismatch: got %v, want %v", info.Mode().Perm(), customMode)
		}
	})

	t.Run("DeleteEmptyDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "emptydir")

		// Create directory
		CreateDirectory(dirPath)

		// Delete it
		if err := DeleteDirectory(dirPath); err != nil {
			t.Fatalf("Failed to delete empty directory: %v", err)
		}

		// Verify it's gone
		if DirectoryExist(dirPath) {
			t.Error("Directory should not exist after deletion")
		}
	})

	t.Run("DeleteNonEmptyDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "nonemptydir")
		filePath := filepath.Join(dirPath, "file.txt")

		// Create directory with file
		CreateDirectory(dirPath)
		CreateFile(filePath, []byte("content"))

		// Try to delete without force (should fail)
		err := DeleteDirectory(dirPath)
		if err == nil {
			t.Error("Deleting non-empty directory should fail without force")
		}

		// Delete with force
		if err := DeleteDirectory(dirPath, WithForce()); err != nil {
			t.Fatalf("Failed to delete non-empty directory with force: %v", err)
		}

		// Verify it's gone
		if DirectoryExist(dirPath) {
			t.Error("Directory should not exist after force deletion")
		}
	})

	t.Run("RenameDirectory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "oldname")
		newPath := filepath.Join(tmpDir, "newname")

		// Create directory
		CreateDirectory(oldPath)

		// Add some content
		CreateFile(filepath.Join(oldPath, "file.txt"), []byte("content"))

		// Rename it
		if err := RenameDirectory(oldPath, newPath); err != nil {
			t.Fatalf("Failed to rename directory: %v", err)
		}

		// Verify old doesn't exist and new does
		if DirectoryExist(oldPath) {
			t.Error("Old directory should not exist after rename")
		}
		if !DirectoryExist(newPath) {
			t.Error("New directory should exist after rename")
		}

		// Verify content moved
		if !FileExist(filepath.Join(newPath, "file.txt")) {
			t.Error("File should exist in renamed directory")
		}
	})

	t.Run("ListDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "listdir")

		// Create directory with content
		CreateDirectory(dirPath)
		CreateFile(filepath.Join(dirPath, "file1.txt"), []byte("content1"))
		CreateFile(filepath.Join(dirPath, "file2.txt"), []byte("content2"))
		CreateDirectory(filepath.Join(dirPath, "subdir"))

		// List directory
		entries, err := ListDirectory(dirPath)
		if err != nil {
			t.Fatalf("Failed to list directory: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(entries))
		}

		// Verify entries
		foundFile1, foundFile2, foundSubdir := false, false, false
		for _, entry := range entries {
			switch entry.Name {
			case "file1.txt":
				foundFile1 = true
				if entry.IsDir {
					t.Error("file1.txt should not be a directory")
				}
			case "file2.txt":
				foundFile2 = true
				if entry.IsDir {
					t.Error("file2.txt should not be a directory")
				}
			case "subdir":
				foundSubdir = true
				if !entry.IsDir {
					t.Error("subdir should be a directory")
				}
			}
		}

		if !foundFile1 || !foundFile2 || !foundSubdir {
			t.Error("Not all expected entries found")
		}
	})

	t.Run("ListDirectoryRecursive", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "recursivelist")

		// Create nested structure
		CreateDirectories(filepath.Join(dirPath, "sub1", "sub2"))
		CreateFile(filepath.Join(dirPath, "root.txt"), []byte("root"))
		CreateFile(filepath.Join(dirPath, "sub1", "sub1.txt"), []byte("sub1"))
		CreateFile(filepath.Join(dirPath, "sub1", "sub2", "sub2.txt"), []byte("sub2"))

		// List recursively
		entries, err := ListDirectory(dirPath, WithRecursive())
		if err != nil {
			t.Fatalf("Failed to list directory recursively: %v", err)
		}

		// Should have: root.txt, sub1, sub1.txt, sub2, sub2.txt
		if len(entries) < 5 {
			t.Errorf("Expected at least 5 entries, got %d", len(entries))
		}
	})

	t.Run("GetDirectoryInfo", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "infodir")

		// Create directory with content
		CreateDirectory(dirPath)
		CreateFile(filepath.Join(dirPath, "file1.txt"), []byte("hello"))
		CreateFile(filepath.Join(dirPath, "file2.txt"), []byte("world"))
		CreateDirectory(filepath.Join(dirPath, "subdir"))

		// Get info
		info, err := GetDirectoryInfo(dirPath)
		if err != nil {
			t.Fatalf("Failed to get directory info: %v", err)
		}

		if info.FileCount != 2 {
			t.Errorf("Expected 2 files, got %d", info.FileCount)
		}
		if info.DirCount != 1 {
			t.Errorf("Expected 1 subdirectory, got %d", info.DirCount)
		}
		if info.TotalSize != 10 { // "hello" + "world" = 10 bytes
			t.Errorf("Expected total size 10, got %d", info.TotalSize)
		}
	})

	t.Run("IsEmptyDirectory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		nonEmptyDir := filepath.Join(tmpDir, "nonempty")

		// Create empty directory
		CreateDirectory(emptyDir)

		// Create non-empty directory
		CreateDirectory(nonEmptyDir)
		CreateFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"))

		// Test empty
		isEmpty, err := IsEmptyDirectory(emptyDir)
		if err != nil {
			t.Fatalf("Failed to check empty directory: %v", err)
		}
		if !isEmpty {
			t.Error("Directory should be empty")
		}

		// Test non-empty
		isEmpty, err = IsEmptyDirectory(nonEmptyDir)
		if err != nil {
			t.Fatalf("Failed to check non-empty directory: %v", err)
		}
		if isEmpty {
			t.Error("Directory should not be empty")
		}
	})

	t.Run("ChangeDirectoryPermissions", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "permdir")
		subDir := filepath.Join(dirPath, "subdir")

		// Create directories
		CreateDirectory(dirPath)
		CreateDirectory(subDir)

		// Change permissions
		newMode := os.FileMode(0700)
		if err := ChangeDirectoryPermissions(dirPath, newMode); err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}

		// Verify
		info, _ := os.Stat(dirPath)
		if info.Mode().Perm() != newMode {
			t.Errorf("Permission mismatch: got %v, want %v", info.Mode().Perm(), newMode)
		}
	})

	t.Run("ListDirectorySorted", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "sortdir")

		// Create directory with files of different sizes
		CreateDirectory(dirPath)
		CreateFile(filepath.Join(dirPath, "c.txt"), []byte("xxx"))   // 3 bytes
		CreateFile(filepath.Join(dirPath, "a.txt"), []byte("x"))     // 1 byte
		CreateFile(filepath.Join(dirPath, "b.txt"), []byte("xxxxx")) // 5 bytes

		// Test sort by name
		entries, err := ListDirectoryByName(dirPath, true)
		if err != nil {
			t.Fatalf("Failed to list directory by name: %v", err)
		}

		if len(entries) != 3 {
			t.Fatalf("Expected 3 entries, got %d", len(entries))
		}

		if entries[0].Name != "a.txt" || entries[1].Name != "b.txt" || entries[2].Name != "c.txt" {
			t.Error("Entries not sorted by name correctly")
		}

		// Test sort by size
		entries, err = ListDirectoryBySize(dirPath, true)
		if err != nil {
			t.Fatalf("Failed to list directory by size: %v", err)
		}

		if entries[0].Size != 1 || entries[1].Size != 3 || entries[2].Size != 5 {
			t.Error("Entries not sorted by size correctly")
		}
	})
}
