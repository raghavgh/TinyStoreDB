package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectDiskRecords(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize store in temp directory
	cfg := &config.Config{
		DataDir: tempDir,
	}
	s, err := store.NewKVStore(cfg)
	require.NoError(t, err)

	// 2. Perform disk operations:
	// Record 1: Set user:1 -> Alice (PUT)
	err = s.Set("user:1", "Alice", nil)
	require.NoError(t, err)

	// Record 2: Set user:2 -> Bob (PUT with future TTL)
	futureTTL := uint64(time.Now().Unix() + 3600)
	err = s.Set("user:2", "Bob", &futureTTL)
	require.NoError(t, err)

	// Record 3: Overwrite user:1 -> Alice Cooper (user:1 record 1 is now stale)
	err = s.Set("user:1", "Alice Cooper", nil)
	require.NoError(t, err)

	// Record 4: Delete user:2 (user:2 record 2 is now stale, record 4 is TOMBSTONE)
	err = s.Delete("user:2")
	require.NoError(t, err)

	// 3. Inspect the disk directory (Fast metadata scan)
	records, summary, walPath, valPath, err := ReadDiskRecords(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, walPath)
	assert.NotEmpty(t, valPath)

	// Check total records
	require.Len(t, records, 4)

	// Verify Record 1: user:1 (PUT, stale)
	assert.Equal(t, "user:1", records[0].Key)
	assert.Equal(t, "PUT", records[0].RecordType)
	assert.Equal(t, "stale", records[0].Liveness)
	assert.False(t, records[0].Deleted)

	// Verify Record 2: user:2 (PUT, stale by tombstone)
	assert.Equal(t, "user:2", records[1].Key)
	assert.Equal(t, "PUT", records[1].RecordType)
	assert.Equal(t, "stale", records[1].Liveness)
	assert.False(t, records[1].Deleted)

	// Verify Record 3: user:1 (PUT, live)
	assert.Equal(t, "user:1", records[2].Key)
	assert.Equal(t, "PUT", records[2].RecordType)
	assert.Equal(t, "live", records[2].Liveness)
	assert.False(t, records[2].Deleted)

	// Verify Record 4: user:2 (TOMBSTONE)
	assert.Equal(t, "user:2", records[3].Key)
	assert.Equal(t, "TOMBSTONE", records[3].RecordType)
	assert.Equal(t, "-", records[3].Liveness)
	assert.Equal(t, "<tombstone>", records[3].Value)
	assert.True(t, records[3].Deleted)

	// Verify Summary
	assert.Equal(t, 4, summary.TotalRecords)
	assert.Equal(t, 3, summary.Puts)
	assert.Equal(t, 1, summary.Tombstones)
	assert.Equal(t, 1, summary.LiveRecords)
	assert.Equal(t, 2, summary.StaleRecords)
	assert.Equal(t, 3, summary.CompactionWaste) // 2 stale + 1 tombstone

	// Test Lazy Value Loading only for displayed records
	displayed := records[2:] // records 3 & 4
	loadValues(valPath, displayed)
	assert.Equal(t, "Alice Cooper", displayed[0].Value)
	assert.Equal(t, "<tombstone>", displayed[1].Value)
	assert.Greater(t, displayed[0].DataLength, uint32(0))

	// Load all values for full render test
	loadValues(valPath, records)
	assert.Equal(t, "Alice", records[0].Value)
	assert.Equal(t, "Bob", records[1].Value)

	// Test Filtering
	tombstones := filterRecords(records, "", true, false)
	require.Len(t, tombstones, 1)
	assert.Equal(t, "user:2", tombstones[0].Key)

	active := filterRecords(records, "", false, true)
	require.Len(t, active, 1)
	assert.Equal(t, "user:1", active[0].Key)
	assert.Equal(t, "Alice Cooper", active[0].Value)

	byKey := filterRecords(records, "user:1", false, false)
	require.Len(t, byKey, 2)

	// Test Human-Readable Table Rendering
	var buf bytes.Buffer
	PrintInspectTable(&buf, tempDir, walPath, valPath, records, summary, len(records), false)
	out := buf.String()
	assert.Contains(t, out, "TinyStoreDB Storage Engine Inspector")
	assert.Contains(t, out, "RECORD TYPE")
	assert.Contains(t, out, "LIVENESS")
	assert.Contains(t, out, "PUT")
	assert.Contains(t, out, "TOMBSTONE")
	assert.Contains(t, out, "live")
	assert.Contains(t, out, "stale")
	assert.Contains(t, out, "byte 0")
	assert.Contains(t, out, "- (no payload)")
	assert.Contains(t, out, "user:1")
	assert.Contains(t, out, "Alice Cooper")
	assert.Contains(t, out, "3 stale/dead records can be reclaimed via 'tinystoredb-cli compact'")

	// Test Hex Mode Table Rendering
	var hexBuf bytes.Buffer
	PrintInspectTable(&hexBuf, tempDir, walPath, valPath, records, summary, len(records), true)
	hexOut := hexBuf.String()
	assert.Contains(t, hexOut, "0x000000")
	assert.Contains(t, hexOut, "Format: Hexadecimal Offsets")
}

func TestInspectEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// When WAL file doesn't exist
	_, _, _, _, err := ReadDiskRecords(tempDir)
	assert.Error(t, err)

	// Create empty WAL file
	_ = os.WriteFile(filepath.Join(tempDir, "wal.bin"), []byte{}, 0644)
	records, summary, _, _, err := ReadDiskRecords(tempDir)
	assert.NoError(t, err)
	assert.Empty(t, records)
	assert.Equal(t, 0, summary.TotalRecords)
}
