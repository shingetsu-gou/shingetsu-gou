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

package util

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//ConfList represents regexp list.
//    One regexp per one line.
type ConfList struct {
	mtime *time.Time
	path  string
	data  []string
	mutex sync.RWMutex
}

//NewConfList makes a confList instance from path.
func NewConfList(path string, defaultList []string) *ConfList {
	r := &ConfList{
		path: path,
	}
	r.update()
	if len(r.data) == 0 {
		r.data = defaultList
	}
	return r
}

//GetData retuns a copy of lines in the file.
func (r *ConfList) GetData() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	d := make([]string, len(r.data))
	copy(d, r.data)
	return d
}

//update read the file if newer, and stores all lines in the file.
func (r *ConfList) update() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.path == "" {
		return false
	}
	s, err := os.Stat(r.path)
	if err != nil {
		r.data = nil
		return false
	}
	mtime := s.ModTime()
	if r.mtime != nil && !mtime.After(*r.mtime) {
		return false
	}
	r.mtime = &mtime
	r.data = r.data[:0]

	err = EachLine(r.path, func(line string, i int) error {
		if line != "" && !strings.HasPrefix(line, "#") {
			r.data = append(r.data, line)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return true
}

//RegexpList represents RegExp list.
//    One regexp per one line.
type RegexpList struct {
	*ConfList
	regs []*regexp.Regexp
}

//NewRegexpList makes a regexpList and regexp.comples each lines in the file.
func NewRegexpList(path string) *RegexpList {
	c := &ConfList{
		path: path,
	}
	r := &RegexpList{
		ConfList: c,
	}
	r.update()
	return r
}

//Check returns true if target matches one of all regexps or not.
func (r *RegexpList) Check(target string) bool {
	r.update()
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, re := range r.regs {
		if re.MatchString(target) {
			return true
		}
	}
	return false
}

//update read the file and regexp.comples each lines in the file if file is newer.
func (r *RegexpList) update() {
	if !r.ConfList.update() {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.regs = r.regs[:0]
	for i, line := range r.ConfList.data {
		reg := regexp.MustCompile(`\\u([0-9a-fA-F]+)`)
		line = reg.ReplaceAllStringFunc(line, func(l string) string {
			m := reg.FindStringSubmatch(l)
			code, err := strconv.ParseInt(m[1], 16, 64)
			if err != nil {
				fmt.Println(err)
				return ""
			}
			return fmt.Sprintf("%c", code)
		})
		re, err := regexp.Compile(line)
		if err != nil {
			log.Println("cannot compile regexp", line, "line", i)
		} else {
			r.regs = append(r.regs, re)
		}
	}
}
