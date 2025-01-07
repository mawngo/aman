package audio

import (
	"errors"
	"github.com/charlievieth/fastwalk"
	"github.com/samber/lo"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type processAudioConfig struct {
	depth       int
	concurrency int
	sort        fastwalk.SortMode
	forcedDir   string
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

func ProcessAudio(root string, handler func(audio ProbedAudio), opts ...ProcessAudioOption) (int64, error) {
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
	c := &fastwalk.Config{NumWorkers: conf.concurrency, Follow: true, Sort: conf.sort}
	err = fastwalk.Walk(c, root, func(path string, d fs.DirEntry, err error) error {
		path = lo.Must(filepath.Rel(root, path))
		if err != nil {
			return err
		}

		// Check depth limit.
		if conf.depth >= 0 && d.IsDir() && strings.Count(path, string(os.PathSeparator)) > conf.depth {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		audio, err := Probe(path)
		if err != nil {
			slog.Debug("Error probing audio", slog.Any("err", err))
			return nil
		}
		count.Add(1)
		handler(audio)
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipDir) {
		return count.Load(), err
	}
	return count.Load(), nil
}
