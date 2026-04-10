package cmd

import (
	"aman/internal/audio"
	"aman/internal/sliceutils"
	"context"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"
)

func newFillCommand() *cobra.Command {
	f := fillFlags{
		groups:      []string{audio.Group320Mp3},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:     "conv <dir>",
		Short:   "Fill missing audio bitrate groups by converting from FLAC",
		Aliases: []string{"fill"},
		Args:    cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			f.groups = sliceutils.FlatMapArgs(f.groups)

			start := time.Now()
			checkMap, cnt, err := audio.ScanMap(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
				if check == nil {
					check = make(map[string]string, len(f.groups))
				}

				if a.Group == audio.GroupFLAC {
					check[audio.GroupFLAC] = a.Filename
					return check
				}

				check[a.Group] = ""
				return check
			},
				audio.WithDepth(f.depth),
				audio.WithProgress(true),
				audio.WithConcurrency(f.concurrency))

			if err != nil {
				slog.Error("Error scanning audio files", slog.Any("err", err))
				return
			}

			slog.Info("Finding missing files to fill...", slog.Int64("files", cnt))
			jobs := make([]convertMeta, 0, 10)
			for basename, availableBitrates := range checkMap {
				if _, ok := availableBitrates[audio.GroupFLAC]; !ok {
					continue
				}

				source := availableBitrates[audio.GroupFLAC]
				delete(availableBitrates, audio.GroupFLAC)
				if source == "" {
					slog.Warn("Missing source", slog.String("file", basename))
					return
				}

				meta := convertMeta{
					Source:   source,
					Basename: basename,
					Missing:  make([]string, 0, len(f.groups)),
				}
				if !f.overwrite {
					for _, bitrate := range f.groups {
						if _, ok := availableBitrates[bitrate]; ok {
							continue
						}
						meta.Missing = append(meta.Missing, bitrate)
					}
				} else {
					meta.Missing = f.groups
				}
				if len(meta.Missing) == 0 {
					continue
				}
				jobs = append(jobs, meta)
			}

			sema := semaphore.NewWeighted(int64(f.concurrency))
			convertedCnt := atomic.Int64{}
			for _, job := range jobs {
				lo.Must0(sema.Acquire(context.Background(), 1))
				go func() {
					defer sema.Release(1)
					cnt := audio.Convert(audio.ConvertConfig{
						Src:          job.Source,
						TargetGroups: job.Missing,
						Overwrite:    f.overwrite,
						DryRun:       f.dryRun,
					})
					convertedCnt.Add(cnt)
				}()
			}

			if err := sema.Acquire(context.Background(), int64(f.concurrency)); err != nil {
				slog.Error("Error waiting for conversion to complete",
					slog.Any("err", err),
					slog.Int64("count", cnt),
					slog.Int("total", len(jobs)),
					slog.Int64("completed", convertedCnt.Load()),
					slog.String("took", time.Since(start).String()))
				return
			}
			slog.Info("Fill completed",
				slog.Int64("count", cnt),
				slog.Int("total", len(jobs)),
				slog.Int64("converted", convertedCnt.Load()),
				slog.String("took", time.Since(start).String()))
		},
	}

	command.Flags().StringSliceVarP(&f.groups, "groups", "b", f.groups, "bitrate groups to fill for")
	command.Flags().BoolVarP(&f.overwrite, "overwrite", "w", f.overwrite, "overwrite existing files")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "number of thread to use")
	command.Flags().BoolVar(&f.dryRun, "dry-run", f.dryRun, "test run without converting files")
	return &command
}

type fillFlags struct {
	groups      []string
	depth       int
	concurrency int
	dryRun      bool
	overwrite   bool
}

type convertMeta struct {
	Source   string
	Basename string
	Missing  []string
}
