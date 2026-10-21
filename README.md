# 🚀 TinyStoreDB

**TinyStoreDB** is a lightweight, persistent key-value storage system built as a learning project. It provides basic operations to store and retrieve string-based key-value pairs using an append-only storage model.  
🚧 **Note:** This project is under active development and approaching an MVP-ready state.


## 🐳 Docker Usage (Server)

Run the database server with Docker:

```bash
docker run -d -p 7389:7389 --name tinystore -e TINYSTOREDB_PORT=7389 -e TINYSTOREDB_DATA_DIR=/data raghavgh/tinystoredb:latest
```

---

## 💻 Terminal CLI Usage (`tinystoredb-cli`)

TinyStoreDB includes an official terminal client (`tinystoredb-cli`) with an interactive shell (REPL) and one-shot commands, similar to `redis-cli`.

### 1. Install the CLI

Install with the 1-line installer (Mac & Linux):
```bash
curl -fsSL https://raw.githubusercontent.com/raghavgh/TinyStoreDB/master/install.sh | sh
```

Or install with Go:
```bash
go install github.com/raghavgh/TinyStoreDB/cmd/tinystoredb-cli@latest
```

### 2. Connect to the Database

In a separate terminal, simply run:
```bash
tinystoredb-cli
```

You are immediately dropped into the interactive shell:
```text
Connected to TinyStoreDB at localhost:7389
Type 'help' for commands, 'exit' or 'quit' to quit.
localhost:7389> SET user:1 Alice
OK
localhost:7389> GET user:1
"Alice"
localhost:7389> EXIST user:1
(integer) 1
localhost:7389> DEL user:1
OK
localhost:7389> COMPACT
OK (compaction complete)
localhost:7389> quit
Bye!
```

Or execute one-shot commands directly from your terminal:
```bash
tinystoredb-cli set key1 value1
tinystoredb-cli get key1
tinystoredb-cli delete key1
tinystoredb-cli compact
```

Connect to a remote or custom host:
```bash
tinystoredb-cli --host 192.168.1.50 --port 7389 get key1
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
	// Create a new client instance, unsecured for simplicity
	// if you want secure communication,
	// use cli, err := client.New("localhost:7389", <token>)
	cli, err := client.New("localhost:7389", nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	err = cli.Set(ctx, "key1", "value1", nil)
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

## 📦 Features Implemented

- ✅ Set, Get, and Delete operations
- ✅ Append-only log-structured storage
- ✅ In-memory index for fast lookups
- ✅ Tombstone support for deletes
- ✅ Compaction to reclaim disk space
- ✅ Concurrency-safe operations (read/write)
- ✅ Interactive CLI shell (`tinystoredb-cli`) like `redis-cli`
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

## 🤝 Contribution

TinyStoreDB is a learning-focused project, but collaboration is encouraged!  
If you're interested in building features, reviewing design decisions, or exploring the internals of a key-value DB, feel free to connect and pair up.

---

## Contact

For any queries or feedback, feel free to reach out via:
- **👤 Linkedin:** *https://www.linkedin.com/in/raghavpaliwal/*
- **🐙 GitHub Issues:** *[Github Issues page](https://github.com/raghavgh/TinyStoreDB/issues)*

---
