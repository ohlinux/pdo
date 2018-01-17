// Copyright © 2017 Ajian
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ohlinux/pdo/pkg/pdo"
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "copy file or directory to destination",
	Long: `the command like rsync ,can copy file or directory, please notice file and directory different format.`,
	Example: `pdo copy a.file /tmp/b.file` ,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// command
		cmdArgs:=cmd.Flags().Args()

		cmdArgstr:=fmt.Sprintf("copy %s --> %s",cmdArgs[0],cmdArgs[1])
		//cmdArgstr:=strings.Join("copy ",cmdArgs[0],"to")
		copyCmd:=fmt.Sprint("rsync -e \"ssh -q -p %s -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3\" ",cmdArgs[0]," %s@%s:",cmdArgs[1])
		newpdo.Command=pdo.Command{
			Inputcmd: cmdArgstr,
			Display: cmdArgstr,
			Execmd: copyCmd,
			Args: cmdArgs[1:],
		}
		fmt.Println(newpdo.Command)

		newpdo.Command.Local = true

		newpdo.PrepareInput(cmd ,args)

		newpdo.PrepareOutput(cmd ,args)

		//todo: validate host list
		//todo: filter host list

		if err:=newpdo.CreateJobList();err!=nil {
			fmt.Errorf("input host list fail %v",err)
		}
		newpdo.Run()
	},
}

func init() {
	RootCmd.AddCommand(copyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// copyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// copyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
