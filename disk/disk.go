package disk

import (
	"encoding/binary"
	"io"
	"os"
	"sync"

	"google.golang.org/protobuf/proto"
)

func (w *TinyFile) ReadAll(newMessage func() proto.Message) ([]proto.Message, error) {
	var messages []proto.Message

	fi, _ := w.file.Stat()
	if fi == nil || fi.Size() == 0 {
		return nil, nil
	}

	var offset uint64

	for {

		var lengthPrefix uint32

		err := binary.Read(w.file, binary.BigEndian, &lengthPrefix)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		data := make([]byte, lengthPrefix)

		_, err = w.file.Read(data)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		message := newMessage()
		err = proto.Unmarshal(data, message)
		if err != nil {
			return nil, err
		}

		offset += uint64(len(data) + 4)

		messages = append(messages, message)
	}

	return messages, nil
}

func (w *TinyFile) ReadAt(offset uint64, messageData proto.Message) error {

	var lengthPrefix uint32

	lengthBuf := make([]byte, 4)
	_, err := w.file.ReadAt(lengthBuf, int64(offset))
	if err != nil {
		if err == io.EOF {
			return nil
		}

		return err
	}

	lengthPrefix = binary.BigEndian.Uint32(lengthBuf)

	data := make([]byte, lengthPrefix)

	_, err = w.file.ReadAt(data, int64(offset+4))
	if err != nil {
		if err == io.EOF {
			return nil
		}

		return err
	}

	err = proto.Unmarshal(data, messageData)
	if err != nil {
		return err
	}

	return nil
}

func (w *TinyFile) Append(message proto.Message) (uint64, error) {
	data, err := MarshalBinary(message)
	if err != nil {
		return 0, err
	}

	// Lock the file to prevent concurrent writes.
	w.Lock()
	defer w.Unlock()

	offset := w.currentOffset

	n, err := w.file.Write(data)
	if err != nil {
		return 0, err
	}

	// Update the current offset.
	w.currentOffset += uint64(n)

	// Sync the file to disk.
	err = w.file.Sync()
	if err != nil {
		return 0, err
	}

	return offset, nil
}

func MarshalBinary(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(data)))

	return append(lengthPrefix, data...), nil
}

func (w *TinyFile) SetOffset(offset uint64) {
	// Lock the file to prevent concurrent writes.
	w.Lock()
	defer w.Unlock()

	// Update the current offset.
	w.currentOffset = offset
}

func (w *TinyFile) Close(delete bool) error {
	if delete {
		err := os.Remove(w.file.Name())
		if err != nil {
			return err
		}
	}

	err := w.file.Close()
	if err != nil {
		return err
	}

	return nil
}

func New(name string) (*TinyFile, error) {
	file, err := os.OpenFile(name,
		os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &TinyFile{
		file:          file,
		currentOffset: uint64(stat.Size()),
	}, nil
}

type TinyFile struct {
	// TinyFile represents a disk WAL.
	file          *os.File
	currentOffset uint64
	sync.Mutex
}
