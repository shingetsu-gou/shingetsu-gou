package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

func main() {
	db, err := bolt.Open("run/gou_bolt.db", 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(os.Args[1]))
		if b == nil {
			return errors.New("no bucket")
		}
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			return nil
		})
		fmt.Println("")
		b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%b, value=%b\n", k, v)
			return nil
		})
		return nil
	})
	log.Println(err)
}

// output:
// Hello World!
