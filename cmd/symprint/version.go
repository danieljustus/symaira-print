package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/versionkit"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the symprint version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := versionkit.New("symprint", version, 1)
			if jsonOut {
				return info.Write(os.Stdout)
			}
			fmt.Println(info.String())
			return nil
		},
	}
}
