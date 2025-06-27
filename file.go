package fsx

import "os"

func FileExist(path string) bool {
	stat, _ := os.Stat(path)
	if stat == nil {
		return false
	}

	return !stat.IsDir()
}

func AnyFileExist(paths ...string) bool {
	if len(paths) == 0 {
		return false
	}

	for _, path := range paths {
		if FileExist(path) {
			return true
		}
	}

	return false
}
