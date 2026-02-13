package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func isJSONOutput(cmd *cobra.Command) bool {
	enabled, err := cmd.Flags().GetBool("json")
	if err != nil {
		return false
	}
	return enabled
}

func writeJSON(cmd *cobra.Command, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
	return err
}
