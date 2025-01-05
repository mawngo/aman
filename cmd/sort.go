package cmd

import (
	"aman/internal/audio"
	"aman/internal/utils"
	"errors"
	"github.com/spf13/cobra"
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
		rootLevel:   "320",
		depth:       5,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "sort <dir>",
		Short: "Organize audio files by bitrate",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
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
			count := atomic.Int64{}
			parents := sync.Map{}
			err := audio.ProcessAudio(args[0], func(a audio.ProbedAudio) {
				parentDir := filepath.Dir(a.Filename)
				parentDirName := filepath.Base(parentDir)
				if _, ok := audio.Groups[parentDirName]; ok {
					parentDir = filepath.Dir(parentDir)
				}
				group := a.Group
				if group == f.rootLevel {
					group = ""
				}

				parents.Store(parentDir, struct{}{})
				dest := filepath.Join(parentDir, group, filepath.Base(a.Filename))
				if utils.MoveFile(a.Filename, dest) {
					count.Add(1)
				}
			},
				audio.WithDepth(f.depth),
				audio.WithConcurrency(f.concurrency),
				audio.WithForcedDir(f.rootLevel))
			if err != nil {
				slog.Error("Error sorting audio files", slog.Any("err", err))
				return
			}
			parents.Range(func(parent, _ any) bool {
				for group := range audio.Groups {
					dir := filepath.Join(parent.(string), group)
					if !removableGroupDir(dir) {
						return true
					}
					if err := os.RemoveAll(dir); err != nil {
						slog.Error("Error removing empty directory", slog.String("dir", dir), slog.Any("err", err))
						return true
					}
					slog.Info("Removed empty directory", slog.String("dir", dir))
				}
				return true
			})
			slog.Info("Audio files sorted",
				slog.Int64("moved", count.Load()),
				slog.String("took", time.Since(start).String()))
		},
	}
	command.Flags().StringVarP(&f.rootLevel, "root-group", "g", f.rootLevel, "Group of audio that will be placed at root directory")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	return &command
}

type sortFlags struct {
	rootLevel   string
	depth       int
	concurrency int
}

func removableGroupDir(path string) bool {
	s, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if !s.IsDir() {
		return false
	}
	size := utils.DirSize(path)
	// Remove if size < 100Kb.
	// We don't check for empty because there may be hidden albums/cover files.
	return size < 100_000
}
