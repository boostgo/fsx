# FSX - File System Extended

[![Go Reference](https://pkg.go.dev/badge/github.com/boostgo/fsx.svg)](https://pkg.go.dev/github.com/boostgo/fsx)
[![Go Report Card](https://goreportcard.com/badge/github.com/boostgo/fsx)](https://goreportcard.com/report/github.com/boostgo/fsx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

FSX is a powerful and comprehensive file system library for Go that provides extended functionality beyond the standard library. It offers a clean, intuitive API for file and directory operations, search capabilities, atomic operations, compression, and much more.

## Features

- 🚀 **Simple and intuitive API** - Easy to use with sensible defaults
- 🔒 **Thread-safe operations** - File locking and atomic operations
- 🔍 **Powerful search** - Find files by name, content, size, time, and more
- 📁 **Advanced directory operations** - Copy, sync, compare directories
- 🗜️ **Compression support** - Gzip and zip archive handling
- 🔄 **Stream processing** - Handle large files efficiently
- ✅ **Comprehensive error handling** - Using the errorx package
- 🛡️ **Production-ready** - Extensive test coverage

## Installation

```bash
go get github.com/boostgo/fsx
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/boostgo/fsx"
)

func main() {
    // Write a file
    err := fsx.WriteFileString("hello.txt", "Hello, FSX!")
    if err != nil {
        log.Fatal(err)
    }

    // Read it back
    content, err := fsx.ReadFileString("hello.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(content) // Output: Hello, FSX!

    // Copy a directory
    err = fsx.CopyDirectory("source", "destination", fsx.WithOverwrite())
    if err != nil {
        log.Fatal(err)
    }

    // Find all .go files
    results, err := fsx.FindFiles(".", "*.go")
    if err != nil {
        log.Fatal(err)
    }
    for _, result := range results {
        fmt.Println(result.Path)
    }
}
```

## Core Operations

### File Operations

#### Basic File Operations

```go
// Check if file exists
exists := fsx.FileExist("config.json")

// Create/Write files
fsx.CreateFile("new.txt", []byte("content"))
fsx.WriteFileString("config.json", `{"key": "value"}`)
fsx.WriteFileLines("list.txt", []string{"item1", "item2", "item3"})

// Read files
data, _ := fsx.ReadFile("data.bin")
text, _ := fsx.ReadFileString("readme.txt")
lines, _ := fsx.ReadFileLines("list.txt")

// Append to files
fsx.AppendFileString("log.txt", "New log entry\n")

// Copy, move, delete
fsx.CopyFile("src.txt", "dst.txt", fsx.WithBackup())
fsx.MoveFile("old.txt", "new.txt", fsx.WithCreateDirs())
fsx.DeleteFile("temp.txt")

// Get file info
info, _ := fsx.GetFileInfo("document.pdf")
fmt.Printf("Size: %d bytes, Modified: %s\n", info.Size, info.ModTime)

// Change permissions
fsx.ChangeFilePermissions("script.sh", 0755)
```

#### Advanced File Operations

```go
// Atomic write (write to temp file, then rename)
fsx.AtomicWriteFile("important.conf", configData, 0644)

// Create temporary files
tmpFile, _ := fsx.CreateTempFile("", "upload-*.tmp", data)
defer os.Remove(tmpFile)

// File locking
lock, _ := fsx.LockFile("database.db")
lock.Write([]byte("exclusive data"))
lock.Unlock()

// Stream processing for large files
fsx.StreamProcessFile("large.log", func(line string, lineNum int) error {
    if strings.Contains(line, "ERROR") {
        fmt.Printf("Error on line %d: %s\n", lineNum, line)
    }
    return nil
})

// Calculate checksums
md5sum, _ := fsx.CalculateFileChecksum("file.zip", fsx.HashMD5)
sha256sum, _ := fsx.CalculateFileChecksum("file.zip", fsx.HashSHA256)

// Verify checksum
valid, _ := fsx.VerifyFileChecksum("file.zip", expectedMD5, fsx.HashMD5)

// Compress/decompress files
fsx.CompressFile("large.txt", "large.txt.gz")
fsx.DecompressFile("archive.gz", "extracted.txt")

// Split and merge files
chunks, _ := fsx.SplitFile("huge.bin", 1024*1024*100) // 100MB chunks
fsx.MergeFiles(chunks, "reconstructed.bin")
```

### Directory Operations

#### Basic Directory Operations

```go
// Check if directory exists
exists := fsx.DirectoryExist("/path/to/dir")

// Create directories
fsx.CreateDirectory("newdir", fsx.WithDirPermissions(0755))
fsx.CreateDirectories("path/to/nested/dir") // Creates all parent directories

// List directory contents
entries, _ := fsx.ListDirectory("/home/user")
for _, entry := range entries {
    fmt.Printf("%s - Size: %d, IsDir: %v\n", entry.Name, entry.Size, entry.IsDir)
}

// List with sorting
entries, _ = fsx.ListDirectoryBySize("/downloads", true) // ascending order
entries, _ = fsx.ListDirectoryByModTime("/documents", false) // descending order

// Delete directories
fsx.DeleteDirectory("emptydir")
fsx.DeleteDirectory("fulldir", fsx.WithForce()) // Delete even if not empty

// Rename/move directories
fsx.RenameDirectory("oldname", "newname")

// Get directory info
info, _ := fsx.GetDirectoryInfo("/home/user/projects")
fmt.Printf("Total size: %d bytes, Files: %d, Dirs: %d\n", 
    info.TotalSize, info.FileCount, info.DirCount)
```

#### Advanced Directory Operations

```go
// Copy entire directory tree
fsx.CopyDirectory("source", "backup", 
    fsx.WithOverwrite(),
    fsx.WithProgress(func(current, total int64, file string) {
        fmt.Printf("Progress: %d/%d bytes - %s\n", current, total, file)
    }))

// Sync directories (one-way sync)
fsx.SyncDirectories("source", "mirror")

// Compare directories
differences, _ := fsx.CompareDirectories("dir1", "dir2")
for _, diff := range differences {
    switch diff.Type {
    case fsx.DiffAdded:
        fmt.Printf("Added: %s\n", diff.Path)
    case fsx.DiffRemoved:
        fmt.Printf("Removed: %s\n", diff.Path)
    case fsx.DiffModified:
        fmt.Printf("Modified: %s\n", diff.Path)
    }
}

// Calculate directory size
size, _ := fsx.CalculateDirectorySize("/home/user/downloads")
fmt.Printf("Total size: %d MB\n", size/1024/1024)

// Find duplicate files
duplicates, _ := fsx.FindDuplicateFiles("/photos")
for hash, files := range duplicates {
    fmt.Printf("Duplicate files (hash: %s):\n", hash)
    for _, file := range files {
        fmt.Printf("  - %s\n", file)
    }
}

// Clean empty directories
fsx.CleanEmptyDirectories("/temp")

// Walk directory with custom function
fsx.WalkDirectory("/data", func(path string, info os.FileInfo, err error) error {
    if err != nil {
        return err
    }
    if !info.IsDir() && info.Size() > 1024*1024*100 { // Files > 100MB
        fmt.Printf("Large file: %s (%d MB)\n", path, info.Size()/1024/1024)
    }
    return nil
})
```

### Search Operations

```go
// Find files by name pattern
results, _ := fsx.FindFiles("/home", "*.txt", fsx.WithIgnoreHidden())

// Find with multiple criteria
results, _ = fsx.FindFiles("/project", "*.go",
    fsx.WithMaxDepth(3),
    fsx.WithIgnoreHidden(),
    fsx.WithExcludePatterns("*_test.go", "vendor/*"))

// Find by regex
results, _ = fsx.FindFilesByRegex("/logs", `error_\d{4}\.log`)

// Find by content
results, _ = fsx.FindFilesByContent("/docs", "TODO", 
    fsx.WithCaseSensitive(false),
    fsx.WithWholeWord())

for _, result := range results {
    fmt.Printf("Found in %s at line %d: %s\n", 
        result.Path, result.LineNumber, result.Line)
}

// Find by size
largeFiles, _ := fsx.FindFilesBySize("/downloads", 
    1024*1024*100, // min 100MB
    -1)            // no max

// Find by modification time
recentFiles, _ := fsx.FindFilesByTime("/documents",
    time.Now().Add(-24*time.Hour), // after 24 hours ago
    time.Now())                    // before now

// Find by permissions
executableFiles, _ := fsx.FindFilesByPermissions("/bin", 0111, false)
```

## Options and Configurations

FSX uses functional options pattern for flexible configuration:

### File Options
- `WithPermissions(mode)` - Set custom file permissions
- `WithCreateDirs()` - Create parent directories if needed
- `WithBackup()` - Create backup before overwriting
- `WithBufferSize(size)` - Set buffer size for operations

### Directory Options
- `WithDirPermissions(mode)` - Set directory permissions
- `WithRecursive()` - Enable recursive operations
- `WithForce()` - Force operations (e.g., delete non-empty dirs)

### Copy Options
- `WithOverwrite()` - Allow overwriting existing files
- `WithPreservePermissions()` - Preserve original permissions
- `WithPreserveTimes()` - Preserve modification times
- `WithSkipErrors()` - Continue on errors
- `WithFilter(func)` - Filter files during copy
- `WithProgress(func)` - Track copy progress

### Search Options
- `WithMaxDepth(n)` - Maximum directory depth
- `WithMinDepth(n)` - Minimum directory depth
- `WithCaseSensitive(bool)` - Case sensitivity
- `WithIgnoreHidden()` - Ignore hidden files
- `WithLimitResults(n)` - Limit number of results
- `WithIncludePatterns(...)` - Include patterns
- `WithExcludePatterns(...)` - Exclude patterns

## Compression and Archives

```go
// Create zip archive
files := []string{"file1.txt", "file2.txt", "file3.txt"}
fsx.CreateZipArchive("archive.zip", files)

// Extract zip archive
fsx.ExtractZipArchive("archive.zip", "/tmp/extracted")

// Gzip compression
fsx.CompressFile("large.log", "large.log.gz")
fsx.DecompressFile("data.gz", "data.txt")
```

## Error Handling

FSX uses the errorx package for enhanced error handling:

```go
err := fsx.CopyFile("src.txt", "dst.txt")
if err != nil {
    switch {
    case errors.Is(err, fsx.ErrFileNotExist):
        fmt.Println("Source file does not exist")
    case errors.Is(err, fsx.ErrPermissionDenied):
        fmt.Println("Permission denied")
    default:
        fmt.Printf("Copy failed: %v\n", err)
    }
}
```

## Performance Considerations

- Use streaming operations for large files to avoid loading entire content into memory
- Use buffered operations with appropriate buffer sizes
- Enable `WithSkipErrors()` for resilient batch operations
- Use `WithLimitResults()` when searching in large directory trees
- Consider using goroutines with file locking for concurrent operations

## Thread Safety

FSX provides thread-safe operations through file locking:

```go
// Safe concurrent writes
lock1, _ := fsx.LockFile("shared.txt")
go func() {
    lock1.Write([]byte("goroutine 1 data"))
    lock1.Unlock()
}()

lock2, _ := fsx.LockFile("shared.txt") // Will wait until lock1 is released
lock2.Write([]byte("goroutine 2 data"))
lock2.Unlock()
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with ❤️ using Go
- Error handling powered by [errorx](https://github.com/boostgo/errorx)
- Inspired by the need for a more comprehensive file system library in Go