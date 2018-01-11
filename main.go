package main

import (
	"fmt"
	"os"

	"gitlab.sftcwl.com/sf-op/pdov3/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}