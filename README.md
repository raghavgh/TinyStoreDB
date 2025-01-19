# TinyStoreDB

**TinyStoreDB** is a lightweight, persistent key-value storage system built as a learning project. It provides basic operations to store and retrieve string-based key-value pairs using an append-only storage model.  
**Note:** This project is still in progress and not ready for production use.

---

## Features

- **Persistent Storage:**  
  Data is stored in a `.bin` file format for durability.
- **Simple Key-Value API:**  
  Supports storing and retrieving string-based key-value pairs.
- **Append-Only Design:**  
  Efficient writes with a sequential append approach.
- **GOB Encoding:**  
  Utilizes Go's GOB encoding for flexible data serialization.

---

## Data Storage Format

Each key-value entry is stored in the following format:

1. **Length of the data** (4 bytes) – Specifies the total size of the stored data.
2. **Encoded data type** (variable size) – Determined by Go's GOB encoding.
3. **Actual data** (variable size) – The encoded key or value.

Both keys and values are stored using this format, ensuring consistency in data retrieval.

---

## Installation

Ensure you have Go 1.23 or later installed on your system.

### Clone the Repository

```bash
git clone git@github.com:raghavgh/TinyStoreDB.git
cd TinyStoreDB
```

### Install Dependencies

```bash
go mod tidy
```

---

## Usage

To get started with TinyStoreDB, follow the example below:

```go
package main

import (
    "fmt"
    "client"
)

func main() {
    db := client.NewTinyStoreDBClient()
    
    // Store key-value pair
    db.Set("exampleKey", "exampleValue")

    // Retrieve value
    value := db.Get("exampleKey")
    fmt.Println("Retrieved Value:", value)
}
```

---

## Known Limitations

- **Limited Data Type Support:**  
  Currently supports only string keys and values.

- **No Update or Delete Operations:**  
  The current version does not support modifying or removing key-value pairs.

- **Sequential Reads:**  
  `Get` operations iterate over the entire file, which may impact performance as data grows.

---

## Roadmap

Future releases will focus on enhancing the database with the following features:

- **Update and Delete Operations:**  
  Enable modification and removal of existing key-value pairs.
- **Optimized Read Performance:**  
  Implementing indexing mechanisms for faster lookups.
- **Concurrency Handling:**  
  Ensuring safe operations across multiple clients.

---

## Contribution

TinyStoreDB is an **in-progress learning project**, and while formal contributions are not currently accepted,  
**collaboration and discussions are welcome!**

If you're interested in learning together, sharing ideas, or continuing this project collaboratively,  
feel free to reach out to discuss and explore potential improvements.

---

## Contact

For any queries or feedback, feel free to reach out via:
- **Linkedin:** *https://www.linkedin.com/in/raghavpaliwal/*
- **GitHub Issues:** *[Github Issues page](https://github.com/raghavgh/TinyStoreDB/issues)*

---