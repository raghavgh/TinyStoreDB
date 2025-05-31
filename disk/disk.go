package disk

import "google.golang.org/protobuf/proto"

type Disk interface {
	// Append appends a message to the disk and returns the offset of the message.
	Append(message proto.Message) (uint64, error)

	// ReadAt reads a message from the disk at the given offset.
	ReadAt(offset uint64, message proto.Message) error

	// ReadAll reads all messages from the disk and returns them as a slice of proto.Message.
	ReadAll(newMessage func() proto.Message) ([]proto.Message, error)

	// SetOffset sets the offset for the next write operation.
	SetOffset(offset uint64)

	// Close closes the disk and optionally deletes the underlying file.
	Close(delete bool) error

	// Lock locks the disk for writing.
	Lock()

	Rename(newName string) error

	// Unlock unlocks the disk for writing.
	Unlock()
}
