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
	"strings"
)

// copyCmd represents the copy command
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "execute local script in remote server",
	Long: `execut local script file , any format. step 1: copy the local file to remote server , step 2: execute the file.`,
	Example: `pdo script example.sh arg1 arg2` ,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// command
		cmdArgs:=cmd.Flags().Args()
		cmdArgstr:=fmt.Sprintf("script %s %s",cmdArgs[0],strings.Join(cmdArgs[1:]," "))
		remoteFile:=fmt.Sprintf("/tmp/pdo_script.%s",pdo.PID)
		remoteExe:=fmt.Sprintf("chmod +x %s && %s %s || cd /tmp/ && rm -f %s",remoteFile,remoteFile,strings.Join(cmdArgs[1:]," "),remoteFile)
		copyCmd:=fmt.Sprint("rsync -e \"ssh -q -p {{.Port}} -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3\" ",cmdArgs[0]," {{.User}}@{{.Host}}:",remoteFile)
		newpdo.Command=pdo.Command{
			Inputcmd: cmdArgstr,
			Display: cmdArgstr,
			Execmd: remoteExe,
			Args: cmdArgs[1:],
			PreCmd: copyCmd,
		}

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
	RootCmd.AddCommand(scriptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// copyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// copyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
