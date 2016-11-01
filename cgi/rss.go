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
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

//Item represents RSS contents.
type Item struct {
	Title       string
	Link        string
	Description string
	Creator     string
	Date        int64
	content     string
	Content     string
	Subject     []string
}

//RSS represents RSS info.
type RSS struct {
	Encode      string
	Lang        string
	Title       string
	Link        string
	Description string
	Feeds       []*Item
	parent      string
	URI         string
	XSL         string
}

//Swap swaps feed[i] and feed[j]
func (r *RSS) Swap(i, j int) {
	r.Feeds[j], r.Feeds[i] = r.Feeds[i], r.Feeds[j]
}

//Less returns true if date of feed[i]<one of [j]
func (r *RSS) Less(i, j int) bool {
	return r.Feeds[i].Date < r.Feeds[j].Date
}

//Len returns # of feeds.
func (r *RSS) Len() int {
	return len(r.Feeds)
}

//NewRSS makes RSS object.
func NewRSS(encode, lang, title, parent, link, uri, description, xsl string) *RSS {
	if encode == "" {
		encode = "utf-8"
	}
	if lang == "" {
		lang = "en"
	}
	r := &RSS{
		Encode:      encode,
		Lang:        lang,
		Title:       title,
		Description: description,
		XSL:         xsl,
		Link:        link,
		URI:         uri,
		parent:      parent,
	}
	if parent != "" && parent[len(parent)-1] != '/' {
		r.parent += "/"
	}
	if link == "" {
		r.Link = parent
	}
	if uri == "" {
		r.URI = parent + "rss.xml"
	}

	return r
}

//Append adds RSS an item.
func (r *RSS) Append(link, title, creator, description, content string, subject []string, date int64, abs bool) {
	if !abs {
		link = r.parent + link
	}
	i := &Item{
		Title:       strings.TrimSpace(title),
		Link:        link,
		Description: strings.TrimSpace(description),
		Creator:     creator,
		Date:        date,
		Subject:     subject,
		content:     content,
	}
	r.Feeds = append(r.Feeds, i)
}

//MakeRSS1 renders template.
func (r *RSS) MakeRSS1(wr io.Writer) {
	for _, c := range r.Feeds {
		c.Content = strings.Replace(c.content, "]]", "&#93;&#93;>", -1)
	}
	sort.Sort(sort.Reverse(r))
	RenderRSS(*r, wr)
}

//W3cdate returns RSS formated date string.
//used in templates
func (r RSS) W3cdate(dat int64) string {
	t := time.Unix(dat, 0)
	return t.Format("2006-01-02T15:04:05Z")
}

//RSSTextFormat formats plain string to stirng usable in html.
func RSSTextFormat(plain string) string {
	buf := strings.Replace(plain, "<br>", " ", -1)
	buf = strings.Replace(buf, "&", "&amp;", -1)
	reg := regexp.MustCompile(`&amp;(#\d+|lt|gt|amp);`)
	buf = reg.ReplaceAllString(buf, "&$1;")
	buf = strings.Replace(buf, "<", "&lt;", -1)
	buf = strings.Replace(buf, ">", "&gt;", -1)
	buf = strings.Replace(buf, "\r", "", -1)
	buf = strings.Replace(buf, "\n", "", -1)
	return buf
}
