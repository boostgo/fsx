package fsx

// SearchOption represents options for search operations
type SearchOption func(*searchOptions)

type searchOptions struct {
	maxDepth        int
	minDepth        int
	followSymlinks  bool
	caseSensitive   bool
	wholeWord       bool
	ignoreHidden    bool
	limitResults    int
	includePatterns []string
	excludePatterns []string
}

// defaultSearchOptions returns default search options
func defaultSearchOptions() *searchOptions {
	return &searchOptions{
		maxDepth:        -1, // No limit
		minDepth:        0,
		followSymlinks:  false,
		caseSensitive:   true,
		wholeWord:       false,
		ignoreHidden:    false,
		limitResults:    -1, // No limit
		includePatterns: []string{},
		excludePatterns: []string{},
	}
}

// WithMaxDepth sets maximum directory depth for search
func WithMaxDepth(depth int) SearchOption {
	return func(opts *searchOptions) {
		opts.maxDepth = depth
	}
}

// WithMinDepth sets minimum directory depth for search
func WithMinDepth(depth int) SearchOption {
	return func(opts *searchOptions) {
		opts.minDepth = depth
	}
}

// WithSearchFollowSymlinks enables following symbolic links
func WithSearchFollowSymlinks() SearchOption {
	return func(opts *searchOptions) {
		opts.followSymlinks = true
	}
}

// WithCaseSensitive sets case sensitivity for searches
func WithCaseSensitive(sensitive bool) SearchOption {
	return func(opts *searchOptions) {
		opts.caseSensitive = sensitive
	}
}

// WithWholeWord matches whole words only in content search
func WithWholeWord() SearchOption {
	return func(opts *searchOptions) {
		opts.wholeWord = true
	}
}

// WithIgnoreHidden ignores hidden files and directories
func WithIgnoreHidden() SearchOption {
	return func(opts *searchOptions) {
		opts.ignoreHidden = true
	}
}

// WithLimitResults limits the number of results returned
func WithLimitResults(limit int) SearchOption {
	return func(opts *searchOptions) {
		opts.limitResults = limit
	}
}

// WithIncludePatterns adds patterns that files must match
func WithIncludePatterns(patterns ...string) SearchOption {
	return func(opts *searchOptions) {
		opts.includePatterns = append(opts.includePatterns, patterns...)
	}
}

// WithExcludePatterns adds patterns that files must not match
func WithExcludePatterns(patterns ...string) SearchOption {
	return func(opts *searchOptions) {
		opts.excludePatterns = append(opts.excludePatterns, patterns...)
	}
}
