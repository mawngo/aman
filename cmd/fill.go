package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"context"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
		Use:   "fill <dir>",
		Short: "Fill missing audio bitrate groups by converting from FLAC (require ffmpeg)",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			f.groups = lo.FlatMap(f.groups, func(item string, _ int) []string {
				return strings.Split(item, ",")
			})

			start := time.Now()
			checkMap, cnt, err := audio.ProcessMapAudio(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
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
				convertAudio(job, sema, &convertedCnt, f.dryRun, f.overwrite)
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

	command.Flags().StringSliceVarP(&f.groups, "groups", "b", f.groups, "Bitrate groups to fill for")
	command.Flags().BoolVarP(&f.overwrite, "overwrite", "w", f.overwrite, "Overwrite existing files")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	command.Flags().BoolVar(&f.dryRun, "dry-run", f.dryRun, "Test run without converting files")
	return &command
}

func convertAudio(job convertMeta, sema *semaphore.Weighted, cnt *atomic.Int64, dryRun bool, overwrite bool) {
	parentDir := filepath.Dir(job.Source)
	if _, ok := audio.Groups[filepath.Base(parentDir)]; ok {
		parentDir = filepath.Dir(parentDir)
	}

	missing := lo.Uniq(job.Missing)
	for _, group := range missing {
		if !strings.HasSuffix(group, audio.TypeMp3) {
			slog.Error("Output format not supported", slog.String("basename", job.Source), slog.String("group", group))
			continue
		}

		dest := filepath.Join(parentDir, group, job.Basename+"."+audio.TypeMp3)
		lo.Must0(sema.Acquire(context.Background(), 1))
		if !overwrite {
			if _, err := os.Stat(dest); err == nil {
				slog.Warn("File already exists", slog.String("file", dest))
				continue
			} else if !os.IsNotExist(err) {
				slog.Error("Error checking file", slog.String("file", dest), slog.Any("err", err))
				continue
			}
		}

		go func() {
			defer sema.Release(1)
			slog.Info("Converting",
				slog.String("basename", job.Basename),
				slog.String("group", group))

			dir := filepath.Dir(dest)
			if !dryRun {
				cmd := exec.Command("ffmpeg",
					"-v", "error",
					"-i", job.Source,
					"-ab", "320k",
					"-map_metadata", "0",
					"-id3v2_version", "3",
					dest)

				fileutils.EnsureDir(dir)
				if err := cmd.Run(); err != nil {
					slog.Error("Error converting",
						slog.String("basename", job.Basename),
						slog.String("group", group),
						slog.String("todir", dir),
						slog.Any("err", err))
					return
				}
				cnt.Add(1)
			}

			slog.Info("Converted",
				slog.String("basename", job.Basename),
				slog.String("group", group),
				slog.String("todir", dir))
		}()
	}
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
