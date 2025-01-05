package audio

import (
	"errors"
	"github.com/charlievieth/fastwalk"
	"github.com/samber/lo"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

func WithForcedDir(name string) ProcessAudioOption {
	return func(config *processAudioConfig) {
		config.forcedDir = name
	}
}

func ProcessAudio(root string, handler func(audio ProbedAudio), opts ...ProcessAudioOption) error {
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
		return err
	}

	if !stats.IsDir() {
		audio, err := Probe(root)
		if err != nil {
			return err
		}
		handler(audio)
		return nil
	}
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
			if enabled, ok := Groups[d.Name()]; ok && d.Name() != conf.forcedDir && enabled {
				return fs.SkipDir
			}
			return nil
		}

		audio, err := Probe(path)
		if err != nil {
			return nil
		}
		handler(audio)
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipDir) {
		return err
	}
	return nil
}
