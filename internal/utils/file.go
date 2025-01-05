package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var createdDir sync.Map

type MoveConfig struct {
	Src string
	Dst string
}

func MoveFile(src string, dest string) bool {
	if src == dest {
		return false
	}
	ensureParentDir(dest)
	slog.Info("Move", slog.String("src", src), slog.String("dst", dest))
	err := os.Rename(src, dest)
	if err != nil {
		slog.Error("Move Error", slog.String("src", src), slog.String("dst", dest), slog.Any("err", err))
	}
	return true
}

func ensureParentDir(file string) {
	dir := filepath.Dir(file)
	if _, ok := createdDir.Load(dir); ok {
		return
	}
	if _, serr := os.Stat(dir); serr != nil {
		merr := os.MkdirAll(dir, os.ModePerm)
		if merr != nil {
			panic(merr)
		}
		slog.Info("Created", slog.String("dir", dir))
		createdDir.Store(dir, struct{}{})
	}
}

func DirSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})

	if err != nil {
		panic(err)
	}
	return size
}
