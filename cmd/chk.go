package cmd

import (
	"aman/internal/audio"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func newChkCommand() *cobra.Command {
	f := chkFlags{
		bitrates:    []string{"flac", "320"},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "bchk <dir>",
		Short: "Checking missing audio files by bitrate",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			f.bitrates = lo.FlatMap(f.bitrates, func(item string, _ int) []string {
				return strings.Split(item, ",")
			})
			bitrates := lo.SliceToMap(f.bitrates, func(item string) (string, struct{}) {
				return item, struct{}{}
			})

			f.excludes = lo.FlatMap(f.excludes, func(item string, _ int) []string {
				return strings.Split(item, ",")
			})
			excludes := lo.SliceToMap(f.excludes, func(item string) (string, struct{}) {
				return item, struct{}{}
			})

			lock := sync.Mutex{}
			checkMap := make(map[string]map[string]struct{})
			cnt := atomic.Int64{}

			slog.Info("Scanning audio files...")
			done := make(chan struct{})
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				for {
					select {
					case <-done:
						ticker.Stop()
						return
					case _ = <-ticker.C:
						slog.Info("Scanning...", slog.Int64("files", cnt.Load()))
					}
				}
			}()

			_, err := audio.ProcessAudio(args[0], func(a audio.ProbedAudio) {
				cnt.Add(1)
				basename := filepath.Base(a.Filename)
				basename = strings.TrimSuffix(basename, filepath.Ext(basename))
				lock.Lock()
				defer lock.Unlock()
				check := checkMap[basename]
				if check == nil {
					check = make(map[string]struct{}, len(bitrates))
				}
				check[a.Group] = struct{}{}
				checkMap[basename] = check
			},
				audio.WithDepth(f.depth),
				audio.WithConcurrency(f.concurrency))

			done <- struct{}{}
			if err != nil {
				slog.Error("Error checking audio files", slog.Any("err", err))
				return
			}

			slog.Info("Checking missing audio bitrates...", slog.Int64("files", cnt.Load()))
			for file, availableBitrates := range checkMap {
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
					)
					continue
				}

				// Excludes list exist.
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
					)
				}
			}
		},
	}

	command.Flags().StringSliceVarP(&f.bitrates, "bitrates", "b", f.bitrates, "Bitrates to check for [flac, 320, 128]")
	command.Flags().StringSliceVarP(&f.excludes, "excludes", "e", f.excludes, "Bitrates to exclude from [flac, 320, 128]")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	return &command
}

type chkFlags struct {
	bitrates    []string
	excludes    []string
	depth       int
	concurrency int
}
