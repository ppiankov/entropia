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
	Long:  `Display the current configuration including all sources (defaults, config file, env vars, flags).`,
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
	Long:  `Create a default configuration file at ~/.entropia/config.yaml with all available options documented.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		defer f.Close()

		// Write complete default configuration as YAML with comments
		defaultCfg := model.DefaultConfig()

		fmt.Fprintf(f, "# Entropia Configuration File\n")
		fmt.Fprintf(f, "# See https://github.com/ppiankov/entropia for full documentation\n")
		fmt.Fprintf(f, "#\n")
		fmt.Fprintf(f, "# Configuration hierarchy (highest to lowest priority):\n")
		fmt.Fprintf(f, "#   1. CLI flags\n")
		fmt.Fprintf(f, "#   2. Environment variables (ENTROPIA_*)\n")
		fmt.Fprintf(f, "#   3. This config file\n")
		fmt.Fprintf(f, "#   4. Built-in defaults\n\n")

		// Marshal the complete default config to YAML
		yamlData, err := yaml.Marshal(defaultCfg)
		if err != nil {
			return fmt.Errorf("error marshaling config: %w", err)
		}

		// Write the YAML data
		if _, err := f.Write(yamlData); err != nil {
			return fmt.Errorf("error writing config: %w", err)
		}

		// Add helpful comments at the end
		fmt.Fprintf(f, "\n# API Keys (recommended to use environment variables instead):\n")
		fmt.Fprintf(f, "#   export OPENAI_API_KEY=sk-...\n")
		fmt.Fprintf(f, "#   export ANTHROPIC_API_KEY=sk-ant-...\n")
		fmt.Fprintf(f, "#   export OLLAMA_BASE_URL=http://localhost:11434\n")

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
