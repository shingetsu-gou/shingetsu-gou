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

	"golang.org/x/text/language"
)

//message hold string map.
type message map[string]string

//newMessage reads from the file excpet #comment and stores them with url unescaping value.
func newMessage(file string) message {
	var m message
	re := regexp.MustCompile("^\\s*#")
	err := eachLine(file, func(line string, i int) error {
		line = strings.Trim(line, "\r\n")
		var err error
		if re.MatchString(line) {
			return nil
		}
		buf := strings.Split(line, "<>")
		if len(buf) == 2 {
			buf[1], err = url.QueryUnescape(buf[1])
			m[buf[0]] = buf[1]
		}
		return err
	})
	if err != nil {
		log.Println(file, err)
	}
	return m
}

//searchMessage parse Accept-Language header ,selects most-weighted(biggest q)
//language ,reads the associated message file, and creates and returns message obj.
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

//DefaultVariable is default variables for html templates.
type DefaultVariable struct {
	CGI         *cgi
	Environment http.Header
	UA          string
	Message     message
	Lang        string
	Aappl       map[string]string
	GatewayCGI  string
	ThreadCGI   string
	AdminCGI    string
	RootPath    string
	Types       []string
	Isadmin     bool
	Isfriend    bool
	Isvisitor   bool
	Dummyquery  int64
	filter      string
	tag         string
}

//MakeGatewayLink makes "{{cginame}}/{{command}}"  link with tile=description.
func (d DefaultVariable) MakeGatewayLink(cginame, command string) string {
	g := struct {
		CGIname     string
		Command     string
		Description string
	}{
		cginame,
		command,
		d.Message["desc_"+command],
	}
	var doc bytes.Buffer
	renderTemplate("gateway_link", g, &doc)
	return doc.String()
}

//checkCache checks cache ca has specified tag and datfile doesn't contains filterd string.
func (d *DefaultVariable) checkCache(ca *cache, target string) (string, bool) {
	x := fileDecode(ca.datfile)
	if x == "" {
		return "", false
	}
	if d.filter != "" && !strings.Contains(d.filter, strings.ToLower(x)) {
		return "", false
	}
	if d.tag != "" {
		switch {
		case ca.tags.hasTagstr(strings.ToLower(d.tag)):
		case target == "recent" && ca.sugtags.hasTagstr(strings.ToLower(d.tag)):
		default:
			return "", false
		}
	}
	return x, true
}

func (d DefaultVariable) MakeListItem(ca *cache, remove bool, target string, search bool) string {
	if target == "" {
		target = "changes"
	}
	x, ok := d.checkCache(ca, target)
	if !ok {
		return ""
	}
	y := strEncode(x)
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
		Cache    *cache
		Title    string
		StrTitle string
		Tags     *tagList
		Sugtags  []*tag
		Target   string
		Remove   bool
		StrOpts  string
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
