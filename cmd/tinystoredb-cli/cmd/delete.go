/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "❌ Delete a key from TinyStoreDB",
	Long: `Delete a key from your TinyStoreDB server.

You can remove a specific key by passing it as an argument:

  tinystoredb delete myKey

This will permanently remove the key from the store if it exists.`,
	Example: `  tinystoredb delete sessionToken
  tinystoredb delete user:1234`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Determine the key from positional argument or flag
		if len(args) == 1 {
			key = args[0]
		} else if key == "" {
			log.Fatal("❌ Please provide a key either as a positional argument or using --key/-k flag")
		}

		cli, err := getClient()
		if err != nil {
			log.Fatal(err)
		}

		start := time.Now()

		_, err = cli.Delete(cmd.Context(), args[0])
		if err != nil {
			log.Fatalf("❌ Delete failed: %v", err)
		}

		fmt.Printf("🗑️  Deleted key '%s' successfully!\n", args[0])
		fmt.Printf("⏱️  Took %v\n", time.Since(start))
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
