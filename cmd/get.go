/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/store"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the value for a specific key",
	Long: `Retrieve the value stored for a given key in TinyStoreDB.
and usage of using your command. For example:
get <key>

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Usage: tinystore get <key>")

			return
		}

		key := args[0]

		kvStore, err := store.NewKVStore(config.Load())
		if err != nil {
			fmt.Println("Failed to create KV store:", err)

			return
		}

		if err := kvStore.Replay(); err != nil {
			fmt.Println("Replay error:", err)

			return
		}

		value, err := kvStore.Get(key)
		if err != nil {
			fmt.Println("Error:", err)

			return
		}

		if value == "" {
			fmt.Println("Key not found")
		} else {
			fmt.Println("Value:", value)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
