/*
 * Copyright (c) 2015, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package gou

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func md5digest(dat string) string {
	sum := md5.Sum([]byte(dat))
	return hex.EncodeToString(sum[:])
}

//from spam.py

func spamCheck(recstr string) bool {
	if cached_rule == nil {
		cached_rule = newRegexpList(spam_list)
	} else {
		cached_rule.update()
	}
	return cached_rule.check(recstr)
}

//fsdiff checks a difference between file and string.
//Return same data or not.
func fsdiff(f, s string) bool {
	cont, err := ioutil.ReadFile(f)
	if err != nil {
		log.Println(err)
		return false
	}
	if string(cont) == s {
		return true
	}
	return false
}

//from attachutil.py

//Type of path is same as mimetype or not.
func isValidImage(mimetype, path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	realType := mime.TypeByExtension(ext)
	if realType == mimetype {
		return true
	}
	if realType == "image/jpeg" && mimetype == "image/pjpeg" {
		return true
	}
	return false
}

func eachIOLine(f io.ReadCloser, handler func(line string, num int) error) error {
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan(); i++ {
		err := handler(scanner.Text(), i)
		if err != nil {
			return err
		}
	}
	return scanner.Err()
}

func eachLine(path string, handler func(line string, num int) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return eachIOLine(f, handler)
}

func eachKeyValueLine(path string, handler func(key string, value []string, num int) error) error {
	err := eachLine(path, func(line string, i int) error {
		kv := strings.Split(line, "<>")
		if len(kv) != 2 {
			log.Fatal("illegal line in", lookup)
		}
		vs := strings.Split(kv[1], " ")
		err := handler(kv[0], vs, i)
		return err
	})
	return err
}

type stringerSlice interface {
	Len() int
	Get(int) string
}

type stringSlice []string

func (s stringSlice) Len() int {
	return len(s)
}

func (s stringSlice) Get(i int) string {
	return s[i]
}

func hasString(ary stringerSlice, val string) bool {
	return findString(ary, val) != -1
}

func findString(ary stringerSlice, val string) int {
	for i := 0; i < ary.Len(); i++ {
		if ary.Get(i) == val {
			return i
		}
	}
	return -1
}

func writeSlice(path string, ary stringerSlice) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 0; i < ary.Len(); i++ {
		f.WriteString(ary.Get(i) + "\n")
	}
	return nil
}

func writeMap(path string, ary map[string][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range ary {
		f.WriteString(k + "<>")
		for i, s := range v {
			f.WriteString(s)
			if i != len(v)-1 {
				f.WriteString(" ")
			}
		}
		f.WriteString("\n")
	}
	return nil
}

func executeTemplate(file string, st interface{}, wr io.Writer) {
	basename := template_dir + "/" + file + "/" + template_suffix
	tpl, err := template.ParseFiles(basename)
	if err != nil {
		log.Println(err)
		return
	}
	funcMap := template.FuncMap{
		"add":  func(a, b int) int { return a + b },
		"mul":  func(a, b int) int { return a * b },
		"toKB": func(a int) float64 { return float64(a) / 1024 },
		"toMB": func(a int) float64 { return float64(a) / (1024 * 1024) },
	}
	tpl.Funcs(funcMap)
	if err := tpl.Execute(wr, st); err != nil {
		fmt.Println(err)
	}
}

func eachFiles(dir string, handler func(dir os.FileInfo) error) error {
	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, i := range dirs {
		if err := handler(i); err != nil {
			return err
		}
	}
	return nil
}

func isFile(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fs.IsDir()
}

func isDir(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fs.IsDir()
}

func sortKeys(m map[string]string) []string {
	mk := make([]string, len(m))
	i := 0
	for k, _ := range m {
		mk[i] = k
		i++
	}
	sort.Strings(mk)
	return mk
}

func moveFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return os.Remove(src)
}

type shufflable interface{
	Len() int
	Swap(i int,j int)
}

func shuffle(slc shufflable) {
	N := slc.Len()
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rand.Intn(N-i)
		slc.Swap(r, i)
	}
}
