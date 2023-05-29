package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nechama",
	Short: "nechama is a program that will do something",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("running something!!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
