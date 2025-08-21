package store

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/raghavgh/TinyStoreDB/client"
	"github.com/stretchr/testify/require"
)

func TestConcurrentSetAndGet(t *testing.T) { // your grpc client wrapper
	client, _ := client.New(
		"localhost:7389",
		toPointer("g6a5g65dfgasd65gdvwej5wr5hw6rh4w"),
	)

	runtime.GOMAXPROCS(runtime.NumCPU())

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			val := fmt.Sprintf("val%d", i)
			err := client.Set(context.Background(), key, val, nil)
			if err != nil {
				t.Errorf("Set failed: %v", err)
			}

			got, err := client.Get(context.Background(), key)
			if err != nil || got != val {
				t.Errorf("Get failed: expected %s, got %s", val, got)
			}
		}(i)
	}
	wg.Wait()
}

func TestSimple(t *testing.T) {
	client, err := client.New("localhost:7389", toPointer("g6a5g65dfgasd65gdvwej5wr5hw6rh4w"))
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = client.Set(ctx, "hello", "world", nil)
	require.NoError(t, err)

	val, err := client.Get(ctx, "hello")
	require.NoError(t, err)
	require.Equal(t, "world", val)
}

func TestCompactionWithEdgeCases(t *testing.T) {
	client, err := client.New("localhost:7389", toPointer("g6a5g65dfgasd65gdvwej5wr5hw6rh4w"))
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Set initial values
	err = client.Set(ctx, "key1", "val1", nil)
	require.NoError(t, err)

	err = client.Set(ctx, "key2", "val2", nil)
	require.NoError(t, err)

	// Update key1
	err = client.Set(ctx, "key1", "val1_updated", nil)
	require.NoError(t, err)

	// Delete key2
	_, err = client.Delete(ctx, "key2")
	require.NoError(t, err)

	// Write new keys
	for i := 3; i <= 10; i++ {
		err := client.Set(ctx, fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i), nil)
		require.NoError(t, err)
	}

	// Run concurrent reads while compaction is happening
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			_, _ = client.Get(ctx, "key1") // Read frequently updated key
			_, _ = client.Get(ctx, "key5") // Read steady key
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Trigger compaction
	err = client.Compact(ctx)
	require.NoError(t, err)

	wg.Wait()

	// Validate key1 is updated
	val, err := client.Get(ctx, "key1")
	require.NoError(t, err)
	require.Equal(t, "val1_updated", val)

	// Validate key2 is deleted
	_, err = client.Get(ctx, "key2")
	require.Error(t, err)

	// Validate other keys exist
	for i := 3; i <= 10; i++ {
		val, err := client.Get(ctx, fmt.Sprintf("key%d", i))
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("val%d", i), val)
	}
}

func toPointer[T any](val T) *T {
	return &val
}
