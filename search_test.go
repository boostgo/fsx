package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearchOperations(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "fsx_search_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test structure
	setupSearchTestStructure(t, tmpDir)

	t.Run("FindFiles", func(t *testing.T) {
		// Find all .txt files (excluding hidden)
		results, err := FindFiles(tmpDir, "*.txt", WithIgnoreHidden())
		if err != nil {
			t.Fatalf("Failed to find files: %v", err)
		}

		// Count non-hidden .txt files in all directories
		// Based on setupSearchTestStructure: test.txt, test1.txt, test2.txt,
		// subdir2/another.txt, .hidden_dir/secret.txt (hidden)
		// Total non-hidden: 4
		if len(results) != 4 {
			t.Errorf("Expected 4 non-hidden .txt files, got %d", len(results))
			for _, r := range results {
				t.Logf("Found: %s", r.Path)
			}
		}

		// Find with specific name
		results, err = FindFiles(tmpDir, "test.txt")
		if err != nil {
			t.Fatalf("Failed to find test.txt: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 test.txt file, got %d", len(results))
		}
	})

	t.Run("FindFilesWithDepthLimit", func(t *testing.T) {
		// Find files with max depth 1 (root and immediate subdirs only)
		results, err := FindFiles(tmpDir, "*", WithMaxDepth(1))
		if err != nil {
			t.Fatalf("Failed to find files with depth limit: %v", err)
		}

		// Should not include files in deep/nested
		for _, result := range results {
			if strings.Contains(result.Path, "nested") {
				t.Error("Should not include files from nested directory with maxDepth=1")
			}
		}
	})

	t.Run("FindFilesWithMinDepth", func(t *testing.T) {
		// Find files with min depth 2 (skip root and immediate subdirs)
		results, err := FindFiles(tmpDir, "*", WithMinDepth(2))
		if err != nil {
			t.Fatalf("Failed to find files with min depth: %v", err)
		}

		// Should only include files in deep/nested
		foundNested := false
		for _, result := range results {
			relPath, _ := filepath.Rel(tmpDir, result.Path)
			depth := len(strings.Split(relPath, string(os.PathSeparator)))
			if depth < 2 {
				t.Error("Found file with depth less than 2")
			}
			if strings.Contains(result.Path, "nested.log") {
				foundNested = true
			}
		}

		if !foundNested {
			t.Error("Should find nested.log with minDepth=2")
		}
	})

	t.Run("FindFilesCaseInsensitive", func(t *testing.T) {
		// Case insensitive search
		results, err := FindFiles(tmpDir, "TEST.TXT", WithCaseSensitive(false))
		if err != nil {
			t.Fatalf("Failed to find files case insensitive: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 file with case insensitive search, got %d", len(results))
		}
	})

	t.Run("FindFilesWithPatterns", func(t *testing.T) {
		// Include .txt and .log, exclude hidden files
		results, err := FindFiles(tmpDir, "*",
			WithIncludePatterns("*.txt", "*.log"),
			WithExcludePatterns(".*"))
		if err != nil {
			t.Fatalf("Failed to find files with patterns: %v", err)
		}

		// Should have txt and log files but no hidden files
		foundTxt, foundLog, foundHidden := false, false, false
		for _, result := range results {
			if strings.HasSuffix(result.Path, ".txt") {
				foundTxt = true
			}
			if strings.HasSuffix(result.Path, ".log") {
				foundLog = true
			}
			if strings.Contains(result.Path, ".hidden") {
				foundHidden = true
			}
		}

		if !foundTxt || !foundLog {
			t.Error("Should find both .txt and .log files")
		}
		if foundHidden {
			t.Error("Should not find hidden files")
		}
	})

	t.Run("FindFilesByRegex", func(t *testing.T) {
		// Find files matching regex pattern
		results, err := FindFilesByRegex(tmpDir, `test\d+\.txt`)
		if err != nil {
			t.Fatalf("Failed to find files by regex: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 files matching test\\d+.txt, got %d", len(results))
		}

		// Verify matched files
		for _, result := range results {
			name := filepath.Base(result.Path)
			if name != "test1.txt" && name != "test2.txt" {
				t.Errorf("Unexpected file matched: %s", name)
			}
		}
	})

	t.Run("FindFilesByContent", func(t *testing.T) {
		// Find files containing "Hello"
		results, err := FindFilesByContent(tmpDir, "Hello")
		if err != nil {
			t.Fatalf("Failed to find files by content: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 file containing 'Hello', got %d", len(results))
		}

		if len(results) > 0 {
			result := results[0]
			if result.LineNumber != 1 {
				t.Errorf("Expected match on line 1, got line %d", result.LineNumber)
			}
			if !strings.Contains(result.Line, "Hello") {
				t.Error("Matched line should contain 'Hello'")
			}
		}
	})

	t.Run("FindFilesByContentWholeWord", func(t *testing.T) {
		// Create test file with partial matches
		testFile := filepath.Join(tmpDir, "whole_word_test.txt")
		if err := WriteFileString(testFile, "testing test tested"); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Find whole word "test"
		results, err := FindFilesByContent(tmpDir, "test", WithWholeWord())
		if err != nil {
			t.Fatalf("Failed to find files by whole word: %v", err)
		}

		found := false
		for _, result := range results {
			if result.Path == testFile {
				found = true
				if !strings.Contains(result.Line, "test") {
					t.Error("Should match line containing whole word 'test'")
				}
			}
		}

		if !found {
			t.Error("Should find file with whole word 'test'")
		}
	})

	t.Run("FindFilesBySize", func(t *testing.T) {
		// Find files between 10 and 100 bytes
		results, err := FindFilesBySize(tmpDir, 10, 100)
		if err != nil {
			t.Fatalf("Failed to find files by size: %v", err)
		}

		// Verify all results are within size range
		for _, result := range results {
			size := result.Info.Size()
			if size < 10 || size > 100 {
				t.Errorf("File %s has size %d, outside range 10-100", result.Path, size)
			}
		}

		// Find files larger than 100 bytes
		results, err = FindFilesBySize(tmpDir, 100, -1)
		if err != nil {
			t.Fatalf("Failed to find large files: %v", err)
		}

		for _, result := range results {
			if result.Info.Size() < 100 {
				t.Errorf("File %s is smaller than 100 bytes", result.Path)
			}
		}
	})

	t.Run("FindFilesByTime", func(t *testing.T) {
		// Create a file with specific modification time
		oldFile := filepath.Join(tmpDir, "old_file.txt")
		if err := CreateFile(oldFile, []byte("old content")); err != nil {
			t.Fatalf("Failed to create old file: %v", err)
		}

		// Set modification time to 2 days ago
		oldTime := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}

		// Find files modified in the last 24 hours
		yesterday := time.Now().Add(-24 * time.Hour)
		results, err := FindFilesByTime(tmpDir, yesterday, time.Now())
		if err != nil {
			t.Fatalf("Failed to find files by time: %v", err)
		}

		// Should not include the old file
		for _, result := range results {
			if result.Path == oldFile {
				t.Error("Should not find file older than 24 hours")
			}
		}

		// Find files modified more than 24 hours ago
		results, err = FindFilesByTime(tmpDir, time.Time{}, yesterday)
		if err != nil {
			t.Fatalf("Failed to find old files: %v", err)
		}

		foundOld := false
		for _, result := range results {
			if result.Path == oldFile {
				foundOld = true
			}
		}

		if !foundOld {
			t.Error("Should find file older than 24 hours")
		}
	})

	t.Run("FindFilesByPermissions", func(t *testing.T) {
		// Create files with different permissions
		readOnlyFile := filepath.Join(tmpDir, "readonly.txt")
		if err := CreateFile(readOnlyFile, []byte("readonly"), WithPermissions(0444)); err != nil {
			t.Fatalf("Failed to create readonly file: %v", err)
		}

		executableFile := filepath.Join(tmpDir, "executable.sh")
		if err := CreateFile(executableFile, []byte("#!/bin/bash"), WithPermissions(0755)); err != nil {
			t.Fatalf("Failed to create executable file: %v", err)
		}

		// Find files with exact permissions 0444
		results, err := FindFilesByPermissions(tmpDir, 0444, true)
		if err != nil {
			t.Fatalf("Failed to find files by exact permissions: %v", err)
		}

		foundReadOnly := false
		for _, result := range results {
			if result.Path == readOnlyFile {
				foundReadOnly = true
			}
		}

		if !foundReadOnly {
			t.Error("Should find file with exact permissions 0444")
		}

		// Find files with at least read permission for owner (0400)
		results, err = FindFilesByPermissions(tmpDir, 0400, false)
		if err != nil {
			t.Fatalf("Failed to find files by permissions: %v", err)
		}

		// Should find most files
		if len(results) < 2 {
			t.Error("Should find multiple files with read permission")
		}
	})

	t.Run("FindFilesWithLimit", func(t *testing.T) {
		// Find only first 2 files
		results, err := FindFiles(tmpDir, "*", WithLimitResults(2))
		if err != nil {
			t.Fatalf("Failed to find files with limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected exactly 2 results with limit, got %d", len(results))
		}
	})

	t.Run("FindFilesIgnoreHidden", func(t *testing.T) {
		// Find all files including hidden
		resultsWithHidden, err := FindFiles(tmpDir, "*")
		if err != nil {
			t.Fatalf("Failed to find all files: %v", err)
		}

		// Find files ignoring hidden
		resultsNoHidden, err := FindFiles(tmpDir, "*", WithIgnoreHidden())
		if err != nil {
			t.Fatalf("Failed to find files ignoring hidden: %v", err)
		}

		// Should have fewer results when ignoring hidden
		if len(resultsNoHidden) >= len(resultsWithHidden) {
			t.Error("Should have fewer results when ignoring hidden files")
		}

		// Verify no hidden files in results
		for _, result := range resultsNoHidden {
			name := filepath.Base(result.Path)
			if strings.HasPrefix(name, ".") {
				t.Errorf("Found hidden file %s when ignoring hidden", name)
			}
		}
	})

	t.Run("ComplexSearch", func(t *testing.T) {
		// Complex search: .txt files, max 50 bytes, modified recently, not in deep directories
		results, err := FindFiles(tmpDir, "*.txt",
			WithMaxDepth(1),
			WithIgnoreHidden(),
			WithLimitResults(5))
		if err != nil {
			t.Fatalf("Failed complex search: %v", err)
		}

		// Verify all conditions
		for _, result := range results {
			// Check it's a .txt file
			if !strings.HasSuffix(result.Path, ".txt") {
				t.Error("Result should be a .txt file")
			}

			// Check it's not hidden
			name := filepath.Base(result.Path)
			if strings.HasPrefix(name, ".") {
				t.Error("Should not include hidden files")
			}

			// Check depth
			relPath, _ := filepath.Rel(tmpDir, result.Path)
			depth := len(strings.Split(relPath, string(os.PathSeparator))) - 1
			if depth > 1 {
				t.Error("Should not include files deeper than maxDepth")
			}
		}

		// Check limit
		if len(results) > 5 {
			t.Error("Should respect result limit")
		}
	})
}

// setupSearchTestStructure creates a test directory structure
func setupSearchTestStructure(t *testing.T, root string) {
	// Create directories
	dirs := []string{
		"subdir1",
		"subdir2",
		"deep/nested",
		".hidden_dir",
	}

	for _, dir := range dirs {
		if err := CreateDirectories(filepath.Join(root, dir)); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files with various attributes
	files := map[string]string{
		"test.txt":               "Hello World",
		"test1.txt":              "Content of test1",
		"test2.txt":              "Content of test2",
		"document.md":            "# Markdown Document\n\nThis is a test document.",
		"data.json":              `{"name": "test", "value": 123}`,
		"script.sh":              "#!/bin/bash\necho 'test'",
		".hidden_file":           "Hidden content",
		"subdir1/file.log":       "Log entry 1\nLog entry 2",
		"subdir2/another.txt":    "Another file",
		"deep/nested/nested.log": "Deeply nested content",
		".hidden_dir/secret.txt": "Secret content",
		"large_file.dat":         strings.Repeat("x", 1000), // 1KB file
	}

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := CreateFile(fullPath, []byte(content)); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Set different permissions on some files
	if err := os.Chmod(filepath.Join(root, "script.sh"), 0755); err != nil {
		t.Logf("Failed to set script permissions: %v", err)
	}
}
