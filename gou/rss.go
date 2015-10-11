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
	"io"
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
	Subject     []string
	Date        int64
	content     string
	Content     string
}

//RSS represents RSS info.
type RSS struct {
	Encode      string
	Lang        string
	Title       string
	Link        string
	URI         string
	Description string
	XSL         string
	Feeds       map[string]*Item
	parent      string
}

//newRSS makes RSS object.
func newRss(encode, lang, title, parent, link, uri, description, xsl string) *RSS {
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
		Feeds:       make(map[string]*Item),
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

//append adds RSS an item.
func (r *RSS) append(link, title, creator, description, content string, subject []string, date int64, abs bool) {
	if abs {
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

	r.Feeds[link] = i
}

//keys returns keys of feeds i.e. link .
func (r *RSS) keys() []string {
	items := make([]string, len(r.Feeds))
	i := 0
	for k := range r.Feeds {
		items[i] = k
		i++
	}
	sort.Strings(items)
	return items
}

//makeRSS renders template.
func (r *RSS) makeRSS1(wr io.Writer) {
	for _, c := range r.Feeds {
		c.Content = strings.Replace(c.content, "]]", "&#93;&#93;>", -1)
	}
	renderTemplate("rss1", *r, wr)
}

//W3cdate returns RSS formated date string.
func (r *RSS) W3cdate(dat int64) string {
	t := time.Unix(dat, 0)
	return t.Format("2006-01-02T15:04:05Z")
}
