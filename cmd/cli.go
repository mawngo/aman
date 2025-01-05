package cmd

import (
	"fmt"
	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"time"
)

type CLI struct {
	command *cobra.Command
}

// NewCLI create new CLI instance and setup application config.
func NewCLI() *CLI {
	command := cobra.Command{
		Use:   "aman",
		Short: "Various audio management tools.",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			levelFlag, err := cmd.Flags().GetString("log")
			if err != nil {
				panic(err)
			}
			level := slog.LevelInfo
			switch levelFlag {
			case "verbose":
				level = slog.LevelDebug
			case "quiet":
				level = slog.LevelWarn
			}
			slog.SetDefault(slog.New(
				console.NewHandler(os.Stderr, &console.HandlerOptions{
					Level:      level,
					TimeFormat: time.Kitchen,
				}),
			))
		},
	}
	command.AddCommand(newMetaCommand())
	command.AddCommand(newSortCommand())
	command.PersistentFlags().String("log", "default", "Configure log level [default/verbose/quiet]")
	return &CLI{command: &command}
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
