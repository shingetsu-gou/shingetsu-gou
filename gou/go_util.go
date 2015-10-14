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
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

//eachIOLine iterates each line to  a ReadCloser and calls func.
func eachIOLine(f io.ReadCloser, handler func(line string, num int) error) error {
	defer close(f)
	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan(); i++ {
		err := handler(scanner.Text(), i)
		if err != nil {
			return err
		}
	}
	return scanner.Err()
}

//eachLine iterates each line and calls a func.
func eachLine(path string, handler func(line string, num int) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return eachIOLine(f, handler)
}

//eachKeyValueLine calls func for each line which contains key and value separated with "<>"
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

//hasString returns true if ary has val.
func hasString(s []string, val string) bool {
	return findString(s, val) != -1
}

//findString search val in ary and returns index. it returns -1 if not found.
func findString(s []string, val string) int {
	for i, v := range s {
		if v == val {
			return i
		}
	}
	return -1
}

//writeSlice write ary into a path.
func writeSlice(path string, ary []string) error {
	f, err := os.Create(path)
	defer close(f)
	if err != nil {
		log.Println(err)
		return err
	}

	for _, v := range ary {
		_, err := f.WriteString(v + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

//writeSlice write map into a path.
func writeMap(path string, ary map[string][]string) error {
	f, err := os.Create(path)
	if err != nil {
		log.Println(err)
		return err
	}
	defer close(f)

	for k, v := range ary {
		_, err := f.WriteString(k + "<>" + strings.Join(v, " ") + "\n")
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

//renderTemplate executes template and write to wr.
func renderTemplate(file string, st interface{}, wr io.Writer) {
	if err := templates.ExecuteTemplate(wr, file, st); err != nil {
		fmt.Println(err)
	}
}

//executeTemplate executes template and returns it as string.
func executeTemplate(file string, st interface{}) string {
	var doc bytes.Buffer
	renderTemplate(file, st, &doc)
	return doc.String()
}

//eachFiles iterates each files in dir and calls handler.
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

//IsFile returns true is path is an existing file.
func IsFile(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fs.IsDir()
}

//IsDir returns true is path is an existing dir.
func IsDir(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fs.IsDir()
}

//moveFile moves a file from src to dest.
func moveFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer close(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer close(out)

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return os.Remove(src)
}

//shufflable interface is for shuffle ary.
type shufflable interface {
	Len() int
	Swap(i int, j int)
}

//shuffle shuffles shufflable ary.
func shuffle(slc shufflable) {
	N := slc.Len()
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rand.Intn(N-i)
		slc.Swap(r, i)
	}
}

//close closes io.Close, if err exists ,println err.
func close(f io.Closer) {
	if err := f.Close(); err != nil {
		log.Println(err)
	}
}

//compressHandler returns handlers.CompressHandler to simplfy.
func registToRouter(s *mux.Router, path string, fn func(w http.ResponseWriter, r *http.Request)) {
	s.Handle(path, handlers.CompressHandler(http.HandlerFunc(fn)))
}

//fileSize returns file size of file.
//returns 0 if file is not found.
func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return 0
	}
	return st.Size()
}

//writeFile rite date to path.
func writeFile(path, data string) error {
	err := ioutil.WriteFile(path, []byte(data), 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

//makeThumbnail makes thumbnail to suffix image format with thumbnailSize.
func makeThumbnail(from, to, suffix string, x, y uint) {
	file, err := os.Open(from)
	if err != nil {
		log.Println(err)
		return
	}
	defer close(file)

	img, _, err := image.Decode(file)
	if err != nil {
		log.Println(err)
		return
	}
	m := resize.Resize(x, y, img, resize.Lanczos3)
	out, err := os.Create(to)
	if err != nil {
		log.Println(err)
		return
	}
	defer close(out)
	switch suffix {
	case "jpg", "jpeg":
		err = jpeg.Encode(out, m, nil)
	case "png":
		err = png.Encode(out, m)
	case "gif":
		err = gif.Encode(out, m, nil)
	default:
		log.Println("illegal format", suffix)
	}
	if err != nil {
		log.Println(err)
	}
}

//touch makes an empty file "name".
func touch(fname string) {
	f, err := os.Create(fname)
	defer close(f)
	if err != nil {
		log.Println(err)
	}
}
