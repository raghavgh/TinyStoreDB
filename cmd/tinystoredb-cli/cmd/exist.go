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

// existCmd represents the exist command
var existCmd = &cobra.Command{
	Use:   "exist [key]",
	Short: "🔍 Check if a key exists in TinyStoreDB",
	Long: `Check whether a key exists in TinyStoreDB.

Example:
  tinystoredb exist myKey
  tinystoredb exist -k myKey
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
		ok, err := cli.Exist(cmd.Context(), key)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Printf("⚠️ Key '%s' not found in store.\n", key)
				return
			}
			log.Fatalf("❌ Failed to retrieve key: %v", err)
		}

		if !ok {
			fmt.Printf("%s key does not exist in store.\n", key)
			return
		}

		duration := time.Since(start)
		fmt.Printf("✅ Exists\n")
		fmt.Printf("⏱️ Time taken: %s\n", duration)
	},
}

func init() {
	rootCmd.AddCommand(existCmd)

	existCmd.Flags().StringVarP(&key, "key", "k", "", "Key to check in the store")
}
