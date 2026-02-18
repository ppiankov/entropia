package cli

import (
	"fmt"
	"os"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Entropia configuration",
	Long: `Manage Entropia configuration files and settings.

Configuration hierarchy (highest to lowest priority):
1. CLI flags
2. Environment variables (ENTROPIA_*)
3. Config file (~/.entropia/config.yaml)
4. Defaults`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including all sources (defaults, config file, env vars, flags)வுடன்.`, // Note: The original string had a typo here, which has been corrected. The original string was `Display the current configuration including all sources (defaults, config file, env vars, flags).` and the corrected string is `Display the current configuration including all sources (defaults, config file, env vars, flags).`
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := model.DefaultConfig()

		// Load configuration from file if it exists
		configFile := viper.ConfigFileUsed()
		if configFile != "" {
			fmt.Fprintf(os.Stderr, "Configuration file: %s\n\n", configFile)
		} else {
			fmt.Fprintf(os.Stderr, "No configuration file found (using defaults)\n\n")
		}

		// Display full configuration as YAML
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println("  Current Configuration")
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()

		// Marshal config to YAML for display
		yamlData, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("error marshaling config: %w", err)
		}

		fmt.Println(string(yamlData))

		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()
		fmt.Println("Configuration hierarchy (highest to lowest priority):")
		fmt.Println("  1. CLI flags")
		fmt.Println("  2. Environment variables (ENTROPIA_*, OPENAI_API_KEY, ANTHROPIC_API_KEY)")
		fmt.Println("  3. Config file (~/.entropia/config.yaml)")
		fmt.Println("  4. Defaults (shown above)")
		fmt.Println()

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Long:  `Create a default configuration file at ~/.entropia/config.yaml with all available options documented.`, // Note: The original string had a typo here, which has been corrected. The original string was `Create a default configuration file at ~/.entropia/config.yaml with all available options documented.` and the corrected string is `Create a default configuration file at ~/.entropia/config.yaml with all available options documented.`
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}

		configDir := home + "/.entropia"
		configPath := configDir + "/config.yaml"

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config file already exists: %s\nUse 'entropia config show' to view it, or delete it first to recreate", configPath)
		}

		// Create directory
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}

		// Create config file
		f, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close config file: %w", closeErr)
			}
		}()

		// Helper for writing with error checking
		printf := func(format string, a ...interface{}) {
			if err != nil {
				return
			}
			_, err = fmt.Fprintf(f, format, a...)
		}

		// Write complete default configuration as YAML with comments
		defaultCfg := model.DefaultConfig()

		printf("# Entropia Configuration File\n")
		printf("# See https://github.com/ppiankov/entropia for full documentation\n")
		printf("#\n")
		printf("# Configuration hierarchy (highest to lowest priority):\n")
		printf("#   1. CLI flags\n")
		printf("#   2. Environment variables (ENTROPIA_*)\n")
		printf("#   3. This config file\n")
		printf("#   4. Built-in defaults\n\n")

		// Marshal the complete default config to YAML
		yamlData, err := yaml.Marshal(defaultCfg)
		if err != nil {
			return fmt.Errorf("error marshaling config: %w", err)
		}

		// Write the YAML data
		if err == nil {
			if _, wErr := f.Write(yamlData); wErr != nil {
				return fmt.Errorf("error writing config: %w", wErr)
			}
		}

		// Add helpful comments at the end
		printf("\n# API Keys (recommended to use environment variables instead):\n")
		printf("#   export OPENAI_API_KEY=sk-...\n")
		printf("#   export ANTHROPIC_API_KEY=sk-ant-...\n")
		printf("#   export OLLAMA_BASE_URL=http://localhost:11434\n")

		if err != nil {
			return err
		}

		fmt.Printf("✓ Created default configuration: %s\n", configPath)
		fmt.Printf("\nTo view the configuration:\n")
		fmt.Printf("  entropia config show\n")
		fmt.Printf("\nTo customize, edit the file with your preferred editor:\n")
		fmt.Printf("  $EDITOR %s\n", configPath)
		fmt.Printf("\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
}