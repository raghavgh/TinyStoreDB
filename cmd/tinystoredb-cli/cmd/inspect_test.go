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
	// Record 1: Set user:1 -> Alice
	err = s.Set("user:1", "Alice", nil)
	require.NoError(t, err)

	// Record 2: Set user:2 -> Bob (with future TTL)
	futureTTL := uint64(time.Now().Unix() + 3600)
	err = s.Set("user:2", "Bob", &futureTTL)
	require.NoError(t, err)

	// Record 3: Overwrite user:1 -> Alice Cooper (user:1 record 1 is now SUPERSEDED)
	err = s.Set("user:1", "Alice Cooper", nil)
	require.NoError(t, err)

	// Record 4: Delete user:2 (user:2 record 2 is now SUPERSEDED, record 4 is TOMBSTONE)
	err = s.Delete("user:2")
	require.NoError(t, err)

	// 3. Inspect the disk directory
	records, summary, walPath, valPath, err := ReadDiskRecords(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, walPath)
	assert.NotEmpty(t, valPath)

	// Check total records
	require.Len(t, records, 4)

	// Verify Record 1: user:1 -> Alice (SUPERSEDED)
	assert.Equal(t, "user:1", records[0].Key)
	assert.Equal(t, "Alice", records[0].Value)
	assert.Equal(t, "SUPERSEDED", records[0].Status)
	assert.False(t, records[0].Deleted)

	// Verify Record 2: user:2 -> Bob (SUPERSEDED by tombstone)
	assert.Equal(t, "user:2", records[1].Key)
	assert.Equal(t, "Bob", records[1].Value)
	assert.Equal(t, "SUPERSEDED", records[1].Status)
	assert.False(t, records[1].Deleted)

	// Verify Record 3: user:1 -> Alice Cooper (ACTIVE)
	assert.Equal(t, "user:1", records[2].Key)
	assert.Equal(t, "Alice Cooper", records[2].Value)
	assert.Equal(t, "ACTIVE", records[2].Status)
	assert.False(t, records[2].Deleted)

	// Verify Record 4: user:2 (TOMBSTONE)
	assert.Equal(t, "user:2", records[3].Key)
	assert.Equal(t, "<tombstone>", records[3].Value)
	assert.Equal(t, "TOMBSTONE", records[3].Status)
	assert.True(t, records[3].Deleted)

	// Verify Summary
	assert.Equal(t, 4, summary.TotalRecords)
	assert.Equal(t, 1, summary.ActiveRecords)
	assert.Equal(t, 2, summary.Superseded)
	assert.Equal(t, 1, summary.Tombstones)
	assert.Equal(t, 3, summary.CompactionWaste) // 2 superseded + 1 tombstone

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

	// Test Table Rendering output
	var buf bytes.Buffer
	PrintInspectTable(&buf, tempDir, walPath, valPath, records, summary)
	out := buf.String()
	assert.Contains(t, out, "TinyStoreDB Storage Engine Inspector")
	assert.Contains(t, out, "user:1")
	assert.Contains(t, out, "Alice Cooper")
	assert.Contains(t, out, "💀 TOMBSTONE")
	assert.Contains(t, out, "♻️  SUPERSEDED")
	assert.Contains(t, out, "✅ ACTIVE")
	assert.Contains(t, out, "3 dead/stale records can be reclaimed via 'tinystoredb-cli compact'")
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
