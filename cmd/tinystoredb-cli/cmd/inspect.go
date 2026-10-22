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
	inspectLimitFlag      int
	inspectAllFlag        bool
	inspectHexFlag        bool
)

// InspectedRecord captures the on-disk state of a single WAL entry and its corresponding ValueLog payload.
type InspectedRecord struct {
	Index      int       `json:"index"`
	WalOffset  uint64    `json:"wal_offset"`
	WalLength  uint32    `json:"wal_length"`
	DataOffset uint64    `json:"data_offset"`
	DataLength uint32    `json:"data_length"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Deleted    bool      `json:"deleted"`
	RecordType string    `json:"record_type"` // PUT, TOMBSTONE (on-disk reality)
	Liveness   string    `json:"liveness"`    // live, stale, expired, - (derived lineage)
	Timestamp  time.Time `json:"timestamp"`
	HasExpiry  bool      `json:"has_expiry"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	TTLSeconds int64     `json:"ttl_seconds,omitempty"`
	Status     string    `json:"status"` // legacy alias for liveness/record type
}

// InspectSummary aggregates metrics for disk analysis.
type InspectSummary struct {
	TotalRecords    int `json:"total_records"`
	Puts            int `json:"puts"`
	Tombstones      int `json:"tombstones"`
	LiveRecords     int `json:"live_records"`
	StaleRecords    int `json:"stale_records"`
	ExpiredRecords  int `json:"expired_records"`
	CompactionWaste int `json:"compaction_waste"`
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "🔍 Inspect raw on-disk data files (WAL, ValueLog, Tombstones, Revisions)",
	Long: `Inspect provides offline and live interactive diagnostic views of TinyStoreDB's on-disk storage engine.

It scans wal.bin and value_log.bin, decoding entry metadata, tracking key revision history,
identifying tombstones (deletions), stale overwritten entries, live values, and TTL expirations.

Terminology maps directly to physical storage:
- RECORD TYPE: What is physically on disk: PUT (write) or TOMBSTONE (delete).
- LIVENESS: Current status in the database index: live, stale (overwritten), or expired.

Optimized for bulk storage: Performs a fast sequential metadata scan and lazily loads
heavy value payloads only for displayed records.

Examples:
  tinystoredb-cli inspect
  tinystoredb-cli inspect --dir /data
  tinystoredb-cli inspect --limit 100
  tinystoredb-cli inspect --all
  tinystoredb-cli inspect --hex
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
		totalFiltered := len(filtered)

		// Apply limit (default 50 recent writes unless --all is passed)
		displayed := filtered
		if !inspectAllFlag && inspectLimitFlag > 0 && len(displayed) > inspectLimitFlag {
			displayed = displayed[len(displayed)-inspectLimitFlag:]
		}

		// Lazy load value payloads only for the records being displayed
		loadValues(valPath, displayed)

		if inspectJsonFlag {
			output := struct {
				DataDir string            `json:"data_dir"`
				WalFile string            `json:"wal_file"`
				ValFile string            `json:"value_log_file"`
				Summary InspectSummary    `json:"summary"`
				Showing int               `json:"showing"`
				Total   int               `json:"total_filtered"`
				Records []InspectedRecord `json:"records"`
			}{
				DataDir: dir,
				WalFile: walPath,
				ValFile: valPath,
				Summary: summary,
				Showing: len(displayed),
				Total:   totalFiltered,
				Records: displayed,
			}
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
			return
		}

		PrintInspectTable(os.Stdout, dir, walPath, valPath, displayed, summary, totalFiltered, inspectHexFlag)
	},
}

func init() {
	inspectCmd.Flags().StringVarP(&inspectDirFlag, "dir", "d", "", "Directory containing TinyStoreDB data files (default: data or TINYSTOREDB_DATA_DIR)")
	inspectCmd.Flags().StringVarP(&inspectKeyFlag, "key", "k", "", "Filter records by key")
	inspectCmd.Flags().BoolVar(&inspectTombstonesFlag, "tombstones", false, "Display only tombstone (deleted) records")
	inspectCmd.Flags().BoolVar(&inspectActiveOnlyFlag, "active-only", false, "Display only currently active live records")
	inspectCmd.Flags().IntVarP(&inspectLimitFlag, "limit", "n", 50, "Limit number of displayed records to most recent N (default: 50)")
	inspectCmd.Flags().BoolVar(&inspectAllFlag, "all", false, "Display all records without limit")
	inspectCmd.Flags().BoolVar(&inspectHexFlag, "hex", false, "Display offsets in raw hexadecimal format (e.g. 0x00002A)")
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
		altWal := filepath.Join(dir, "wal_test.bin")
		altVal := filepath.Join(dir, "value_log_test.bin")
		if _, errAlt := os.Stat(altWal); errAlt == nil {
			return altWal, altVal, nil
		}
		return "", "", fmt.Errorf("WAL file not found at %s", walPath)
	}

	return walPath, valPath, nil
}

// ReadDiskRecords performs a fast sequential scan of wal.bin to reconstruct full database state without reading value payloads.
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

		entryTotalLen := 4 + lengthPrefix

		var walRecord walpb.WALRecord
		if err := proto.Unmarshal(data, &walRecord); err != nil {
			currentWalOffset += uint64(entryTotalLen)
			continue
		}

		record := InspectedRecord{
			Index:      len(records) + 1,
			WalOffset:  currentWalOffset,
			WalLength:  entryTotalLen,
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
		}

		records = append(records, record)
		currentWalOffset += uint64(entryTotalLen)
	}

	// Compute lifecycle status across all records
	summary := computeLifecycleStatuses(records)

	return records, summary, walPath, valPath, nil
}

// loadValues lazily fetches the actual value bytes from value_log.bin only for the records being rendered.
func loadValues(valPath string, records []InspectedRecord) {
	valFile, err := os.Open(valPath)
	if err != nil {
		return
	}
	defer valFile.Close()

	for i := range records {
		r := &records[i]
		if r.Deleted {
			r.Value = "<tombstone>"
			continue
		}

		valLenBuf := make([]byte, 4)
		if _, err := valFile.ReadAt(valLenBuf, int64(r.DataOffset)); err == nil {
			valLen := binary.BigEndian.Uint32(valLenBuf)
			r.DataLength = 4 + valLen
			valBuf := make([]byte, valLen)
			if _, err := valFile.ReadAt(valBuf, int64(r.DataOffset+4)); err == nil {
				var vRec value_log_pb.ValueLogRecord
				if proto.Unmarshal(valBuf, &vRec) == nil {
					r.Value = string(vRec.Value)
				}
			}
		}
		if r.Value == "" {
			r.Value = "<pending/empty>"
		}
	}
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
			r.RecordType = "TOMBSTONE"
			r.Liveness = "-"
			r.Status = "TOMBSTONE"
			summary.Tombstones++
			summary.CompactionWaste++
			continue
		}

		r.RecordType = "PUT"
		summary.Puts++

		if lastSeenIndex[r.Key] != i {
			r.Liveness = "stale"
			r.Status = "SUPERSEDED"
			summary.StaleRecords++
			summary.CompactionWaste++
			continue
		}

		if r.HasExpiry && r.ExpiresAt.Before(now) {
			r.Liveness = "expired"
			r.Status = "EXPIRED"
			summary.ExpiredRecords++
			summary.CompactionWaste++
			continue
		}

		r.Liveness = "live"
		r.Status = "ACTIVE"
		summary.LiveRecords++
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
		if activeOnly && r.Liveness != "live" {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// PrintInspectTable renders a formatted table showing disk records and summary statistics.
func PrintInspectTable(w io.Writer, dir, walPath, valPath string, records []InspectedRecord, summary InspectSummary, totalFiltered int, useHex bool) {
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
	fmt.Fprintf(w, "----------------------------------------------------------------------------------------------------\n")
	if useHex {
		fmt.Fprintf(w, "  📖 Format: Hexadecimal Offsets (0x...)\n")
	} else {
		fmt.Fprintf(w, "  📖 Storage Layout:\n")
		fmt.Fprintf(w, "  • WAL Position:   Byte location in wal.bin (+record length)\n")
		fmt.Fprintf(w, "  • Value Position: Byte location in value_log.bin (+payload length)\n")
		fmt.Fprintf(w, "  • Record Type:    PUT (written record) or TOMBSTONE (soft-deleted entry)\n")
		fmt.Fprintf(w, "  • Liveness:       live (current active value), stale (overwritten), or expired (TTL)\n")
	}
	fmt.Fprintf(w, "====================================================================================================\n")

	if len(records) == 0 {
		fmt.Fprintf(w, "  (No matching records found in storage files)\n\n")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if useHex {
		fmt.Fprintf(tw, "  #\tWAL-OFFSET\tDATA-OFFSET\tKEY\tVALUE\tTIMESTAMP\tTTL\tRECORD TYPE\tLIVENESS\n")
		fmt.Fprintf(tw, "  -\t----------\t-----------\t---\t-----\t---------\t---\t-----------\t--------\n")
	} else {
		fmt.Fprintf(tw, "  #\tWAL POSITION\tVALUE POSITION\tKEY\tVALUE\tTIMESTAMP\tTTL\tRECORD TYPE\tLIVENESS\n")
		fmt.Fprintf(tw, "  -\t------------\t--------------\t---\t-----\t---------\t---\t-----------\t--------\n")
	}

	for _, r := range records {
		var walPosStr, valPosStr string
		if useHex {
			walPosStr = fmt.Sprintf("0x%06X", r.WalOffset)
			if r.Deleted {
				valPosStr = "-"
			} else {
				valPosStr = fmt.Sprintf("0x%06X", r.DataOffset)
			}
		} else {
			walPosStr = fmt.Sprintf("byte %d (%d B)", r.WalOffset, r.WalLength)
			if r.Deleted {
				valPosStr = "- (no payload)"
			} else if r.DataLength > 0 {
				valPosStr = fmt.Sprintf("byte %d (%d B)", r.DataOffset, r.DataLength)
			} else {
				valPosStr = fmt.Sprintf("byte %d", r.DataOffset)
			}
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

		typeStr := r.RecordType
		if r.RecordType == "TOMBSTONE" {
			typeStr = "💀 TOMBSTONE"
		} else {
			typeStr = "✍️  PUT"
		}

		livenessStr := r.Liveness
		switch r.Liveness {
		case "live":
			livenessStr = "✅ live"
		case "stale":
			livenessStr = "♻️  stale"
		case "expired":
			livenessStr = "⏰ expired"
		case "-":
			livenessStr = "-"
		}

		fmt.Fprintf(tw, "  %d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Index, walPosStr, valPosStr, r.Key, valStr, timeStr, ttlStr, typeStr, livenessStr)
	}
	tw.Flush()

	fmt.Fprintf(w, "----------------------------------------------------------------------------------------------------\n")
	if len(records) < totalFiltered {
		fmt.Fprintf(w, "  ℹ️  Showing most recent %d of %d matching records. (Use --all to display all)\n", len(records), totalFiltered)
	}
	fmt.Fprintf(w, "  Summary: %d records (%d PUTs, %d TOMBSTONEs) | %d live | %d stale | %d expired\n",
		summary.TotalRecords, summary.Puts, summary.Tombstones, summary.LiveRecords, summary.StaleRecords, summary.ExpiredRecords)
	if summary.CompactionWaste > 0 {
		fmt.Fprintf(w, "  ⚡ Compaction: %d stale/dead records can be reclaimed via 'tinystoredb-cli compact'\n",
			summary.CompactionWaste)
	} else {
		fmt.Fprintf(w, "  ✨ Compaction: Storage is fully clean and optimal (no stale records)\n")
	}
	fmt.Fprintf(w, "====================================================================================================\n\n")
}

func runWatchMode(ctx context.Context, dir string) {
	fmt.Printf("👀 Watching TinyStoreDB storage in %q... (Press Ctrl+C to exit)\n", dir)

	records, summary, walPath, valPath, err := ReadDiskRecords(dir)
	if err == nil {
		displayed := records
		if len(displayed) > 20 {
			displayed = displayed[len(displayed)-20:]
		}
		loadValues(valPath, displayed)
		PrintInspectTable(os.Stdout, dir, walPath, valPath, displayed, summary, len(records), false)
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
			currentRecords, _, _, valPath, err := ReadDiskRecords(dir)
			if err != nil {
				continue
			}

			if len(currentRecords) > lastCount {
				newBatch := currentRecords[lastCount:]
				loadValues(valPath, newBatch)
				for _, r := range newBatch {
					timeStr := time.Now().Format("15:04:05")
					if r.Deleted {
						fmt.Printf("[%s] 💀 TOMBSTONE      | KEY: %-15s | WAL: byte %d (%d B)\n",
							timeStr, r.Key, r.WalOffset, r.WalLength)
					} else {
						fmt.Printf("[%s] ✍️  PUT            | KEY: %-15s | VALUE: %-15s | VALUE POS: byte %d (%d B) | LIVENESS: %s\n",
							timeStr, r.Key, r.Value, r.DataOffset, r.DataLength, r.Liveness)
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
