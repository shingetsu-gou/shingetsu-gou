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
	"html"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shingetsu-gou/shingetsu-gou/util"
	"golang.org/x/text/language"
)

//Message hold string map.
type Message map[string]string

//newMessage reads from the file excpet #comment and stores them with url unescaping value.
func newMessage(filedir, fname string) Message {
	var err error
	m := make(map[string]string)
	var dat []byte
	file := path.Join("file", fname)
	if dat, err = util.Asset(file); err != nil {
		log.Println(err)
	}
	file = filepath.Join(filedir, fname)
	if util.IsFile(fname) {
		dat1, err := ioutil.ReadFile(file)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("loaded", file)
			dat = dat1
		}
	}
	if dat == nil {
		log.Fatal("message file was not found")
	}

	re := regexp.MustCompile(`^\s*#`)
	for i, line := range strings.Split(string(dat), "\n") {
		line = strings.Trim(line, "\r\n")
		if line == "" || re.MatchString(line) {
			continue
		}
		buf := strings.Split(line, "<>")
		if len(buf) != 2 {
			log.Fatalf("illegal format at line %d in the message file", i)
		}
		buf[1] = html.UnescapeString(buf[1])
		m[buf[0]] = buf[1]
	}
	return m
}

//SearchMessage parse Accept-Language header ,selects most-weighted(biggest q)
//language ,reads the associated message file, and creates and returns message obj.
func SearchMessage(acceptLanguage, filedir string) Message {
	const defaultLanguage = "en" // Language code (see RFC3066)

	var lang []string
	if acceptLanguage != "" {
		tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
		if err != nil {
			log.Println(err)
		} else {
			for _, tag := range tags {
				lang = append(lang, tag.String())
			}
		}
	}
	lang = append(lang, defaultLanguage)
	for _, l := range lang {
		slang := strings.Split(l, "-")[0]
		for _, j := range []string{l, slang} {
			if m := newMessage(filedir, "message-"+j+".txt"); m != nil {
				return m
			}
		}
	}
	log.Fatalf("no messages are found.")
	return nil
}
