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

var (
	key        string
	value      string
	ttlSeconds int64
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "📝 Set a key-value pair in TinyStoreDB",
	Long: `📌 Set stores the specified key-value pair into TinyStoreDB.

You can pass the key and value directly as positional arguments or via flags.

You can also optionally include a TTL (time-to-live) in seconds, after which the key will automatically expire.
`,
	Example: `
  # Set a key and value using positional args
  tinystoredb set myKey myValue

  # Set using flags
  tinystoredb set --key=myKey --value=myValue

  # Set with TTL (key expires in 5 seconds)
  tinystoredb set --key=myKey --value=myValue --ttl=5
`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()

		if len(args) >= 2 {
			key = args[0]
			value = args[1]
		} else if key == "" || value == "" {
			log.Fatal("❌ Either provide key and value as positional arguments or use --key and --value flags")
		}

		var ttlPtr *uint64
		if ttlSeconds > 0 {
			expiry := uint64(time.Now().Unix() + ttlSeconds)
			ttlPtr = &expiry
		}

		cli, err := getClient()
		if err != nil {
			log.Fatalf("❌ failed to get client: %v", err)
		}
		defer cli.Close()

		if err := cli.Set(cmd.Context(), key, value, ttlPtr); err != nil {
			log.Fatalf("❌ failed to set key: %v", err)
		}

		duration := time.Since(start)
		fmt.Printf("✅ Key '%s' set successfully. ⏱️ Took %s\n", key, duration)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)

	setCmd.Flags().StringVar(&key, "key", "", "Key to set")
	setCmd.Flags().StringVar(&value, "value", "", "Value to set")
	setCmd.Flags().Int64Var(&ttlSeconds, "ttl", 0, "Optional TTL in seconds")
}
