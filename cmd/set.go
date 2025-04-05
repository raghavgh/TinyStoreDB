/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/raghavgh/TinyStoreDB/store"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a key-value pair",
	Long: `Store a key and its corresponding value into TinyStoreDB. 
For example:

set <key> <value>

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Here you will define your command's action.
		// For example, you can call a function to perform the action.
		// fmt.Println("set called")

		// copilot implement this function
		if len(args) != 2 {
			cmd.Help()
			return
		}

		key := args[0]
		value := args[1]

		kvStore, err := store.NewKVStore()
		if err != nil {
			fmt.Println("Failed to create KV store:", err)

			return
		}

		err = kvStore.Set(key, value)
		if err != nil {
			fmt.Println("Error:", err)

			return
		}

		fmt.Println("Key-value pair set successfully")
	},
}

func init() {
	rootCmd.AddCommand(setCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// setCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// setCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
