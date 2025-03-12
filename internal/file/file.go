package file

import (
	"os"
	"runtime"
	"strings"
)

// IsExist 文件/路径存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// IsNotExist 文件/路径不存在
func IsNotExist(path string) bool {
	_, err := os.Stat(path)
	return err != nil && os.IsNotExist(err)
}

// FolderName 本文件所在最下级文件夹名
func FolderName() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "<unk>"
	}
	i := strings.LastIndex(file, "/")
	if i <= 0 {
		return file
	}
	file = file[:i]
	i = strings.LastIndex(file, "/")
	if i <= 0 || i+1 >= len(file) {
		return file
	}
	return file[i+1:]
}
