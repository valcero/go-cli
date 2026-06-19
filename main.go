package main

import (
	"fmt"
	"log"
)

func main() {
	db, err := ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to the database!")
}
