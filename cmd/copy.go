package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"aman/internal/sliceutils"
	"context"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func newCopyCommand() *cobra.Command {
	f := copyFlags{
		order:       []string{audio.GroupFLAC, audio.Group320aac, audio.Group320Mp3},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "copy <src> <target>",
		Short: "Selectively copy music files by bitrate group",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			f.order = sliceutils.FlatMapArgs(f.order)
			bitrates := sliceutils.ToSet(f.order)
			excludes := sliceutils.FlatMapArgsToSet(f.excludes)

			target := args[1]
			if s, err := os.Stat(target); err == nil && !s.IsDir() {
				slog.Error("Target is not a directory", slog.String("target", target))
				return
			}

			start := time.Now()
			checkMap, cnt, err := audio.ScanMap(args[0], func(best audio.ProbedAudio, a audio.ProbedAudio) audio.ProbedAudio {
				if _, ok := excludes[a.Group]; ok {
					return best
				}
				_, included := bitrates[a.Group]
				if f.strict && !included {
					return best
				}

				if best.Filename == "" {
					// Initialize.
					return a
				}

				// Compare.
				_, bestIncluded := bitrates[best.Group]
				if bestIncluded && !included {
					return best
				}
				if !bestIncluded && included {
					return a
				}

				if audio.Groups[a.Group] > audio.Groups[best.Group] {
					return a
				}
				return best
			},
				audio.WithDepth(f.depth),
				audio.WithProgress(true),
				audio.WithConcurrency(f.concurrency))

			if err != nil {
				slog.Error("Error scanning audio files", slog.Any("err", err))
				return
			}

			slog.Info("Copying files",
				slog.Int64("files", cnt),
				slog.String("order", strings.Join(f.order, ">")))

			copied := atomic.Int64{}
			sema := semaphore.NewWeighted(int64(f.concurrency))
			for _, a := range checkMap {
				if a.Filename == "" {
					continue
				}

				filename := filepath.Base(a.Filename)
				dest := filepath.Join(target, filename)
				if !f.flat {
					rel := lo.Must(filepath.Rel(args[0], a.Location()))
					dest = filepath.Join(target, rel, filename)
				}

				lo.Must0(sema.Acquire(context.Background(), 1))
				go func() {
					defer sema.Release(1)
					ok := fileutils.CopyFile(fileutils.MoveConfig{
						Src:    a.Filename,
						Dest:   dest,
						DryRun: f.dryRun,
					})
					if ok {
						copied.Add(1)
					}
				}()
			}

			if err := sema.Acquire(context.Background(), int64(f.concurrency)); err != nil {
				slog.Error("Error waiting for copy to complete",
					slog.Any("err", err),
					slog.Int64("count", cnt),
					slog.Int("total", len(checkMap)),
					slog.Int64("copied", copied.Load()),
					slog.String("took", time.Since(start).String()))
				return
			}

			slog.Info("Audio files copied",
				slog.Int64("count", cnt),
				slog.Int("total", len(checkMap)),
				slog.Int64("copied", copied.Load()),
				slog.String("took", time.Since(start).String()))
		},
	}
	command.Flags().StringSliceVarP(&f.order, "order", "o", f.order, "Preferred quality (group) order")
	command.Flags().StringSliceVarP(&f.order, "excludes", "e", f.excludes, "Excluded quality group")
	command.Flags().BoolVar(&f.strict, "strict", f.strict, "Only copy files with the extract quality specified")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().BoolVar(&f.flat, "flat", f.flat, "Flatten directory structure")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	command.Flags().BoolVar(&f.dryRun, "dry-run", f.dryRun, "Test run without coping files")
	return &command
}

type copyFlags struct {
	order       []string
	excludes    []string
	flat        bool
	depth       int
	concurrency int
	dryRun      bool
	convert     bool
	strict      bool
}
