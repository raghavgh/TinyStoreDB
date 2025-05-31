package store

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/disk"
	"github.com/raghavgh/TinyStoreDB/disk/files"
	"github.com/raghavgh/TinyStoreDB/disk/record/value_log/proto/value_log_pb"
	"github.com/raghavgh/TinyStoreDB/disk/record/wal/proto/walpb"
	"github.com/raghavgh/TinyStoreDB/index"
	"github.com/raghavgh/TinyStoreDB/index/go_map"
	"github.com/raghavgh/TinyStoreDB/index/thread_safe_map"
	"github.com/raghavgh/TinyStoreDB/pair"
	"google.golang.org/protobuf/proto"
)

var (
	_test bool

	_writerBarrier atomic.Bool

	_readBarrier atomic.Bool
)

func (kv *KVStore) Compact() error {
	newWal, err := files.New(kv.fullPath("wal_new.bin"))
	if err != nil {
		return err
	}

	newValueLog, err := files.New(kv.fullPath("value_log_new.bin"))
	if err != nil {
		return err
	}

	// acquire the writer barrier, to freeze the writes
	_writerBarrier.Store(true)
	defer _writerBarrier.Store(false)

	walRecord := &walpb.WALRecord{}

	kv.wal.ReadAt(0, walRecord)

	snapShotWalRecords, err := kv.wal.ReadAll(func() proto.Message {
		return &walpb.WALRecord{}
	})
	if err != nil {
		return err
	}

	seenMap := go_map.New[string, struct{}]()

	for _, record := range snapShotWalRecords {
		walRecord := record.(*walpb.WALRecord)
		if walRecord.Deleted {
			// skip deleted records
			continue
		}

		if _, ok := seenMap.Get(walRecord.Key); ok {
			// skip duplicate records
			continue
		}

		val, ok := kv.inMemoryIndex.Get(walRecord.Key)
		if !ok {
			// it means the key was deleted
			continue
		}

		offset := val

		value := &value_log_pb.ValueLogRecord{}
		err := kv.valueLog.ReadAt(offset, value)
		if err != nil {
			return err
		}

		if value == nil {
			continue
		}

		newOffset, appendErr := newValueLog.Append(value)
		if appendErr != nil {
			return appendErr
		}

		_, appendErr = newWal.Append(&walpb.WALRecord{
			Key:       walRecord.Key,
			Offset:    newOffset,
			Timestamp: walRecord.Timestamp,
		})
		if appendErr != nil {
			return appendErr
		}

		seenMap.Put(walRecord.Key, struct{}{})
	}

	err = kv.wal.Rename(kv.fullPath("wal_old.bin"))
	if err != nil {
		return err
	}

	err = kv.valueLog.Rename(kv.fullPath("value_log_old.bin"))
	if err != nil {
		return err
	}

	oldWal := kv.wal
	oldValueLog := kv.valueLog

	kv.wal = newWal
	kv.valueLog = newValueLog

	_ = kv.wal.Rename(kv.fullPath("wal.bin"))
	_ = kv.valueLog.Rename(kv.fullPath("value_log.bin"))

	_readBarrier.Store(true)

	_ = kv.Replay()

	_readBarrier.Store(false)

	_writerBarrier.Store(false)

	// close old files
	err = oldWal.Close(true)
	if err != nil {
		log.Printf("error while closing old wal file: %s", err.Error())
	}

	err = oldValueLog.Close(true)
	if err != nil {
		log.Printf("error while closing old value log file: %s", err.Error())
	}

	return nil
}

func (kv *KVStore) Replay() error {
	walRecords, err := kv.wal.ReadAll(func() proto.Message {
		return &walpb.WALRecord{}
	})
	if err != nil {
		return err
	}

	var pairList pair.List[string, uint64]

	deletedKeys := map[string]struct{}{}

	for _, record := range walRecords {
		walRecord := record.(*walpb.WALRecord)
		if walRecord.Deleted {
			deletedKeys[walRecord.Key] = struct{}{}

			continue
		}

		// if key exist in deletedMap remove it since it set again as per logs
		delete(deletedKeys, walRecord.Key)

		pairList = append(pairList, &pair.Pair[string, uint64]{Key: walRecord.Key, Value: walRecord.Offset})
	}

	kv.inMemoryIndex.Rebuild(pairList)

	for deletedKey := range deletedKeys {
		kv.inMemoryIndex.Delete(deletedKey)
	}

	return nil
}

func (kv *KVStore) Get(key string) (string, error) {
	if _readBarrier.Load() {
		return "", errors.New("retryable error")
	}

	val, ok := kv.inMemoryIndex.Get(key)
	if !ok {
		return "", errors.New("key not found")
	}

	value := &value_log_pb.ValueLogRecord{}

	err := kv.valueLog.ReadAt(val, value)
	if err != nil {
		return "", err
	}

	if value == nil {
		return "", nil
	}

	return string(value.Value), nil
}

func (kv *KVStore) Set(key string, value string) error {
	if _writerBarrier.Load() {
		return errors.New("retryable error")
	}

	offset, err := kv.valueLog.Append(&value_log_pb.ValueLogRecord{
		Value:     []byte(value),
		Timestamp: uint64(time.Now().Unix()),
	})
	if err != nil {
		return err
	}

	kv.inMemoryIndex.Put(key, offset)
	_, err = kv.wal.Append(&walpb.WALRecord{
		Key:       key,
		Offset:    offset,
		Timestamp: uint64(time.Now().Unix()),
	})
	if err != nil {
		// remove from in-memory index, to prevent garbage data.
		kv.inMemoryIndex.Delete(key)

		//TODO: also remove from value log, once we have a way to do that.

		return err
	}

	return nil
}

func (kv *KVStore) fullPath(filename string) string {
	return filepath.Join(kv.dataDirectory, filename)
}

func (kv *KVStore) Delete(key string) error {
	if _writerBarrier.Load() {
		return errors.New("retryable error")
	}

	_, ok := kv.inMemoryIndex.Get(key)
	if !ok {
		return errors.New("key not found")
	}

	_, err := kv.wal.Append(&walpb.WALRecord{
		Key:       key,
		Timestamp: uint64(time.Now().Unix()),
		Deleted:   true,
	})
	if err != nil {
		return err
	}

	kv.inMemoryIndex.Delete(key)
	_, err = kv.wal.Append(&walpb.WALRecord{
		Key:       key,
		Timestamp: uint64(time.Now().Unix()),
		Deleted:   true,
	})

	if err != nil {
		return err
	}

	return nil
}

func ensureDirExists(path string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Directory does not exist, so create it
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("failed to create directory: %v", err)
		}
	} else if err != nil {
		log.Fatalf("failed to stat directory: %v", err)
	} else if !info.IsDir() {
		log.Fatalf("%s exists but is not a directory", path)
	}
}

func NewKVStore(cfg *config.Config) (*KVStore, error) {
	ensureDirExists(cfg.DataDir)

	kvStore := &KVStore{
		inMemoryIndex: thread_safe_map.New[string, uint64](),
		dataDirectory: cfg.DataDir,
	}

	walFileName := func() string {
		if _test {
			return kvStore.fullPath("wal_test.bin")
		}
		return kvStore.fullPath("wal.bin")
	}()

	writeAheadLog, err := files.New(walFileName)
	if err != nil {
		return nil, err
	}

	valueLogFileName := func() string {
		if _test {
			return kvStore.fullPath("value_log_test.bin")
		}
		return kvStore.fullPath("value_log.bin")
	}()

	valueLog, err := files.New(valueLogFileName)
	if err != nil {
		return nil, err
	}

	kvStore.valueLog = valueLog
	kvStore.wal = writeAheadLog

	return kvStore, nil
}

type KVStore struct {
	inMemoryIndex index.Index[string, uint64]
	wal           disk.Disk
	valueLog      disk.Disk
	dataDirectory string
}
