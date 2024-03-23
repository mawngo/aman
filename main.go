package main

import (
	"aman/cmd"
)

func main() {
	cli := cmd.NewCLI()
	cli.Execute()
}
