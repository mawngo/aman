package main

import (
	"aman/cmd"
	"github.com/spf13/cobra"
)

func main() {
	cobra.EnableCommandSorting = false

	cli := cmd.NewCLI()
	cli.Execute()
}
