package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print academy version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(displayVersion())
		},
	}
}
