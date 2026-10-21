package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
		return ""
	}
	configPath := filepath.Join(configDir, "tinystoredb")
	_ = os.MkdirAll(configPath, 0o755)
	return filepath.Join(configPath, "config.json")
}

func saveConfig(cfg Config) error {
	path := getConfigPath()
	if path == "" {
		return fmt.Errorf("unable to determine user config directory")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadConfig() (Config, error) {
	cfg := Config{
		Host: "localhost",
		Port: 7389,
	}
	if h := os.Getenv("TINYSTOREDB_HOST"); h != "" {
		cfg.Host = h
	}
	if p := os.Getenv("TINYSTOREDB_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			cfg.Port = val
		}
	}
	if s := os.Getenv("TINYSTOREDB_SECRET"); s != "" {
		cfg.Secret = s
	}

	path := getConfigPath()
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &cfg)
		}
	}
	return cfg, nil
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
		cfg, _ := loadConfig()
		if hostFlag != "" {
			cfg.Host = hostFlag
		}
		if portFlag != 0 {
			cfg.Port = portFlag
		}
		if secretFlag != "" {
			cfg.Secret = secretFlag
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
