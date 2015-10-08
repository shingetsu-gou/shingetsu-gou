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
	"bytes"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/language"
)

/*
//Connection Counter.
type counter struct {
	N     int
	mutex sync.Mutex
}

func (c *counter) increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.N++
}

func (c *counter) decrement() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.N--
}
*/

type message map[string]string

func newMessage(file string) message {
	var m message
	re := regexp.MustCompile("^#$")
	err := eachLine(file, func(line string, i int) error {
		line = strings.Trim(line, "\r\n")
		var err error
		if !re.MatchString(line) {
			buf := strings.Split(line, "<>")
			if len(buf) == 2 {
				buf[1], err = url.QueryUnescape(buf[1])
				m[buf[0]] = buf[1]
			}
		}
		return err
	})
	if err != nil {
		log.Println(file, err)
	}
	return m
}

func (m message) get(k string) string {
	if v, exist := m[k]; exist {
		return v
	}
	return ""
}

func searchMessage(acceptLanguage string) message {
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
			file := path.Join(fileDir, "message-"+j+".txt")
			if isFile(file) {
				return newMessage(file)
			}
		}
	}
	return nil
}

type DefaultVariable struct {
	CGI         *cgi
	Environment http.Header
	UA          string
	Message     message
	Lang        string
	Aappl       map[string]string
	GatewayCgi  string
	ThreadCgi   string
	AdminCgi    string
	RootPath    string
	Types       []string
	Isadmin     bool
	Isfriend    bool
	Isvisitor   bool
	Dummyquery  int64
	filter      string
	tag         string
}

func (d DefaultVariable) Add(a, b int) int {
	return a + b
}
func (d DefaultVariable) Mul(a, b int) int {
	return a * b
}
func (d DefaultVariable) ToKB(a int) float64 {
	return float64(a) / 1024
}
func (d DefaultVariable) ToMB(a int) float64 {
	return float64(a) / (1024 * 1024)
}
func (d DefaultVariable) Localtime(stamp int64) string {
	return time.Unix(stamp, 0).Format("2006-01-02 15:04")
}
func (d DefaultVariable) StrEncode(query string) string {
	return strEncode(query)
}

func (d DefaultVariable) Escape(msg string) string {
	return escape(msg)
}

func (d DefaultVariable) EscapeSpace(msg string) string {
	return escapeSpace(msg)
}

func (d DefaultVariable) FileDecode(query, t string) string {
	q := strings.Split(query, "_")
	if len(q) < 2 {
		return t
	}
	return q[0]
}

func (d DefaultVariable) MakeGatewayLink(cginame, command string) string {
	g := struct {
		CGIname     string
		Command     string
		Description string
	}{
		cginame,
		command,
		d.Message.get("desc_" + command),
	}
	var doc bytes.Buffer
	renderTemplate("gateway_link", g, &doc)
	return doc.String()
}

//toolong
func (d DefaultVariable) MakeListItem(ca *cache, remove bool, target string, search bool) string {
	x := fileDecode(ca.datfile)
	if x == "" {
		return ""
	}
	y := strEncode(x)
	if d.filter != "" && !strings.Contains(d.filter, strings.ToLower(x)) {
		return ""
	}
	if d.tag != "" {
		var cacheTags []*tag
		matchtag := false
		cacheTags = append(cacheTags, ca.tags.tags...)
		if target == "recent" {
			cacheTags = append(cacheTags, ca.sugtags.tags...)
		}
		for _, t := range cacheTags {
			if strings.ToLower(t.tagstr) == d.tag {
				matchtag = true
				break
			}
		}
		if !matchtag {
			return ""
		}
	}
	x = escapeSpace(x)
	var strOpts string
	if search {
		strOpts = "?search_new_file=yes"
	}
	var sugtags []*tag
	if target == "recent" {
		strTags := make([]string, ca.tags.Len())
		for i, v := range ca.tags.tags {
			strTags[i] = strings.ToLower(v.tagstr)
		}
		for _, st := range ca.sugtags.tags {
			if !hasString(strTags, strings.ToLower(st.tagstr)) {
				sugtags = append(sugtags, st)
			}
		}
	}
	var doc bytes.Buffer
	g := struct {
		*DefaultVariable
		cache    *cache
		title    string
		strTitle string
		tags     *tagList
		sugtags  []*tag
		target   string
		remove   bool
		strOpts  string
	}{
		&d,
		ca,
		x,
		y,
		ca.tags,
		sugtags,
		target,
		remove,
		strOpts,
	}
	renderTemplate("list_item", g, &doc)
	return doc.String()
}
