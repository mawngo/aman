package cmd

import (
	"aman/internal/audio"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func newLsCommand() *cobra.Command {
	f := chkFlags{
		groups:      []string{"*"},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "ls <dir>",
		Short: "List audio files",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			f.groups = lo.FlatMap(f.groups, func(item string, _ int) []string {
				return strings.Split(item, ",")
			})
			includes := lo.SliceToMap(f.groups, func(item string) (string, struct{}) {
				return item, struct{}{}
			})
			listAll := false
			if _, ok := includes["*"]; ok {
				listAll = true
			}

			f.excludes = lo.FlatMap(f.excludes, func(item string, _ int) []string {
				return strings.Split(item, ",")
			})
			excludes := lo.SliceToMap(f.excludes, func(item string) (string, struct{}) {
				return item, struct{}{}
			})

			start := time.Now()
			slog.Info("Scanning audio files...")
			checkMap, cnt, err := audio.ProcessMapAudio(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
				if _, ok := excludes[a.Group]; ok {
					return check
				}
				if !listAll {
					if _, ok := includes[a.Group]; !ok {
						return check
					}
				}
				if check == nil {
					check = make(map[string]string, len(includes))
				}
				if _, ok := check["_loc"]; !ok {
					loc := filepath.Dir(a.Filename)
					if _, ok := audio.Groups[filepath.Base(loc)]; ok {
						loc = filepath.Dir(loc)
					}
					check["_loc"] = loc
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

			for basename, bitrates := range checkMap {
				loc := bitrates["_loc"]
				delete(bitrates, "_loc")
				slog.Info("Audio file",
					slog.String("basename", basename),
					slog.String("available", strings.Join(lo.Keys(bitrates), ",")),
					slog.String("location", loc))
			}

			slog.Info("Listing completed",
				slog.Int64("files", cnt),
				slog.Int("total", len(checkMap)),
				slog.String("took", time.Since(start).String()),
			)
		},
	}

	command.Flags().StringSliceVarP(&f.groups, "groups", "g", f.groups, "Included groups")
	command.Flags().StringSliceVarP(&f.excludes, "excludes", "e", f.excludes, "Excluded groups")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	return &command
}
