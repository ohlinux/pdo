package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of pdo",
	Long:  `All software has versions. This is pdo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("201712211018")
	},
}
