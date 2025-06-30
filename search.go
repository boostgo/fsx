package fsx

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FindFiles finds files by name pattern (supports wildcards)
func FindFiles(root string, pattern string, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	var results []SearchResult
	currentDepth := 0
	resultsFound := 0

	err := walkWithDepth(root, currentDepth, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF // Stop walking
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply exclude patterns first
		for _, excludePattern := range opts.excludePatterns {
			matched, err := matchPattern(info.Name(), excludePattern, opts.caseSensitive)
			if err != nil {
				return err
			}
			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Apply include patterns
		if len(opts.includePatterns) > 0 {
			included := false
			for _, includePattern := range opts.includePatterns {
				matched, err := matchPattern(info.Name(), includePattern, opts.caseSensitive)
				if err != nil {
					return err
				}
				if matched {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		// Match main pattern
		matched, err := matchPattern(info.Name(), pattern, opts.caseSensitive)
		if err != nil {
			return err
		}

		if matched && !info.IsDir() {
			results = append(results, SearchResult{
				Path:      path,
				Info:      info,
				MatchedBy: "name",
			})
			resultsFound++
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchFiles.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// FindFilesByRegex finds files by regex pattern
func FindFilesByRegex(root string, pattern string, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Compile regex
	var re *regexp.Regexp
	var err error
	if opts.caseSensitive {
		re, err = regexp.Compile(pattern)
	} else {
		re, err = regexp.Compile("(?i)" + pattern)
	}
	if err != nil {
		return nil, ErrInvalidRegex.
			SetError(err).
			SetData(struct {
				Pattern string `json:"pattern"`
				Error   error  `json:"error"`
			}{
				Pattern: pattern,
				Error:   err,
			})
	}

	var results []SearchResult
	resultsFound := 0

	err = walkWithDepth(root, 0, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if re.MatchString(info.Name()) && !info.IsDir() {
			results = append(results, SearchResult{
				Path:      path,
				Info:      info,
				MatchedBy: "regex",
			})
			resultsFound++
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchFiles.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// FindFilesByContent finds files containing specific content
func FindFilesByContent(root string, content string, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Prepare search pattern
	searchPattern := content
	if !opts.caseSensitive {
		searchPattern = strings.ToLower(searchPattern)
	}

	var results []SearchResult
	resultsFound := 0

	err := walkWithDepth(root, 0, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories and binary files
		if info.IsDir() || !isTextFile(path) {
			return nil
		}

		// Search in file content
		lines, err := ReadFileLines(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		for lineNum, line := range lines {
			searchLine := line
			if !opts.caseSensitive {
				searchLine = strings.ToLower(searchLine)
			}

			found := false
			if opts.wholeWord {
				// Whole word search
				words := strings.Fields(searchLine)
				for _, word := range words {
					if word == searchPattern {
						found = true
						break
					}
				}
			} else {
				// Substring search
				found = strings.Contains(searchLine, searchPattern)
			}

			if found {
				results = append(results, SearchResult{
					Path:       path,
					Info:       info,
					MatchedBy:  "content",
					LineNumber: lineNum + 1,
					Line:       line,
				})
				resultsFound++

				// If limit reached, stop
				if opts.limitResults > 0 && resultsFound >= opts.limitResults {
					return io.EOF
				}

				break // Move to next file after first match
			}
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchContent.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// FindFilesBySize finds files by size criteria
func FindFilesBySize(root string, minSize, maxSize int64, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	var results []SearchResult
	resultsFound := 0

	err := walkWithDepth(root, 0, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			size := info.Size()
			if (minSize < 0 || size >= minSize) && (maxSize < 0 || size <= maxSize) {
				results = append(results, SearchResult{
					Path:      path,
					Info:      info,
					MatchedBy: "size",
				})
				resultsFound++
			}
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchFiles.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// FindFilesByTime finds files by modification time
func FindFilesByTime(root string, after, before time.Time, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	var results []SearchResult
	resultsFound := 0

	err := walkWithDepth(root, 0, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			modTime := info.ModTime()
			if (after.IsZero() || modTime.After(after)) && (before.IsZero() || modTime.Before(before)) {
				results = append(results, SearchResult{
					Path:      path,
					Info:      info,
					MatchedBy: "time",
				})
				resultsFound++
			}
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchFiles.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// FindFilesByPermissions finds files by permission bits
func FindFilesByPermissions(root string, mode os.FileMode, exact bool, options ...SearchOption) ([]SearchResult, error) {
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	var results []SearchResult
	resultsFound := 0

	err := walkWithDepth(root, 0, func(path string, info os.FileInfo, depth int, err error) error {
		if err != nil {
			return err
		}

		// Check depth limits
		if opts.maxDepth >= 0 && depth > opts.maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depth < opts.minDepth {
			return nil
		}

		// Check result limit
		if opts.limitResults > 0 && resultsFound >= opts.limitResults {
			return io.EOF
		}

		// Handle hidden files
		if opts.ignoreHidden && isHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			fileMode := info.Mode().Perm()
			matched := false

			if exact {
				matched = fileMode == mode
			} else {
				// Check if file has at least the specified permissions
				matched = fileMode&mode == mode
			}

			if matched {
				results = append(results, SearchResult{
					Path:      path,
					Info:      info,
					MatchedBy: "permissions",
				})
				resultsFound++
			}
		}

		return nil
	}, opts.followSymlinks)

	if err != nil && err != io.EOF {
		return nil, ErrSearchFiles.
			SetError(err).
			SetData(pathErrorContext{
				Path:  root,
				Error: err,
			})
	}

	return results, nil
}

// Helper functions

// walkWithDepth is a helper that walks directory tree tracking depth
func walkWithDepth(root string, currentDepth int, fn func(path string, info os.FileInfo, depth int, err error) error, followSymlinks bool) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fn(root, nil, currentDepth, err)
	}

	// Handle symlinks
	if info.Mode()&os.ModeSymlink != 0 && followSymlinks {
		info, err = os.Stat(root)
		if err != nil {
			return fn(root, nil, currentDepth, err)
		}
	}

	err = fn(root, info, currentDepth, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return fn(root, info, currentDepth, err)
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		err = walkWithDepth(path, currentDepth+1, fn, followSymlinks)
		if err != nil {
			if err == io.EOF {
				return err
			}
			// Continue on error unless it's a stop signal
			continue
		}
	}

	return nil
}

// matchPattern matches a pattern against a name (supports * and ? wildcards)
func matchPattern(name, pattern string, caseSensitive bool) (bool, error) {
	if !caseSensitive {
		name = strings.ToLower(name)
		pattern = strings.ToLower(pattern)
	}

	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false, ErrInvalidPattern.
			SetError(err).
			SetData(struct {
				Pattern string `json:"pattern"`
				Error   error  `json:"error"`
			}{
				Pattern: pattern,
				Error:   err,
			})
	}

	return matched, nil
}

// isHidden checks if a file/directory is hidden
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// isTextFile checks if a file is likely a text file (simple heuristic)
func isTextFile(path string) bool {
	// Check by extension first
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := []string{
		".txt", ".log", ".md", ".json", ".xml", ".yaml", ".yml",
		".go", ".js", ".py", ".java", ".c", ".cpp", ".h", ".hpp",
		".html", ".css", ".scss", ".less", ".vue", ".jsx", ".tsx",
		".sh", ".bash", ".zsh", ".fish", ".conf", ".cfg", ".ini",
		".csv", ".sql", ".rs", ".rb", ".php", ".swift", ".kt",
	}

	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	// Could implement more sophisticated detection by reading first few bytes
	// and checking for binary content, but this is good enough for most cases
	return false
}
