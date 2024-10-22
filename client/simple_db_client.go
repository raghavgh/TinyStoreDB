package client

import (
	"bytes"
	"encoding/gob"

	"simple-db-go/simple_db"
)

type SimpleDBClient struct {
	db *simple_db.SimpleDB
}

func NewSimpleDBClient() *SimpleDBClient {
	db := simple_db.NewSimpleDB()
	return &SimpleDBClient{
		db: db,
	}
}

func (sdc *SimpleDBClient) Set(key, val string) error {
	keyBytes, err := serialize(key)
	if err != nil {
		return err
	}

	valueBytes, err := serialize(val)
	if err != nil {
		return err
	}

	return sdc.db.Set(keyBytes, valueBytes)
}

func (sdc *SimpleDBClient) Get(key string) (*string, error) {
	keyBytes, err := serialize(key)
	if err != nil {
		return nil, err
	}

	var valBytes []byte

	valBytes, err = sdc.db.Get(keyBytes)
	if err != nil {
		return nil, err
	}

	var value string

	// Deserialize and return the value
	value, err = deserialize(valBytes)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

// serialize encodes a value using gob
func serialize(value string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(value)
	return buf.Bytes(), err
}

// deserialize decodes a value using gob
func deserialize(data []byte) (string, error) {
	var value string
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&value)
	return value, err
}
