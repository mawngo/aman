package cmd

import (
	"aman/internal/audio"
	"encoding/json"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
)

func newMetaCommand() *cobra.Command {
	showJSON := false

	command := cobra.Command{
		Use:   "meta <file/dir>",
		Short: "Print metadata of audio files",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			_, err := audio.Scan(args[0], func(r audio.ProbedAudio) {
				if showJSON {
					println(string(lo.Must(json.MarshalIndent(r, "", "  "))))
					return
				}
				r.Print()
				if r.Tags.Title != "" {
					slog.Info("Tags",
						slog.String("title", r.Tags.Title),
						slog.String("album", r.Tags.Album),
						slog.String("genre", r.Tags.Genre),
						slog.String("track", r.Tags.Track),
						slog.String("publisher", r.Tags.Publisher),
					)
				}
			}, audio.WithDepth(-1))
			if err != nil {
				slog.Error("Error probing audio", slog.Any("err", err))
			}
		},
	}

	command.Flags().BoolVar(&showJSON, "show-json", showJSON, "show all meta as json")
	command.Flags().SortFlags = false
	return &command
}
