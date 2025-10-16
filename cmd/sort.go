package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"errors"
	"github.com/samber/lo"
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
		rootLevel:   audio.GroupFLAC,
		depth:       5,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "sort <dir>",
		Short: "Organize audio files to groups by bitrate",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if f.rootLevel != "" {
				if _, ok := audio.Groups[f.rootLevel]; !ok {
					slog.Error("Invalid root group",
						slog.String("group", f.rootLevel),
						slog.String("supported", strings.Join(lo.Keys(audio.Groups), ", ")))
					return
				}
			}

			slog.Info("Scanning audio files...")
			start := time.Now()
			moved := atomic.Int64{}
			lock := sync.Mutex{}
			parents := make(map[string]struct{}, 100)
			bestMap := make(map[string]map[string]string, 100)

			count, err := audio.Scan(args[0], func(a audio.ProbedAudio) {
				location := a.Location()
				basename := a.Basename()

				lock.Lock()
				parents[location] = struct{}{}
				if f.best {
					// Marking alternative best quality audio files if enabled.
					best, ok := bestMap[basename]
					if !ok {
						best = make(map[string]string)
					}
					if best != nil {
						best[a.Group] = a.Filename
						bestMap[basename] = best
					}
				}

				group := a.Group
				if group == f.rootLevel {
					group = ""
					// Reset the best map if the root group is found.
					bestMap[basename] = nil
				}
				lock.Unlock()

				if !f.quiet {
					a.Print()
				}
				mv := fileutils.MoveConfig{
					Src:    a.Filename,
					Dest:   filepath.Join(location, group, filepath.Base(a.Filename)),
					DryRun: f.dryRun,
				}
				if fileutils.MoveFile(mv) {
					moved.Add(1)
				}
			},
				audio.WithDepth(f.depth),
				audio.WithProgress(f.quiet),
				audio.WithConcurrency(f.concurrency))
			if err != nil {
				slog.Error("Error sorting audio files", slog.Any("err", err))
				return
			}

			if f.best {
				moveBackBestFile(bestMap, f)
			}

			if !f.dryRun {
				RemoveAllEmptyGroupDirs(parents)
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
	command.Flags().BoolVar(&f.quiet, "quiet", f.quiet, "Only show moved files")
	command.Flags().BoolVar(&f.best, "best", f.best, "Move alternative best quality audio files to root directory (except FLAC)")
	return &command
}

type sortFlags struct {
	rootLevel   string
	depth       int
	concurrency int
	dryRun      bool
	quiet       bool
	best        bool
}

func moveBackBestFile(bestMap map[string]map[string]string, f sortFlags) {
	moveBack := func(file string) {
		mv := fileutils.MoveConfig{
			Src:    file,
			Dest:   filepath.Join(audio.LocationDir(file), filepath.Base(file)),
			DryRun: f.dryRun,
		}
		fileutils.MoveFile(mv)
	}

	for _, best := range bestMap {
		if best == nil {
			continue
		}

		if len(best) > 1 {
			// Does not include FLAC when there are other alternates.
			delete(best, audio.GroupFLAC)
		}

		if len(best) == 1 {
			for _, file := range best {
				// Get the first (only) file.
				moveBack(file)
				break
			}
			continue
		}

		// Select for the best alternative.
		bestQuality := 0
		bestFile := ""
		for bitrate, file := range best {
			quality := audio.Groups[bitrate]
			if quality <= bestQuality {
				continue
			}
			bestQuality = quality
			bestFile = file
		}
		moveBack(bestFile)
	}
}

func RemoveAllEmptyGroupDirs[T any](dirs map[string]T) {
	for parent := range dirs {
		for group := range audio.Groups {
			dir := filepath.Join(parent, group)
			if !removableGroupDir(dir) {
				continue
			}
			if err := os.RemoveAll(dir); err != nil {
				slog.Error("Error removing empty directory",
					slog.String("dir", dir), slog.Any("err", err))
				continue
			}
			slog.Info("Removed empty directory", slog.String("dir", dir))
		}
	}
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
