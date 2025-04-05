package store

import (
	"errors"
	"time"

	"github.com/raghavgh/TinyStoreDB/disk"
	"github.com/raghavgh/TinyStoreDB/disk/value_log/proto/value_log_pb"
	"github.com/raghavgh/TinyStoreDB/disk/wal/proto/walpb"
	"github.com/raghavgh/TinyStoreDB/index"
	"github.com/raghavgh/TinyStoreDB/index/go_map"
	"github.com/raghavgh/TinyStoreDB/pair"
	"google.golang.org/protobuf/proto"
)

var (
	_test bool
)

func (kv *KVStore) Replay() error {
	walRecords, err := kv.wal.ReadAll(func() proto.Message {
		return &walpb.WALRecord{}
	})
	if err != nil {
		return err
	}

	var pairList pair.List[string, uint64]

	for _, record := range walRecords {
		walRecord := record.(*walpb.WALRecord)
		pairList = append(pairList, &pair.Pair[string, uint64]{Key: walRecord.Key, Value: walRecord.Offset})
	}

	kv.inMemoryIndex.Rebuild(pairList)

	return nil
}

func (kv *KVStore) Get(key string) (string, error) {
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

func NewKVStore() (*KVStore, error) {
	walFileName := func() string {
		if _test {
			return "wal_test.bin"
		}
		return "wal.bin"
	}()
	writeAheadLog, err := disk.New(walFileName)
	if err != nil {
		return nil, err
	}

	valueLogFileName := func() string {
		if _test {
			return "value_log_test.bin"
		}
		return "value_log.bin"
	}()

	valueLog, err := disk.New(valueLogFileName)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		inMemoryIndex: go_map.New[string, uint64](),
		wal:           writeAheadLog,
		valueLog:      valueLog,
	}, nil
}

type KVStore struct {
	inMemoryIndex index.Index[string, uint64]
	wal           *disk.TinyFile
	valueLog      *disk.TinyFile
}
