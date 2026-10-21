package store_test

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/raghavgh/TinyStoreDB/client"
	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/server"
	"github.com/raghavgh/TinyStoreDB/store"
	"github.com/stretchr/testify/require"
)

func startTestServer(t *testing.T, secret *string) (*client.TinyStoreClient, *store.KVStore, string) {
	t.Helper()
	tempDir := t.TempDir()
	cfg := &config.Config{
		Port:     "0",
		DataDir:  tempDir,
		LogLevel: "info",
		Secret:   secret,
	}

	kv, err := store.NewKVStore(cfg)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := server.NewGRPCServer(kv, secret)
	go func() {
		_ = grpcServer.Serve(lis)
	}()

	cli, err := client.New(lis.Addr().String(), secret)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = cli.Close()
		grpcServer.GracefulStop()
		_ = lis.Close()
	})

	return cli, kv, tempDir
}

func TestConcurrentSetAndGet(t *testing.T) {
	token := "test-secret-token"
	client, _, _ := startTestServer(t, &token)

	runtime.GOMAXPROCS(runtime.NumCPU())

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			val := fmt.Sprintf("val%d", i)
			err := client.Set(context.Background(), key, val, nil)
			if err != nil {
				t.Errorf("Set failed: %v", err)
				return
			}

			got, err := client.Get(context.Background(), key)
			if err != nil || got != val {
				t.Errorf("Get failed: expected %s, got %s, err: %v", val, got, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestSimple(t *testing.T) {
	token := "test-secret-token"
	client, _, _ := startTestServer(t, &token)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Set(ctx, "hello", "world", nil)
	require.NoError(t, err)

	val, err := client.Get(ctx, "hello")
	require.NoError(t, err)
	require.Equal(t, "world", val)

	exists, err := client.Exist(ctx, "hello")
	require.NoError(t, err)
	require.True(t, exists)

	deleted, err := client.Delete(ctx, "hello")
	require.NoError(t, err)
	require.True(t, deleted)

	existsAfterDel, err := client.Exist(ctx, "hello")
	require.NoError(t, err)
	require.False(t, existsAfterDel)
}

func TestCompactionWithEdgeCases(t *testing.T) {
	token := "test-secret-token"
	client, _, _ := startTestServer(t, &token)

	ctx := context.Background()

	// Set initial values
	err := client.Set(ctx, "key1", "val1", nil)
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

	// Trigger compaction
	err = client.Compact(ctx)
	require.NoError(t, err)

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

func TestReplayAndRecovery(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		DataDir: tempDir,
	}

	// 1. First session: write data
	kv1, err := store.NewKVStore(cfg)
	require.NoError(t, err)

	err = kv1.Set("persist1", "val1", nil)
	require.NoError(t, err)
	err = kv1.Set("persist2", "val2", nil)
	require.NoError(t, err)
	err = kv1.Set("to_delete", "val_delete", nil)
	require.NoError(t, err)
	err = kv1.Delete("to_delete")
	require.NoError(t, err)

	// 2. Re-open second session on same dataDir
	kv2, err := store.NewKVStore(cfg)
	require.NoError(t, err)

	err = kv2.Replay()
	require.NoError(t, err)

	val1, err := kv2.Get("persist1")
	require.NoError(t, err)
	require.Equal(t, "val1", val1)

	val2, err := kv2.Get("persist2")
	require.NoError(t, err)
	require.Equal(t, "val2", val2)

	_, err = kv2.Get("to_delete")
	require.Error(t, err)
}
