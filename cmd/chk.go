package cmd

import (
	"aman/internal/audio"
	"aman/internal/sliceutils"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

func newChkCommand() *cobra.Command {
	f := chkFlags{
		groups:      []string{audio.GroupFLAC},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "chk <dir>",
		Short: "Checking missing audio files by bitrate group",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			bitrates := sliceutils.FlatMapArgsToSet(f.groups)
			excludes := sliceutils.FlatMapArgsToSet(f.excludes)

			start := time.Now()
			slog.Info("Scanning audio files...")
			checkMap, cnt, err := audio.ScanMap(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
				if check == nil {
					check = make(map[string]string, len(bitrates))
				}

				if _, ok := check["_loc"]; !ok {
					check["_loc"] = a.Location()
				}
				check[a.Group] = ""
				return check
			},
				audio.WithDepth(f.depth),
				audio.WithConcurrency(f.concurrency),
				audio.WithProgress(true))
			if err != nil {
				slog.Error("Error checking audio files", slog.Any("err", err))
				return
			}

			missingCnt := 0
			slog.Info("Checking missing audio bitrates...", slog.Int64("files", cnt))
			for file, availableBitrates := range checkMap {
				location := availableBitrates["_loc"]
				delete(availableBitrates, "_loc")
				miss := make([]string, 0, len(bitrates))

				for bitrate := range bitrates {
					if _, ok := availableBitrates[bitrate]; ok {
						continue
					}
					miss = append(miss, bitrate)
				}

				if len(miss) == 0 {
					continue
				}

				avail := lo.Keys(availableBitrates)
				if len(excludes) == 0 {
					slog.Info("Missing bitrate",
						slog.String("basename", file),
						slog.String("available", strings.Join(avail, ",")),
						slog.String("missing", strings.Join(miss, ",")),
						slog.String("loc", location),
					)
					missingCnt++
					continue
				}

				// Excludes list exists.
				// Only show files that missing more than one bitrate.
				excluded := false
				for availBitRate := range availableBitrates {
					if _, ok := excludes[availBitRate]; !ok {
						excluded = true
						break
					}
				}
				if excluded {
					slog.Info("Missing bitrate",
						slog.String("basename", file),
						slog.String("available", strings.Join(avail, ",")),
						slog.String("missing", strings.Join(miss, ",")),
						slog.String("loc", location),
					)
					missingCnt++
				}
			}
			slog.Info("Checking completed",
				slog.Int64("count", cnt),
				slog.Int("missing", missingCnt),
				slog.String("took", time.Since(start).String()))
		},
	}

	command.Flags().StringSliceVarP(&f.groups, "groups", "g", f.groups, "groups to check for")
	command.Flags().StringSliceVarP(&f.excludes, "excludes", "e", f.excludes, "groups to exclude from")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "number of thread to use")
	return &command
}

type chkFlags struct {
	groups      []string
	excludes    []string
	depth       int
	concurrency int
}
