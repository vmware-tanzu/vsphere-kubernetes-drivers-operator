package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete vSphere Kubernetes Driver Operator",
	Long:    `This command helps to delete the VDO Resources which are associated with VDO Controller.`,
	Example: "vdoctl delete vdo",
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
