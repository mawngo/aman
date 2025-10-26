package fileutils

import (
	"encoding/csv"
	"errors"
	"github.com/dustin/go-humanize"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type CleanupFunc func() error

var createdDir sync.Map

type MoveConfig struct {
	Src       string
	Dest      string
	Overwrite bool
	DryRun    bool

	Indent int
}

func MoveFile(conf MoveConfig) bool {
	if conf.Src == conf.Dest {
		return false
	}

	indent := strings.Repeat("   ", conf.Indent)
	if !conf.Overwrite {
		_, err := os.Stat(conf.Dest)
		if err == nil {
			slog.Warn(indent+"Copy cancelled",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.String("err", "file already exists"))
			return false
		} else if !os.IsNotExist(err) {
			slog.Error(indent+"Move Error",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.Any("err", err))
			return false
		}
	}

	destDir := filepath.Dir(conf.Dest)
	destName := filepath.Base(conf.Dest)
	if destName == filepath.Base(conf.Src) {
		destName = destDir
	} else {
		destName = filepath.Join(destDir, destName)
	}
	slog.Info(indent+"Move", slog.String("src", conf.Src), slog.String("to", destName))

	if conf.DryRun {
		return false
	}

	if EnsureDir(destDir) {
		slog.Info(indent+"Created", slog.String("dir", destDir))
	}
	err := os.Rename(conf.Src, conf.Dest)
	if err != nil {
		slog.Error(indent+"Move Error",
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
	indent := strings.Repeat("   ", conf.Indent)
	if !conf.Overwrite {
		_, err := os.Stat(conf.Dest)
		if err == nil {
			slog.Warn(indent+"Copy cancelled",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.String("err", "file already exists"))
			return false
		} else if !os.IsNotExist(err) {
			slog.Error(indent+"Copy Error",
				slog.String("src", conf.Src),
				slog.String("dst", conf.Dest),
				slog.Any("err", err))
			return false
		}
	}

	destDir := filepath.Dir(conf.Dest)
	destName := filepath.Base(conf.Dest)
	if destName == filepath.Base(conf.Src) {
		destName = destDir
	} else {
		destName = filepath.Join(destDir, destName)
	}
	slog.Info(indent+"Copy",
		slog.String("src", conf.Src),
		slog.String("to", destName))

	if conf.DryRun {
		return false
	}

	if EnsureDir(destDir) {
		slog.Info(indent+"Created", slog.String("dir", destDir))
	}
	r, err := os.Open(conf.Src)
	if err != nil {
		slog.Error(indent+"Cannot open source file", slog.String("src", conf.Src), slog.Any("err", err))
		return false
	}
	defer r.Close()
	w, err := os.Create(conf.Dest)
	if err != nil {
		slog.Error(indent+"Cannot create destination file", slog.String("src", conf.Dest), slog.Any("err", err))
		return false
	}
	defer w.Close()
	written, err := io.Copy(w, r)
	if err != nil {
		slog.Error(indent+"Copy Error",
			slog.String("src", conf.Src),
			slog.String("dst", conf.Dest),
			slog.String("written", humanize.Bytes(uint64(written))),
			slog.Any("err", err))
		return false
	}
	return true
}

func EnsureDir(dir string) bool {
	if _, ok := createdDir.Load(dir); ok {
		return false
	}
	if _, serr := os.Stat(dir); serr != nil {
		if !os.IsNotExist(serr) {
			panic(serr)
		}

		merr := os.MkdirAll(dir, os.ModePerm)
		if merr != nil {
			panic(merr)
		}
		createdDir.Store(dir, struct{}{})
		return true
	}
	// Already exists.
	return false
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

func CreateCsvFile(path string) (writer *csv.Writer, cleanup CleanupFunc, err error) {
	file, err := os.Create(path)
	if err != nil {
		return
	}

	writer = csv.NewWriter(file)
	cleanup = func() (err error) {
		defer func() {
			if ferr := file.Close(); ferr != nil {
				if err != nil {
					err = errors.Join(err, ferr)
				}
			}
		}()

		writer.Flush()
		err = writer.Error()
		return err
	}
	return writer, cleanup, nil
}
