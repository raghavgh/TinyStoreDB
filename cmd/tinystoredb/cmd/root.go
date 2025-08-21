/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tinystoredb",
	Short: "TinyStoreDB CLI - Interact with your lightweight key-value database",
	Long: `TinyStoreDB is a lightweight, embeddable key-value database built in Go. 
It supports fast disk-based persistence with optional TTL-based expiry and concurrency-safe operations.

This CLI tool lets you set, get, and delete keys, configure your connection, and inspect metrics.

Example usage:
  tinystoredb set mykey myvalue
  tinystoredb get mykey
  tinystoredb delete mykey

Use "--help" on any command to learn more.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tinystoredb.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
