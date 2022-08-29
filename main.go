package main

import (
	"os"

	"github.com/pimisiak/goblock/cli"
)

func main() {
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()
}
