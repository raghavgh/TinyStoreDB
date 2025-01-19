package main

import (
	"fmt"
	"math/rand"
	_ "net/http/pprof"
	"time"

	"TinyStoreDB/client"
)

func main() {
	db := client.NewTinyStoreDBClient()

	loadBulkDataForLoadTesting(db)
	getMultipleData(db)
}

// loadBulkDataForLoadTesting loads bulk data for load testing
func loadBulkDataForLoadTesting(db *client.TinyStoreDBClient) {
	n := 10000
	tempStore := map[string]string{}
	for i := range n {
		tempStore[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	now := time.Now()
	for k, v := range tempStore {
		if err := db.Set(k, v); err != nil {
			panic(err)
		}
	}

	fmt.Printf("time : %.2f\n", time.Since(now).Seconds())

}

// Call multiple get and log the performance
func getMultipleData(db *client.TinyStoreDBClient) {
	for i := 0; i < 5; i++ {
		now := time.Now()
		key := fmt.Sprintf("key%d", i+rand.Intn(766))
		var (
			val *string
			err error
		)
		if val, err = db.Get(key); err != nil {
			panic(err)
		}
		fmt.Printf("value : %s, time : %d\n", *val, time.Since(now).Milliseconds())
	}
}
