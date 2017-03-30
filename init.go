package main

import "github.com/spf13/cobra"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize the driver",
	Run: func(cmd *cobra.Command, args []string) {
		result{}.emit(0, "")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
