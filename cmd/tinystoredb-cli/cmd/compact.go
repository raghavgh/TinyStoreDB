package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
)

var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "⚡ Trigger storage compaction to reclaim disk space",
	Long:  `Run a full compaction cycle on the database to reclaim space from deleted and overwritten keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli, err := getClient()
		if err != nil {
			log.Fatalf("❌ Failed to initialize client: %v", err)
		}
		defer cli.Close()

		start := time.Now()
		if err := cli.Compact(cmd.Context()); err != nil {
			log.Fatalf("❌ Compaction failed: %v", err)
		}

		fmt.Printf("✅ Compaction completed successfully in %v\n", time.Since(start))
	},
}

func init() {
	rootCmd.AddCommand(compactCmd)
}
