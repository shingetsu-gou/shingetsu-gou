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

package cgi

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/shingetsu-gou/shingetsu-gou/util"
)

//finfo contains file info,including mtime and content.
type finfo struct {
	mtime *time.Time
	cont  []byte
	exist bool
}

//jsCache contains js inf, i.e. path info and  finfo of each js files..
type jsCache struct {
	path  string
	files map[string]*finfo
}

//newJsCache return jsCache instnace and parse all js files under path dir.
func newJsCache(path string) *jsCache {
	j := &jsCache{path: path, files: make(map[string]*finfo)}
	j.update()
	return j
}

//GetLatest gets latest mtime of all jsCache.files.
func (j *jsCache) GetLatest() int64 {
	var l *time.Time
	for _, v := range j.files {
		if l == nil || v.mtime.After(*l) {
			l = v.mtime
		}
	}
	return l.Unix()
}

//getContent concat contents of all js files sorted by keys(filenames).
func (j *jsCache) getContent() string {
	j.update()
	var cont string
	keys := make([]string, len(j.files))
	i := 0
	for k := range j.files {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		cont += string(j.files[k].cont)
	}
	return cont
}

//update reloads all js files if mtime is newer.
func (j *jsCache) update() {
	for k := range j.files {
		j.files[k].exist = false
	}
	err := util.EachFiles(j.path, func(f os.FileInfo) error {
		var err error
		name := f.Name()
		if !strings.HasSuffix(name, ".js") || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			return nil
		}
		oldfi, exist := j.files[name]
		if !exist || f.ModTime().After(*oldfi.mtime) {
			m := f.ModTime()
			fi := finfo{mtime: &m, exist: true}
			fi.cont, err = ioutil.ReadFile(path.Join(j.path, name))
			j.files[name] = &fi
		} else {
			oldfi.exist = true
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	for k := range j.files {
		if !j.files[k].exist {
			delete(j.files, k)
		}
	}
}
