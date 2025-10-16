package fileutils

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
	Src       string
	Dest      string
	Overwrite bool
	DryRun    bool
}

func MoveFile(conf MoveConfig) bool {
	if conf.Src == conf.Dest {
		return false
	}
	if !conf.Overwrite {
		_, err := os.Stat(conf.Dest)
		if err == nil {
			slog.Warn("Copy cancelled",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.String("err", "file already exists"))
			return false
		} else if !os.IsNotExist(err) {
			slog.Error("Move Error",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.Any("err", err))
			return false
		}
	}
	destDir := filepath.Dir(conf.Dest)
	slog.Info("Move", slog.String("src", conf.Src), slog.String("to", destDir))

	if conf.DryRun {
		return false
	}

	EnsureDir(destDir)
	err := os.Rename(conf.Src, conf.Dest)
	if err != nil {
		slog.Error("Move Error",
			slog.String("src", conf.Src),
			slog.String("dst", conf.Dest),
			slog.Any("err", err))
		return false
	}
	return true
}

func CopyFile(conf MoveConfig) bool {
	if conf.Src == conf.Dest {
		return false
	}
	if !conf.Overwrite {
		_, err := os.Stat(conf.Dest)
		if err == nil {
			slog.Warn("Copy cancelled",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.String("err", "file already exists"))
			return false
		} else if !os.IsNotExist(err) {
			slog.Error("Copy Error",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.Any("err", err))
			return false
		}
	}

	destDir := filepath.Dir(conf.Dest)
	EnsureDir(destDir)
	slog.Info("Copy",
		slog.String("src", conf.Src),
		slog.String("to", destDir))
	if conf.DryRun {
		return false
	}

	r, err := os.Open(conf.Src)
	if err != nil {
		slog.Error("Cannot open source file", slog.String("src", conf.Src), slog.Any("err", err))
		return false
	}
	defer r.Close()
	w, err := os.Create(conf.Dest)
	if err != nil {
		slog.Error("Cannot create destination file", slog.String("src", conf.Dest), slog.Any("err", err))
		return false
	}
	defer w.Close()
	written, err := io.Copy(w, r)
	if err != nil {
		slog.Error("Copy Error",
			slog.String("src", conf.Src),
			slog.String("dst", conf.Dest),
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
