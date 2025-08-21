package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Config struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Failed to get config directory:", err)
		os.Exit(1)
	}
	configPath := filepath.Join(configDir, "tinystoredb")
	_ = os.MkdirAll(configPath, 0o755)
	return filepath.Join(configPath, "config.json")
}

func saveConfig(cfg Config) error {
	configFile := getConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}

func loadConfig() (Config, error) {
	var cfg Config
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

var (
	hostFlag   string
	portFlag   int
	secretFlag string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage TinyStoreDB CLI config",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set config values",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := Config{
			Host:   hostFlag,
			Port:   portFlag,
			Secret: secretFlag,
		}
		if err := saveConfig(cfg); err != nil {
			fmt.Println("Failed to save config:", err)
			return
		}
		fmt.Println("✅ Config saved")
	},
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current config",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("Failed to load config:", err)
			return
		}
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configViewCmd)

	configSetCmd.Flags().StringVar(&hostFlag, "host", "localhost", "TinyStoreDB host")
	configSetCmd.Flags().IntVar(&portFlag, "port", 7389, "TinyStoreDB port")
	configSetCmd.Flags().StringVar(&secretFlag, "secret", "", "TinyStoreDB secret")
}
