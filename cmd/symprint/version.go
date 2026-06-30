package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the symprint version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOut {
				return printJSON(map[string]string{"version": version})
			}
			fmt.Printf("symprint %s\n", version)
			return nil
		},
	}
}
