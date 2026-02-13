package cli

import (
	"fmt"
	"os"

	"github.com/Perttulands/agent-academy/internal/store"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"

	logger = log.NewWithOptions(os.Stderr, log.Options{Prefix: "academy"})
)

func Execute() error {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		logger.Error("command failed", "err", err)
		return err
	}

	return nil
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "academy",
		Short:         "Agent Academy CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("db", store.DefaultDBPath(), "Path to SQLite database")
	_ = viper.BindPFlag("db", rootCmd.PersistentFlags().Lookup("db"))

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newSessionCmd())

	return rootCmd
}

func initConfig() {
	viper.SetEnvPrefix("ACADEMY")
	viper.AutomaticEnv()
}

func displayVersion() string {
	return fmt.Sprintf("%s (commit=%s buildDate=%s)", Version, Commit, BuildDate)
}
