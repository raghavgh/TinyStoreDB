package main

import (
	"fmt"
	"time"

	"TinyStoreDB/client"
)

func main() {
	db := client.NewTinyStoreDBClient()

	getMultipleData(db)
}

// loadBulkDataForLoadTesting loads bulk data for load testing
func loadBulkDataForLoadTesting(db *client.TinyStoreDBClient) {
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)

		if err := db.Set(key, value); err != nil {
			panic(err)
		}
	}

}

// Call multiple get and log the performance
func getMultipleData(db *client.TinyStoreDBClient) {
	start := time.Now()
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("key%d", i)

		if _, err := db.Get(key); err != nil {
			panic(err)
		}
		//fmt.Printf("Time taken to get record %d: %s\n", i, elapsedForEach)
	}
	elapsed := time.Since(start)
	fmt.Printf("Time taken to get 5000 records: %s\n", elapsed)
}
