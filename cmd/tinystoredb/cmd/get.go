/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// getCmd represents the get command for TinyStoreDB.
var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "🔍 Get a value by key from the TinyStoreDB",
	Long:  `Retrieve a value from TinyStoreDB by providing the associated key.`,
	Example: `
  # Basic usage with positional argument
  tinystoredb get myKey

  # Using --key flag
  tinystoredb get --key=myKey

  # Using shorthand -k flag
  tinystoredb get -k myKey
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine the key from positional argument or flag
		if len(args) == 1 {
			key = args[0]
		} else if key == "" {
			log.Fatal("❌ Please provide a key either as a positional argument or using --key/-k flag")
		}

		cli, err := getClient()
		if err != nil {
			log.Fatalf("❌ Failed to initialize client: %v", err)
		}
		defer cli.Close()

		// Start measuring execution time
		start := time.Now()
		val, err := cli.Get(cmd.Context(), key)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Printf("⚠️ Key '%s' not found in store.\n", key)
				return
			}
			log.Fatalf("❌ Failed to retrieve key: %v", err)
		}

		duration := time.Since(start)
		fmt.Printf("✅ Value: %s\n", val)
		fmt.Printf("⏱️ Time taken: %s\n", duration)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVarP(&key, "key", "k", "", "Key to retrieve from the store")
}
