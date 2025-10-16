package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"errors"
	"github.com/spf13/cobra"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func newSortCommand() *cobra.Command {
	f := sortFlags{
		rootLevel:   audio.Group320Mp3,
		depth:       5,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "sort <dir>",
		Short: "Organize audio files to groups by bitrate",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if f.rootLevel != "" {
				if enabled, ok := audio.Groups[f.rootLevel]; !ok || !enabled {
					supported := make([]string, 0, len(audio.Groups))
					for group, enabled := range audio.Groups {
						if enabled {
							supported = append(supported, group)
						}
					}
					slog.Error("Invalid root group",
						slog.String("group", f.rootLevel),
						slog.String("supported", strings.Join(supported, ", ")))
					return
				}
			}

			slog.Info("Scanning audio files...")
			start := time.Now()
			moved := atomic.Int64{}
			lock := sync.Mutex{}
			parents := make(map[string]struct{}, 100)
			count, err := audio.ProcessAudio(args[0], func(a audio.ProbedAudio) {
				parentDir := filepath.Dir(a.Filename)
				parentDirName := filepath.Base(parentDir)
				if _, ok := audio.Groups[parentDirName]; ok {
					parentDir = filepath.Dir(parentDir)
				}
				lock.Lock()
				parents[parentDir] = struct{}{}
				lock.Unlock()

				group := a.Group
				if group == f.rootLevel {
					group = ""
				}
				a.Print()
				mv := fileutils.MoveConfig{
					Src:    a.Filename,
					Dest:   filepath.Join(parentDir, group, filepath.Base(a.Filename)),
					DryRun: f.dryRun,
				}
				if fileutils.MoveFile(mv) {
					moved.Add(1)
				}
			},
				audio.WithDepth(f.depth),
				audio.WithConcurrency(f.concurrency))
			if err != nil {
				slog.Error("Error sorting audio files", slog.Any("err", err))
				return
			}

			if !f.dryRun {
				for parent := range parents {
					for group := range audio.Groups {
						dir := filepath.Join(parent, group)
						if !removableGroupDir(dir) {
							continue
						}
						if err := os.RemoveAll(dir); err != nil {
							slog.Error("Error removing empty directory", slog.String("dir", dir), slog.Any("err", err))
							continue
						}
						slog.Info("Removed empty directory", slog.String("dir", dir))
					}
				}
			}

			slog.Info("Audio files sorted",
				slog.Int64("count", count),
				slog.Int64("moved", moved.Load()),
				slog.String("took", time.Since(start).String()))
		},
	}
	command.Flags().StringVarP(&f.rootLevel, "root-group", "g", f.rootLevel, "Group of audio that will be placed at root directory")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	command.Flags().BoolVar(&f.dryRun, "dry-run", f.dryRun, "Test run without moving files")
	return &command
}

type sortFlags struct {
	rootLevel   string
	depth       int
	concurrency int
	dryRun      bool
}

func removableGroupDir(path string) bool {
	s, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if !s.IsDir() {
		return false
	}
	size := fileutils.DirSize(path)
	// Remove if size < 100Kb.
	// We don't check for empty because there may be hidden albums/cover files.
	return size < 100_000
}
