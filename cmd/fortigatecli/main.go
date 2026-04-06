package main

import (
	"os"

	"fortigatecli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
