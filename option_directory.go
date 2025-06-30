package fsx

import "os"

// DirectoryOption represents optional parameters for directory operations
type DirectoryOption func(*directoryOptions)

type directoryOptions struct {
	perm      os.FileMode
	recursive bool
	force     bool
}

// defaultDirectoryOptions returns default options for directory operations
func defaultDirectoryOptions() *directoryOptions {
	return &directoryOptions{
		perm:      0755,
		recursive: false,
		force:     false,
	}
}

// WithDirPermissions sets custom directory permissions
func WithDirPermissions(perm os.FileMode) DirectoryOption {
	return func(opts *directoryOptions) {
		opts.perm = perm
	}
}

// WithRecursive enables recursive operations
func WithRecursive() DirectoryOption {
	return func(opts *directoryOptions) {
		opts.recursive = true
	}
}

// WithForce forces operation (e.g., delete non-empty directories)
func WithForce() DirectoryOption {
	return func(opts *directoryOptions) {
		opts.force = true
	}
}
