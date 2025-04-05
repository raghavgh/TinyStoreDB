package store

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func (s *KVStoreTestSuite) SetupTest() {
	_test = true

	s.store, _ = NewKVStore()
}

func (s *KVStoreTestSuite) TearDownTest() {
	_ = s.store.wal.Close(true)
	_ = s.store.valueLog.Close(true)
	s.store.inMemoryIndex.Clear()
}

// TestSet tests the Set method of the KVStore.
func (s *KVStoreTestSuite) TestSet() {
	err := s.store.Set("key1", "value1")
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
	setErr := s.store.Set("TestSetAndGetKey", "TestSetAndGetValue")
	s.NoError(setErr)

	getVal, getErr := s.store.Get("TestSetAndGetKey")
	s.NoError(getErr)
	s.Equal("TestSetAndGetValue", getVal)
}

// TestReplay tests the Replay method of the KVStore.
// set random values, and then clear the store, and then replay the values,
// and check if the values are still there.
func (s *KVStoreTestSuite) TestReplay() {
	// Set some values
	_ = s.store.Set("key1", "value1")
	_ = s.store.Set("key2", "value2")
	_ = s.store.Set("key3", "value3")

	// Clear the store
	s.store.inMemoryIndex.Clear()

	_ = s.store.wal.Close(false)
	_ = s.store.valueLog.Close(false)

	s.store, _ = NewKVStore()

	// Replay the values
	err := s.store.Replay()
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

func TestKVStoreSuite(t *testing.T) {
	suite.Run(t, new(KVStoreTestSuite))
}

type KVStoreTestSuite struct {
	suite.Suite
	store *KVStore
}
