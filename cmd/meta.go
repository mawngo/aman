package cmd

import (
	"aman/internal/audio"
	"encoding/json"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func newMetaCommand() *cobra.Command {
	showJSON := false

	command := cobra.Command{
		Use:   "meta <file>",
		Short: "Print metadata of audio file",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			r := lo.Must(audio.Probe(args[0]))
			if showJSON {
				println(string(lo.Must(json.MarshalIndent(r, "", "  "))))
				return
			}
			r.Print()
			if r.Tags.Title != "" {
				println(string(lo.Must(json.MarshalIndent(r.Tags, "", "  "))))
			}
		},
	}

	command.Flags().BoolVar(&showJSON, "show-json", showJSON, "show all meta as json")
	command.Flags().SortFlags = false
	return &command
}
