package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func Assert(err error) {
	if err != nil {
		panic(err)
	}
}

func RunningDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Dir(currentFile)
}

func IsExist(path string) bool {
	f, e := os.Open(path)
	defer f.Close()
	return os.IsExist(e)
}

func IsNotExist(path string) bool {
	f, e := os.Open(path)
	defer f.Close()
	return os.IsNotExist(e)
}

func IsPermission(path string) bool {
	f, e := os.Open(path)
	defer f.Close()
	return os.IsPermission(e)
}

func EnsureDirs(paths ...string) {
	for _, path := range paths {
		if IsNotExist(path) {
			Assert(os.MkdirAll(path, os.ModePerm))
		}
	}
}

func ForceCopyFile(srcName, dstName string) (written int64, err error) {
	src, err := os.Open(srcName)
	Assert(err)
	defer src.Close()

	info, err := src.Stat()
	Assert(err)
	mode := info.Mode()

	dst, err := os.Create(dstName)
	Assert(err)
	defer dst.Close()

	Assert(os.Chmod(dstName, mode))

	return io.Copy(dst, src)
}

func CopyDir(srcDir, dstDir string) error {
	copypath := func(basepath, targpath, newBasepath string) string {
		relpath, _ := filepath.Rel(basepath, targpath)
		return filepath.Join(newBasepath, relpath)
	}

	walk := func(oriPath string, info os.FileInfo, err error) error {
		newPath := copypath(srcDir, oriPath, dstDir)
		if info.IsDir() {
			return os.MkdirAll(newPath, info.Mode())
		} else {
			_, err := ForceCopyFile(oriPath, newPath)
			return err
		}
	}
	return filepath.Walk(srcDir, walk)
}
