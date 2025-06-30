package fsx

// CopyOption represents options for copy operations
type CopyOption func(*copyOptions)

type copyOptions struct {
	overwrite       bool
	preservePerms   bool
	preserveTimes   bool
	skipErrors      bool
	followSymlinks  bool
	filter          FilterFunc
	progressHandler ProgressFunc
}

// defaultCopyOptions returns default copy options
func defaultCopyOptions() *copyOptions {
	return &copyOptions{
		overwrite:      false,
		preservePerms:  true,
		preserveTimes:  true,
		skipErrors:     false,
		followSymlinks: false,
	}
}

// WithOverwrite allows overwriting existing files
func WithOverwrite() CopyOption {
	return func(opts *copyOptions) {
		opts.overwrite = true
	}
}

// WithPreservePermissions preserves original file permissions
func WithPreservePermissions(preserve bool) CopyOption {
	return func(opts *copyOptions) {
		opts.preservePerms = preserve
	}
}

// WithPreserveTimes preserves original modification times
func WithPreserveTimes(preserve bool) CopyOption {
	return func(opts *copyOptions) {
		opts.preserveTimes = preserve
	}
}

// WithSkipErrors continues operation on errors
func WithSkipErrors() CopyOption {
	return func(opts *copyOptions) {
		opts.skipErrors = true
	}
}

// WithFollowSymlinks follows symbolic links
func WithFollowSymlinks() CopyOption {
	return func(opts *copyOptions) {
		opts.followSymlinks = true
	}
}

// WithFilter sets a filter function for selective operations
func WithFilter(filter FilterFunc) CopyOption {
	return func(opts *copyOptions) {
		opts.filter = filter
	}
}

// WithProgress sets a progress handler
func WithProgress(handler ProgressFunc) CopyOption {
	return func(opts *copyOptions) {
		opts.progressHandler = handler
	}
}
