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

Running 'tinystoredb' with no arguments starts an interactive shell (REPL).
Alternatively, you can run one-shot commands:

  tinystoredb set mykey myvalue
  tinystoredb get mykey
  tinystoredb delete mykey
  tinystoredb exist mykey
  tinystoredb compact

Use "--help" on any command to learn more.`,
	Run: func(cmd *cobra.Command, args []string) {
		runREPL()
	},
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
	rootCmd.PersistentFlags().StringVarP(&hostFlag, "host", "H", "", "TinyStoreDB server host (default: localhost or env TINYSTOREDB_HOST)")
	rootCmd.PersistentFlags().IntVarP(&portFlag, "port", "P", 0, "TinyStoreDB server port (default: 7389 or env TINYSTOREDB_PORT)")
	rootCmd.PersistentFlags().StringVarP(&secretFlag, "secret", "s", "", "Auth secret (default: none or env TINYSTOREDB_SECRET)")
}
