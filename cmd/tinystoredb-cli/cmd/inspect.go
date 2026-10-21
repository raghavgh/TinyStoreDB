package cmd

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/raghavgh/TinyStoreDB/disk/record/value_log/proto/value_log_pb"
	"github.com/raghavgh/TinyStoreDB/disk/record/wal/proto/walpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var (
	inspectDirFlag        string
	inspectKeyFlag        string
	inspectTombstonesFlag bool
	inspectActiveOnlyFlag bool
	inspectJsonFlag       bool
	inspectWatchFlag      bool
)

// InspectedRecord captures the on-disk state of a single WAL entry and its corresponding ValueLog payload.
type InspectedRecord struct {
	Index      int       `json:"index"`
	WalOffset  uint64    `json:"wal_offset"`
	DataOffset uint64    `json:"data_offset"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Deleted    bool      `json:"deleted"`
	Timestamp  time.Time `json:"timestamp"`
	HasExpiry  bool      `json:"has_expiry"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	TTLSeconds int64     `json:"ttl_seconds,omitempty"`
	Status     string    `json:"status"` // ACTIVE, TOMBSTONE, SUPERSEDED, EXPIRED
}

// InspectSummary aggregates metrics for disk analysis.
type InspectSummary struct {
	TotalRecords    int `json:"total_records"`
	ActiveRecords   int `json:"active_records"`
	Tombstones      int `json:"tombstones"`
	Superseded      int `json:"superseded"`
	Expired         int `json:"expired"`
	CompactionWaste int `json:"compaction_waste"`
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "🔍 Inspect raw on-disk data files (WAL, ValueLog, Tombstones, Revisions)",
	Long: `Inspect provides offline and live interactive diagnostic views of TinyStoreDB's on-disk storage engine.

It walks through wal.bin and value_log.bin sequentially, decoding every entry, tracking key revision
history, identifying tombstones (deletions), superseded entries, active values, and TTL expirations.

Examples:
  tinystoredb-cli inspect
  tinystoredb-cli inspect --dir /data
  tinystoredb-cli inspect --key user:100
  tinystoredb-cli inspect --tombstones
  tinystoredb-cli inspect --watch
  tinystoredb-cli inspect --json`,
	Run: func(cmd *cobra.Command, args []string) {
		dir := resolveDataDir(inspectDirFlag)
		if inspectWatchFlag {
			runWatchMode(cmd.Context(), dir)
			return
		}

		records, summary, walPath, valPath, err := ReadDiskRecords(dir)
		if err != nil {
			fmt.Printf("❌ Failed to inspect data directory %q: %v\n", dir, err)
			return
		}

		// Apply filters
		filtered := filterRecords(records, inspectKeyFlag, inspectTombstonesFlag, inspectActiveOnlyFlag)

		if inspectJsonFlag {
			output := struct {
				DataDir string            `json:"data_dir"`
				WalFile string            `json:"wal_file"`
				ValFile string            `json:"value_log_file"`
				Summary InspectSummary    `json:"summary"`
				Records []InspectedRecord `json:"records"`
			}{
				DataDir: dir,
				WalFile: walPath,
				ValFile: valPath,
				Summary: summary,
				Records: filtered,
			}
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
			return
		}

		PrintInspectTable(os.Stdout, dir, walPath, valPath, filtered, summary)
	},
}

func init() {
	inspectCmd.Flags().StringVarP(&inspectDirFlag, "dir", "d", "", "Directory containing TinyStoreDB data files (default: data or TINYSTOREDB_DATA_DIR)")
	inspectCmd.Flags().StringVarP(&inspectKeyFlag, "key", "k", "", "Filter records by key")
	inspectCmd.Flags().BoolVar(&inspectTombstonesFlag, "tombstones", false, "Display only tombstone (deleted) records")
	inspectCmd.Flags().BoolVar(&inspectActiveOnlyFlag, "active-only", false, "Display only currently active live records")
	inspectCmd.Flags().BoolVar(&inspectJsonFlag, "json", false, "Output results in JSON format")
	inspectCmd.Flags().BoolVarP(&inspectWatchFlag, "watch", "w", false, "Live watch mode: tail storage files and display writes/tombstones in real-time")

	rootCmd.AddCommand(inspectCmd)
}

func resolveDataDir(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if envVal := os.Getenv("TINYSTOREDB_DATA_DIR"); envVal != "" {
		return envVal
	}
	return "data"
}

// locateFiles returns the paths to wal and value_log files in dir.
func locateFiles(dir string) (string, string, error) {
	walPath := filepath.Join(dir, "wal.bin")
	valPath := filepath.Join(dir, "value_log.bin")

	if _, err := os.Stat(walPath); err != nil {
		// Check test files if regular not present
		altWal := filepath.Join(dir, "wal_test.bin")
		altVal := filepath.Join(dir, "value_log_test.bin")
		if _, errAlt := os.Stat(altWal); errAlt == nil {
			return altWal, altVal, nil
		}
		return "", "", fmt.Errorf("WAL file not found at %s", walPath)
	}

	return walPath, valPath, nil
}

// ReadDiskRecords parses the WAL and ValueLog files to reconstruct the disk state.
func ReadDiskRecords(dir string) ([]InspectedRecord, InspectSummary, string, string, error) {
	walPath, valPath, err := locateFiles(dir)
	if err != nil {
		return nil, InspectSummary{}, "", "", err
	}

	walFile, err := os.Open(walPath)
	if err != nil {
		return nil, InspectSummary{}, walPath, valPath, fmt.Errorf("open WAL: %w", err)
	}
	defer walFile.Close()

	valFile, _ := os.Open(valPath)
	if valFile != nil {
		defer valFile.Close()
	}

	var records []InspectedRecord
	var currentWalOffset uint64 = 0

	for {
		var lengthPrefix uint32
		err := binary.Read(walFile, binary.BigEndian, &lengthPrefix)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			break
		}

		data := make([]byte, lengthPrefix)
		_, err = io.ReadFull(walFile, data)
		if err != nil {
			break
		}

		var walRecord walpb.WALRecord
		if err := proto.Unmarshal(data, &walRecord); err != nil {
			currentWalOffset += uint64(4 + lengthPrefix)
			continue
		}

		record := InspectedRecord{
			Index:      len(records) + 1,
			WalOffset:  currentWalOffset,
			DataOffset: walRecord.Offset,
			Key:        walRecord.Key,
			Deleted:    walRecord.Deleted,
		}

		if walRecord.Timestamp != nil && *walRecord.Timestamp > 0 {
			record.Timestamp = time.Unix(int64(*walRecord.Timestamp), 0)
		}

		if walRecord.Ttl != nil && *walRecord.Ttl > 0 {
			record.HasExpiry = true
			record.ExpiresAt = time.Unix(int64(*walRecord.Ttl), 0)
			record.TTLSeconds = int64(*walRecord.Ttl) - time.Now().Unix()
		}

		if walRecord.Deleted {
			record.Value = "<tombstone>"
		} else if valFile != nil {
			// Read payload from ValueLog
			valLenBuf := make([]byte, 4)
			if _, err := valFile.ReadAt(valLenBuf, int64(walRecord.Offset)); err == nil {
				valLen := binary.BigEndian.Uint32(valLenBuf)
				valBuf := make([]byte, valLen)
				if _, err := valFile.ReadAt(valBuf, int64(walRecord.Offset+4)); err == nil {
					var vRec value_log_pb.ValueLogRecord
					if proto.Unmarshal(valBuf, &vRec) == nil {
						record.Value = string(vRec.Value)
					}
				}
			}
			if record.Value == "" {
				record.Value = "<pending/empty>"
			}
		}

		records = append(records, record)
		currentWalOffset += uint64(4 + lengthPrefix)
	}

	// Compute status and summary
	summary := computeLifecycleStatuses(records)

	return records, summary, walPath, valPath, nil
}

func computeLifecycleStatuses(records []InspectedRecord) InspectSummary {
	now := time.Now()
	lastSeenIndex := make(map[string]int)

	for i, r := range records {
		lastSeenIndex[r.Key] = i
	}

	var summary InspectSummary
	summary.TotalRecords = len(records)

	for i := range records {
		r := &records[i]
		if r.Deleted {
			r.Status = "TOMBSTONE"
			summary.Tombstones++
			summary.CompactionWaste++
			continue
		}

		if lastSeenIndex[r.Key] != i {
			r.Status = "SUPERSEDED"
			summary.Superseded++
			summary.CompactionWaste++
			continue
		}

		if r.HasExpiry && r.ExpiresAt.Before(now) {
			r.Status = "EXPIRED"
			summary.Expired++
			summary.CompactionWaste++
			continue
		}

		r.Status = "ACTIVE"
		summary.ActiveRecords++
	}

	return summary
}

func filterRecords(records []InspectedRecord, keyFilter string, tombstonesOnly bool, activeOnly bool) []InspectedRecord {
	var filtered []InspectedRecord
	for _, r := range records {
		if keyFilter != "" && !strings.Contains(r.Key, keyFilter) {
			continue
		}
		if tombstonesOnly && !r.Deleted {
			continue
		}
		if activeOnly && r.Status != "ACTIVE" {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// PrintInspectTable renders a formatted table showing disk records and summary statistics.
func PrintInspectTable(w io.Writer, dir, walPath, valPath string, records []InspectedRecord, summary InspectSummary) {
	walFi, _ := os.Stat(walPath)
	valFi, _ := os.Stat(valPath)
	var walSize, valSize int64
	if walFi != nil {
		walSize = walFi.Size()
	}
	if valFi != nil {
		valSize = valFi.Size()
	}

	fmt.Fprintf(w, "\n====================================================================================================\n")
	fmt.Fprintf(w, "  🔍 TinyStoreDB Storage Engine Inspector (Directory: %s)\n", dir)
	fmt.Fprintf(w, "  WAL: %s (%s) | ValueLog: %s (%s)\n",
		filepath.Base(walPath), formatBytes(walSize),
		filepath.Base(valPath), formatBytes(valSize))
	fmt.Fprintf(w, "====================================================================================================\n")

	if len(records) == 0 {
		fmt.Fprintf(w, "  (No records found in storage files)\n\n")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  #\tWAL-OFFSET\tDATA-OFFSET\tKEY\tVALUE\tTIMESTAMP\tTTL\tSTATUS\n")
	fmt.Fprintf(tw, "  -\t----------\t-----------\t---\t-----\t---------\t---\t------\n")

	for _, r := range records {
		dataOffsetStr := fmt.Sprintf("0x%06X", r.DataOffset)
		if r.Deleted {
			dataOffsetStr = "-"
		}

		valStr := r.Value
		if len(valStr) > 24 {
			valStr = valStr[:21] + "..."
		}
		valStr = strings.ReplaceAll(valStr, "\n", "↵")

		timeStr := "-"
		if !r.Timestamp.IsZero() {
			timeStr = r.Timestamp.Format("2006-01-02 15:04:05")
		}

		ttlStr := "-"
		if r.HasExpiry {
			if r.TTLSeconds > 0 {
				ttlStr = fmt.Sprintf("%ds", r.TTLSeconds)
			} else {
				ttlStr = "expired"
			}
		}

		statusStr := r.Status
		switch r.Status {
		case "ACTIVE":
			statusStr = "✅ ACTIVE"
		case "TOMBSTONE":
			statusStr = "💀 TOMBSTONE"
		case "SUPERSEDED":
			statusStr = "♻️  SUPERSEDED"
		case "EXPIRED":
			statusStr = "⏰ EXPIRED"
		}

		fmt.Fprintf(tw, "  %d\t0x%06X\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Index, r.WalOffset, dataOffsetStr, r.Key, valStr, timeStr, ttlStr, statusStr)
	}
	tw.Flush()

	fmt.Fprintf(w, "----------------------------------------------------------------------------------------------------\n")
	fmt.Fprintf(w, "  Summary: %d total records | %d active (live) | %d superseded | %d tombstones | %d expired\n",
		summary.TotalRecords, summary.ActiveRecords, summary.Superseded, summary.Tombstones, summary.Expired)
	if summary.CompactionWaste > 0 {
		fmt.Fprintf(w, "  ⚡ Compaction: %d dead/stale records can be reclaimed via 'tinystoredb-cli compact'\n",
			summary.CompactionWaste)
	} else {
		fmt.Fprintf(w, "  ✨ Compaction: Storage is fully clean and optimal (no stale records)\n")
	}
	fmt.Fprintf(w, "====================================================================================================\n\n")
}

func runWatchMode(ctx context.Context, dir string) {
	fmt.Printf("👀 Watching TinyStoreDB storage in %q... (Press Ctrl+C to exit)\n", dir)

	// First print snapshot
	records, summary, walPath, valPath, err := ReadDiskRecords(dir)
	if err == nil {
		PrintInspectTable(os.Stdout, dir, walPath, valPath, records, summary)
	} else {
		fmt.Printf("Waiting for data files to be created in %s...\n", dir)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastCount := len(records)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			fmt.Println("\nStopped watch mode.")
			return
		case <-ticker.C:
			currentRecords, _, _, _, err := ReadDiskRecords(dir)
			if err != nil {
				continue
			}

			if len(currentRecords) > lastCount {
				for i := lastCount; i < len(currentRecords); i++ {
					r := currentRecords[i]
					timeStr := time.Now().Format("15:04:05")
					if r.Deleted {
						fmt.Printf("[%s] 💀 TOMBSTONE      | KEY: %-15s | WAL-OFFSET: 0x%06X\n",
							timeStr, r.Key, r.WalOffset)
					} else {
						fmt.Printf("[%s] ✍️  WRITE          | KEY: %-15s | VALUE: %-15s | OFFSET: 0x%06X | STATUS: %s\n",
							timeStr, r.Key, r.Value, r.DataOffset, r.Status)
					}
				}
				lastCount = len(currentRecords)
			}
		}
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
