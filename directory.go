package fsx

import "os"

func DirectoryExist(path string) bool {
	stat, _ := os.Stat(path)
	if stat == nil {
		return false
	}

	return stat.IsDir()
}
