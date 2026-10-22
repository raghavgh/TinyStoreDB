package store

import (
	"testing"
	"time"

	"github.com/raghavgh/TinyStoreDB/config"
	"github.com/raghavgh/TinyStoreDB/disk/record/wal/proto/walpb"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func (s *KVStoreTestSuite) SetupTest() {
	_test = true
	s.dataDir = s.T().TempDir()
	cfg := &config.Config{
		DataDir: s.dataDir,
	}
	var err error
	s.store, err = NewKVStore(cfg)
	s.Require().NoError(err)
}

func (s *KVStoreTestSuite) TearDownTest() {
	_ = s.store.wal.Close(true)
	_ = s.store.valueLog.Close(true)
	s.store.inMemoryIndex.Clear()
}

// TestSet tests the Set method of the KVStore.
func (s *KVStoreTestSuite) TestSet() {
	err := s.store.Set("key1", "value1", nil)
	s.NoError(err)
}

// TestGetNotFound tests the Get method of the KVStore when the key is not found.
func (s *KVStoreTestSuite) TestGetNotFound() {
	val, err := s.store.Get("randomKey")
	s.Error(err)
	s.Equal("", val)
}

// TestSetAndGet tests the Set and Get methods of the KVStore.
func (s *KVStoreTestSuite) TestSetAndGet() {
	setErr := s.store.Set("TestSetAndGetKey", "TestSetAndGetValue", nil)
	s.NoError(setErr)

	getVal, getErr := s.store.Get("TestSetAndGetKey")
	s.NoError(getErr)
	s.Equal("TestSetAndGetValue", getVal)
}

// TestReplay tests the Replay method of the KVStore.
func (s *KVStoreTestSuite) TestReplay() {
	// Set some values
	_ = s.store.Set("key1", "value1", nil)
	_ = s.store.Set("key2", "value2", nil)
	_ = s.store.Set("key3", "value3", nil)

	// Clear the store
	s.store.inMemoryIndex.Clear()

	_ = s.store.wal.Close(false)
	_ = s.store.valueLog.Close(false)

	cfg := &config.Config{
		DataDir: s.dataDir,
	}
	var err error
	s.store, err = NewKVStore(cfg)
	s.Require().NoError(err)

	// Replay the values
	err = s.store.Replay()
	s.NoError(err)

	// Check if the values are still there
	val, err := s.store.Get("key1")
	s.NoError(err)
	s.Equal("value1", val)

	val, err = s.store.Get("key2")
	s.NoError(err)
	s.Equal("value2", val)

	val, err = s.store.Get("key3")
	s.NoError(err)
	s.Equal("value3", val)
}

// TestCompactionPurgesExpiredKeysAndPreservesTTL tests that expired keys are completely discarded during compaction,
// and live keys with future TTL preserve their TTL.
func (s *KVStoreTestSuite) TestCompactionPurgesExpiredKeysAndPreservesTTL() {
	// 1. Set key without TTL
	err := s.store.Set("liveKey", "liveVal", nil)
	s.NoError(err)

	// 2. Set key with future TTL (3600s)
	futureTTL := uint64(time.Now().Unix() + 3600)
	err = s.store.Set("futureTTLKey", "futureVal", &futureTTL)
	s.NoError(err)

	// 3. Set key with already-expired TTL (in the past)
	pastTTL := uint64(time.Now().Unix() - 10)
	err = s.store.Set("expiredKey", "expiredVal", &pastTTL)
	s.NoError(err)

	// 4. Run compaction
	err = s.store.Compact()
	s.NoError(err)

	// 5. Verify liveKey is present
	val, err := s.store.Get("liveKey")
	s.NoError(err)
	s.Equal("liveVal", val)

	// 6. Verify futureTTLKey is present
	val, err = s.store.Get("futureTTLKey")
	s.NoError(err)
	s.Equal("futureVal", val)

	// 7. Verify expiredKey was completely purged by compaction
	val, err = s.store.Get("expiredKey")
	s.Error(err)
	s.Equal("", val)

	// 8. Verify the compacted WAL records directly
	walRecords, err := s.store.wal.ReadAll(func() proto.Message {
		return &walpb.WALRecord{}
	})
	s.NoError(err)

	foundKeys := make(map[string]*walpb.WALRecord)
	for _, rec := range walRecords {
		wr := rec.(*walpb.WALRecord)
		foundKeys[wr.Key] = wr
	}

	// expiredKey must not be in the compacted WAL at all
	s.NotContains(foundKeys, "expiredKey")

	// futureTTLKey must have its TTL preserved
	s.Contains(foundKeys, "futureTTLKey")
	s.NotNil(foundKeys["futureTTLKey"].Ttl)
	s.Equal(futureTTL, *foundKeys["futureTTLKey"].Ttl)

	// liveKey must have no TTL
	s.Contains(foundKeys, "liveKey")
	s.Nil(foundKeys["liveKey"].Ttl)
}

func TestKVStoreSuite(t *testing.T) {
	suite.Run(t, new(KVStoreTestSuite))
}

type KVStoreTestSuite struct {
	suite.Suite
	store   *KVStore
	dataDir string
}
