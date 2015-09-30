package main

import (
	"html/template"
	"log"
	"os"
)

func main() {
	_, err := template.ParseFiles(os.Args[1])
	log.Println(err)
}
