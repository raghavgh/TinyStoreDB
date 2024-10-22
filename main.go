package main

import (
	"fmt"
	"time"

	"simple-db-go/client"
)

//TIP To run your code, right-click the code and select <b>Run</b>. Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.

func main() {
	db := client.NewSimpleDBClient()

	getMultipleData(db)
}

// loadBulkDataForLoadTesting loads bulk data for load testing
func loadBulkDataForLoadTesting(db *client.SimpleDBClient) {
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)

		if err := db.Set(key, value); err != nil {
			panic(err)
		}
	}

}

// Call multiple get and log the performance
func getMultipleData(db *client.SimpleDBClient) {
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

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
