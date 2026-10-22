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
	"github.com/raghavgh/TinyStoreDB/utils"
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
		_ = newWal.Close(true)
		return err
	}

	// acquire the writer barrier, to freeze the writes
	_writerBarrier.Store(true)
	defer _writerBarrier.Store(false)

	snapShotWalRecords, err := kv.wal.ReadAll(func() proto.Message {
		return &walpb.WALRecord{}
	})
	if err != nil {
		_ = newWal.Close(true)
		_ = newValueLog.Close(true)
		return err
	}

	seenMap := go_map.New[string, struct{}]()
	nowUnix := uint64(time.Now().Unix())

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
			_ = newWal.Close(true)
			_ = newValueLog.Close(true)
			return err
		}

		if value == nil {
			continue
		}

		// Determine true TTL from latest payload or WAL record
		var recTTL *uint64
		if value.Ttl != nil && *value.Ttl > 0 {
			recTTL = value.Ttl
		} else if walRecord.Ttl != nil && *walRecord.Ttl > 0 {
			recTTL = walRecord.Ttl
		}

		// Check if record has expired: if so, purge it during compaction!
		if recTTL != nil && *recTTL > 0 && *recTTL < nowUnix {
			seenMap.Put(walRecord.Key, struct{}{})
			continue
		}

		newOffset, appendErr := newValueLog.Append(value)
		if appendErr != nil {
			_ = newWal.Close(true)
			_ = newValueLog.Close(true)
			return appendErr
		}

		_, appendErr = newWal.Append(&walpb.WALRecord{
			Key:       walRecord.Key,
			Offset:    newOffset,
			Timestamp: walRecord.Timestamp,
			Ttl:       recTTL,
		})
		if appendErr != nil {
			_ = newWal.Close(true)
			_ = newValueLog.Close(true)
			return appendErr
		}

		seenMap.Put(walRecord.Key, struct{}{})
	}

	_readBarrier.Store(true)
	defer _readBarrier.Store(false)

	oldWal := kv.wal
	oldValueLog := kv.valueLog

	// Close old files before atomically replacing them
	_ = oldWal.Close(false)
	_ = oldValueLog.Close(false)

	targetWal := kv.fullPath(kv.walFileName)
	targetValueLog := kv.fullPath(kv.valueFileName)

	if err := newWal.Rename(targetWal); err != nil {
		return err
	}

	if err := newValueLog.Rename(targetValueLog); err != nil {
		return err
	}

	kv.wal = newWal
	kv.valueLog = newValueLog

	_ = kv.Replay()

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

		pairList = append(
			pairList,
			&pair.Pair[string, uint64]{Key: walRecord.Key, Value: walRecord.Offset},
		)
	}

	kv.inMemoryIndex.Rebuild(pairList)

	for deletedKey := range deletedKeys {
		kv.inMemoryIndex.Delete(deletedKey)
	}

	return nil
}

func (kv *KVStore) getValue(key string) (*value_log_pb.ValueLogRecord, error) {
	if _readBarrier.Load() {
		return nil, errors.New("retryable error")
	}

	val, ok := kv.inMemoryIndex.Get(key)
	if !ok {
		return nil, errors.New("key not found")
	}

	value := &value_log_pb.ValueLogRecord{}

	err := kv.valueLog.ReadAt(val, value)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	if value.Ttl != nil && value.GetTtl() < uint64(time.Now().Unix()) {

		err = kv.Delete(key)
		if err != nil {
			log.Printf("error while deleting key %s: %s\n", key, err.Error())
			return nil, errors.New("key not found")
		}
	}

	return value, nil
}

func (kv *KVStore) Get(key string) (string, error) {
	value, err := kv.getValue(key)
	if err != nil {
		return "", err
	}

	return string(value.Value), nil
}

func (kv *KVStore) Set(key string, value string, ttl *uint64) error {
	if _writerBarrier.Load() {
		return errors.New("retryable error")
	}

	offset, err := kv.valueLog.Append(&value_log_pb.ValueLogRecord{
		Value:     []byte(value),
		Timestamp: utils.ToPointer(uint64(time.Now().Unix())),
		Ttl:       ttl,
	})
	if err != nil {
		return err
	}

	_, err = kv.wal.Append(&walpb.WALRecord{
		Key:       key,
		Offset:    offset,
		Timestamp: utils.ToPointer(uint64(time.Now().Unix())),
		Ttl:       ttl,
	})
	if err != nil {
		return err
	}

	kv.inMemoryIndex.Put(key, offset)
	return nil
}

func (kv *KVStore) fullPath(filename string) string {
	return filepath.Join(kv.dataDirectory, filename)
}

func (kv *KVStore) Exist(key string) (bool, error) {
	if _readBarrier.Load() {
		return false, errors.New("retryable error")
	}

	val, err := kv.getValue(key)
	if err != nil {
		return false, nil
	}

	if val.Ttl != nil && val.GetTtl() < uint64(time.Now().Unix()) {
		err = kv.Delete(key)
		if err != nil {
			log.Printf("error while deleting key %s: %s\n", key, err.Error())
		}

		return false, nil
	}

	return true, nil
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
		Timestamp: utils.ToPointer(uint64(time.Now().Unix())),
		Deleted:   true,
	})
	if err != nil {
		return err
	}

	kv.inMemoryIndex.Delete(key)

	return nil
}

func ensureDirExists(path string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Directory does not exist, so create it
		if err := os.MkdirAll(path, 0o755); err != nil {
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

	walBase := "wal.bin"
	valueLogBase := "value_log.bin"
	if _test {
		walBase = "wal_test.bin"
		valueLogBase = "value_log_test.bin"
	}

	kvStore := &KVStore{
		inMemoryIndex: thread_safe_map.New[string, uint64](),
		dataDirectory: cfg.DataDir,
		walFileName:   walBase,
		valueFileName: valueLogBase,
	}

	writeAheadLog, err := files.New(kvStore.fullPath(walBase))
	if err != nil {
		return nil, err
	}

	valueLog, err := files.New(kvStore.fullPath(valueLogBase))
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
	walFileName   string
	valueFileName string
}
