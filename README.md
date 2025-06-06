# 🚀 TinyStoreDB

**TinyStoreDB** is a lightweight, persistent key-value storage system built as a learning project. It provides basic operations to store and retrieve string-based key-value pairs using an append-only storage model.  
🚧 **Note:** This project is under active development and approaching an MVP-ready state.

## 📦 Features Implemented

- ✅ Set, Get, and Delete operations
- ✅ Append-only log-structured storage
- ✅ In-memory index for fast lookups
- ✅ Tombstone support for deletes
- ✅ Compaction to reclaim disk space
- ✅ Concurrency-safe operations (read/write)
- ✅ Basic observability with Prometheus metrics
- ✅ Configurable port and data directory via environment variables
- ✅ Dockerized setup for easy deployment
- ✅ Go client SDK for easy integration in Go projects

## 🚧 Upcoming Improvements

- 📦 client SDK for other languages
- 🌐 New operations
- 🚀 Lock Free Auto Compaction
- 🔐 Basic auth support using shared secret
- 🌐 Optional HTTP server interface
- 🔄 Auto-triggered compaction logic
- 🧪 End-to-end integration tests
- 📚 Better documentation and usage guides

---

## 🤝 Contribution

TinyStoreDB is a learning-focused project, but collaboration is encouraged!  
If you're interested in building features, reviewing design decisions, or exploring the internals of a key-value DB, feel free to connect and pair up.

## 🐳 Docker Usage

You can run the database with:

```bash
docker run -p 7389:7389 -e TINYSTOREDB_PORT=7389 -e TINYSTOREDB_DATA_DIR=/data tinystoredb/tinystoredb:latest
```

---

## 📦 Go Client SDK Usage

TinyStoreDB provides a simple Go client that you can import in your projects.

### Installation

```bash
go get github.com/raghavgh/TinyStoreDB/client
```

### Usage Example

```go
package main

import (
	"context"
	"log"

	"github.com/raghavgh/TinyStoreDB/client"
)

func main() {
	cli, err := client.New("localhost:7389")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	err = cli.Set(ctx, "key1", "value1")
	if err != nil {
		log.Fatal(err)
	}

	val, err := cli.Get(ctx, "key1")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Got value:", val)

	ok, err := cli.Delete(ctx, "key1")
	if err != nil {
		log.Fatal(err)
	}

	if !ok {
		log.Println("Delete failed")
	}
}
```

---

## Contact

For any queries or feedback, feel free to reach out via:
- **👤 Linkedin:** *https://www.linkedin.com/in/raghavpaliwal/*
- **🐙 GitHub Issues:** *[Github Issues page](https://github.com/raghavgh/TinyStoreDB/issues)*

---</file>
