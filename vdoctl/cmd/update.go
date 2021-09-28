/*
Copyright © 2021

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u"},
	Short:   "Update the VDO Resources",
	Long: `This command helps to update the VDO Resources which is created by VDO Controller. 
For example:

vdoctl update matrix https://sample/sample.yaml
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please select the sub-command, to get more help run" +
			"vdoctl update -h")
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)

}
