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
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

//confList represents regexp list.
//    One regexp per one line.
type confList struct {
	mtime *time.Time
	path  string
	data  []string
}

//newConfList makes a confList instance from path.
func newConfList(path string, defaultList []string) *confList {
	r := &confList{path: path}
	r.update()
	if len(r.data) == 0 {
		r.data = defaultList
	}
	return r
}

//update read the file if newer, and stores all lines in the file.
func (r *confList) update() {
	if r.path == "" {
		return
	}
	s, err := os.Stat(r.path)
	if err != nil {
		r.data = nil
		return
	}
	mtime := s.ModTime()
	if r.mtime != nil && !mtime.After(*r.mtime) {
		return
	}
	r.mtime = &mtime
	r.data = r.data[:0]
	err = eachLine(r.path, func(line string, i int) error {
		if line != "" && !strings.HasPrefix(line, "#") {
			r.data = append(r.data, line)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

//regexpList represents RegExp list.
//    One regexp per one line.
type regexpList struct {
	*confList
	regs []*regexp.Regexp
}

//newRegExpList make a regexpList and regexp.comples each lines in the file.
func newRegexpList(path string) *regexpList {
	c := newConfList(path, []string{})
	r := &regexpList{}
	r.regs = make([]*regexp.Regexp, 0)
	r.confList = c
	r.update()
	return r
}

//check checks whethere target matches one of all regexps or not.
func (r *regexpList) check(target string) bool {
	r.update()
	for _, r := range r.regs {
		if r.MatchString(target) {
			return true
		}
	}
	return false
}

//update read the file and regexp.comples each lines in the file if file is newer.
func (r *regexpList) update() {
	r.confList.update()
	r.regs = r.regs[:0]
	for i, line := range r.confList.data {
		re, err := regexp.Compile(line)
		if err != nil {
			log.Println("cannot compile regexp", line, "line", i)
		} else {
			r.regs = append(r.regs, re)
		}
	}
}
