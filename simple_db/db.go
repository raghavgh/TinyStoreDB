package simple_db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

type SimpleDB struct {
	file *os.File
}

const dbPath = "simple_db/db.bin"

/*// Initialize the database and register types
func init() {
	// Register the types you intend to store
	gob.Register(string(""))
	gob.Register(int(0))
	gob.Register(float64(0))
	// Add more types as needed
}*/

// NewSimpleDB initializes and returns a new SimpleDB instance
func NewSimpleDB() *SimpleDB {
	file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}

	return &SimpleDB{file: file}
}

// Set stores the key-value pair in the database
func (db *SimpleDB) Set(key, value []byte) error {
	// Write the length of keyBytes
	if err := writeBytesWithLength(db.file, key); err != nil {
		return err
	}

	// Write the length of valueBytes
	if err := writeBytesWithLength(db.file, value); err != nil {
		return err
	}

	return nil
}

// Get retrieves the value associated with the given key
func (db *SimpleDB) Get(keyBytes []byte) ([]byte, error) {
	// Reset file pointer to the beginning
	_, err := db.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	for {
		// Read key length
		kLen, err := readLength(db.file)
		if err == io.EOF {
			break // Reached end of file, key not found
		}
		if err != nil {
			return nil, err
		}

		// Read key bytes
		currentKeyBytes := make([]byte, kLen)
		_, err = io.ReadFull(db.file, currentKeyBytes)
		if err != nil {
			return nil, err
		}

		// Compare keys
		if bytes.Equal(currentKeyBytes, keyBytes) {
			// Read value length
			vLen, err := readLength(db.file)
			if err != nil {
				return nil, err
			}

			// Read value bytes
			valueBytes := make([]byte, vLen)
			_, err = io.ReadFull(db.file, valueBytes)
			if err != nil {
				return nil, err
			}

			return valueBytes, nil
		} else {
			// Skip the value associated with the non-matching key
			vLen, err := readLength(db.file)
			if err != nil {
				return nil, err
			}
			_, err = db.file.Seek(int64(vLen), io.SeekCurrent)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, errors.New("key not found")
}

// writeBytesWithLength writes the length of the byte slice followed by the bytes themselves
func writeBytesWithLength(file *os.File, data []byte) error {
	length := uint32(len(data))
	// Write length as 8 bytes (uint32)
	err := binaryWrite(file, length)
	if err != nil {
		return err
	}
	// Write the actual Data
	_, err = file.Write(data)
	return err
}

// readLength reads the first 4 bytes and interprets them as an uint32 length
func readLength(file *os.File) (uint32, error) {
	var length uint32
	err := binaryRead(file, &length)
	return length, err
}

// binaryWrite writes binary Data to the file
func binaryWrite(file *os.File, data any) error {
	return binary.Write(file, binary.BigEndian, data)
}

// binaryRead reads binary Data from the file
func binaryRead(file *os.File, data any) error {
	return binary.Read(file, binary.BigEndian, data)
}
