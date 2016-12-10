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
	defer func() {
		errr := db.Close()
		if errr != nil {
			log.Println(err)
		}
	}()

	err = db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(os.Args[1]))
		if b == nil {
			return errors.New("no bucket")
		}
		errr := b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%s, value=%s\n", k, v)
			fmt.Printf("key=%02x, value=%x\n", k, v)
			return nil
		})
		if errr != nil {
			log.Println(errr)
		}
		fmt.Println("")
		errr = b.ForEach(func(k, v []byte) error {
			fmt.Printf("key=%b, value=%b\n", k, v)
			return nil
		})
		if errr != nil {
			log.Println(errr)
		}
		return nil
	})
	log.Println(err)
}

// output:
// Hello World!
