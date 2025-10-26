package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"aman/internal/sliceutils"
	"encoding/csv"
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
	export := false

	command := cobra.Command{
		Use:   "ls <dir>",
		Short: "List audio files",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			includes := sliceutils.FlatMapArgsToSet(f.groups)
			excludes := sliceutils.FlatMapArgsToSet(f.excludes)

			_, listAll := includes["*"]
			start := time.Now()
			slog.Info("Scanning audio files...")
			checkMap, cnt, err := audio.ScanMap(args[0], func(check map[string]string, a audio.ProbedAudio) map[string]string {
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
					check["_loc"] = a.Location()
				}
				check[a.Group] = lo.Ternary(export, a.Filename, "")
				return check
			},
				audio.WithDepth(f.depth),
				audio.WithConcurrency(f.concurrency),
				audio.WithProgress(true))
			if err != nil {
				slog.Error("Error checking audio files", slog.Any("err", err))
				return
			}

			var writer *csv.Writer
			if export {
				exportFile := filepath.Join(args[0], audio.ExportFilename)
				var cleanup fileutils.CleanupFunc
				writer, cleanup, err = fileutils.CreateCsvFile(exportFile)
				if err != nil {
					slog.Error("Error creating export file",
						slog.String("file", exportFile),
						slog.Any("err", err))
					return
				}
				defer func() {
					if err := cleanup(); err != nil {
						slog.Error("Error writing export file",
							slog.String("file", exportFile),
							slog.Any("err", err))
						return
					}
					slog.Info("Tracklist exported", slog.String("file", exportFile))
				}()
			}

			for basename, bitrates := range checkMap {
				loc := bitrates["_loc"]
				delete(bitrates, "_loc")

				if export {
					for bitrate, filename := range bitrates {
						if err := writer.Write([]string{filename, bitrate, basename}); err != nil {
							return
						}
					}
				}

				slog.Info("Audio file",
					slog.String("basename", basename),
					slog.String("available", strings.Join(lo.Keys(bitrates), ",")),
					slog.String("loc", loc))
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
	command.Flags().BoolVar(&export, "export", export, "Export tracklist")
	return &command
}
