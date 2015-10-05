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
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

func gatewaySetup(s *http.ServeMux) {
	s.Handle("/gateway.cgi/motd", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printMotd()
	})))
	s.Handle("/gateway.cgi/mergedjs", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printMergedJS()
	})))
	s.Handle("/gateway.cgi/rss", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printRSS()
	})))
	s.Handle("/gateway.cgi/recent_rss", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printRecentRSS()
	})))
	s.Handle("/gateway.cgi/index", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printIndex()
	})))
	s.Handle("/gateway.cgi/changes", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printChanges()
	})))
	s.Handle("/gateway.cgi/recent", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printRecent()
	})))
	s.Handle("/gateway.cgi/new", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printNew()
	})))
	s.Handle("/gateway.cgi/thread", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printThread()
	})))
	s.Handle("/gateway.cgi/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printTitle()
	})))
	s.Handle("/gateway.cgi/csv/index/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printCSVIndex()
	})))
	s.Handle("/gateway.cgi/csv/changes/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printCSVChanges()
	})))
	s.Handle("/gateway.cgi/csv/recent/", handlers.CompressHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := newGatewayCGI(w, r)
		if a == nil {
			return
		}
		a.printCSVRecent()
	})))
}

type gatewayCGI struct {
	*cgi
}

func newGatewayCGI(w http.ResponseWriter, r *http.Request) *gatewayCGI {
	c := newCGI(w, r)
	r.ParseForm()

	filter := r.FormValue("filter")
	tag := r.FormValue("tag")

	if filter != "" {
		c.filter = strings.ToLower(filter)
		c.strFilter = cgiEscape(filter, true)
	} else {
		c.tag = strings.ToLower(tag)
		c.strTag = cgiEscape(tag, true)
	}
	c.host = server_name
	if c.host == "" {
		c.host = r.Host
	}
	if !c.checkVisitor() {
		c.print403("")
		return nil
	}
	return &gatewayCGI{
		c,
	}
}

func (g *gatewayCGI) renderCSV(cl *cacheList) {
	g.wr.Header().Set("Content-Type", "text/comma-separated-values;charset=UTF-8")
	p := strings.Split(g.req.URL.Path, "/")
	if len(p) < 3 {
		g.print404(nil, "")
		return
	}
	cols := strings.Split(p[2], ",")
	cwr := csv.NewWriter(g.wr)
	for _, ca := range cl.caches {
		title, _ := fileDecode(ca.datfile)
		var t, p string
		if hasString(stringSlice(types), ca.typee) {
			t = ca.typee
			p = application[t] + query_separator + strEncode(title)
		}
		row := make([]string, len(cols))
		for i, c := range cols {
			switch c {
			case "file":
				row[i] = ca.datfile
			case "stamp":
				row[i] = strconv.FormatInt(ca.validStamp, 10)
			case "date":
				row[i] = time.Unix(ca.validStamp, 0).String()
			case "path":
				row[i] = p
			case "uri":
				if g.host != "" && p != "" {
					row[i] = "http://" + g.host + p
				} else {
					row[i] = ""
				}
			case "type":
				row[i] = ca.typee
			case "title":
				row[i] = title
			case "records":
				row[i] = strconv.Itoa(ca.Len())
			case "size":
				row[i] = strconv.Itoa(ca.size)
			case "tag":
				row[i] = ca.tags.string()
			case "sugtag":
				row[i] = ca.sugtags.string()
			default:
			}
		}
		cwr.Write(row)
	}
	cwr.Flush()
}

func (g *gatewayCGI) printThread() {
	reg := regexp.MustCompile("^/(thread)/?([^/]*)$")
	m := reg.FindStringSubmatch(g.req.URL.Path)
	var uri string
	switch {
	case m == nil:
		g.printTitle()
		return
	case m[2] != "":
		uri = application["thread"] + query_separator + strEncode(m[2])
	case g.req.URL.RawQuery != "":
		uri = application["thread"] + query_separator + g.req.URL.RawQuery
	default:
		g.printTitle()
		return
	}
	g.print302(uri)
}

func (g *gatewayCGI) printCSVIndex() {
	cl := newCacheList()
	g.renderCSV(cl)
}

func (g *gatewayCGI) printCSVChanges() {
	cl := newCacheList()
	cl.sort("validStamp", true)
	g.renderCSV(cl)
}
func (g *gatewayCGI) printCSVRecent() {
	if !g.isFriend && !g.isAdmin {
		g.print403("")
		return
	}
	cl := g.makeRecentCachelist()
	g.renderCSV(cl)
}
func (g *gatewayCGI) printRecentRSS() {
	rsss := newRss("UTF-8", "", fmt.Sprintf("%s - %s", g.m["recent"], g.m["logo"]),
		"http://"+g.host, "",
		"http://"+g.host+gateway_cgi+query_separator+"recent_rss", g.m["description"], xsl)
	cl := g.makeRecentCachelist()
	for _, ca := range cl.caches {
		f, _ := fileDecode(ca.datfile)
		title := escape(f)
		tags := make([]string, ca.tags.Len()+ca.sugtags.Len())
		for i, t := range ca.tags.tags {
			tags[i] = t.tagstr
		}
		for i, t := range ca.sugtags.tags {
			tags[i+ca.tags.Len()] = t.tagstr
		}
		if _, exist := application[ca.typee]; !exist {
			continue
		}
		rsss.append(application[ca.typee][1:]+query_separator+strEncode(title),
			title, "", "", cgiEscape(title, false), tags, ca.recentStamp, false)
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	k := rsss.keys()
	if len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.items[k[0]].date))
	}
	rsss.makeRSS1(g.wr)

}

func (g *gatewayCGI) printRSS() {
	now := time.Now().Unix()
	rsss := newRss("UTF-8", "", g.m["logo"], "http://"+g.host, "",
		"http://"+g.host+gateway_cgi+query_separator+"rss", g.m["description"], xsl)
	cl := newCacheList()
	for _, ca := range cl.caches {
		if ca.validStamp+int64(rss_range) >= now {
			f, _ := fileDecode(ca.datfile)
			title := escape(f)
			path := application[ca.typee] + query_separator + strEncode(title)
			for _, r := range ca.recs {
				if r.stamp+int64(rss_range) < now {
					continue
				}
				r.loadBody()
				desc := g.rssTextFormat(r.Get("body", ""))
				content := g.rssHtmlFormat(r.Get("body", ""), application[ca.typee], title)
				if attach := r.Get("attach", ""); attach != "" {
					suffix := r.Get("suffix", "")
					if reg := regexp.MustCompile("^[0-9A-Za-z]+$"); reg.MatchString(suffix) {
						suffix = "txt"
					}
					content += fmt.Sprintf("\n    <p><a href=\"http://%s%s%s%s/%s/%d.%s\">%d.%s</a></p>",
						g.host, application[ca.typee], query_separator, ca.datfile, r.id, r.stamp, suffix, r.stamp, suffix)
				}
				permpath := path[1:]
				if ca.typee == "thread" {
					permpath = fmt.Sprintf("%s/%s", path[1:], r.id[:8])
				}
				rsss.append(permpath, title, g.rssTextFormat(r.Get("name", "")), desc, content, tagSliceTostringSlice(ca.tags.tags), r.stamp, false)
				r.free()
			}
		}
	}
	g.wr.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	k := rsss.keys()
	if len(k) != 0 {
		g.wr.Header().Set("Last-Modified", g.rfc822Time(rsss.items[k[0]].date))
	}
	rsss.makeRSS1(g.wr)

}

func (g *gatewayCGI) printMergedJS() {
	g.wr.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
	g.wr.Header().Set("Last-Modified", g.rfc822Time(g.jc.getLatest()))
	g.wr.Write([]byte(g.jc.getContent()))
}
func (g *gatewayCGI) printMotd() {
	g.wr.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	c, err := ioutil.ReadFile(motd)
	if err != nil {
		log.Println(err)
		return
	}
	g.wr.Write(c)

}
func (g *gatewayCGI) printNew() {
	g.header(g.m["new"], "", nil, true, nil)
	g.printNewElementForm()
	g.footer(nil)
}
func (g *gatewayCGI) printTitle() {
	cl := newCacheList()
	cl.sort("validStamp", true)
	outputCachelist := make([]*cache, 0, cl.Len())
	for _, ca := range cl.caches {
		if time.Now().Unix() <= ca.validStamp+int64(top_recent_range) {
			outputCachelist = append(outputCachelist, ca)
		}
	}
	g.header(g.m["logo"]+" - "+g.m["description"], "", nil, false, nil)
	s := struct {
		Cachelist     []*cache
		Target        string
		Taglist       *userTagList
		mchUrl        string
		mchCategories []*mchCategory
	}{
		outputCachelist,
		"changes",
		newUserTagList(),
		g.mchUrl(),
		g.mchCategories(),
	}
	renderTemplate("top", s, g.wr)
	g.printNewElementForm()
	g.footer(nil)
}

func (g *gatewayCGI) printIndex() {
	title := g.m["index"]
	if g.strFilter != "" {
		title = fmt.Sprintf("%s : %s", g.m["index"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_index"])
	cl := newCacheList()
	cl.sort("velocity_count", true)
	g.printIndexList(cl, "index", false, false)
}

func (g *gatewayCGI) printChanges() {
	title := g.m["changes"]
	if g.strFilter != "" {
		title = fmt.Sprintf("%s : %s", g.m["changes"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_changes"])
	cl := newCacheList()
	cl.sort("validStamp", true)
	g.printIndexList(cl, "changes", false, false)
}

func (g *gatewayCGI) makeRecentCachelist() *cacheList {
	rl := newRecentList()
	sort.Sort(sort.Reverse(rl))
	cl := make([]*cache, 0)
	check := make([]string, 0)
	for _, rec := range rl.tiedlist {
		if !hasString(stringSlice(check), rec.datfile) {
			ca := newCache(rec.datfile, nil, nil)
			ca.recentStamp = rec.stamp
			cl = append(cl, ca)
			check = append(check, rec.datfile)
		}
	}
	return &cacheList{caches: cl}
}

func (g *gatewayCGI) printRecent() {
	title := g.m["recent"]
	if g.strFilter != "" {
		title = fmt.Sprintf("%s : %s", g.m["recent"], g.filter)
	}
	g.header(title, "", nil, true, nil)
	g.printParagraph(g.m["desc_recent"])
	cl := g.makeRecentCachelist()
	g.printIndexList(cl, "recent", false, false)
}

func (g *gatewayCGI) jumpNewFile() {
	link := g.req.FormValue("link")
	t := g.req.FormValue("type")
	switch {
	case link == "":
		g.header(g.m["null_title"], "", nil, true, nil)
		g.footer(nil)
	case strings.ContainsAny(link, "/[]<>"):
		g.header(g.m["bad_title"], "", nil, true, nil)
		g.footer(nil)
	case t == "":
		g.header(g.m["null_type"], "", nil, true, nil)
		g.footer(nil)
	case hasString(stringSlice(types), t):
		tag := strEncode(g.req.FormValue("tag"))
		search := strEncode(g.req.FormValue("search_new_file"))
		g.print302(application[t] + query_separator + strEncode(link) + "?tag=" + tag + "&search_new_filter" + search)
	default:
		g.print404(nil, "")
	}
}
func (g *gatewayCGI) rssTextFormat(plain string) string {
	buf := strings.Replace(plain, "<br>", " ", -1)
	buf = strings.Replace(buf, "&", "&amp;", -1)
	reg := regexp.MustCompile("&amp;(#\\d+|lt|gt|amp);")
	buf = reg.ReplaceAllString(buf, "&\\1")
	buf = strings.Replace(buf, "<", "&lt;", -1)
	buf = strings.Replace(buf, ">", "&gt;", -1)
	buf = strings.Replace(buf, "\r", "", -1)
	buf = strings.Replace(buf, "\n", "", -1)
	return buf
}

func (g *gatewayCGI) rssHtmlFormat(plain, appli, path string) string {
	title := strDecode(path)
	buf := g.htmlFormat(plain, appli, title, true)
	if buf != "" {
		buf = fmt.Sprintf("<p>%s</p>", buf)
	}
	return buf
}
