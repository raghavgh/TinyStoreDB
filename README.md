# 🚀 TinyStoreDB

**TinyStoreDB** is a lightweight, persistent key-value storage system built as a learning project. It provides basic operations to store and retrieve string-based key-value pairs using an append-only storage model.  
🚧 **Note:** This project is under active development and approaching an MVP-ready state.


## 🐳 Docker Usage (Server)

Run the database server with Docker (mounting a local `./data` directory ensures your data persists on your host):

```bash
docker run -d -p 7389:7389 -v $(pwd)/data:/data --name tinystore -e TINYSTOREDB_PORT=7389 -e TINYSTOREDB_DATA_DIR=/data raghavgh/tinystoredb:latest
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
localhost:7389> INSPECT
====================================================================================================
  🔍 TinyStoreDB Storage Engine Inspector (Directory: data)
...
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

### 3. Inspect On-Disk Storage & Tombstones (`tinystoredb-cli inspect`)

Debugging raw binary `.bin` files directly can be challenging. TinyStoreDB provides a built-in diagnostic inspector (similar to `sst_dump` in RocksDB or `badger dump` in BadgerDB) to decode `wal.bin` and `value_log.bin` into a human-readable table:

```bash
tinystoredb-cli inspect --dir ./data
```

**Human-Readable Output:**
```text
====================================================================================================
  🔍 TinyStoreDB Storage Engine Inspector (Directory: ./data)
  WAL: wal.bin (180 B) | ValueLog: value_log.bin (240 B)
----------------------------------------------------------------------------------------------------
  📖 Storage Layout:
  • WAL Position:   Byte location in wal.bin (+record length)
  • Value Position: Byte location in value_log.bin (+payload length)
  • Record Type:    PUT (written record) or TOMBSTONE (soft-deleted entry)
  • Liveness:       live (current active value), stale (overwritten), or expired (TTL)
====================================================================================================
  #  WAL POSITION    VALUE POSITION  KEY     VALUE         TIMESTAMP            TTL   RECORD TYPE   LIVENESS
  -  ------------    --------------  ---     -----         ---------            ---   -----------   --------
  1  byte 0 (13 B)   byte 0 (15 B)   user:1  Alice         2026-10-21 18:00:01  -     ✍️  PUT        ♻️  stale
  2  byte 13 (16 B)  byte 15 (17 B)  user:2  Bob           2026-10-21 18:00:05  300s  ✍️  PUT        ♻️  stale
  3  byte 29 (18 B)  byte 32 (22 B)  user:1  Alice Cooper  2026-10-21 18:01:10  -     ✍️  PUT        ✅ live
  4  byte 47 (16 B)  - (no payload)  user:2  <tombstone>   2026-10-21 18:02:00  -     💀 TOMBSTONE  -
----------------------------------------------------------------------------------------------------
  Summary: 4 records (3 PUTs, 1 TOMBSTONEs) | 1 live | 2 stale | 0 expired
  ⚡ Compaction: 3 stale/dead records can be reclaimed via 'tinystoredb-cli compact'
====================================================================================================
```

#### Diagnostic & Optimization Flags:
* **`--limit <N>` / `-n`**: Limit output to the most recent N records (default: 50). Keeps terminal fast on multi-gigabyte databases.
* **`--all`**: Display all records without pagination.
* **`--hex`**: Switch from decimal byte layout to raw hexadecimal offsets (e.g. `0x00002A`).
* **`--watch` / `-w`**: Live interactive mode. Continuously tails storage files and prints writes and tombstones in real-time as they arrive.
* **`--tombstones`**: Filter to show only deleted records.
* **`--active-only`**: Show only current live records.
* **`--key <name>` / `-k`**: Trace revision history and overwrites for a specific key.
* **`--json`**: Output structured JSON for automation or test assertions.

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
- ✅ On-disk storage inspector (`tinystoredb-cli inspect`) with live watch & tombstone tracking
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
