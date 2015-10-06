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
	"html"
	"io"
	"sort"
	"strings"
	"time"
)

type item struct {
	title       string
	link        string
	description string
	creator     string
	subject     []string
	date        int64
	content     string
}

func newItem(link, title, creator, description, content string, subject []string, date int64) *item {
	i := &item{
		link:    link,
		creator: creator,
		date:    date,
		content: content,
	}
	if subject == nil {
		i.subject = make([]string, 0)
	}
	r := strings.NewReplacer("\r", "", "\n", "")
	i.title = r.Replace(title)
	i.description = r.Replace(description)

	return i
}

type rss struct {
	encode      string
	lang        string
	title       string
	parent      string
	link        string
	uri         string
	description string
	xsl         string
	items       map[string]*item
}

func newRss(encode, lang, title, parent, link, uri, description, xsl string) *rss {
	r := &rss{
		encode:      encode,
		lang:        lang,
		title:       title,
		description: description,
		parent:      parent,
		xsl:         xsl,
		link:        link,
		uri:         uri,
		items:       make(map[string]*item),
	}
	if parent != "" && parent[len(parent)-1] != '/' {
		r.parent += "/"
	}
	if link == "" {
		r.link = parent
	}
	if uri == "" {
		r.uri = parent + "rss.xml"
	}
	return r
}

func (r *rss) append(link, title, creator, description, content string, subject []string, date int64, abs bool) {
	if abs {
		link = r.parent + link
	}
	i := newItem(link, title, creator, description, content, subject, date)
	r.items[link] = i
}
func (r *rss) keys() []string {
	items := make([]string, len(r.items))
	i := 0
	for k := range r.items {
		items[i] = k
		i++
	}
	sort.Strings(items)
	return items
}

func (r *rss) makeRSS1(wr io.Writer) {
	items := make([]*item, len(r.items))
	i := 0
	for _, v := range r.items {
		items[i] = v
		i++
	}

	param := rssParam{r, items}
	renderTemplate("rss1", param, wr)
}

type rssParam struct {
	Rss  *rss
	Feed []*item
}

func (r *rssParam) W3cdate(dat int64) string {
	t := time.Unix(dat, 0)
	return t.Format("2006-01-02T15:04:05Z")
}
func (r *rssParam) Escape(str string) string {
	return html.EscapeString(str)
}
