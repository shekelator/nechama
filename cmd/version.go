package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newVersionCommand(output io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(output, Version)
			return err
		},
	}

	cmd.SetOut(output)
	return cmd
}
