package main

import (
	"html/template"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	t := template.New("")
	funcMap := template.FuncMap{
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"mul":         func(a, b int) int { return a * b },
		"div":         func(a, b int) int { return a / b },
		"toMB":        func(a int) float64 { return float64(a) / (1024 * 1024) },
		"toKB":        func(a int) float64 { return float64(a) / (1024) },
		"strEncode":   func(s string) string { return s },
		"escape":      func(s string) string { return s },
		"escapeSpace": func(s string) string { return s },
		"localtime":   func(stamp int64) string { return time.Unix(stamp, 0).Format("2006-01-02 15:04") },
		"fileDecode": func(query, t string) string {
			q := strings.Split(query, "_")
			if len(q) < 2 {
				return t
			}
			return q[0]
		},
	}
	_, err := t.Funcs(funcMap).ParseFiles(os.Args[1])
	log.Println(err)
}
