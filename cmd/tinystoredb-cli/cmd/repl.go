package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/raghavgh/TinyStoreDB/client"
)

func runREPL() {
	cfg, _ := loadConfig()
	if hostFlag != "" {
		cfg.Host = hostFlag
	}
	if portFlag != 0 {
		cfg.Port = portFlag
	}
	if secretFlag != "" {
		cfg.Secret = secretFlag
	}

	var secretPtr *string
	if cfg.Secret != "" {
		secretPtr = &cfg.Secret
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	cli, err := client.New(addr, secretPtr)
	if err != nil {
		fmt.Printf("❌ Failed to connect to %s: %v\n", addr, err)
		return
	}
	defer cli.Close()

	fmt.Printf("Connected to TinyStoreDB at %s\n", addr)
	fmt.Println("Type 'help' for commands, 'exit' or 'quit' to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s> ", addr)
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		switch cmd {
		case "exit", "quit":
			cancel()
			fmt.Println("Bye!")
			return

		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  SET <key> <val> [ttl_seconds]  - Store a key-value pair")
			fmt.Println("  GET <key>                      - Retrieve value for a key")
			fmt.Println("  DEL <key>                      - Delete a key")
			fmt.Println("  EXIST <key>                    - Check if a key exists")
			fmt.Println("  COMPACT                        - Trigger storage compaction")
			fmt.Println("  EXIT / QUIT                    - Close the shell")

		case "get":
			if len(parts) < 2 {
				fmt.Println("Usage: GET <key>")
				cancel()
				continue
			}
			val, err := cli.Get(ctx, parts[1])
			if err != nil {
				fmt.Printf("(error) %v\n", err)
			} else {
				fmt.Printf("%q\n", val)
			}

		case "set":
			if len(parts) < 3 {
				fmt.Println("Usage: SET <key> <value> [ttl_seconds]")
				cancel()
				continue
			}
			var ttl *uint64
			if len(parts) >= 4 {
				if sec, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
					expiry := uint64(time.Now().Unix()) + sec
					ttl = &expiry
				}
			}
			err := cli.Set(ctx, parts[1], parts[2], ttl)
			if err != nil {
				fmt.Printf("(error) %v\n", err)
			} else {
				fmt.Println("OK")
			}

		case "del", "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: DEL <key>")
				cancel()
				continue
			}
			ok, err := cli.Delete(ctx, parts[1])
			if err != nil {
				fmt.Printf("(error) %v\n", err)
			} else if ok {
				fmt.Println("OK")
			} else {
				fmt.Println("(key not found)")
			}

		case "exist", "exists":
			if len(parts) < 2 {
				fmt.Println("Usage: EXIST <key>")
				cancel()
				continue
			}
			ok, err := cli.Exist(ctx, parts[1])
			if err != nil {
				fmt.Printf("(error) %v\n", err)
			} else if ok {
				fmt.Println("(integer) 1")
			} else {
				fmt.Println("(integer) 0")
			}

		case "compact":
			err := cli.Compact(ctx)
			if err != nil {
				fmt.Printf("(error) %v\n", err)
			} else {
				fmt.Println("OK (compaction complete)")
			}

		default:
			fmt.Printf("Unknown command %q. Type 'help' for usage.\n", cmd)
		}
		cancel()
	}
}
