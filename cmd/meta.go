package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"aman/internal/sliceutils"
	"encoding/json"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
)

func newMetaCommand() *cobra.Command {
	showJSON := false
	sort := false
	sortGenres := []string{"dance", "electro", "classical"}

	command := cobra.Command{
		Use:   "meta <file/dir>",
		Short: "Print metadata of audio files",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			var replacer *strings.Replacer
			if sort {
				sortGenres = slices.Compact(sliceutils.FlatMapArgs(sortGenres))
				replacer = strings.NewReplacer(".", "", "'", "")
			}

			_, err := audio.Scan(args[0], func(r audio.ProbedAudio) {
				if showJSON {
					println(string(lo.Must(json.MarshalIndent(r, "", "  "))))
					return
				}
				r.Print()
				if r.Tags.Title != "" {
					slog.Info("   Tags",
						slog.String("title", r.Tags.Title),
						slog.String("album", r.Tags.Album),
						slog.String("genre", r.Tags.Genre),
						slog.String("track", r.Tags.Track),
						slog.String("publisher", r.Tags.Publisher),
					)
				}

				if !sort {
					return
				}

				if r.Tags.Title == "" {
					slog.Warn("   Missing title metadata")
					renameUnsorted(r.Filename)
					return
				}

				basename := r.Basename()
				dir := filepath.Dir(r.Filename)
				newName := ""

				r.Tags.Title = replacer.Replace(r.Tags.Title)
				title := strings.ToLower(r.Tags.Title)
				if basenameLower := strings.ToLower(basename); strings.HasPrefix(basenameLower, title) {
					// Already in the correct format.
					newName = filepath.Base(r.Filename)
				} else if i := strings.Index(basenameLower, " - "+title); i >= 0 {
					// Move title part to the start of the filename.
					name := basename[:i] + basename[i+len(" - ")+len(r.Tags.Title):]
					name = r.Tags.Title + " - " + name
					newName = name + filepath.Ext(r.Filename)
				}

				if newName == "" {
					slog.Warn("   No title match found", slog.String("title", r.Tags.Title))
					renameUnsorted(r.Filename)
					return
				}

				if len(sortGenres) > 0 && r.Tags.Genre != "" {
					genre := strings.ToLower(r.Tags.Genre)
					dirname := filepath.Base(dir)

					for _, targetDir := range sortGenres {
						matchname := ""
						if strings.Contains(targetDir, ":") {
							spl := strings.Split(targetDir, ":")
							matchname = strings.ToLower(spl[0])
							targetDir = spl[1]
						} else {
							matchname = strings.ToLower(targetDir)
						}

						if !strings.Contains(genre, matchname) {
							continue
						}
						if dirname == targetDir {
							break
						}
						newName = filepath.Join(targetDir, newName)
						break
					}
				}

				fileutils.MoveFile(fileutils.MoveConfig{
					Src:    r.Filename,
					Dest:   filepath.Join(dir, newName),
					Indent: 1,
				})
			}, audio.WithDepth(-1))
			if err != nil {
				slog.Error("Error probing audio", slog.Any("err", err))
			}
		},
	}

	command.Flags().BoolVar(&showJSON, "show-json", showJSON, "show all meta as json")
	command.Flags().BoolVar(&sort, "sort", sort, "rename and categorize by genres")
	command.Flags().StringSliceVar(&sortGenres, "sort-genres", sortGenres, "genres to categorize by")
	command.Flags().SortFlags = false
	return &command
}

func renameUnsorted(filename string) {
	dir := filepath.Dir(filename)
	name := strings.TrimPrefix(filepath.Base(filename), "(unsorted) ")
	fileutils.MoveFile(fileutils.MoveConfig{
		Src:    filename,
		Dest:   filepath.Join(dir, "(unsorted) "+name),
		Indent: 1,
	})
}
