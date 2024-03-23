package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	cobra.EnableCommandSorting = false
}

type CLI struct {
	command *cobra.Command
}

// NewCLI create new CLI instance and setup application config
func NewCLI() *CLI {
	command := cobra.Command{
		Use:   "aman",
		Short: "Various audio management tools.",
	}
	return &CLI{command: &command}
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
