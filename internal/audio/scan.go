package audio

import (
	"aman/internal/sliceutils"
	"errors"
	"github.com/charlievieth/fastwalk"
	"github.com/samber/lo"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var SupportedExtensions = []string{".flac", ".mp3", ".m4a"}

type processAudioConfig struct {
	depth       int
	concurrency int
	sort        fastwalk.SortMode
	progress    bool
	excludeDirs map[string]struct{}
}

type ProcessAudioOption func(*processAudioConfig)

func WithDepth(depth int) ProcessAudioOption {
	return func(config *processAudioConfig) {
		config.depth = depth
	}
}

func WithConcurrency(concurrency int) ProcessAudioOption {
	return func(config *processAudioConfig) {
		config.concurrency = concurrency
	}
}

func WithProgress(show bool) ProcessAudioOption {
	return func(config *processAudioConfig) {
		config.progress = show
	}
}

func WithSkipDirectories(dirs ...string) ProcessAudioOption {
	return func(config *processAudioConfig) {
		config.excludeDirs = sliceutils.ToSet(dirs)
	}
}

func Scan(root string, handler func(audio ProbedAudio), opts ...ProcessAudioOption) (int64, error) {
	conf := processAudioConfig{
		depth:       5,
		concurrency: 0,
		sort:        fastwalk.SortDirsFirst,
	}
	for _, opt := range opts {
		opt(&conf)
	}

	stats, err := os.Stat(root)
	if err != nil {
		return 0, err
	}

	if !stats.IsDir() {
		audio, err := Probe(root)
		if err != nil {
			return 0, err
		}
		handler(audio)
		return 1, nil
	}

	count := atomic.Int64{}
	done := make(chan struct{})
	if conf.progress {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-done:
					ticker.Stop()
					return
				case <-ticker.C:
					slog.Info("Scanning audios...", slog.Int64("files", count.Load()))
				}
			}
		}()
	}

	c := &fastwalk.Config{NumWorkers: conf.concurrency, Follow: true, Sort: conf.sort}
	err = fastwalk.Walk(c, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.Clean(path)
		// Check depth limit.
		rel := lo.Must(filepath.Rel(root, path))
		if conf.depth >= 0 && d.IsDir() && strings.Count(rel, string(os.PathSeparator)) > conf.depth {
			return fs.SkipDir
		}

		if d.IsDir() {
			if _, ok := conf.excludeDirs[d.Name()]; ok {
				return fs.SkipDir
			}
			return nil
		}

		if !slices.Contains(SupportedExtensions, filepath.Ext(path)) {
			slog.Debug("Not supported file", slog.Any("file", path))
			return nil
		}

		audio, err := Probe(path)
		if err != nil {
			slog.Error("Error probing audio",
				slog.String("file", path),
				slog.Any("err", err))
			return nil
		}
		count.Add(1)
		handler(audio)
		return nil
	})

	if conf.progress {
		done <- struct{}{}
	}
	if err != nil && !errors.Is(err, fs.SkipDir) {
		return count.Load(), err
	}
	return count.Load(), nil
}

func ScanMap[T any](root string, handler func(m T, audio ProbedAudio) T, opts ...ProcessAudioOption) (map[string]T, int64, error) {
	res := make(map[string]T, 100)
	lock := sync.Mutex{}
	count, err := Scan(root, func(audio ProbedAudio) {
		basename := audio.Basename()
		lock.Lock()
		defer lock.Unlock()
		v, ok := res[basename]
		if !ok {
			var n T
			v = n
		}
		v = handler(v, audio)
		res[basename] = v
	}, opts...)
	return res, count, err
}
