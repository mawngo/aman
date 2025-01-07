package utils

import (
	"github.com/dustin/go-humanize"
	"io"
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
	destDir := filepath.Dir(dest)
	EnsureDir(destDir)
	slog.Info("Move", slog.String("src", src), slog.String("to", destDir))
	err := os.Rename(src, dest)
	if err != nil {
		slog.Error("Move Error", slog.String("src", src), slog.String("dst", dest), slog.Any("err", err))
	}
	return true
}

func CopyFile(src string, dest string) bool {
	destDir := filepath.Dir(dest)
	EnsureDir(destDir)
	slog.Info("Copy", slog.String("src", src), slog.String("to", destDir))

	r, err := os.Open(src)
	if err != nil {
		slog.Error("Cannot open source file", slog.String("src", src), slog.Any("err", err))
		return false
	}
	defer r.Close()
	w, err := os.Create(dest)
	if err != nil {
		slog.Error("Cannot create destination file", slog.String("src", src), slog.Any("err", err))
		return false
	}
	defer w.Close()
	written, err := io.Copy(w, r)
	if err != nil {
		slog.Error("Move Error",
			slog.String("src", src),
			slog.String("dst", dest),
			slog.String("written", humanize.Bytes(uint64(written))),
			slog.Any("err", err))
		return false
	}
	return true
}

func EnsureDir(dir string) {
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
