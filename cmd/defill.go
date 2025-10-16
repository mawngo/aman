package cmd

import (
	"aman/internal/audio"
	"aman/internal/sliceutils"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

func newDeFillCommand() *cobra.Command {
	f := fillFlags{
		groups:      []string{audio.Group256aac, audio.Group128Mp3, audio.LowQuality},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "defill <dir>",
		Short: "Remove audio of bitrate groups if it already exists in other quality",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			bitrates := sliceutils.FlatMapArgsToSet(f.groups)
			delete(bitrates, audio.GroupFLAC)

			start := time.Now()
			checkMap, cnt, err := audio.ScanMap(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
				if check == nil {
					check = make(map[string]string, len(bitrates))
				}
				check[a.Group] = a.Filename
				return check
			},
				audio.WithDepth(f.depth),
				audio.WithProgress(true),
				audio.WithConcurrency(f.concurrency))

			if err != nil {
				slog.Error("Error scanning audio files", slog.Any("err", err))
				return
			}

			slog.Info("Finding redundant audio files...", slog.Int64("files", cnt))
			dirs := make(map[string]struct{})
			rmCnt := int64(0)

			for basename, availableBitrates := range checkMap {
				maxQuality := 0
				for bitrate := range availableBitrates {
					maxQuality = max(maxQuality, audio.Groups[bitrate])
				}

				files := make(map[string]string, len(bitrates))
				for bitrate := range bitrates {
					if _, ok := availableBitrates[bitrate]; !ok {
						continue
					}
					if quality := audio.Groups[bitrate]; quality >= maxQuality {
						continue
					}

					file := availableBitrates[bitrate]
					files[bitrate] = file
					dirs[audio.LocationDir(file)] = struct{}{}
					delete(availableBitrates, bitrate)
				}

				if len(files) > 0 {
					slog.Info("Removing audio",
						slog.String("basename", basename),
						slog.String("removes", strings.Join(lo.Keys(files), ",")),
						slog.String("keep", strings.Join(lo.Keys(availableBitrates), ",")),
					)
					for group, file := range files {
						if !f.dryRun {
							if err := os.Remove(file); err != nil {
								slog.Error("Error removing file",
									slog.String("file", file),
									slog.Any("err", err))
								continue
							}
						}
						rmCnt++
						slog.Info("> Removed",
							slog.String("file", file),
							slog.String("group", group))
					}
				}
			}

			if !f.dryRun {
				RemoveAllEmptyGroupDirs(dirs)
			}

			slog.Info("Remove completed",
				slog.Int64("count", cnt),
				slog.Int("total", len(checkMap)),
				slog.Int64("removed", rmCnt),
				slog.String("took", time.Since(start).String()))
		},
	}

	command.Flags().StringSliceVarP(&f.groups, "groups", "b", f.groups, "Bitrate groups to remove from")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	command.Flags().BoolVar(&f.dryRun, "dry-run", f.dryRun, "Test run without converting files")
	return &command
}
