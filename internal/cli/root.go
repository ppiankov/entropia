package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "entropia",
	Short: "Entropia - Evidence support & decay diagnostics (non-normative)",
	Long: `Entropia is an open-source diagnostic tool for detecting entropy, decay,
and support gaps in public claims.

It does not determine what is true, correct, legal, authentic, or valid.

Entropia evaluates how well claims are supported by available, current,
and authoritative sources, and highlights conflicts, gaps, and drift
between claims and evidence.

Entropia is a mirror, not an oracle.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version number and build information for Entropia.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("entropia v0.1.13")
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.entropia/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			return
		}

		// Search for config in home directory
		viper.AddConfigPath(home + "/.entropia")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read in environment variables that match ENTROPIA_*
	viper.SetEnvPrefix("ENTROPIA")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}
